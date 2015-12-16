// Copyright (c) 2014-2015 The Notify Authors. All rights reserved.
// Use of this source code is governed by the MIT license that can be
// found in the LICENSE file.

// +build darwin,kqueue dragonfly freebsd netbsd openbsd solaris

// TODO: Doc + cleanup

package notify

import (
	"os"
	"path/filepath"
	"strings"
)

func newWatcher(c chan<- EventInfo) watcher {
	t := newTrg(c)
	if err := t.init(); err != nil {
		panic(err)
	}
	go t.monitor()
	return t
}

// Close implements watcher.
func (t *trg) Close() error {
	return t.close()
}

// sendEvents sends reported events one by one through chan.
func (t *trg) sendEvents(evn []event) {
	for i := range evn {
		t.c <- &evn[i]
	}
}

// watch starts to watch given p file/directory.
func (t *trg) singlewatch(p string, e Event, direct bool, fi os.FileInfo) (err error) {
	w, ok := t.pthLkp[p]
	if !ok {
		if w, err = t.addwatch(p, fi); err != nil {
			return
		}
	}
	if direct {
		w.eDir |= e
	} else {
		w.eNonDir |= e
	}
	var ee int64
	if e&Create != 0 && fi.IsDir() {
		ee = int64(nWrite)
	}
	if err = t.natwatch(fi, w, encode(w.eDir|w.eNonDir)|ee); err != nil {
		return
	}
	if !ok {
		t.assign(w)
		return nil
	}
	return errAlreadyWatched
}

// decode converts event received from native to notify.Event
// representation taking into account requested events (w).
func decode(o int64, w Event) (e Event) {
	for f, n := range nat2not {
		if o&int64(f) != 0 {
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

func (t *trg) watch(p string, e Event, fi os.FileInfo) error {
	if err := t.singlewatch(p, e, true, fi); err != nil {
		if err != errAlreadyWatched {
			return nil
		}
	}
	if fi.IsDir() {
		err := t.walk(p, func(fi os.FileInfo) (err error) {
			if err = t.singlewatch(filepath.Join(p, fi.Name()), e, false,
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

// walk runs f func on each file/dir from p directory.
func (t *trg) walk(p string, fn func(os.FileInfo) error) error {
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

func (t *trg) unwatch(p string, fi os.FileInfo) error {
	if fi.IsDir() {
		err := t.walk(p, func(fi os.FileInfo) error {
			if !fi.IsDir() {
				err := t.singleunwatch(filepath.Join(p, fi.Name()), false)
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
	return t.singleunwatch(p, true)
}

// Watch implements Watcher interface.
func (t *trg) Watch(p string, e Event) error {
	fi, err := os.Stat(p)
	if err != nil {
		return err
	}
	t.Lock()
	err = t.watch(p, e, fi)
	t.Unlock()
	return err
}

// Unwatch implements Watcher interface.
func (t *trg) Unwatch(p string) error {
	fi, err := os.Stat(p)
	if err != nil {
		return err
	}
	t.Lock()
	err = t.unwatch(p, fi)
	t.Unlock()
	return err
}

// Rewatch implements Watcher interface.
//
// TODO(rjeczalik): This is a naive hack. Rewrite might help.
func (t *trg) Rewatch(p string, _, e Event) error {
	fi, err := os.Stat(p)
	if err != nil {
		return err
	}
	t.Lock()
	if err = t.unwatch(p, fi); err == nil {
		// TODO(rjeczalik): If watch fails then we leave kqueue in inconsistent
		// state. Handle? Panic? Native version of rewatch?
		err = t.watch(p, e, fi)
	}
	t.Unlock()
	return nil
}

func (*trg) file(w *watched, n interface{}, e Event) (evn []event) {
	evn = append(evn, event{w.p, e, w.fi.IsDir(), n})
	return
}

func (t *trg) dir(w *watched, n interface{}, e, ge Event) (evn []event) {
	// If it's dir and delete we have to send it and continue, because
	// other processing relies on opening (in this case not existing) dir.
	// Events for contents of this dir are reported by kqueue.
	// However events for rename must be generated for all monitored files
	// inside of moved directory, because kqueue does not report it independently
	// for each file descriptor being moved in result of move action on
	// parent dirLiczba dostępnych dni urlopowych: 0ectory.
	if (ge & (nRename | nRemove)) != 0 {
		// Write is reported also for Remove on directory. Because of that
		// we have to filter it out explicitly.
		evn = append(evn, event{w.p, e & ^Write & ^nWrite, true, n})
		if ge&nRename != 0 {
			for p, wt := range t.pthLkp {
				if strings.HasPrefix(p, w.p+string(os.PathSeparator)) {
					if err := t.unwatch(p, wt.fi); err != nil && err != errNotWatched &&
						!os.IsNotExist(err) {
						dbgprintf("trg: failed stop watching moved file (%q): %q\n",
							p, err)
					}
					if (w.eDir|w.eNonDir)&(nRename|Rename) != 0 {
						evn = append(evn, event{
							p, (w.eDir | w.eNonDir) & e &^ Write &^ nWrite,
							w.fi.IsDir(), nil,
						})
					}
				}
			}
		}
		t.del(w)
		return
	}
	if (ge & nWrite) != 0 {
		switch err := t.walk(w.p, func(fi os.FileInfo) error {
			p := filepath.Join(w.p, fi.Name())
			switch err := t.singlewatch(p, w.eDir, false, fi); {
			case os.IsNotExist(err) && ((w.eDir & Remove) != 0):
				evn = append(evn, event{p, Remove, fi.IsDir(), n})
			case err == errAlreadyWatched:
			case err != nil:
				dbgprintf("trg: watching %q failed: %q", p, err)
			case (w.eDir & Create) != 0:
				evn = append(evn, event{p, Create, fi.IsDir(), n})
			default:
			}
			return nil
		}); {
		case os.IsNotExist(err):
			return
		case err != nil:
			dbgprintf("trg: dir processing failed: %q", err)
		default:
		}
	}
	return
}

// unwatch stops watching p file/directory.
func (t *trg) singleunwatch(p string, direct bool) error {
	w, ok := t.pthLkp[p]
	if !ok {
		return errNotWatched
	}
	if direct {
		w.eDir = 0
	} else {
		w.eNonDir = 0
	}
	if err := t.natunwatch(w); err != nil {
		return err
	}
	if w.eNonDir|w.eDir != 0 {
		if err := t.singlewatch(p, w.eNonDir|w.eDir, w.eNonDir == 0,
			w.fi); err != nil && err != errAlreadyWatched {
			return err
		}
	} else {
		t.del(w)
	}
	return nil
}

// process event returned by port_get call.
func (t *trg) process(n interface{}) (evn []event) {
	t.Lock()
	w, ge := t.watched(n)
	if w == nil {
		t.Unlock()
		dbgprintf("trg: %v event for not registered ", Event(ge))
		return
	}

	e := decode(ge, w.eDir|w.eNonDir)
	if ge&int64(nDelRen) == 0 {
		switch fi, err := os.Stat(w.p); {
		case err != nil:
		default:
			if err = t.natwatch(fi, w, (encode(w.eDir | w.eNonDir))); err != nil {
				dbgprintf("trg: %q is no longer watched: %q", w.p, err)
				t.del(w)
			}
		}
	}

	if w.fi.IsDir() {
		evn = append(evn, t.dir(w, n, e, Event(ge))...)
	} else {
		evn = append(evn, t.file(w, n, e)...)
	}
	if Event(ge)&nDelRen != 0 {
		t.del(w)
	}
	t.Unlock()
	return
}
