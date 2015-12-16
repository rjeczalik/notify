// Copyright (c) 2014-2015 The Notify Authors. All rights reserved.
// Use of this source code is governed by the MIT license that can be
// found in the LICENSE file.

// +build solaris

package notify

import (
	"os"
	"sync"
	"syscall"
)

func newTrg(c chan<- EventInfo) *trg {
	return &trg{
		pthLkp: make(map[string]*watched, 0),
		c:      c,
		s:      make(chan struct{}, 1),
		cf:     newCfen(),
	}
}

func (t *trg) close() (err error) {
	if err = t.cf.port_alert(t.p); err != nil {
		return
	}
	<-t.s
	t.pthLkp = make(map[string]*watched, 0)
	err = syscall.Close(t.p)
	t.cf.free()
	return
}

// encode converts notify's events to FEN's representation.
func encode(e Event) (o int64) {
	// Create event is not supported by FEN.
	o = int64(e &^ Create)
	if e&Write != 0 {
		o = (o &^ int64(Write)) | int64(FileModified)
	}
	// Following events are 'exception events' and as such cannot be requested
	// explicitly for monitoring or filtered out.
	o &= int64(^Rename & ^Remove &^ FileDelete &^ FileRenameTo &^
		FileRenameFrom &^ Unmounted &^ MountedOver)
	return
}

var nat2not = map[Event]Event{
	FileModified:   Write,
	FileRenameFrom: Rename,
	FileDelete:     Remove,
	FileAccess:     Event(0),
	FileAttrib:     Event(0),
	FileRenameTo:   Event(0),
	FileTrunc:      Event(0),
	FileNoFollow:   Event(0),
	Unmounted:      Event(0),
	MountedOver:    Event(0),
}

func (t *trg) monitor() {
	var (
		pe  PortEvent
		err error
	)
	for {
		pe = PortEvent{}
		err = t.cf.port_get(t.p, &pe)
		switch {
		case err == syscall.EINTR:
		case err == syscall.EBADF:
			t.s <- struct{}{}
			return
		case err != nil:
			dbgprintf("trg: failed to read fen events: %q\n", err)
		case pe.PortevSource == srcAlert:
			t.s <- struct{}{}
			return
		default:
			t.sendEvents(t.process(pe))
		}
	}
}

var (
	nRename = FileRenameFrom
	nRemove = FileDelete
	nWrite  = FileModified
	nDelRen = FileDelete | FileRenameFrom
)

func (*trg) addwatch(p string, fi os.FileInfo) (*watched, error) {
	return &watched{p: p, fi: fi}, nil
}

func (t *trg) assign(w *watched) {
	t.pthLkp[w.p] = w
}

func (t *trg) del(w *watched) {
	delete(t.pthLkp, w.p)
}

func (t *trg) watched(n interface{}) (*watched, int64) {
	pe, ok := n.(PortEvent)
	if !ok {
		panic("trg: invalid native type")
	}
	fo, ok := pe.PortevObject.(*FileObj)
	if !ok || fo == nil {
		panic("fen: invalid response from port_get")
	}
	w, ok := t.pthLkp[fo.Name]
	if !ok {
		return nil, 0
	}
	return w, int64(pe.PortevEvents)
}

// fen is a type holding data for fen watcher.
type trg struct {
	sync.Mutex
	// p is a FEN port identifier
	p int
	// pthLkp is a data structure mapping file names with data about watching
	// represented by them files/directories.
	pthLkp map[string]*watched
	// c is a channel used to pass events further.
	c chan<- EventInfo
	// s is a channel used to stop monitoring.
	s chan struct{}
	// cf wraps C operations.
	cf cfen
}

// watched is a data structure representing watched file/directory.
type watched struct {
	// p is a path to watched file/directory.
	p string
	// fi provides information about watched file/dir.
	fi os.FileInfo
	// eDir represents events watched directly.
	eDir Event
	// eNonDir represents events watched indirectly.
	eNonDir Event
}

// init initializes FEN.
func (t *trg) init() (err error) {
	t.p, err = t.cf.port_create()
	return
}

func fi2fo(fi os.FileInfo, p string) FileObj {
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		panic("fen: invalid stat type")
	}
	return FileObj{Name: p, Atim: st.Atim, Mtim: st.Mtim, Ctim: st.Ctim}
}

func (t *trg) natunwatch(w *watched) error {
	return t.cf.port_dissociate(t.p, FileObj{Name: w.p})
}

func (t *trg) natwatch(fi os.FileInfo, w *watched, e int64) error {
	return t.cf.port_associate(t.p, fi2fo(fi, w.p), int(e))
}
