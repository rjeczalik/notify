// +build linux,!fsnotify

package notify

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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
	name string
	mask uint32
}

// TODO(ppknap) : doc.
func init() {
	handlers = newInotify()
	notify = NewRuntime(handlers)
	go loop()
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
	runtime.SetFinalizer(h, func(h *handlersType) { syscall.Close(*h.fd) })
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
		}, impl: watched{name: name}})
	}
	send(events)
}

// TODO(ppknap) : doc.
func watch(name string, event Event) error {
	if event&invalid != 0 {
		return errors.New("invalid event")
	}
	wd, err := syscall.InotifyAddWatch(*handlers.fd, name, makemask(event))
	if err != nil {
		return os.NewSyscallError("InotifyAddWatch", err)
	}

	handlers.RLock()
	w := handlers.m[int32(wd)]
	handlers.RUnlock()

	if w == nil {
		w = &watched{name: name, mask: uint32(event)}
		handlers.Lock()
		handlers.m[int32(wd)] = w
		handlers.Unlock()
	} else {
		atomic.StoreUint32(&w.mask, uint32(event))
	}
	return nil
}

// TODO(ppknap) : doc.
func unwatch(name string) error {
	wd := int32(-1)
	handlers.RLock()
	for wdkey, w := range handlers.m {
		if w.name == name {
			wd = wdkey
			break
		}
	}
	handlers.RUnlock()

	if wd < 0 {
		return errors.New("file/directory " + name + " is unwatched")
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
			if event.impl.name == "" {
				event.impl.name = w.name
			} else {
				event.impl.name = filepath.Join(w.name, event.impl.name)
			}
			if event.sys.Mask&makemask(Event(w.mask)) != 0 {
				event.impl.mask = w.mask
			} else {
				events[i] = nil
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
	impl watched
}

func (e *event) Event() Event     { return decodemask(e.impl.mask, e.sys.Mask) }
func (e *event) IsDir() bool      { return e.sys.Mask&syscall.IN_ISDIR != 0 }
func (e *event) Name() string     { return e.impl.name }
func (e *event) Sys() interface{} { return e.sys }

// TODO(ppknap) : doc.
func makemask(e Event) uint32 {
	if e&Create != 0 {
		e = (e ^ Create) | IN_CREATE | IN_MOVED_TO
	}
	if e&Delete != 0 {
		e = (e ^ Delete) | IN_DELETE | IN_DELETE_SELF
	}
	if e&Write != 0 {
		e = (e ^ Write) | IN_MODIFY
	}
	if e&Move != 0 {
		e = (e ^ Move) | IN_MOVED_FROM | IN_MOVE_SELF
	}
	return uint32(e)
}

// TODO(ppknap) : doc.
func decodemask(mask, syse uint32) Event {
	imask := makemask(Event(mask))
	switch {
	case mask&syse != 0:
		return Event(syse)
	case imask&makemask(Create)&syse != 0:
		return Create
	case imask&makemask(Delete)&syse != 0:
		return Delete
	case imask&makemask(Write)&syse != 0:
		return Write
	case imask&makemask(Move)&syse != 0:
		return Move
	}
	panic("notify: cannot decode internal mask")
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
