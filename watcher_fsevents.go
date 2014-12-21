// +build darwin,!kqueue
// +build !fsnotify

package notify

import (
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
)

var errAlreadyWatched = errors.New("path is already watched")
var errNotWatched = errors.New("path is not being watched")

type watch struct {
	c      chan<- EventInfo
	stream *Stream
	path   string
	events uint32
	isrec  int32
}

func (w *watch) Dispatch(ev []FSEvent) {
	events := atomic.LoadUint32(&w.events)
	isrec := (atomic.LoadInt32(&w.isrec) == 1)
	for i := range ev {
		e := Event(ev[i].Flags & events)
		if e == 0 {
			continue
		}
		if !strings.HasPrefix(ev[i].Path, w.path) {
			continue
		}
		if n := len(w.path); len(ev[i].Path) > n {
			if ev[i].Path[n] != '/' {
				continue
			}
			if !isrec && strings.IndexByte(ev[i].Path[n+1:], '/') != -1 {
				continue
			}
		}
		select {
		case w.c <- &event{
			fse:   ev[i],
			event: e,
			isdir: ev[i].Flags&FSEventsIsDir != 0,
		}:
		default:
			fmt.Println(i, "[DEBUG] (*watch).Dispatch: blocked", w.c)
		}
	}
}

type fsevents struct {
	watches map[string]*watch
	c       chan<- EventInfo
}

func newWatcher() Watcher {
	return &fsevents{
		watches: make(map[string]*watch),
	}
}

func (fse *fsevents) watch(path string, event Event, isrec int32) error {
	if _, ok := fse.watches[path]; ok {
		return errAlreadyWatched
	}
	w := &watch{
		c:      fse.c,
		path:   path,
		events: uint32(event),
		isrec:  isrec,
	}
	w.stream = NewStream(path, w.Dispatch)
	if err := w.stream.Start(); err != nil {
		return err
	}
	fse.watches[path] = w
	return nil
}

func (fse *fsevents) Watch(path string, event Event) error {
	return fse.watch(path, event, 0)
}

func (fse *fsevents) Unwatch(path string) error {
	w, ok := fse.watches[path]
	if !ok {
		return errNotWatched
	}
	w.stream.Stop()
	delete(fse.watches, path)
	return nil
}

func (fse *fsevents) Rewatch(path string, oldevent, newevent Event) error {
	w, ok := fse.watches[path]
	if !ok {
		return errNotWatched
	}
	if !atomic.CompareAndSwapUint32(&w.events, uint32(oldevent), uint32(newevent)) {
		return errors.New("invalid event state diff")
	}
	return nil
}

func (fse *fsevents) Dispatch(c chan<- EventInfo, stop <-chan struct{}) {
	fse.c = c
	go func() {
		<-stop
		fse.Stop()
	}()
}

func (fse *fsevents) RecursiveWatch(path string, event Event) error {
	return fse.watch(path, event, 1)
}

func (fse *fsevents) RecursiveUnwatch(path string) error {
	// TODO(rjeczalik): fail if w.isrec == 0?
	return fse.Unwatch(path)
}

func (fse *fsevents) RecursiveRewatch(oldpath, newpath string, oldevent, newevent Event) error {
	switch [2]bool{oldpath == newpath, oldevent == newevent} {
	case [2]bool{true, true}:
		w, ok := fse.watches[oldpath]
		if !ok {
			return errNotWatched
		}
		atomic.CompareAndSwapInt32(&w.isrec, 0, 1)
		return nil
	case [2]bool{true, false}:
		w, ok := fse.watches[oldpath]
		if !ok {
			return errNotWatched
		}
		if !atomic.CompareAndSwapUint32(&w.events, uint32(oldevent), uint32(newevent)) {
			return errors.New("invalid event state diff")
		}
		atomic.CompareAndSwapInt32(&w.isrec, 0, 1)
		return nil
	default:
		// TODO(rjeczalik): rewatch newpath only if exists?
		if _, ok := fse.watches[newpath]; ok {
			return errAlreadyWatched
		}
		if err := fse.Unwatch(oldpath); err != nil {
			return err
		}
		// TODO(rjeczalik): revert unwatch if watch fails?
		return fse.watch(newpath, newevent, 1)
	}
}

func (fse *fsevents) Stop() {
	for _, w := range fse.watches {
		w.stream.Stop()
	}
}
