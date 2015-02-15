// Copyright (c) 2014-2015 The Notify Authors. All rights reserved.
// Use of this source code is governed by the MIT license that can be
// found in the LICENSE file.

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

// watched is a pair of file path and inotify mask used as a value in
// watched files map.
type watched struct {
	path string
	mask uint32
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
func (i *inotify) Watch(path string, e Event) error {
	return i.watch(path, e)
}

// Rewatch implements notify.Watcher interface.
func (i *inotify) Rewatch(path string, _, newevent Event) error {
	return i.watch(path, newevent)
}

// TODO : pknap
func (i *inotify) watch(path string, e Event) (err error) {
	// Adding new filters with IN_MASK_ADD mask is not supported.
	e &^= Event(syscall.IN_MASK_ADD)
	if e&^(All|Event(syscall.IN_ALL_EVENTS)) != 0 {
		return errors.New("notify: unknown event")
	}
	if err = i.lazyinit(); err != nil {
		return
	}
	iwd, err := syscall.InotifyAddWatch(int(i.fd), path, encode(e))
	if err != nil {
		return
	}
	i.RLock()
	wd := i.m[int32(iwd)]
	i.RUnlock()
	if wd == nil {
		i.Lock()
		if i.m[int32(iwd)] == nil {
			i.m[int32(iwd)] = &watched{path: path, mask: uint32(e)}
		}
		i.Unlock()
	} else {
		i.Lock()
		wd.mask = uint32(e)
		i.Unlock()
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
				atomic.StoreInt32(&i.fd, int32(invalidDescriptor))
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
	if err = syscall.EpollCtl(i.epfd, syscall.EPOLL_CTL_ADD, int(i.fd),
		&i.epes[0]); err != nil {
		return
	}
	return syscall.EpollCtl(i.epfd, syscall.EPOLL_CTL_ADD, i.pipefd[0],
		&i.epes[1])
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
		if _, err := syscall.EpollWait(i.epfd, epes, -1); err != nil &&
			err != syscall.EINTR {
			// Panic? error?
			panic(err)
		}
		switch epes[0].Fd {
		case fd:
			esch <- i.read()
		case int32(i.pipefd[0]):
			i.Lock()
			defer i.Unlock()
			if err := syscall.Close(int(fd)); err != nil {
				// Panic? error?
				panic(err)
			}
			atomic.StoreInt32(&i.fd, invalidDescriptor)
			if err := i.epollclose(); err != nil {
				// Panic? error?
				panic(err)
			}
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
		// Panic? error?
		panic(err)
	case n < syscall.SizeofInotifyEvent:
		return
	}
	var sys *syscall.InotifyEvent
	nmin := n - syscall.SizeofInotifyEvent
	for pos, path := 0, ""; pos <= nmin; {
		sys = (*syscall.InotifyEvent)(unsafe.Pointer(&i.buffer[pos]))
		pos += syscall.SizeofInotifyEvent
		if path = ""; sys.Len > 0 {
			endpos := pos + int(sys.Len)
			path = string(bytes.TrimRight(i.buffer[pos:endpos], "\x00"))
			pos = endpos
		}
		es = append(es, &event{sys: syscall.InotifyEvent{
			Wd:     sys.Wd,
			Mask:   sys.Mask,
			Cookie: sys.Cookie,
		}, path: path})
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
	var multi []*event
	i.RLock()
	for idx, e := range es {
		if e.sys.Mask&(syscall.IN_IGNORED|syscall.IN_Q_OVERFLOW) != 0 {
			es[idx] = nil
			continue
		}
		wd, ok := i.m[e.sys.Wd]
		if !ok || e.sys.Mask&encode(Event(wd.mask)) == 0 {
			es[idx] = nil
			continue
		}
		if e.path == "" {
			e.path = wd.path
		} else {
			e.path = filepath.Join(wd.path, e.path)
		}
		multi = append(multi, decode(e, Event(wd.mask))...)
		if e.event == 0 {
			es[idx] = nil
		}
	}
	i.RUnlock()
	es = append(es, multi...)
	return es
}

// TODO(ppknap) : doc.
func encode(e Event) uint32 {
	if e&Create != 0 {
		e = (e ^ Create) | InCreate | InMovedTo
	}
	if e&Remove != 0 {
		e = (e ^ Remove) | InDelete | InDeleteSelf
	}
	if e&Write != 0 {
		e = (e ^ Write) | InModify
	}
	if e&Rename != 0 {
		e = (e ^ Rename) | InMovedFrom | InMoveSelf
	}
	return uint32(e)
}

// TODO(ppknap) : doc.
func decode(e *event, mask Event) (multi []*event) {
	if syse := uint32(mask) & e.sys.Mask; syse != 0 {
		multi = append(multi, &event{sys: syscall.InotifyEvent{
			Wd:     e.sys.Wd,
			Mask:   e.sys.Mask,
			Cookie: e.sys.Cookie,
		}, event: Event(syse), path: e.path})
	}
	imask := encode(mask)
	switch {
	case mask&Create != 0 &&
		imask&uint32(InCreate|InMovedTo)&e.sys.Mask != 0:
		e.event = Create
	case mask&Remove != 0 &&
		imask&uint32(InDelete|InDeleteSelf)&e.sys.Mask != 0:
		e.event = Remove
	case mask&Write != 0 &&
		imask&uint32(InModify)&e.sys.Mask != 0:
		e.event = Write
	case mask&Rename != 0 &&
		imask&uint32(InMovedFrom|InMoveSelf)&e.sys.Mask != 0:
		e.event = Rename
	}
	return
}

// Unwatch implements notify.Watcher interface.
func (i *inotify) Unwatch(path string) (err error) {
	iwd := int32(-1)
	i.RLock()
	for iwdkey, wd := range i.m {
		if wd.path == path {
			iwd = iwdkey
			break
		}
	}
	i.RUnlock()
	if iwd < 0 {
		return errors.New("notify: file/dir " + path + " is unwatched")
	}
	if _, err = syscall.InotifyRmWatch(int(atomic.LoadInt32(&i.fd)),
		uint32(iwd)); err != nil {
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
		if _, e := syscall.InotifyRmWatch(int(i.fd),
			uint32(iwdkey)); e != nil && err == nil {
			err = e
		}
		delete(i.m, iwdkey)
	}
	if _, e := syscall.Write(i.pipefd[1], []byte{0x00}); e != nil &&
		err == nil {
		err = e
	}
	i.Unlock()
	i.wg.Wait()
	return
}
