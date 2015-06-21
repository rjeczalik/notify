// Copyright (c) 2014-2015 The Notify Authors. All rights reserved.
// Use of this source code is governed by the MIT license that can be
// found in the LICENSE file.

// +build darwin,kqueue dragonfly freebsd netbsd openbsd

package notify

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
)

// newWatcher returns kqueue Watcher implementation.
func newWatcher(c chan<- EventInfo) watcher {
	k := &kqueue{
		idLkp:  make(map[int]*watched, 0),
		pthLkp: make(map[string]*watched, 0),
		c:      c,
		s:      make(chan struct{}, 1),
	}
	if err := k.init(); err != nil {
		panic(err)
	}
	go k.monitor()
	return k
}

// Close closes all still open file descriptors and kqueue.
func (k *kqueue) Close() (err error) {
	// trigger event used to interrup Kevent call.
	if _, err = syscall.Write(k.pipefds[1], []byte{0x00}); err != nil {
		return
	}
	<-k.s
	k.Lock()
	var e error
	for _, w := range k.idLkp {
		if e = k.unwatch(w.p, w.fi); e != nil && err == nil {
			dbgprintf("kqueue: unwatch %q failed: %q", w.p, e)
			err = e
		}
	}
	if e := error(syscall.Close(k.fd)); e != nil && err == nil {
		dbgprintf("kqueue: closing kqueu fd failed: %q", e)
		err = e
	}
	k.idLkp, k.pthLkp = nil, nil
	k.Unlock()
	return
}

// sendEvents sends reported events one by one through chan.
func (k *kqueue) sendEvents(evn []event) {
	for i := range evn {
		k.c <- &evn[i]
	}
}

// encode converts requested events to kqueue representation.
func encode(e Event) (o uint32) {
	o = uint32(e &^ Create)
	if e&Write != 0 {
		o = (o &^ uint32(Write)) | uint32(NoteWrite)
	}
	if e&Rename != 0 {
		o = (o &^ uint32(Rename)) | uint32(NoteRename)
	}
	if e&Remove != 0 {
		o = (o &^ uint32(Remove)) | uint32(NoteDelete)
	}
	return
}

// decode converts event received from kqueue to notify.Event
// representation taking into account requested events (w).
func decode(o uint32, w Event) (e Event) {
	// TODO(someone) : fix me
	if o&uint32(NoteWrite) != 0 {
		if w&NoteWrite != 0 {
			e |= NoteWrite
		} else {
			e |= Write
		}
	}
	if o&uint32(NoteRename) != 0 {
		if w&NoteRename != 0 {
			e |= NoteRename
		} else {
			e |= Rename
		}
	}
	if o&uint32(NoteDelete) != 0 {
		if w&NoteDelete != 0 {
			e |= NoteDelete
		} else {
			e |= Remove
		}
	}
	e |= Event(o) & w
	return
}

// del closes fd for watched and removes it from internal cache of monitored
// files/directories.
func (k *kqueue) del(w watched) {
	syscall.Close(w.fd)
	delete(k.idLkp, w.fd)
	delete(k.pthLkp, w.p)
}

// monitor reads reported kqueue events and forwards them further after
// performing additional processing. If read event concerns directory,
// it generates Create/Remove event and sent them further instead of directory
// event. This event is detected based on reading contents of analyzed
// directory. If no changes in file list are detected, no event is send further.
// Reading directory structure is less accurate than kqueue and can lead
// to lack of detection of all events.
func (k *kqueue) monitor() {
	var (
		kevn [1]syscall.Kevent_t
		n    int
		err  error
	)
	for {
		kevn[0] = syscall.Kevent_t{}
		switch n, err = syscall.Kevent(k.fd, nil, kevn[:], nil); {
		case err == syscall.EINTR:
		case err != nil:
			dbgprintf("kqueue: failed to read events: %q\n", err)
		case int(kevn[0].Ident) == k.pipefds[0]:
			k.s <- struct{}{}
			return
		case n > 0:
			k.sendEvents(k.process(kevn[0]))
		}
	}
}

