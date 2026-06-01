// Copyright (c) 2014-2015 The Notify Authors. All rights reserved.
// Use of this source code is governed by the MIT license that can be
// found in the LICENSE file.

//go:build darwin && !kqueue && cgo
// +build darwin,!kqueue,cgo

package notify

/*
#include <CoreServices/CoreServices.h>
#include <dispatch/dispatch.h>
#include <stdlib.h>
#include <stdint.h>

void gostream(uintptr_t, uintptr_t, size_t, uintptr_t, uintptr_t, uintptr_t);

static dispatch_queue_t EventDispatchQueue(void) {
	return dispatch_queue_create("com.github.rjeczalik.notify", DISPATCH_QUEUE_SERIAL);
}

static FSEventStreamRef EventStreamCreate(FSEventStreamContext * context, uintptr_t info, CFArrayRef paths, FSEventStreamEventId since, CFTimeInterval latency, FSEventStreamCreateFlags flags) {
	context->info = (void*) info;
	return FSEventStreamCreate(NULL, (FSEventStreamCallback) gostream, context, paths, since, latency, flags);
}

static CFArrayRef PathArrayCreate(CFStringRef path) {
	const void *paths[] = { path };
	return CFArrayCreate(kCFAllocatorDefault, paths, 1, &kCFTypeArrayCallBacks);
}

static const char* EventPathAt(uintptr_t paths, size_t i) {
	return ((char**)paths)[i];
}

static FSEventStreamEventFlags EventFlagsAt(uintptr_t flags, size_t i) {
	return ((FSEventStreamEventFlags*)flags)[i];
}

static FSEventStreamEventId EventIDAt(uintptr_t ids, size_t i) {
	return ((FSEventStreamEventId*)ids)[i];
}

#cgo LDFLAGS: -framework CoreServices
*/
import "C"

import (
	"errors"
	"os"
	"sync"
	"sync/atomic"
	"unsafe"
)

var (
	nilarray  C.CFArrayRef
	nilstring C.CFStringRef
	nilstream C.FSEventStreamRef
)

// Default arguments for FSEventStreamCreate function.
var (
	latency C.CFTimeInterval
	flags   = C.FSEventStreamCreateFlags(C.kFSEventStreamCreateFlagFileEvents | C.kFSEventStreamCreateFlagNoDefer)
	since   = uint64(C.FSEventsGetCurrentEventId())
)

// global dispatch queue which all streams are registered with
var q C.dispatch_queue_t = C.EventDispatchQueue()

// Errors returned when FSEvents functions fail.
var (
	errCreate = os.NewSyscallError("FSEventStreamCreate", errors.New("NULL"))
	errStart  = os.NewSyscallError("FSEventStreamStart", errors.New("false"))
)

//export gostream
func gostream(_, info uintptr, n C.size_t, paths, flags, ids uintptr) {
	if n == 0 {
		return
	}
	fn := streamFuncs.get(info)
	if fn == nil {
		return
	}
	ev := make([]FSEvent, 0, int(n))
	for i := uintptr(0); i < uintptr(n); i++ {
		switch flags := uint32(C.EventFlagsAt(C.uintptr_t(flags), C.size_t(i))); {
		case flags&uint32(FSEventsEventIdsWrapped) != 0:
			atomic.StoreUint64(&since, uint64(C.FSEventsGetCurrentEventId()))
		default:
			ev = append(ev, FSEvent{
				Path:  C.GoString(C.EventPathAt(C.uintptr_t(paths), C.size_t(i))),
				Flags: flags,
				ID:    uint64(C.EventIDAt(C.uintptr_t(ids), C.size_t(i))),
			})
		}

	}
	fn(ev)
}

// StreamFunc is a callback called when stream receives file events.
type streamFunc func([]FSEvent)

var streamFuncs = streamFuncRegistry{m: map[uintptr]streamFunc{}}

type streamFuncRegistry struct {
	mu sync.Mutex
	m  map[uintptr]streamFunc
	i  uintptr
}

func (r *streamFuncRegistry) get(id uintptr) streamFunc {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.m[id]
}

func (r *streamFuncRegistry) add(fn streamFunc) uintptr {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.i++
	r.m[r.i] = fn
	return r.i
}

func (r *streamFuncRegistry) delete(id uintptr) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.m, id)
}

// Stream represents a single watch-point which listens for events scheduled on the global dispatch queue.
type stream struct {
	path string
	ref  C.FSEventStreamRef
	info uintptr
}

// NewStream creates a stream for given path, listening for file events and
// calling fn upon receiving any.
func newStream(path string, fn streamFunc) *stream {
	return &stream{
		path: path,
		info: streamFuncs.add(fn),
	}
}

// Start creates a FSEventStream for the given path and schedules on the global dispatch queue.
// It's a nop if the stream was already started.
func (s *stream) Start() error {
	if s.ref != nilstream {
		return nil
	}
	cpath := C.CString(s.path)
	defer C.free(unsafe.Pointer(cpath))
	p := C.CFStringCreateWithCString(C.kCFAllocatorDefault, cpath, C.kCFStringEncodingUTF8)
	if p == nilstring {
		streamFuncs.delete(s.info)
		return errCreate
	}
	defer C.CFRelease(C.CFTypeRef(p))
	path := C.PathArrayCreate(p)
	if path == nilarray {
		streamFuncs.delete(s.info)
		return errCreate
	}
	defer C.CFRelease(C.CFTypeRef(path))
	ctx := C.FSEventStreamContext{}
	ref := C.EventStreamCreate(&ctx, C.uintptr_t(s.info), path, C.FSEventStreamEventId(atomic.LoadUint64(&since)), latency, flags)
	if ref == nilstream {
		streamFuncs.delete(s.info)
		return errCreate
	}
	C.FSEventStreamSetDispatchQueue(ref, q)
	if C.FSEventStreamStart(ref) == C.Boolean(0) {
		C.FSEventStreamInvalidate(ref)
		streamFuncs.delete(s.info)
		return errStart
	}
	s.ref = ref
	return nil
}

// Stop stops underlying FSEventStream and unregisters it from the global dispatch queue.
func (s *stream) Stop() {
	if s.ref == nilstream {
		return
	}
	C.FSEventStreamStop(s.ref)
	C.FSEventStreamInvalidate(s.ref)
	s.ref = nilstream
	streamFuncs.delete(s.info)
}
