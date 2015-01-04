// +build linux
// +build !fsnotify

package notify

import (
	"bytes"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"unsafe"
)

// eventBufferSize defines the size of the buffer given to read(2) function. One
// should not depend on this value, since it was arbitrary chosen and may be
// changed in the future.
const eventBufferSize = 64 * (syscall.SizeofInotifyEvent + syscall.PathMax + 1)

// consumersCount defines the number of consumers in producer-consumer based
// implementation. Each consumer is run in a separate goroutine and has read
// access to watched files map.
const consumersCount = 2

const invalidDescriptor = -1

// watched is a pair of file pathname and inotify mask used as a value in
// watched files map.
type watched struct {
	pathname string
	mask     uint32
}

// watcher implements Watcher interface.
type watcher struct {
	sync.RWMutex
	m      map[int32]*watched
	fd     int32
	pipefd []int
	epfd   int
	epes   []syscall.EpollEvent
	buffer [eventBufferSize]byte
	wg     sync.WaitGroup
	c      chan<- EventInfo
}

// NewWatcher creates new non-recursive watcher backed by inotify.
func newWatcher(c chan<- EventInfo) (w *watcher) {
	w = &watcher{
		m:      make(map[int32]*watched),
		fd:     invalidDescriptor,
		pipefd: []int{invalidDescriptor, invalidDescriptor},
		epfd:   invalidDescriptor,
		epes:   make([]syscall.EpollEvent, 0, 2),
		c:      c,
	}
	runtime.SetFinalizer(w, func(w *watcher) {
		w.epollclose()
		if w.fd != invalidDescriptor {
			syscall.Close(int(w.fd))
		}
	})
	return
}

// Watch implements notify.Watcher interface.
func (w *watcher) Watch(pathname string, e Event) error {
	return w.watch(pathname, e)
}

// Rewatch implements notify.Watcher interface.
func (w *watcher) Rewatch(pathname string, _, newevent Event) error {
	return w.watch(pathname, newevent)
}

// TODO : pknap
func (w *watcher) watch(pathname string, e Event) (err error) {
	// Adding new filters with IN_MASK_ADD mask is not supported.
	e &^= Event(syscall.IN_MASK_ADD)
	if e&^(All|Event(syscall.IN_ALL_EVENTS)) != 0 {
		return errors.New("notify: unknown event")
	}
	if err = w.lazyinit(); err != nil {
		return
	}
	iwd, err := syscall.InotifyAddWatch(int(w.fd), pathname, encode(e))
	if err != nil {
		return
	}
	w.RLock()
	wd := w.m[int32(iwd)]
	w.RUnlock()
	if wd == nil {
		w.Lock()
		if w.m[int32(iwd)] == nil {
			w.m[int32(iwd)] = &watched{pathname: pathname, mask: uint32(e)}
		}
		w.Unlock()
	} else {
		atomic.StoreUint32(&wd.mask, uint32(e))
	}
	return nil
}

// TODO : pknap
func (w *watcher) lazyinit() error {
	if atomic.LoadInt32(&w.fd) == invalidDescriptor {
		w.Lock()
		defer w.Unlock()
		if atomic.LoadInt32(&w.fd) == invalidDescriptor {
			fd, err := syscall.InotifyInit()
			if err != nil {
				return err
			}
			atomic.StoreInt32(&w.fd, int32(fd))
			if err = w.epollinit(); err != nil {
				w.epollclose()
				return err
			}
			esch := make(chan []*event)
			go w.loop(esch)
			w.wg.Add(consumersCount)
			for i := 0; i < consumersCount; i++ {
				go w.send(esch)
			}
		}
	}
	return nil
}

// TODO : pknap
func (w *watcher) epollinit() (err error) {
	if w.epfd, err = syscall.EpollCreate(2); err != nil {
		return
	}
	if err = syscall.Pipe(w.pipefd); err != nil {
		return
	}
	w.epes = []syscall.EpollEvent{
		syscall.EpollEvent{Events: syscall.EPOLLIN, Fd: int32(w.fd), Pad: 0},
		syscall.EpollEvent{Events: syscall.EPOLLIN, Fd: int32(w.pipefd[0]), Pad: 0},
	}
	if err = syscall.EpollCtl(
		w.epfd, syscall.EPOLL_CTL_ADD, int(w.fd), &w.epes[0]); err != nil {
		return
	}
	return syscall.EpollCtl(
		w.epfd, syscall.EPOLL_CTL_ADD, w.pipefd[0], &w.epes[1])
}

// TODO : pknap
func (w *watcher) epollclose() (err error) {
	if w.epfd != invalidDescriptor {
		if err = syscall.Close(w.epfd); err == nil {
			w.epfd = invalidDescriptor
		}
	}
	for i, fd := range w.pipefd {
		if fd != invalidDescriptor {
			switch e := syscall.Close(fd); {
			case e != nil && err == nil:
				err = e
			case e == nil:
				w.pipefd[i] = invalidDescriptor
			}
		}
	}
	return
}

