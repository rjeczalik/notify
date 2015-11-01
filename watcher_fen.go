// Copyright (c) 2014-2015 The Notify Authors. All rights reserved.
// Use of this source code is governed by the MIT license that can be
// found in the LICENSE file.

// +build solaris

package notify

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
)

// newWatcher returns fen watcher implementation.
func newWatcher(c chan<- EventInfo) watcher {
	f := &fen{
		pthLkp: make(map[string]*watched, 0),
		c:      c,
		s:      make(chan struct{}, 1),
		cf:     newCfen(),
	}
	if err := f.init(); err != nil {
		panic(err)
	}
	go f.monitor()
	return f
}

// Close implements watcher interface. It stops waiting for new events and
// closes FEN's port.
func (f *fen) Close() (err error) {
	if err = f.cf.port_alert(f.p); err != nil {
		return
	}
	<-f.s
	f.pthLkp = make(map[string]*watched, 0)
	err = syscall.Close(f.p)
	f.cf.free()
	return
}

// sendEvents sends reported events one by one through chan.
func (f *fen) sendEvents(evn []event) {
	for i := range evn {
		f.c <- &evn[i]
	}
}

// encode converts notify's events to FEN's representation.
func encode(e Event) (o int) {
	// Create event is not supported by FEN.
	o = int(e &^ Create)
	if e&Write != 0 {
		o = (o &^ int(Write)) | int(FileModified)
	}
	// Following events are 'exception events' and as such cannot be requested
	// explicitly for monitoring or filtered out.
	o &= int(^Rename & ^Remove &^ FileDelete &^ FileRenameTo &^
		FileRenameFrom &^ Unmounted &^ MountedOver)
	return
}

var fen2not = map[Event]Event{
	FileModified:   Write,
	FileRenameFrom: Rename,
	FileDelete:     Remove,
}

// decode converts event received from FEN to notify.Event
// representation taking into account requested events (w).
func decode(o int, w Event) (e Event) {
	// TODO(someone) : fix me
	for f, n := range fen2not {
		if o&int(f) != 0 {
			if w&f != 0 {
				e |= f
			}
			if w&n != 0 {
				e |= n
			}
		}
	}

	return
}

// monitor reads reported fen events and forwards them further after
// performing additional processing. If read event concerns directory,
// it generates Create/Remove event and sent them further instead of directory
// event. This event is detected based on reading contents of analyzed
// directory. If no changes in file list are detected, no event is send further.
// Reading directory structure is less accurate than fen and can lead
// to lack of detection of all events.
func (f *fen) monitor() {
	var (
		pe  PortEvent
		err error
	)
	for {
		pe = PortEvent{}
		err = f.cf.port_get(f.p, &pe)
		switch {
		case err == syscall.EINTR:
		case err == syscall.EBADF:
			f.s <- struct{}{}
			return
		case err != nil:
			dbgprintf("fen: failed to read events: %q\n", err)
		case pe.PortevSource == srcAlert:
			f.s <- struct{}{}
			return
		default:
			f.sendEvents(f.process(pe))
		}
	}
}

func (f *fen) dir(w watched, pe PortEvent, e Event) (evn []event) {
	// If it's dir and delete we have to send it and continue, because
	// other processing relies on opening (in this case not existing) dir.
	// Events for contents of this dir are reported by kqueue.
	// However events for rename must be generated for all monitored files
	// inside of moved directory, because kqueue does not report it independently
	// for each file descriptor being moved in result of move action on
	// parent dirLiczba dostępnych dni urlopowych: 0ectory.
	if (Event(pe.PortevEvents) & (FileDelete | FileRenameFrom)) != 0 {
		// Write is reported also for Remove on directory. Because of that
		// we have to filter it out explicitly.
		evn = append(evn, event{w.p, e & ^Write & ^FileModified, true, &pe})
		if Event(pe.PortevEvents)&FileRenameFrom != 0 {
			for p, wt := range f.pthLkp {
				if strings.HasPrefix(p, w.p+string(os.PathSeparator)) {
					if err := f.unwatch(p, wt.fi); err != nil && err != errNotWatched &&
						!os.IsNotExist(err) {
						dbgprintf("fen: failed stop watching moved file (%q): %q\n",
							p, err)
					}
					if (w.eDir|w.eNonDir)&(FileRenameFrom|Rename) != 0 {
						evn = append(evn, event{
							p, (w.eDir | w.eNonDir) & e &^ Write &^ FileModified,
							w.fi.IsDir(), nil,
						})
					}
				}
			}
		}
		delete(f.pthLkp, w.p)
		return
	}
	if (Event(pe.PortevEvents) & FileModified) != 0 {
		switch err := f.walk(w.p, func(fi os.FileInfo) error {
			p := filepath.Join(w.p, fi.Name())
			switch err := f.singlewatch(p, w.eDir, false, fi); {
			case os.IsNotExist(err) && ((w.eDir & Remove) != 0):
				evn = append(evn, event{p, Remove, fi.IsDir(), &pe})
			case err == errAlreadyWatched:
			case err != nil:
				dbgprintf("fen: watching %q failed: %q", p, err)
			case (w.eDir & Create) != 0:
				evn = append(evn, event{p, Create, fi.IsDir(), &pe})
			default:
			}
			return nil
		}); {
		// If file is already watched, fen will return remove event.
		case os.IsNotExist(err):
			return
		case err != nil:
			dbgprintf("fen: dir processing failed: %q", err)
		default:
		}
	}
	return
}

