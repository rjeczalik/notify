// Copyright (c) 2014-2015 The Notify Authors. All rights reserved.
// Use of this source code is governed by the MIT license that can be
// found in the LICENSE file.

// +build darwin,kqueue dragonfly freebsd netbsd openbsd

package notify

import (
	"os"
	"sync"
	"syscall"
)

func newTrg(c chan<- EventInfo) *trg {
	return &trg{
		idLkp:  make(map[int]*watched, 0),
		pthLkp: make(map[string]*watched, 0),
		c:      c,
		s:      make(chan struct{}, 1),
	}
}

func (t *trg) close() (err error) {
	// trigger event used to interrupt Kevent call.
	if _, err = syscall.Write(t.pipefds[1], []byte{0x00}); err != nil {
		return
	}
	<-t.s
	t.Lock()
	var e error
	for _, w := range t.idLkp {
		if e = t.unwatch(w.p, w.fi); e != nil && err == nil {
			dbgprintf("trg: unwatch %q failed: %q", w.p, e)
			err = e
		}
	}
	if e := error(syscall.Close(t.fd)); e != nil && err == nil {
		dbgprintf("trg: closing kqueue fd failed: %q", e)
		err = e
	}
	t.idLkp, t.pthLkp = nil, nil
	t.Unlock()
	return
}

// encode converts requested events to kqueue representation.
func encode(e Event) (o int64) {
	o = int64(e &^ Create)
	if e&Write != 0 {
		o = (o &^ int64(Write)) | int64(NoteWrite)
	}
	if e&Rename != 0 {
		o = (o &^ int64(Rename)) | int64(NoteRename)
	}
	if e&Remove != 0 {
		o = (o &^ int64(Remove)) | int64(NoteDelete)
	}
	return
}

var nat2not = map[Event]Event{
	NoteWrite:  Write,
	NoteRename: Rename,
	NoteDelete: Remove,
	NoteExtend: Event(0),
	NoteAttrib: Event(0),
	NoteRevoke: Event(0),
	NoteLink:   Event(0),
}

func (t *trg) del(w *watched) {
	syscall.Close(w.fd)
	delete(t.idLkp, w.fd)
	delete(t.pthLkp, w.p)
}

func (t *trg) monitor() {
	var (
		kevn [1]syscall.Kevent_t
		n    int
		err  error
	)
	for {
		kevn[0] = syscall.Kevent_t{}
		switch n, err = syscall.Kevent(t.fd, nil, kevn[:], nil); {
		case err == syscall.EINTR:
		case err != nil:
			dbgprintf("trg: failed to read events: %q\n", err)
		case int(kevn[0].Ident) == t.pipefds[0]:
			t.s <- struct{}{}
			return
		case n > 0:
			t.sendEvents(t.process(kevn[0]))
		}
	}
}

var (
	nRename = NoteRename
	nRemove = NoteDelete
	nWrite  = NoteWrite
	nDelRen = NoteDelete | NoteRename
)

type trg struct {
	sync.Mutex
	// fd is a kqueue file descriptor
	fd int
	// pipefds are file descriptors used to stop `Kevent` call.
	pipefds [2]int
	// idLkp is a data structure mapping file descriptors with data about watching
	// represented by them files/directories.
	idLkp map[int]*watched
	// pthLkp is a data structure mapping file names with data about watching
	// represented by them files/directories.
	pthLkp map[string]*watched
	// c is a channel used to pass events further.
	c chan<- EventInfo
	// s is a channel used to stop monitoring.
	s chan struct{}
}

// watched is a data structure representing watched file/directory.
type watched struct {
	// p is a path to watched file/directory.
	p string
	// fd is a file descriptor for watched file/directory.
	fd int
	// fi provides information about watched file/dir.
	fi os.FileInfo
	// eDir represents events watched directly.
	eDir Event
	// eNonDir represents events watched indirectly.
	eNonDir Event
}

// init initializes kqueue.
func (t *trg) init() (err error) {
	if t.fd, err = syscall.Kqueue(); err != nil {
		return
	}
	// Creates pipe used to stop `Kevent` call by registering it,
	// watching read end and writing to other end of it.
	if err = syscall.Pipe(t.pipefds[:]); err != nil {
		return
	}
	var kevn [1]syscall.Kevent_t
	syscall.SetKevent(&kevn[0], t.pipefds[0], syscall.EVFILT_READ, syscall.EV_ADD)
	_, err = syscall.Kevent(t.fd, kevn[:], nil, nil)
	return
}

func (t *trg) natwatch(fi os.FileInfo, w *watched, e int64) error {
	var kevn [1]syscall.Kevent_t
	syscall.SetKevent(&kevn[0], w.fd, syscall.EVFILT_VNODE,
		syscall.EV_ADD|syscall.EV_CLEAR)
	kevn[0].Fflags = uint32(e)

	if _, err := syscall.Kevent(t.fd, kevn[:], nil, nil); err != nil {
		return err
	}
	return nil
}

func (t *trg) watched(n interface{}) (*watched, int64) {
	kevn, ok := n.(syscall.Kevent_t)
	if !ok {
		panic("trg: invalid native type")
	}
	return t.idLkp[int(kevn.Ident)], int64(kevn.Fflags)
}

func (t *trg) natunwatch(w *watched) error {
	var kevn [1]syscall.Kevent_t
	syscall.SetKevent(&kevn[0], w.fd, syscall.EVFILT_VNODE, syscall.EV_DELETE)

	if _, err := syscall.Kevent(t.fd, kevn[:], nil, nil); err != nil {
		return err
	}
	return nil
}

func (*trg) addwatch(p string, fi os.FileInfo) (*watched, error) {
	fd, err := syscall.Open(p, syscall.O_NONBLOCK|syscall.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	return &watched{fd: fd, p: p, fi: fi}, nil
}

func (t *trg) assign(w *watched) {
	t.idLkp[w.fd], t.pthLkp[w.p] = w, w
}
