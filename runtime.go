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
	ch   map[chan<- EventInfo]map[string]struct{} // cache path registrations per channel
	ev   map[string]Event                         // cache total event set per path
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
		ch:   make(map[chan<- EventInfo]map[string]struct{}),
		ev:   make(map[string]Event),
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
		if re, ok := w.(Rewatcher); ok {
			i.Rewatcher = re
		} else {
			i.Rewatcher = rewatch{
				Watcher: w,
			}
		}
		if rw, ok := w.(RecursiveWatcher); ok {
			i.RecursiveWatcher = rw
		} else {
			i.RecursiveWatcher = recursive{
				Watcher: w,
				Runtime: r,
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
	// TODO Unwatch / rewatch watch-points.
	if paths, ok := r.ch[c]; ok {
		for path := range paths {
			_ = r.i.Unwatch(path) // TODO handle errors?
			// TODO(rjeczalik): If channel c is set to watch the same path, deep
			// in the directory tree at different tree level, doing separate
			// lookups for each is highly inefficient - it should be possible
			// to remove more channel subscription at one lookup.
			var subscribers map[chan<- EventInfo]Event
			parent, name, _ := r.lookup(path) // TODO
			switch v := parent[name].(type) {
			case map[string]interface{}: // dir
				if v, ok := v[""].(map[chan<- EventInfo]Event); ok {
					subscribers = v
				} else {
					// TODO error handling?
				}
			case map[chan<- EventInfo]Event: // file
				subscribers = v
			}
			delete(subscribers, c)
			// BUG Update r.ev[path].
		}
	}
	delete(r.ch, c)
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

func (r *Runtime) tryRewatch(p string, e Event) error {
	if old := r.ev[p]; old&e != e {
		r.ev[p] |= e
		return r.i.Rewatch(p, old, r.ev[p])
	}
	return nil
}

func (r *Runtime) addpath(c chan<- EventInfo, p string) {
	if _, ok := r.ch[c]; !ok {
		r.ch[c] = make(map[string]struct{})
	}
	r.ch[c][p] = struct{}{}
}

// Watch registers user's chan in the notification tree and, if needed, sets
// also a watch-point.
//
// Watching a directory, simple scenarios:
//
//   -
func (r *Runtime) watch(p string, e Event, c chan<- EventInfo, isdir, isrec bool) error {
	parent, name, err := r.lookup(p)
	if err != nil {
		return err
	}
	switch {
	case isdir && isrec:
		return errors.New("(*Runtime).Watch TODO(rjeczalik)")
	case isdir:
		// See whether the channel needs to be subscribded and whether its existing
		// event set is sufficient - only if the directory is being watched.
		if v, ok := parent[name]; ok {
			dir := v.(map[string]interface{})
			// See whether the watch-point and already subscribded channel has
			// sufficient event - expand or rewatch any that hasn't.
			if v, ok = dir[""]; ok {
				subs := v.(map[chan<- EventInfo]Event)
				if subdEvent, ok := subs[c]; ok {
					// If current event set for already subscribded channel is
					// not enough, expand it. If current watch-point has too
					// small event set - rewatch it as well.
					if e&subdEvent != e {
						if err := r.tryRewatch(p, e); err != nil {
							return err
						}
						subs[c] = subdEvent | e
					}
					// Channel already registered, event set ok - move one.
				} else {
					// See if event set is sufficient or expand it if needed.
					if err := r.tryRewatch(p, e); err != nil {
						return err
					}
					subs[c] = e
				}
			} else {
				// Create a watch-point and subscribe the channel. There may
				// be other file subscription within this directory.
				//
				// TODO(rjeczalik): Consider removing watch-points for subscribded
				// channels watching single files within the directory.
				dir[""] = map[chan<- EventInfo]Event{c: e}
				r.addpath(c, p)
				if err := r.i.Watch(p, e); err != nil {
					return err
				}
				r.ev[p] = e
			}
		} else {
			// Create a watch-point and subscribe the channel. There are no other
			// channel subscriptions within the directory.
			parent[name] = map[string]interface{}{"": map[chan<- EventInfo]Event{c: e}}
			r.addpath(c, p)
			if err := r.i.Watch(p, e); err != nil {
				return err
			}
			r.ev[p] = e
		}
	default:
		return errors.New("(*Runtime).Watch TODO(rjeczalik)")
	}
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
