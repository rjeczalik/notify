// +build linux

package notify

import (
	"bytes"
	"errors"
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

// inotify implements Watcher interface.
type inotify struct {
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

// NewWatcher creates new non-recursive inotify backed by inotify.
func newWatcher(c chan<- EventInfo) *inotify {
	i := &inotify{
		m:      make(map[int32]*watched),
		fd:     invalidDescriptor,
		pipefd: []int{invalidDescriptor, invalidDescriptor},
		epfd:   invalidDescriptor,
		epes:   make([]syscall.EpollEvent, 0, 2),
		c:      c,
	}
	runtime.SetFinalizer(i, func(i *inotify) {
		i.epollclose()
		if i.fd != invalidDescriptor {
			syscall.Close(int(i.fd))
		}
	})
	return i
}

// Watch implements notify.Watcher interface.
func (i *inotify) Watch(pathname string, e Event) error {
	return i.watch(pathname, e)
}

// Rewatch implements notify.Watcher interface.
func (i *inotify) Rewatch(pathname string, _, newevent Event) error {
	return i.watch(pathname, newevent)
}

// TODO : pknap
func (i *inotify) watch(pathname string, e Event) (err error) {
	// Adding new filters with IN_MASK_ADD mask is not supported.
	e &^= Event(syscall.IN_MASK_ADD)
	if e&^(All|Event(syscall.IN_ALL_EVENTS)) != 0 {
		return errors.New("notify: unknown event")
	}
	if err = i.lazyinit(); err != nil {
		return
	}
	iwd, err := syscall.InotifyAddWatch(int(i.fd), pathname, encode(e))
	if err != nil {
		return
	}
	i.RLock()
	wd := i.m[int32(iwd)]
	i.RUnlock()
	if wd == nil {
		i.Lock()
		if i.m[int32(iwd)] == nil {
			i.m[int32(iwd)] = &watched{pathname: pathname, mask: uint32(e)}
		}
		i.Unlock()
	} else {
		atomic.StoreUint32(&wd.mask, uint32(e))
	}
	return nil
}

// TODO : pknap
func (i *inotify) lazyinit() error {
	if atomic.LoadInt32(&i.fd) == invalidDescriptor {
		i.Lock()
		defer i.Unlock()
		if atomic.LoadInt32(&i.fd) == invalidDescriptor {
			fd, err := syscall.InotifyInit()
			if err != nil {
				return err
			}
			atomic.StoreInt32(&i.fd, int32(fd))
			if err = i.epollinit(); err != nil {
				i.epollclose()
				return err
			}
			esch := make(chan []*event)
			go i.loop(esch)
			i.wg.Add(consumersCount)
			for n := 0; n < consumersCount; n++ {
				go i.send(esch)
			}
		}
	}
	return nil
}

// TODO : pknap
func (i *inotify) epollinit() (err error) {
	if i.epfd, err = syscall.EpollCreate(2); err != nil {
		return
	}
	if err = syscall.Pipe(i.pipefd); err != nil {
		return
	}
	i.epes = []syscall.EpollEvent{
		{Events: syscall.EPOLLIN, Fd: int32(i.fd), Pad: 0},
		{Events: syscall.EPOLLIN, Fd: int32(i.pipefd[0]), Pad: 0},
	}
	if err = syscall.EpollCtl(
		i.epfd, syscall.EPOLL_CTL_ADD, int(i.fd), &i.epes[0]); err != nil {
		return
	}
	return syscall.EpollCtl(
		i.epfd, syscall.EPOLL_CTL_ADD, i.pipefd[0], &i.epes[1])
}

// TODO : pknap
func (i *inotify) epollclose() (err error) {
	if i.epfd != invalidDescriptor {
		if err = syscall.Close(i.epfd); err == nil {
			i.epfd = invalidDescriptor
		}
	}
	for n, fd := range i.pipefd {
		if fd != invalidDescriptor {
			switch e := syscall.Close(fd); {
			case e != nil && err == nil:
				err = e
			case e == nil:
				i.pipefd[n] = invalidDescriptor
			}
		}
	}
	return
}

// TODO : pknap
func (i *inotify) loop(esch chan<- []*event) {
	epes := make([]syscall.EpollEvent, 1)
	fd := atomic.LoadInt32(&i.fd)
	for {
		if _, err := syscall.EpollWait(i.epfd, epes, -1); err != nil {
			// Panic? error?
		}
		switch epes[0].Fd {
		case fd:
			esch <- i.read()
		case int32(i.pipefd[0]):
			i.Lock()
			defer i.Unlock()
			if err := syscall.Close(int(fd)); err != nil {
				// Panic? error?
			} else {
				atomic.StoreInt32(&i.fd, invalidDescriptor)
			}
			i.epollclose()
			close(esch)
			return
		}
	}
}

// TODO(ppknap) : doc.
func (i *inotify) read() (es []*event) {
	n, err := syscall.Read(int(i.fd), i.buffer[:])
	switch {
	case err != nil || n < 0:
		// TODO(rjeczalik): Panic, error?
		panic(err)
	case n < syscall.SizeofInotifyEvent:
		return
	}
	var sys *syscall.InotifyEvent
	nmin := n - syscall.SizeofInotifyEvent
	for pos, pathname := 0, ""; pos <= nmin; {
		sys = (*syscall.InotifyEvent)(unsafe.Pointer(&i.buffer[pos]))
		pos += syscall.SizeofInotifyEvent
		if pathname = ""; sys.Len > 0 {
			endpos := pos + int(sys.Len)
			pathname = string(bytes.TrimRight(i.buffer[pos:endpos], "\x00"))
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
func (i *inotify) send(esch <-chan []*event) {
	for es := range esch {
		for _, e := range i.strip(es) {
			if e != nil {
				i.c <- e
			}
		}
	}
	i.wg.Done()
}

// TODO(ppknap) : doc.
func (i *inotify) strip(es []*event) []*event {
	i.RLock()
	for n, e := range es {
		if e.sys.Mask&(syscall.IN_IGNORED|syscall.IN_Q_OVERFLOW) != 0 {
			es[n] = nil
			continue
		}
		wd, ok := i.m[e.sys.Wd]
		if !ok || e.sys.Mask&encode(Event(wd.mask)) == 0 {
			es[n] = nil
			continue
		}
		e.impl.mask = wd.mask
		if e.impl.pathname == "" {
			e.impl.pathname = wd.pathname
		} else {
			e.impl.pathname = filepath.Join(wd.pathname, e.impl.pathname)
		}
	}
	i.RUnlock()
	return es
}

// TODO(ppknap) : doc.
func encode(e Event) uint32 {
	if e&Create != 0 {
		e = (e ^ Create) | InCreate | InMovedTo
	}
	if e&Delete != 0 {
		e = (e ^ Delete) | InDelete | InDeleteSelf
	}
	if e&Write != 0 {
		e = (e ^ Write) | InModify
	}
	if e&Move != 0 {
		e = (e ^ Move) | InMovedFrom | InMoveSelf
	}
	return uint32(e)
}

// Unwatch implements notify.Watcher interface.
func (i *inotify) Unwatch(pathname string) (err error) {
	iwd := int32(-1)
	i.RLock()
	for iwdkey, wd := range i.m {
		if wd.pathname == pathname {
			iwd = iwdkey
			break
		}
	}
	i.RUnlock()
	if iwd < 0 {
		return errors.New("notify: file/dir " + pathname + " is unwatched")
	}
	if _, err = syscall.InotifyRmWatch(
		int(atomic.LoadInt32(&i.fd)), uint32(iwd)); err != nil {
		return
	}
	i.Lock()
	delete(i.m, iwd)
	i.Unlock()
	return nil
}

func (i *inotify) Close() (err error) {
	i.Lock()
	if fd := atomic.LoadInt32(&i.fd); fd == invalidDescriptor {
		i.Unlock()
		return nil
	}
	for iwdkey := range i.m {
		if _, e := syscall.InotifyRmWatch(
			int(i.fd), uint32(iwdkey)); e != nil && err == nil {
			err = e
		}
		delete(i.m, iwdkey)
	}
	if _, e := syscall.Write(
		i.pipefd[1], []byte{0x00}); e != nil && err == nil {
		err = e
	}
	i.Unlock()
	i.wg.Wait()
	return
}

// TODO(ppknap) : doc.
func decode(mask, syse uint32) Event {
	imask := encode(Event(mask))
	switch {
	case mask&syse != 0:
		return Event(syse)
	case imask&uint32(InCreate|InMovedTo)&syse != 0:
		return Create
	case imask&uint32(InDelete|InDeleteSelf)&syse != 0:
		return Delete
	case imask&uint32(InModify)&syse != 0:
		return Write
	case imask&uint32(InMovedFrom|InMoveSelf)&syse != 0:
		return Move
	}
	panic("notify: cannot decode internal mask")
}