// TODO : pknap
func (w *watcher) loop(esch chan<- []*event) {
	epes := make([]syscall.EpollEvent, 1)
	fd := atomic.LoadInt32(&w.fd)
	for {
		if _, err := syscall.EpollWait(w.epfd, epes, -1); err != nil {
			// Panic? error?
		}
		switch epes[0].Fd {
		case fd:
			fmt.Println("Send events")
			esch <- w.read()
		case int32(w.pipefd[0]):
			fmt.Println("break the loop")
			w.Lock()
			defer w.Unlock()
			if err := syscall.Close(int(fd)); err != nil {
				// Panic? error?
			} else {
				atomic.StoreInt32(&w.fd, invalidDescriptor)
			}
			w.epollclose()
			close(esch)
			return
		}
	}
}

// TODO(ppknap) : doc.
func (w *watcher) read() (es []*event) {
	n, err := syscall.Read(int(w.fd), w.buffer[:])
	fmt.Println("read :", n, " err ", err, "   ")
	switch {
	case err != nil || n < 0:
		// TODO(rjeczalik): Panic, error?
		fmt.Println("Error:( ", err)
		return
	case n < syscall.SizeofInotifyEvent:
		return
	}
	var sys *syscall.InotifyEvent
	nmin := n - syscall.SizeofInotifyEvent
	for pos, pathname := 0, ""; pos <= nmin; {
		sys = (*syscall.InotifyEvent)(unsafe.Pointer(&w.buffer[pos]))
		pos += syscall.SizeofInotifyEvent
		if pathname = ""; sys.Len > 0 {
			endpos := pos + int(sys.Len)
			pathname = string(bytes.TrimRight(w.buffer[pos:endpos], "\x00"))
			pos = endpos
		}
		es = append(es, &event{sys: syscall.InotifyEvent{
			Wd:     sys.Wd,
			Mask:   sys.Mask,
			Cookie: sys.Cookie,
		}, impl: watched{pathname: pathname}})
	}
	return
}

// TODO(ppknap) : doc.
func (w *watcher) send(esch <-chan []*event) {
	for es := range esch {
		for _, e := range w.strip(es) {
			if e != nil {
				w.c <- e
			}
		}
	}
	fmt.Println("Done")
	w.wg.Done()
}

// TODO(ppknap) : doc.
func (w *watcher) strip(es []*event) []*event {
	w.RLock()
	for i, e := range es {
		if e.sys.Mask&(syscall.IN_IGNORED|syscall.IN_Q_OVERFLOW) != 0 {
			es[i] = nil
			continue
		}
		wd, ok := w.m[e.sys.Wd]
		if !ok || e.sys.Mask&encode(Event(wd.mask)) == 0 {
			es[i] = nil
			continue
		}
		e.impl.mask = wd.mask
		if e.impl.pathname == "" {
			e.impl.pathname = wd.pathname
		} else {
			e.impl.pathname = filepath.Join(wd.pathname, e.impl.pathname)
		}
	}
	w.RUnlock()
	return es
}

// TODO(ppknap) : doc.
func encode(e Event) uint32 {
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

// Unwatch implements notify.Watcher interface.
func (w *watcher) Unwatch(pathname string) (err error) {
	iwd := int32(-1)
	w.RLock()
	for iwdkey, wd := range w.m {
		if wd.pathname == pathname {
			iwd = iwdkey
			break
		}
	}
	w.RUnlock()
	if iwd < 0 {
		return errors.New("notify: file/dir " + pathname + " is unwatched")
	}
	if _, err = syscall.InotifyRmWatch(
		int(atomic.LoadInt32(&w.fd)), uint32(iwd)); err != nil {
		return
	}
	w.Lock()
	delete(w.m, iwd)
	w.Unlock()
	return nil
}

func (w *watcher) Close() (err error) {
	w.Lock()
	if fd := atomic.LoadInt32(&w.fd); fd == invalidDescriptor {
		w.Unlock()
		return nil
	}
	for iwdkey, _ := range w.m {
		if _, e := syscall.InotifyRmWatch(
			int(w.fd), uint32(iwdkey)); e != nil && err == nil {
			err = e
		}
		delete(w.m, iwdkey)
	}
	if _, e := syscall.Write(
		w.pipefd[1], []byte{0x00}); e != nil && err == nil {
		err = e
	}
	w.Unlock()
	w.wg.Wait()
	return
}

// TODO(ppknap) : doc.
func decode(mask, syse uint32) Event {
	imask := encode(Event(mask))
	switch {
	case mask&syse != 0:
		return Event(syse)
	case imask&uint32(IN_CREATE|IN_MOVED_TO)&syse != 0:
		return Create
	case imask&uint32(IN_DELETE|IN_DELETE_SELF)&syse != 0:
		return Delete
	case imask&uint32(IN_MODIFY)&syse != 0:
		return Write
	case imask&uint32(IN_MOVED_FROM|IN_MOVE_SELF)&syse != 0:
		return Move
	}
	panic("notify: cannot decode internal mask")
}