func (*fen) file(w watched, pe PortEvent, e Event) (evn []event) {
	evn = append(evn, event{w.p, e, w.fi.IsDir(), &pe})
	return
}

// process event returned by port_get call.
func (f *fen) process(pe PortEvent) (evn []event) {
	f.Lock()
	fo, ok := pe.PortevObject.(*FileObj)
	if !ok || fo == nil {
		panic("fen: invalid response from port_get")
	}
	w, ok := f.pthLkp[fo.Name]
	if !ok {
		f.Unlock()
		dbgprintf("fen: %v event for not registered ", Event(pe.PortevEvents))
		return
	}

	e := decode(pe.PortevEvents, w.eDir|w.eNonDir)
	if pe.PortevEvents&int(FileDelete) == 0 && pe.PortevEvents&int(FileRenameFrom) == 0 {
		switch fi, err := os.Stat(w.p); {
		case err != nil:
		default:
			if err = f.cf.port_associate(f.p, fi2fo(fi, w.p), encode(w.eDir|w.eNonDir)); err != nil {
				dbgprintf("fen: %q is no longer watched: %q", w.p, err)
				delete(f.pthLkp, w.p)
			}
		}
	}

	if w.fi.IsDir() {
		evn = append(evn, f.dir(*w, pe, e)...)
	} else {
		evn = append(evn, f.file(*w, pe, e)...)
	}
	if (Event(pe.PortevEvents) & (FileDelete | FileRenameFrom)) != 0 {
		delete(f.pthLkp, w.p)
	}
	f.Unlock()
	return
}

// fen is a type holding data for fen watcher.
type fen struct {
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
func (f *fen) init() (err error) {
	f.p, err = f.cf.port_create()
	return
}

func (f *fen) watch(p string, e Event, fi os.FileInfo) error {
	if err := f.singlewatch(p, e, true, fi); err != nil {
		if err != errAlreadyWatched {
			return nil
		}
	}
	if fi.IsDir() {
		err := f.walk(p, func(fi os.FileInfo) (err error) {
			if err = f.singlewatch(filepath.Join(p, fi.Name()), e, false,
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

func fi2fo(fi os.FileInfo, p string) FileObj {
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		panic("fen: invalid stat type")
	}
	return FileObj{Name: p, Atim: st.Atim, Mtim: st.Mtim, Ctim: st.Ctim}
}

// watch starts to watch given p file/directory.
func (f *fen) singlewatch(p string, e Event, direct bool, fi os.FileInfo) error {
	w, ok := f.pthLkp[p]
	if !ok {
		w = &watched{p: p, fi: fi}
	}
	if direct {
		w.eDir |= e
	} else {
		w.eNonDir |= e
	}
	ee := 0
	if e&(Create|FileRenameTo) != 0 && fi.IsDir() {
		ee = int(FileModified)
	}
	if err := f.cf.port_associate(f.p, fi2fo(fi, w.p),
		encode(w.eDir|w.eNonDir)|ee); err != nil {
		return err
	}
	if !ok {
		f.pthLkp[w.p] = w
		return nil
	}
	return errAlreadyWatched
}

// unwatch stops watching p file/directory.
func (f *fen) singleunwatch(p string, direct bool) error {
	w, ok := f.pthLkp[p]
	if !ok {
		return errNotWatched
	}
	if direct {
		w.eDir = 0
	} else {
		w.eNonDir = 0
	}
	fo := FileObj{Name: p}
	if err := f.cf.port_dissociate(f.p, fo); err != nil {
		return err
	}
	if w.eNonDir|w.eDir != 0 {
		if err := f.singlewatch(p, w.eNonDir|w.eDir, w.eNonDir == 0,
			w.fi); err != nil {
			return err
		}
	} else {
		delete(f.pthLkp, w.p)
	}
	return nil
}

// walk runs f func on each file/dir from p directory.
func (f *fen) walk(p string, fn func(os.FileInfo) error) error {
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
		if err := fn(ls[i]); err != nil {
			return err
		}
	}
	return nil
}

func (f *fen) unwatch(p string, fi os.FileInfo) error {
	if fi.IsDir() {
		err := f.walk(p, func(fi os.FileInfo) error {
			if !fi.IsDir() {
				err := f.singleunwatch(filepath.Join(p, fi.Name()), false)
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
	return f.singleunwatch(p, true)
}

// Watch implements Watcher interface.
func (f *fen) Watch(p string, e Event) error {
	fi, err := os.Stat(p)
	if err != nil {
		return err
	}
	f.Lock()
	err = f.watch(p, e, fi)
	f.Unlock()
	return nil
}

// Unwatch implements Watcher interface.
func (f *fen) Unwatch(p string) error {
	fi, err := os.Stat(p)
	if err != nil {
		return err
	}
	f.Lock()
	err = f.unwatch(p, fi)
	f.Unlock()
	return nil
}

// Rewatch implements Watcher interface.
//
// TODO(rjeczalik): This is a naive hack. Rewrite might help.
func (f *fen) Rewatch(p string, _, e Event) error {
	fi, err := os.Stat(p)
	if err != nil {
		return err
	}
	f.Lock()
	if err = f.unwatch(p, fi); err == nil {
		// TODO(rjeczalik): If watch fails then we leave kqueue in inconsistent
		// state. Handle? Panic? Native version of rewatch?
		err = f.watch(p, e, fi)
	}
	f.Unlock()
	return nil
}
