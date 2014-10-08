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
//   var global struct {
//     ...
//   }
type inotify struct {
	sync.RWMutex
	m      map[int32]*watched
	fd     *int
	buffer [64 * maxEventSize]byte
	c      chan<- EventInfo
	wg     sync.WaitGroup
}

// TODO(ppknap) : doc.
var global *inotify

// TODO(ppknap) : doc.
type watched struct {
	name string
	mask uint32
}

// TODO(ppknap) : doc.
func init() {
	global = newInotify()
}

// NewInotify TODO
func newInotify() *inotify {
	fd, err := syscall.InotifyInit()
	if err != nil {
		panic(os.NewSyscallError("InotifyInit", err))
	}
	h := &inotify{
		m:  make(map[int32]*watched),
		fd: &fd,
	}
	runtime.SetFinalizer(h, func(h *inotify) { syscall.Close(*h.fd) })
	return h
}

// NewWatcher creates new non-recursive watcher backed by inotify.
func newWatcher() Watcher {
	return global
}

// TODO(ppknap) : doc.
func loop(stop <-chan struct{}) {
	global.wg.Add(1)
	for {
		select {
		case <-stop:
			global.wg.Done()
			return
		default:
			n, err := syscall.Read(*global.fd, global.buffer[:])
			switch {
			//TODO(pknap) : improve error handling + doc.
			case err != nil || n < 0:
				// TODO(rjeczalik): Panic, error?
				fmt.Println(os.NewSyscallError("Read", err))
				return
			case n < syscall.SizeofInotifyEvent:
				return
			}

			events := make([]*event, 0)
			nmin := n - syscall.SizeofInotifyEvent

			var sys *syscall.InotifyEvent
			for pos, name := 0, ""; pos <= nmin; {
				sys = (*syscall.InotifyEvent)(unsafe.Pointer(&global.buffer[pos]))
				pos += syscall.SizeofInotifyEvent

				if name = ""; sys.Len > 0 {
					endpos := pos + int(sys.Len)
					name = string(bytes.TrimRight(global.buffer[pos:endpos], "\x00"))
					pos = endpos
				}

				events = append(events, &event{sys: syscall.InotifyEvent{
					Wd:     sys.Wd,
					Mask:   sys.Mask,
					Cookie: sys.Cookie,
				}, impl: watched{name: name}})
			}
			send(events)
		}
	}
}

// TODO(ppknap) : doc.
func inotifywatch(name string, event Event) error {
	// TODO(ppknap) : doc. (ignore add mask)
	event &= ^IN_MASK_ADD

	if event&invalid != 0 {
		return errors.New("invalid event")
	}
	wd, err := syscall.InotifyAddWatch(*global.fd, name, makemask(event))
	if err != nil {
		return os.NewSyscallError("InotifyAddWatch", err)
	}

	global.RLock()
	w := global.m[int32(wd)]
	global.RUnlock()

	if w == nil {
		w = &watched{name: name, mask: uint32(event)}
		global.Lock()
		global.m[int32(wd)] = w
		global.Unlock()
	} else {
		atomic.StoreUint32(&w.mask, uint32(event))
	}
	return nil
}

// TODO(ppknap) : doc.
func inotifyunwatch(name string) error {
	wd := int32(-1)
	global.RLock()
	for wdkey, w := range global.m {
		if w.name == name {
			wd = wdkey
			break
		}
	}
	global.RUnlock()

	if wd < 0 {
		return errors.New("file/directory " + name + " is unwatched")
	}
	// BUG(goauthors) : watch descriptor is of type `int`, not `uint32`
	if _, err := syscall.InotifyRmWatch(*global.fd, uint32(wd)); err != nil {
		return os.NewSyscallError("InotifyRmWatch", err)
	}

	global.Lock()
	delete(global.m, wd)
	global.Unlock()
	return nil
}

// TODO(ppknap) : doc
func send(events []*event) {
	global.RLock()
	for i, event := range events {
		if event.sys.Mask&(syscall.IN_IGNORED|syscall.IN_Q_OVERFLOW) != 0 {
			events[i] = nil
			continue
		}
		if w, ok := global.m[event.sys.Wd]; ok {
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
	global.RUnlock()

	for _, event := range events {
		if event != nil {
			sendevent(event)
		}
	}
}

func sendevent(ei EventInfo) {
	global.c <- ei
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
	case imask&(IN_CREATE|IN_MOVED_TO)&syse != 0:
		return Create
	case imask&(IN_DELETE|IN_DELETE_SELF)&syse != 0:
		return Delete
	case imask&(IN_MODIFY)&syse != 0:
		return Write
	case imask&(IN_MOVED_FROM|IN_MOVE_SELF)&syse != 0:
		return Move
	}
	panic("notify: cannot decode internal mask")
}

// Watch implements notify.Watcher interface.
func (i *inotify) Watch(p string, e Event) error {
	return inotifywatch(p, e)
}

// Unwatch implements notify.Watcher interface.
func (i *inotify) Unwatch(p string) error {
	return inotifyunwatch(p)
}

// Fanin implements notify.Watcher interface.
func (i *inotify) Fanin(c chan<- EventInfo, stop <-chan struct{}) {
	i.wg.Wait() // Waits for close of previous loop() - only for test purpose.
	i.c = c
	go loop(stop)
}
