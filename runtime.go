package notify

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/rjeczalik/fs"
)

// Interface TODO
type Interface interface {
	// Watcher provides a minimum functionality required, that must be implemented
	// for each supported platform.
	Watcher

	// Rewatcher provides an interface for modyfing existing watch-points, like
	// expanding its event set. If the native Watcher does not implement it, it
	// is emulated by the notify Runtime.
	Rewatcher

	// RecusiveWatcher provides an interface for watching directories recursively
	// fo events. If the native Watcher does not implement it, it is emulated by
	// the notify Runtime.
	RecursiveWatcher
}

// Runtime TODO
type Runtime struct {
	tree map[string]interface{}
	ch   map[chan<- EventInfo]string
	stop chan struct{}
	c    <-chan EventInfo
	fs   fs.Filesystem
	i    Interface
}

// NewRuntime TODO
func NewRuntime() *Runtime {
	return NewRuntimeWatcher(NewWatcher(), fs.Default)
}

// NewRuntimeWatcher TODO
func NewRuntimeWatcher(w Watcher, fs fs.Filesystem) *Runtime {
	c := make(chan EventInfo)
	r := &Runtime{
		tree: make(map[string]interface{}),
		ch:   make(map[chan<- EventInfo]string),
		stop: make(chan struct{}),
		c:    c,
		fs:   fs,
	}
	if i, ok := w.(Interface); ok {
		r.i = i
	} else {
		i := struct {
			Watcher
			Rewatcher
			RecursiveWatcher
		}{Watcher: w}
		if rw, ok := w.(RecursiveWatcher); ok {
			i.RecursiveWatcher = rw
		} else {
			i.RecursiveWatcher = Recursive{
				Watcher: w,
				Runtime: r,
			}
		}
		if re, ok := w.(Rewatcher); ok {
			i.Rewatcher = re
		} else {
			i.Rewatcher = Re{
				Watcher: w,
			}
		}
		r.i = i
	}
	r.i.Fanin(c, r.stop)
	go r.loop()
	return r
}

// Watch TODO
func (r *Runtime) Watch(p string, c chan<- EventInfo, events ...Event) (err error) {
	if c == nil {
		panic("notify: Watch using nil channel")
	}
	isrec := false
	if strings.HasSuffix(p, "...") {
		p, isrec = p[:len(p)-3], true
	}
	isdir, err := r.isdir(p)
	if err != nil {
		return
	}
	if isrec && !isdir {
		return &os.PathError{
			Op:   "notify.Watch",
			Path: p,
			Err:  os.ErrInvalid,
		}
	}
	return r.watch(p, joinevents(events), c, isdir, isrec)
}

// Stop TODO
func (r *Runtime) Stop(c chan<- EventInfo) {
	if p, ok := r.ch[c]; ok {
		r.i.Unwatch(p)
	}
}

// Close stops the runtime, resulting it does not send any more notifications.
// For test purposes.
//
// TODO(rjeczalik): Close all channels if we decide to close them on error.
// The Go stdlib does not close user channels, maybe because it does not handle
// errors via channels.
func (r *Runtime) Close() (err error) {
	close(r.stop)
	return
}

func (r *Runtime) loop() {
	for {
		select {
		case ei := <-r.c:
			r.dispatch(ei)
		case <-r.stop:
			return
		}
	}
}

func (r *Runtime) dispatch(ei EventInfo) {
	println("TODO: dispatching event", ei)
}

// Watch registers user's chan in the notification tree and, if needed, sets
// also a watch-point. It tries to do this efficiently, that's why it
// implements a strategy, which covers the following scenarios:
//
//   1. Watch-point exist and its event-set is sufficient.
//   2. Watch-point exist but its event-set needs to be expanded.
//   3. Watch-point exist but is-non recursive and recursive one was requested.
//   3a. Watcher is natively recursive.
//   3b. Watcher is not natively recursive.
//   4. Watch-point does not exist and was requested for a directory, but more
//      than 0 watch-points already exist watching files.
//   5. ?
func (r *Runtime) watch(p string, e Event, c chan<- EventInfo, isdir, isrec bool) error {
	dir, s, err := r.lookup(p)
	if err != nil {
		return err
	}
	_, _ = dir, s
	switch {
	case isdir && isrec: // 1, 2, 3, 4
	case isdir: // 1, 2, 4
		if err = r.i.Watch(p, e); err != nil {
			return err
		}
	default: // 1, 2
		if err = r.i.Watch(p, e); err != nil {
			return err
		}
	}
	r.ch[c] = p
	return nil
}

func (r *Runtime) lookup(p string) (dir map[string]interface{}, s string, err error) {
	dir, s = r.tree, filepath.Base(p)
	fn := func(s string) bool {
		d, ok := dir[s]
		if !ok {
			d := make(map[string]interface{})
			dir[s], dir = d, d
			return true
		}
		if d, ok := d.(map[string]interface{}); ok {
			dir = d
			return true
		}
		return false
	}
	if !walkpath(p, fn) {
		return nil, "", &os.PathError{
			Op:   "notify.Watch",
			Path: p,
			Err:  os.ErrInvalid,
		}
	}
	return
}

func (r *Runtime) walkdir(p string, fn func(string) error) error {
	// TODO
	return errors.New("(*Runtime).walkdir TODO(rjeczalik)")
}

func (r *Runtime) isdir(p string) (bool, error) {
	fi, err := r.fs.Stat(p)
	if err != nil {
		return false, err
	}
	return fi.IsDir(), nil
}
