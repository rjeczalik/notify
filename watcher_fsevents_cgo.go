// +build darwin,!kqueue

package notify

// #include <CoreServices/CoreServices.h>
//
// typedef void (*CFRunLoopPerformCallBack)(void*);
//
// void gosource(void *);
// void gostream(void*, void*, size_t, uintptr_t, uintptr_t, uintptr_t);
//
// #cgo LDFLAGS: -framework CoreServices
import "C"

import (
	"errors"
	"os"
	"sync"
	"unsafe"
)

var nilstream C.FSEventStreamRef

// Default arguments for FSEventStreamCreate function. Make them configurable?
var (
	latency C.CFTimeInterval           = 1
	flags   C.FSEventStreamCreateFlags = C.kFSEventStreamCreateFlagFileEvents
	now     C.FSEventStreamEventId     = 1<<64 - 1
)

var runloop C.CFRunLoopRef // global runloop which all streams are registered with
var wg sync.WaitGroup      // used to wait until the runloop starts

// source is used for synchronization purposes - it signals when runloop has
// started and is ready via the wg. It also serves purpose of a dummy source,
// thanks to it the runloop does not return as it also has at least one source
// registered.
var source = C.CFRunLoopSourceCreate(nil, 0, &C.CFRunLoopSourceContext{
	perform: (C.CFRunLoopPerformCallBack)(C.gosource),
})

// Errors returned when FSEvents functions fail.
var (
	errCreate = os.NewSyscallError("FSEventStreamCreate", errors.New("NULL"))
	errStart  = os.NewSyscallError("FSEventStreamStart", errors.New("false"))
)

// initializes the global runloop and ensures any created stream awaits its
// readiness.
func init() {
	wg.Add(1)
	go func() {
		runloop = C.CFRunLoopGetCurrent()
		C.CFRunLoopAddSource(runloop, source, C.kCFRunLoopDefaultMode)
		C.CFRunLoopRun()
		panic("runloop has just unexpectedly stopped")
	}()
	C.CFRunLoopSourceSignal(source)
}

//export gosource
func gosource(unsafe.Pointer) {
	wg.Done()
}

//export gostream
func gostream(_, ctx unsafe.Pointer, n C.size_t, paths, flags, ids uintptr) {
	const (
		offchar = unsafe.Sizeof((*C.char)(nil))
		offflag = unsafe.Sizeof(C.FSEventStreamEventFlags(0))
		offid   = unsafe.Sizeof(C.FSEventStreamEventId(0))
	)
	if n == 0 {
		return
	}
	ev := make([]FSEvent, int(n))
	for i := uintptr(0); i < uintptr(n); i++ {
		ev[i].Path = C.GoString(*(**C.char)(unsafe.Pointer(paths + i*offchar)))
		ev[i].Flags = *(*uint32)(unsafe.Pointer((flags + i*offflag)))
		ev[i].ID = *(*uint64)(unsafe.Pointer(ids + i*offid))
	}
	(*(*StreamFunc)(ctx))(ev)
}

// StreamFunc is a callback called when stream receives file events.
type StreamFunc func([]FSEvent)

// Stream represents single watch-point which listens for events scheduled by
// the global runloop.
type Stream struct {
	path string
	ref  C.FSEventStreamRef
	ctx  C.FSEventStreamContext
}

// NewStream creates a stream for given path, listening for file events and
// calling fn upon receving any.
func NewStream(path string, fn StreamFunc) *Stream {
	return &Stream{
		path: path,
		ctx: C.FSEventStreamContext{
			info: unsafe.Pointer(&fn),
		},
	}
}

// Start creates a FSEventStream for the given path and schedules it with
// global runloop. It's a nop if the stream was already started.
func (s *Stream) Start() error {
	if s.ref != nilstream {
		return nil
	}
	wg.Wait()
	p := C.CFStringCreateWithCStringNoCopy(nil, C.CString(s.path), C.kCFStringEncodingUTF8, nil)
	path := C.CFArrayCreate(nil, (*unsafe.Pointer)(unsafe.Pointer(&p)), 1, nil)
	// TODO(rjeczalik): kFSEventStreamCreateFlagWatchRoot + update canonical(s.path)?
	ref := C.FSEventStreamCreate(nil, (C.FSEventStreamCallback)(C.gostream),
		&s.ctx, path, now, latency, flags)
	if ref == nilstream {
		return errCreate
	}
	C.FSEventStreamScheduleWithRunLoop(ref, runloop, C.kCFRunLoopDefaultMode)
	if C.FSEventStreamStart(ref) == C.Boolean(0) {
		C.FSEventStreamInvalidate(ref)
		return errStart
	}
	C.CFRunLoopWakeUp(runloop)
	s.ref = ref
	return nil
}

// Stop stops underlying FSEventStream and unregisters it from global runloop.
func (s *Stream) Stop() {
	if s.ref == nilstream {
		return
	}
	wg.Wait()
	C.FSEventStreamStop(s.ref)
	C.FSEventStreamInvalidate(s.ref)
	C.CFRunLoopWakeUp(runloop)
	s.ref = nilstream
}
