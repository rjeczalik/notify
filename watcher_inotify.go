// +build linux,!fsnotify

package notify

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	rntm "runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"unsafe"
)

// TODO(ppknap) : doc.
const maxEventSize = syscall.SizeofInotifyEvent + syscall.PathMax + 1

// TODO(ppknap) : This type should be dropped. It is useless since we use it only
// here. Use:
//
//   var handlers struct {
//     ...
//   }
type handlersType struct {
	sync.RWMutex
	m      map[int32]*watched
	fd     *int
	buffer [64 * maxEventSize]byte
	// TODO(ppknap) : I don't like this:<.
	c chan EventInfo
}

// TODO(ppknap) : doc.
var handlers *handlersType

// TODO(ppknap) : doc.
type watched struct {
	path string
	mask uint32
}

// TODO(ppknap) : doc.
func init() {
	handlers = newInotify()
	runtime = NewRuntime(handlers)
}

// NewInotify TODO
func newInotify() *handlersType {
	fd, err := syscall.InotifyInit()
	if err != nil {
		panic(os.NewSyscallError("InotifyInit", err))
	}
	h := &handlersType{
		m:  make(map[int32]*watched),
		fd: &fd,
		c:  make(chan EventInfo),
	}
	rntm.SetFinalizer(h, func(h *handlersType) { syscall.Close(*h.fd) })
	go loop()
	return h
}

// TODO(ppknap) : doc.
func loop() {
	for {
		process()
	}
}

// TODO(ppknap) : doc.
func process() {
	n, err := syscall.Read(*handlers.fd, handlers.buffer[:])
	switch {
	//TODO(pknap) : improve error handling + doc.
	case err != nil || n < 0:
		// TODO(rjeczalik): Panic, error?
		fmt.Println(os.NewSyscallError("Read", err))
	case n < syscall.SizeofInotifyEvent:
		return
	}

	events := make([]*event, 0)
	nmin := n - syscall.SizeofInotifyEvent

	var sys *syscall.InotifyEvent
	for pos, name := 0, ""; pos <= nmin; {
		sys = (*syscall.InotifyEvent)(unsafe.Pointer(&handlers.buffer[pos]))
		pos += syscall.SizeofInotifyEvent

		if name = ""; sys.Len > 0 {
			endpos := pos + int(sys.Len)
			name = string(bytes.TrimRight(handlers.buffer[pos:endpos], "\x00"))
			pos = endpos
		}

		if pos > n {
			fmt.Println("TODO queue overflow")
		}

		events = append(events, &event{sys: syscall.InotifyEvent{
			Wd:     sys.Wd,
			Mask:   sys.Mask,
			Cookie: sys.Cookie,
		}, name: name})
	}
	send(events)
}

// TODO(ppknap) : doc.
func watch(path string, event Event) error {
	wd, err := syscall.InotifyAddWatch(*handlers.fd, path, uint32(event))
	if err != nil {
		return os.NewSyscallError("InotifyAddWatch", err)
	}

	handlers.RLock()
	w := handlers.m[int32(wd)]
	handlers.RUnlock()

	if w == nil {
		w = &watched{path: path, mask: uint32(event)}
		handlers.Lock()
		handlers.m[int32(wd)] = w
		handlers.Unlock()
	} else {
		atomic.StoreUint32(&w.mask, uint32(event))
	}
	return nil
}

// TODO(ppknap) : doc.
func unwatch(path string) error {
	wd := int32(-1)
	handlers.RLock()
	for wdkey, w := range handlers.m {
		if w.path == path {
			wd = wdkey
			break
		}
	}
	handlers.RUnlock()

	if wd < 0 {
		return errors.New("path " + path + " is unwatched")
	}
	// BUG(goauthors) : watch descriptor is of type `int`, not `uint32`
	if _, err := syscall.InotifyRmWatch(*handlers.fd, uint32(wd)); err != nil {
		return os.NewSyscallError("InotifyRmWatch", err)
	}

	handlers.Lock()
	delete(handlers.m, wd)
	handlers.Unlock()
	return nil
}

// TODO(ppknap) : doc
func send(events []*event) {
	handlers.RLock()
	for i, event := range events {
		if w, ok := handlers.m[event.sys.Wd]; ok {
			if event.name == "" {
				event.name = w.path
			} else {
				event.name = filepath.Join(w.path, event.name)
			}
		} else {
			events[i] = nil
		}
	}
	handlers.RUnlock()

	for _, event := range events {
		if event != nil {
			sendevent(event)
		}
	}
}

func sendevent(ei EventInfo) {
	handlers.c <- ei
}

// TODO(ppknap) : doc.
type event struct {
	sys  syscall.InotifyEvent
	name string
}

func (e *event) Event() Event     { return maskevent(e.sys.Mask) }
func (e *event) IsDir() bool      { return e.sys.Mask&syscall.IN_ISDIR != 0 }
func (e *event) Name() string     { return e.name }
func (e *event) Sys() interface{} { return e.sys } //

// TODO(ppknap) : impl/doc.
func maskevent(mask uint32) Event {
	return Event(mask & syscall.IN_ALL_EVENTS)
}

// Watch implements notify.Watcher interface.
func (h *handlersType) Watch(p string, e Event) error {
	return watch(p, e)
}

// Unwatch implements notify.Watcher interface.
func (h *handlersType) Unwatch(p string) error {
	return unwatch(p)
}

// Fanin implements notify.Watcher interface.
func (h *handlersType) Fanin(c chan<- EventInfo) {
	go func() {
		for e := range h.c {
			c <- e
		}
	}()
}
