// +build darwin,!kqueue
// +build !fsnotify

package notify

// #include <CoreServices/CoreServices.h>
// void gocallback(void*, void*, size_t, uintptr_t, uintptr_t, uintptr_t);
// #cgo LDFLAGS: -framework CoreServices
import "C"

import (
	"errors"
	"os"
	"runtime"
	"sync/atomic"
	"unsafe"
)

var (
	nilrl   C.CFRunLoopRef
	nilref  C.FSEventStreamRef
	latency C.CFTimeInterval           = 3
	flags   C.FSEventStreamCreateFlags = C.kFSEventStreamCreateFlagFileEvents
	now     C.FSEventStreamEventId     = 1<<64 - 1 // C constant (-1) overflows C.FSEventStreamEventId
)

//export gocallback
func gocallback(_, ctx unsafe.Pointer, n C.size_t, paths, flags, ids uintptr) {
	if n == 0 {
		return
	}
	ev := make([]FSEvent, int(n))
	for i := range ev {
		ev[i].Path = C.GoString(*(**C.char)(unsafe.Pointer(paths + uintptr(i))))
		ev[i].Flags = *(*uint32)(unsafe.Pointer((flags + uintptr(i))))
		ev[i].ID = *(*uint64)(unsafe.Pointer(ids + uintptr(i)))
	}
	(*(*StreamFunc)(ctx))(ev)
}

type runloop struct {
	n   int32
	ref C.CFRunLoopRef
}

func (ll *runloop) schedule(ref C.FSEventStreamRef) {
	if atomic.AddInt32(&ll.n, 1) == 1 {
		runtime.LockOSThread()
		ll.ref = C.CFRunLoopGetCurrent()
		go C.CFRunLoopRun()
		runtime.UnlockOSThread()
	}
	C.FSEventStreamScheduleWithRunLoop(ref, ll.ref, C.kCFRunLoopDefaultMode)
	C.CFRunLoopWakeUp(ll.ref)
}

func (ll *runloop) unschedule(ref C.FSEventStreamRef) {
	C.FSEventStreamUnscheduleFromRunLoop(ref, ll.ref, C.kCFRunLoopDefaultMode)
	if atomic.AddInt32(&ll.n, -1) == 0 {
		C.CFRunLoopStop(ll.ref)
		ll.ref = nilrl
	}
}

var loop runloop

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
	p := C.CFStringCreateWithCStringNoCopy(nil, C.CString(s.path), C.kCFStringEncodingUTF8, nil)
	path := C.CFArrayCreate(nil, (*unsafe.Pointer)(unsafe.Pointer(&p)), 1, nil)
	s.ctx.info = unsafe.Pointer(&s.fn)
	ref := C.FSEventStreamCreate(nil, (C.FSEventStreamCallback)(C.gocallback),
		&s.ctx, path, now, latency, flags)
	if ref == nilref {
		return errCreate
	}
	loop.schedule(ref)
	if C.FSEventStreamStart(ref) == C.Boolean(0) {
		return errStart
	}
	s.ref = ref
	return nil
}

// Stop TODO
func (s *Stream) Stop() {
	if s.ref == nilref {
		return
	}
	C.FSEventStreamFlushSync(s.ref)
	loop.unschedule(s.ref)
	C.FSEventStreamStop(s.ref)
	s.ref = nilref
}
