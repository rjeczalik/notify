package notify

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rjeczalik/fs"
)

// None is an empty event diff, think null object.
var None EventDiff

// EventDiff describes a change to an event set - EventDiff[0] is an old state,
// while EventDiff[1] is a new state. If event set has not changed (old == new),
// functions typically return None value.
type EventDiff [2]Event

// Subscriber TODO
//
// The nil key holds total event set - logical sum for all registered events.
// It speeds up computing EventDiff for Subscribe method.
type Subscriber map[chan<- EventInfo]Event

// Subscribe TODO
//
// Subscribe assumes neither c nor e are nil or zero values.
func (sub Subscriber) Subscribe(c chan<- EventInfo, e Event) (diff EventDiff) {
	sub[c] |= e
	diff[0] = sub[nil]
	diff[1] = diff[0] | e
	sub[nil] = diff[1]
	if diff[0] == diff[1] {
		return None
	}
	return
}

// Unsubscribe TODO
func (sub Subscriber) Unsubscribe(c chan<- EventInfo) (diff EventDiff) {
	delete(sub, c)
	diff[0] = sub[nil]
	// Recalculate total event set.
	delete(sub, nil)
	for _, e := range sub {
		diff[1] |= e
	}
	sub[nil] = diff[1]
	if diff[0] == diff[1] {
		return None
	}
	return
}

// Total TODO
func (sub Subscriber) Total() Event {
	return sub[nil]
}

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
	path map[chan<- EventInfo][]string // cache path registrations per channel
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
		path: make(map[chan<- EventInfo][]string),
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
	if paths, ok := r.path[c]; ok {
		// TODO(rjeczalik): If channel c is set to watch the same path, deep
		// in the directory tree at different tree level, doing separate
		// lookups for each is highly inefficient - it should be possible
		// to remove more channel subscription at one lookup.
		for _, path := range paths {
			var sub Subscriber
			parent, name, _ := r.lookup(path) // TODO error?
			switch v := parent[name].(type) {
			case map[string]interface{}: // dir
				// NOTE !ok means channel was already removed. Bug?
				sub = v[""].(Subscriber)
			case Subscriber:
				sub = v
			default:
				panic("bug")
			}
			if diff := sub.Unsubscribe(c); diff != None {
				if diff[1] == 0 {
					_ = r.i.Unwatch(path) // TODO error?
				} else {
					_ = r.i.Rewatch(path, diff[0], diff[1]) // TODO error?
				}
			}
		}
		delete(r.path, c)
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

func (r *Runtime) cachepath(c chan<- EventInfo, p string) {
	if paths := r.path[c]; len(paths) == 0 {
		r.path[c] = []string{p}
	} else {
		switch i := sort.StringSlice(paths).Search(p); {
		case i == len(paths):
			r.path[c] = append(r.path[c], p)
		case paths[i] == p:
			return
		default:
			paths = append(paths, "")
			copy(paths[i+1:], paths[i:])
			paths[i], r.path[c] = p, paths
		}
	}
}

// Watch registers user's chan in the notification tree and, if needed, sets
// also a watch-point.
func (r *Runtime) watch(p string, e Event, c chan<- EventInfo, isdir, isrec bool) error {
	parent, name, err := r.lookup(p)
	if err != nil {
		return err
	}
	switch {
	case isdir && isrec:
		return errors.New("(*Runtime).Watch TODO(rjeczalik)")
	case isdir:
		var dir map[string]interface{}
		var sub Subscriber
		var err error
		// Get or create dir for the current watch-point.
		if v, ok := parent[name].(map[string]interface{}); ok {
			dir = v
		} else {
			dir = map[string]interface{}{}
			parent[name] = dir
		}
		// Get or create subscribers map for the current watch-point
		if v, ok := dir[""].(Subscriber); ok {
			sub = v
		} else {
			sub = Subscriber{}
			dir[""] = sub
		}
		// Subscribe channel to the current watch-point.
		if diff := sub.Subscribe(c, e); diff != None {
			if diff[0] == 0 {
				// No existing watch-point for the path, create new one.
				err = r.i.Watch(p, diff[1])
			} else {
				// Not sufficient event set, expand it.
				err = r.i.Rewatch(p, diff[0], diff[1])
			}
			if err != nil {
				return err
			}
		}
		// Cache some values. Probably temporary until the data structure
		// for the watch-point tree gets reimplemented.
		r.cachepath(c, p)
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
