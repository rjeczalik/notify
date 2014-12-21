// +build darwin,!kqueue
// +build !fsnotify

package notify

// #include <CoreServices/CoreServices.h>
// typedef void (*CFRunLoopPerformCallBack)(void*);
// void gosource(void *);
// void gostream(void*, void*, size_t, uintptr_t, uintptr_t, uintptr_t);
// #cgo LDFLAGS: -framework CoreServices
import "C"

import (
	"errors"
	"os"
	"sync"
	"unsafe"
)

// TODO
var (
	nilrl  C.CFRunLoopRef
	nilref C.FSEventStreamRef
)

// TODO
var (
	latency C.CFTimeInterval           = 1
	flags   C.FSEventStreamCreateFlags = C.kFSEventStreamCreateFlagFileEvents
	now     C.FSEventStreamEventId     = 1<<64 - 1
)

// TODO
var wg sync.WaitGroup

// TODO
var runloop C.CFRunLoopRef

// TODO
var source = C.CFRunLoopSourceCreate(nil, 0, &C.CFRunLoopSourceContext{
	perform: (C.CFRunLoopPerformCallBack)(C.gosource),
})

// TODO
func init() {
	wg.Add(1)
	go func() {
		runloop = C.CFRunLoopGetCurrent()
		C.CFRunLoopAddSource(runloop, source, C.kCFRunLoopDefaultMode)
		C.CFRunLoopRun()
		panic("runloop has stopped")
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
	for i := range ev {
		ev[i].Path = C.GoString(*(**C.char)(unsafe.Pointer(paths + offchar*uintptr(i))))
		ev[i].Flags = *(*uint32)(unsafe.Pointer((flags + offflag*uintptr(i))))
		ev[i].ID = *(*uint64)(unsafe.Pointer(ids + offid*uintptr(i)))
	}
	(*(*StreamFunc)(ctx))(ev)
}

// FSEvent TODO
type FSEvent struct {
	Path  string
	ID    uint64
	Flags uint32
}

// StreamFunc TODO
type StreamFunc func([]FSEvent)

// Stream TODO
type Stream struct {
	path string
	fn   StreamFunc
	ref  C.FSEventStreamRef
	ctx  C.FSEventStreamContext
}

// NewStream TODO
func NewStream(path string, fn StreamFunc) *Stream {
	return &Stream{
		path: path,
		fn:   fn,
	}
}

func (s *Stream) String() string {
	return s.path
}

var errCreate = os.NewSyscallError("FSEventStreamCreate", errors.New("NULL"))
var errStart = os.NewSyscallError("FSEventStreamStart", errors.New("false"))

// Start TODO
func (s *Stream) Start() error {
	if s.ref != nilref {
		return nil
	}
	wg.Wait()
	p := C.CFStringCreateWithCStringNoCopy(nil, C.CString(s.path), C.kCFStringEncodingUTF8, nil)
	path := C.CFArrayCreate(nil, (*unsafe.Pointer)(unsafe.Pointer(&p)), 1, nil)
	s.ctx.info = unsafe.Pointer(&s.fn)
	ref := C.FSEventStreamCreate(nil, (C.FSEventStreamCallback)(C.gostream),
		&s.ctx, path, now, latency, flags)
	if ref == nilref {
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

// Stop TODO
func (s *Stream) Stop() {
	// BUG(rjeczalik): Stop gets called twice from TestWatcherBasic:
	//
	//   2014-12-23 17:09 notify.test[4764] (FSEvents.framework) FSEventStream
	//   Invalidate(): failed assertion 'streamRef != NULL'
	//
	// Check out why and fix.
	if s.ref == nilref {
		return
	}
	wg.Wait()
	// TODO(rjeczalik): Do we care about unflushed events? Stop means probably no.
	// The drawback is enabling flush would require fixing the following failures
	// during TestWatcherBasic test:
	//
	//   2014-12-23 13:39 notify.test[5888] (FSEvents.framework) FSEventStreamFlushAsync:
	//   ERROR: f2d_flush_rpc() => (ipc/send) invalid destination port (268435459)
	//
	//   2014-12-23 13:39 notify.test[5888] (FSEvents.framework) FSEventStreamUnschedule
	//   FromRunLoop(): failed assertion 'streamRef != NULL'
	//
	//   2014-12-23 13:39 notify.test[5888] (FSEvents.framework) FSEventStreamStop():
	//   failed assertion 'streamRef != NULL
	//
	// Wtf, stop is legit - starts were successful.
	//
	// C.FSEventStreamFlushAsync(s.ref)
	// C.FSEventStreamUnscheduleFromRunLoop(s.ref, runloop, C.kCFRunLoopDefaultMode)
	// C.CFRunLoopWakeUp(runloop)
	// C.FSEventStreamStop(s.ref)
	C.FSEventStreamStop(s.ref)
	C.FSEventStreamInvalidate(s.ref)
	s.ref = nilref
}