func (k *kqueue) dir(w watched, kevn syscall.Kevent_t, e Event) (evn []event) {
	// If it's dir and delete we have to send it and continue, because
	// other processing relies on opening (in this case not existing) dir.
	// Events for contents of this dir are reported by kqueue.
	// However events for rename must be generated for all monitored files
	// inside of moved directory, because kqueue does not report it independently
	// for each file descriptor being moved in result of move action on
	// parent directory.
	if (Event(kevn.Fflags) & (NoteDelete | NoteRename)) != 0 {
		// Write is reported also for Remove on directory. Because of that
		// we have to filter it out explicitly.
		evn = append(evn, event{w.p,
			e & ^Write & ^NoteWrite, Kevent{&kevn, w.fi}})
		if Event(kevn.Fflags)&NoteRename != 0 {
			for p, wt := range k.pthLkp {
				// TODO: Change the way of storing paths in kqueue.
				if strings.HasPrefix(p, w.p+string(os.PathSeparator)) {
					if err := k.unwatch(p, wt.fi); err != nil && err != errNotWatched &&
						!os.IsNotExist(err) {
						dbgprintf("kqueue: failed stop watching moved file (%q): %q\n",
							p, err)
					}
					if (w.eDir|w.eNonDir)&(NoteRename|Rename) != 0 {
						evn = append(evn, event{
							p, (w.eDir | w.eNonDir) & e &^ Write &^ NoteWrite,
							Kevent{nil, wt.fi},
						})
					}
				}
			}
		}
		k.del(w)
		return
	}
	if (Event(kevn.Fflags) & NoteWrite) != 0 {
		switch err := k.walk(w.p, func(fi os.FileInfo) error {
			p := filepath.Join(w.p, fi.Name())
			switch err := k.singlewatch(p, w.eDir, false, fi); {
			case os.IsNotExist(err) && ((w.eDir & Remove) != 0):
				evn = append(evn, event{p, Remove, Kevent{nil, fi}})
			case err == errAlreadyWatched:
			case err != nil:
				dbgprintf("kqueue: watching %q failed: %q", p, err)
			case (w.eDir & Create) != 0:
				evn = append(evn, event{p, Create, Kevent{nil, fi}})
			}
			return nil
		}); {
		// If file is already watched, kqueue will return remove event.
		case os.IsNotExist(err):
			return
		case err != nil:
			dbgprintf("kqueue: dir processing failed: %q", err)
		default:
		}
	}
	return
}

func (*kqueue) file(w watched, kevn syscall.Kevent_t, e Event) (evn []event) {
	evn = append(evn, event{w.p, e, Kevent{&kevn, w.fi}})
	return
}

// process event returned by Kevent call.
func (k *kqueue) process(kevn syscall.Kevent_t) (evn []event) {
	k.Lock()
	w := k.idLkp[int(kevn.Ident)]
	if w == nil {
		k.Unlock()
		dbgprintf("kqueue: %v event for not registered fd", kevn)
		return
	}
	e := decode(kevn.Fflags, w.eDir|w.eNonDir)
	if w.fi.IsDir() {
		evn = k.dir(*w, kevn, e)
	} else {
		evn = k.file(*w, kevn, e)
	}
	if (Event(kevn.Fflags) & (NoteDelete | NoteRename)) != 0 {
		k.del(*w)
	}
	k.Unlock()
	return
}

