package notify

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rjeczalik/fs"
)

// Interface TODO
type Interface interface {
	// Watcher provides a minimum functionality required, that must be implemented
	// for each supported platform.
	Watcher

	// Rewatcher provides an interface for modyfing existing watch-points, like
	// expanding its event set.
	//
	// If the native Watcher does not implement it, it is emulated by the notify
	// Runtime.
	Rewatcher

	// RecusiveWatcher provides an interface for watching directories recursively
	// for events.
	//
	// If the native Watcher does not implement it, it is emulated by the notify
	// Runtime.
	RecursiveWatcher

	// RecursiveRewatcher provides an interface for relocating and/or modifying
	// existing watch-points.
	//
	// If the native Watcher does not implement it, it is emulated by the notify
	// Runtime.
	RecursiveRewatcher
}

// Runtime TODO
type Runtime struct {
	tree map[string]interface{}
	path map[chan<- EventInfo][]string // cache path registrations per channel
	stop chan struct{}
	c    <-chan EventInfo
	fs   fs.Filesystem
	os   Interface
}

// NewRuntime TODO
func NewRuntime() *Runtime {
	return NewRuntimeWatcher(NewWatcher(), fs.Default)
}

// NewRuntimeWatcher TODO
func NewRuntimeWatcher(w Watcher, fs fs.Filesystem) *Runtime {
	c := make(chan EventInfo, 128)
	r := &Runtime{
		tree: make(map[string]interface{}),
		path: make(map[chan<- EventInfo][]string),
		stop: make(chan struct{}),
		c:    c,
		fs:   fs,
	}
	r.buildos(w)
	r.os.Dispatch(c, r.stop)
	go r.loop()
	return r
}

func (r *Runtime) buildos(w Watcher) {
	if os, ok := w.(Interface); ok {
		r.os = os
		return
	}
	os := struct {
		Watcher
		Rewatcher
		RecursiveWatcher
		RecursiveRewatcher
	}{w, r, r, r}
	if rew, ok := w.(Rewatcher); ok {
		os.Rewatcher = rew
	}
	if rec, ok := w.(RecursiveWatcher); ok {
		os.RecursiveWatcher = rec
	}
	if recrew, ok := w.(RecursiveRewatcher); ok {
		os.RecursiveRewatcher = recrew
	}
	r.os = os
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
		// to remove more channel wpscription at one lookup.
		for _, path := range paths {
			var wp Watchpoint
			parent, name, _ := r.buildpath(path) // TODO error?
			switch v := parent[name].(type) {
			case map[string]interface{}: // dir
				wp = v[""].(Watchpoint)
			case Watchpoint:
				wp = v
			}
			if diff := wp.Del(c); diff != None {
				if diff[1] == 0 {
					if err := r.os.Unwatch(path); err != nil {
						panic("notify: Unwatch failed with: " + err.Error())
					}
				} else {
					if err := r.os.Rewatch(path, diff[0], diff[1]); err != nil {
						panic("notify: Rewatch failed with: " + err.Error())
					}
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

// RecursiveWatch implements notify.RecursiveWatcher interface.
func (r *Runtime) RecursiveWatch(p string, e Event) error {
	return errors.New("RecurisveWatch TODO(rjeczalik)")
}

// RecursiveUnwatch implements notify.RecursiveWatcher interface.
func (r *Runtime) RecursiveUnwatch(p string) error {
	return errors.New("RecurisveUnwatch TODO(rjeczalik)")
}

// Rewatch implements notify.Rewatcher interface.
func (r *Runtime) Rewatch(p string, olde, newe Event) error {
	if err := r.os.Unwatch(p); err != nil {
		return err
	}
	return r.os.Watch(p, newe)
}

// RecursiveRewatch implements notify.RecursiveRewatcher interface.
func (r *Runtime) RecursiveRewatch(oldp, newp string, olde, newe Event) error {
	if err := r.os.RecursiveUnwatch(oldp); err != nil {
		return err
	}
	return r.os.RecursiveWatch(newp, newe)
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
	parent, name, err := r.buildpath(ei.FileName())
	if err != nil {
		fmt.Println(err)
		return
	}
	var wp Watchpoint
	switch v := parent[name].(type) {
	case Watchpoint:
		wp = v
	case map[string]interface{}:
		if v, ok := v[""].(Watchpoint); ok {
			wp = v
		}
	}
	if wp != nil {
		wp.Dispatch(ei, false)
	}
	if wp, ok := parent[""].(Watchpoint); ok {
		wp.Dispatch(ei, false)
	}
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
	parent, name, err := r.buildpath(p)
	if err != nil {
		return err
	}
	switch {
	case isdir && isrec:
		return errors.New("(*Runtime).Watch TODO(rjeczalik)")
	case isdir:
		var dir map[string]interface{}
		var wp Watchpoint
		var err error
		// Get or create dir for the current watch-point.
		if v, ok := parent[name].(map[string]interface{}); ok {
			dir = v
		} else {
			dir = map[string]interface{}{}
			parent[name] = dir
		}
		// Get or create watchpoints map for the current watch-point
		if v, ok := dir[""].(Watchpoint); ok {
			wp = v
		} else {
			wp = Watchpoint{}
			dir[""] = wp
		}
		// Add channel to the current watch-point.
		if diff := wp.Add(c, e); diff != None {
			if diff[0] == 0 {
				// No existing watch-point for the path, create new one.
				//
				// TODO(rjeczalik): Move file watchpoints to newly created dir one?
				err = r.os.Watch(p, diff[1])
			} else {
				// Not sufficient event set, expand it.
				err = r.os.Rewatch(p, diff[0], diff[1])
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

func (r *Runtime) walkwp(p string, fn func(string, wp Watchpoint) bool) bool {
	return false
}

func (r *Runtime) buildpath(p string) (parent map[string]interface{}, s string, err error) {
	s = filepath.Base(p)
	dir := r.tree
	fn := func(s string) bool {
		parent = dir
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
