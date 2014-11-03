package notify

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rjeczalik/fs"
)

// ChanPointsMap TODO
type ChanPointsMap map[chan<- EventInfo]*PointSet

func (m ChanPointsMap) Add(c chan<- EventInfo, pt Point) {
	if pts, ok := m[c]; ok {
		pts.Add(pt)
	} else {
		m[c] = &PointSet{pt}
	}
}

func (m ChanPointsMap) Del(c chan<- EventInfo, pt Point) {
	if pts, ok := m[c]; ok {
		if pts.Del(pt); len(*pts) == 0 {
			delete(m, c)
		}
	}
}

// WatchPointTree TODO
type WatchPointTree struct {
	// FS TODO
	FS fs.Filesystem

	// Cwd TODO
	Cwd Point

	// Root TODO
	Root map[string]interface{}

	cpt  ChanPointsMap
	stop chan struct{}
	os   Interface
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
		Root: make(map[string]interface{}),
		cpt:  make(ChanPointsMap),
		stop: make(chan struct{}),
	}
	w.setos(wat)
	go w.dispatch(c)
	return w
}

func (w *WatchPointTree) isdir(p string) (bool, error) {
	fi, err := w.fs().Stat(p)
	if err != nil {
		return false, err
	}
	return fi.IsDir(), nil
}

// Watch TODO
//
// Watch does not support symlinks as it does not care. If user cares, p should
// be passed to os.Readlink first.
func (w *WatchPointTree) Watch(p string, c chan<- EventInfo, e ...Event) error {
	if c == nil {
		panic("notify: Watch using nil channel")
	}
	// TODO(rjeczalik): Make it notify.All when len(e)==0
	if len(e) == 0 {
		panic("notify: Watch using empty event set")
	}
	isrec := false
	if strings.HasSuffix(p, "...") {
		p, isrec = p[:len(p)-3], true
	}
	isdir, err := w.isdir(p)
	if err != nil {
		return err
	}
	if isrec && !isdir {
		return &os.PathError{
			Op:   "notify.Watch",
			Path: p,
			Err:  os.ErrInvalid,
		}
	}
	if p, err = filepath.Abs(p); err != nil {
		return err
	}
	if isrec {
		return w.watchrec(p, isdir, c, joinevents(e))
	}
	return w.watch(p, isdir, c, joinevents(e))
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

func (w *WatchPointTree) register(pt Point, isdir bool, c chan<- EventInfo, e Event) EventDiff {
	var wp WatchPoint
	if isdir {
		dir, ok := pt.Parent[pt.Name].(map[string]interface{})
		if !ok {
			wp = WatchPoint{}
			dir = map[string]interface{}{"": wp}
			pt.Parent[pt.Name] = dir
		} else if wp, ok = dir[""].(WatchPoint); !ok {
			wp = WatchPoint{}
			dir[""] = wp
		}
	} else {
		var ok bool
		if wp, ok = pt.Parent[pt.Name].(WatchPoint); !ok {
			wp = WatchPoint{}
			pt.Parent[pt.Name] = wp
		}
	}
	w.cpt.Add(c, pt)
	return wp.Add(c, e)
}

func (w *WatchPointTree) unregister(pt Point, c chan<- EventInfo) (diff EventDiff) {
	switch v := pt.Parent[pt.Name].(type) {
	case WatchPoint:
		if diff = v.Del(c); diff != None && diff[1] == 0 {
			delete(pt.Parent, pt.Name)
		}
		// TODO(rjeczalik) if len(pt.Parent)==0 it should be removed from its parent
		// so the GC can collect empty nodes.
	case map[string]interface{}:
		if diff = v[""].(WatchPoint).Del(c); diff != None && diff[1] == 0 {
			if delete(v, ""); len(v) == 0 {
				delete(pt.Parent, pt.Name)
			}
		}
	}
	w.cpt.Del(c, pt)
	return
}

func (w *WatchPointTree) watch(p string, isdir bool, c chan<- EventInfo, e Event) error {
	var pt Point
	err := w.WalkPoint(p, func(tmp Point, last bool) error {
		if last {
			pt = tmp
		}
		return nil
	})
	if err != nil {
		return err
	}
	w.Cwd = Point{Name: p, Parent: pt.Parent}
	if diff := w.register(pt, isdir, c, e); diff != None {
		if diff[0] == 0 {
			err = w.os.Watch(p, diff[1])
		} else {
			err = w.os.Rewatch(p, diff[0], diff[1])
		}
	}
	if err != nil {
		w.unregister(pt, c)
		return err
	}
	return nil
}

func (w *WatchPointTree) watchrec(p string, isdir bool, c chan<- EventInfo, e Event) error {
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