// kqueue is a type holding data for kqueue watcher.
type kqueue struct {
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
func (k *kqueue) init() (err error) {
	if k.fd, err = syscall.Kqueue(); err != nil {
		return
	}
	// Creates pipe used to stop `Kevent` call by registering it,
	// watching read end and writing to other end of it.
	if err = syscall.Pipe(k.pipefds[:]); err != nil {
		return
	}
	var kevn [1]syscall.Kevent_t
	syscall.SetKevent(&kevn[0], k.pipefds[0], syscall.EVFILT_READ, syscall.EV_ADD)
	_, err = syscall.Kevent(k.fd, kevn[:], nil, nil)
	return
}

func (k *kqueue) watch(p string, e Event, fi os.FileInfo) error {
	if err := k.singlewatch(p, e, true, fi); err != nil {
		if err != errAlreadyWatched {
			return nil
		}
	}
	if fi.IsDir() {
		err := k.walk(p, func(fi os.FileInfo) (err error) {
			if err = k.singlewatch(filepath.Join(p, fi.Name()), e, false,
				fi); err != nil {
				if err != errAlreadyWatched {
					return
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// watch starts to watch given p file/directory.
func (k *kqueue) singlewatch(p string, e Event, direct bool,
	fi os.FileInfo) error {
	w, ok := k.pthLkp[p]
	if !ok {
		fd, err := syscall.Open(p, syscall.O_NONBLOCK|syscall.O_RDONLY, 0)
		if err != nil {
			return err
		}
		w = &watched{fd: fd, p: p, fi: fi}
	}
	if direct {
		w.eDir |= e
	} else {
		w.eNonDir |= e
	}
	var kevn [1]syscall.Kevent_t
	syscall.SetKevent(&kevn[0], w.fd, syscall.EVFILT_VNODE,
		syscall.EV_ADD|syscall.EV_CLEAR)
	kevn[0].Fflags = encode(w.eDir | w.eNonDir)
	if _, err := syscall.Kevent(k.fd, kevn[:], nil, nil); err != nil {
		return err
	}
	if !ok {
		k.idLkp[w.fd], k.pthLkp[w.p] = w, w
		return nil
	}
	return errAlreadyWatched
}

// unwatch stops watching p file/directory.
func (k *kqueue) singleunwatch(p string, direct bool) error {
	w := k.pthLkp[p]
	if w == nil {
		return errNotWatched
	}
	if direct {
		w.eDir = 0
	} else {
		w.eNonDir = 0
	}
	var kevn [1]syscall.Kevent_t
	syscall.SetKevent(&kevn[0], w.fd, syscall.EVFILT_VNODE, syscall.EV_DELETE)
	if _, err := syscall.Kevent(k.fd, kevn[:], nil, nil); err != nil {
		return err
	}
	if w.eNonDir&w.eDir != 0 {
		if err := k.singlewatch(p, w.eNonDir|w.eDir, w.eNonDir == 0,
			w.fi); err != nil {
			return err
		}
	} else {
		k.del(*w)
	}
	return nil
}

// walk runs f func on each file/dir from p directory.
func (k *kqueue) walk(p string, f func(os.FileInfo) error) error {
	fp, err := os.Open(p)
	if err != nil {
		return err
	}
	ls, err := fp.Readdir(0)
	fp.Close()
	if err != nil {
		return err
	}
	for i := range ls {
		if err := f(ls[i]); err != nil {
			return err
		}
	}
	return nil
}

func (k *kqueue) unwatch(p string, fi os.FileInfo) error {
	if fi.IsDir() {
		err := k.walk(p, func(fi os.FileInfo) error {
			if !fi.IsDir() {
				err := k.singleunwatch(filepath.Join(p, fi.Name()), false)
				if err != errNotWatched {
					return err
				}
				return nil
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return k.singleunwatch(p, true)
}

// Watch implements Watcher interface.
func (k *kqueue) Watch(p string, e Event) error {
	fi, err := os.Stat(p)
	if err != nil {
		return err
	}
	k.Lock()
	err = k.watch(p, e, fi)
	k.Unlock()
	return nil
}

// Unwatch implements Watcher interface.
func (k *kqueue) Unwatch(p string) error {
	fi, err := os.Stat(p)
	if err != nil {
		return err
	}
	k.Lock()
	err = k.unwatch(p, fi)
	k.Unlock()
	return nil
}

// Rewatch implements Watcher interface.
//
// TODO(rjeczalik): This is a naive hack. Rewrite might help.
func (k *kqueue) Rewatch(p string, _, e Event) error {
	fi, err := os.Stat(p)
	if err != nil {
		return err
	}
	k.Lock()
	if err = k.unwatch(p, fi); err == nil {
		// TODO(rjeczalik): If watch fails then we leave kqueue in inconsistent
		// state. Handle? Panic? Native version of rewatch?
		err = k.watch(p, e, fi)
	}
	k.Unlock()
	return nil
}
