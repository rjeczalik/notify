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

// Point TODO
type Point struct {
	Name   string
	Parent map[string]interface{}
}

// WatchPointTree TODO
type WatchPointTree struct {
	// FS TODO
	FS fs.Filesystem

	// Cwd TODO
	Cwd Point

	// Root TODO
	Root map[string]interface{}

	paths map[chan<- EventInfo][]string
	stop  chan struct{}
	os    Interface
}

func (w *WatchPointTree) fs() fs.Filesystem {
	if w.FS != nil {
		return w.FS
	}
	return fs.Default
}

func (w *WatchPointTree) setos(wat Watcher) {
	if os, ok := wat.(Interface); ok {
		w.os = os
		return
	}
	os := struct {
		Watcher
		Rewatcher
		RecursiveWatcher
		RecursiveRewatcher
	}{wat, w, w, w}
	if rew, ok := wat.(Rewatcher); ok {
		os.Rewatcher = rew
	}
	if rec, ok := wat.(RecursiveWatcher); ok {
		os.RecursiveWatcher = rec
	}
	if recrew, ok := wat.(RecursiveRewatcher); ok {
		os.RecursiveRewatcher = recrew
	}
	w.os = os

}

func (w *WatchPointTree) dispatch(c <-chan EventInfo) {
	for {
		select {
		case ei := <-c:
			fmt.Println(ei)
		case <-w.stop:
			return
		}
	}
}

// NewWatchPointTree TODO
func NewWatchPointTree(wat Watcher) *WatchPointTree {
	c := make(chan EventInfo, 128)
	w := &WatchPointTree{
		Root:  make(map[string]interface{}),
		paths: make(map[chan<- EventInfo][]string),
		stop:  make(chan struct{}),
	}
	w.setos(wat)
	go w.dispatch(c)
	return w
}

// Watch TODO
func (w *WatchPointTree) Watch(p string, c chan<- EventInfo, e ...Event) error {
	return errors.New("Watch not implemented")
}

// Stop TODO
func (w *WatchPointTree) Stop(c chan<- EventInfo) error {
	return errors.New("Stop not implemented")
}

// Close TODO
func (w *WatchPointTree) Close() error {
	close(w.stop)
	return nil
}

func (w *WatchPointTree) cachep(c chan<- EventInfo, p string) {
	if paths := w.paths[c]; len(paths) == 0 {
		w.paths[c] = []string{p}
	} else {
		switch i := sort.StringSlice(paths).Search(p); {
		case paths[i] == p:
			return
		case len(paths) == i:
			w.paths[c] = append(paths, p)
		default:
			paths = append(paths, "")
			copy(paths[i+1:], paths[i:])
			paths[i], w.paths[c] = p, paths
		}
	}
}

func (w *WatchPointTree) watch(p string, isrec bool, c chan<- EventInfo, e Event) error {
	return errors.New("watch TODO(rjeczalik)")
}

// RecursiveWatch implements notify.RecursiveWatcher interface.
func (w *WatchPointTree) RecursiveWatch(p string, e Event) error {
	return errors.New("RecurisveWatch TODO(rjeczalik)")
}

// RecursiveUnwatch implements notify.RecursiveWatcher interface.
func (w *WatchPointTree) RecursiveUnwatch(p string) error {
	return errors.New("RecurisveUnwatch TODO(rjeczalik)")
}

// Rewatch implements notify.Rewatcher interface.
func (w *WatchPointTree) Rewatch(p string, olde, newe Event) error {
	if err := w.os.Unwatch(p); err != nil {
		return err
	}
	return w.os.Watch(p, newe)
}

// RecursiveRewatch implements notify.RecursiveRewatcher interface.
func (w *WatchPointTree) RecursiveRewatch(oldp, newp string, olde, newe Event) error {
	if err := w.os.RecursiveUnwatch(oldp); err != nil {
		return err
	}
	return w.os.RecursiveWatch(newp, newe)
}

// WalkPointFunc TODO
type WalkPointFunc func(pt Point, last bool) error

func issubpath(path, sub string) bool {
	return strings.HasPrefix(path, sub) && len(path) > len(sub) &&
		path[len(sub)] == os.PathSeparator
}

func (w *WatchPointTree) begin(p string) (d map[string]interface{}, n int) {
	if n := len(w.Cwd.Name); n != 0 {
		if p == w.Cwd.Name {
			return w.Cwd.Parent, n
		}
		if issubpath(p, w.Cwd.Name) {
			return w.Cwd.Parent, n + 1
		}
	}
	vol := filepath.VolumeName(p)
	n = len(vol)
	if n == 0 {
		return w.Root, 1
	}
	d, ok := w.Root[vol].(map[string]interface{})
	if !ok {
		d = make(map[string]interface{})
		w.Root[vol] = d
	}
	return d, n
}

// WalkPoint TODO
//
// WalkPoint expectes the `p` path to be clean.
func (w *WatchPointTree) WalkPoint(p string, fn WalkPointFunc) (err error) {
	parent, i := w.begin(p)
	for j := 0; ; {
		if j = strings.IndexRune(p[i:], os.PathSeparator); j == -1 {
			break
		}
		pt := Point{Name: p[i : i+j], Parent: parent}
		if err = fn(pt, false); err != nil {
			return
		}
		// TODO(rjeczalik): handle edge case where parent[pt.Name] is a file
		cd, ok := parent[pt.Name].(map[string]interface{})
		if !ok {
			cd = make(map[string]interface{})
			parent[pt.Name] = cd
		}
		i += j + 1
		parent = cd
	}
	if i < len(p) {
		err = fn(Point{Name: p[i:], Parent: parent}, true)
	}
	return
}
