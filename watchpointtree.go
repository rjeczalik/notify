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

// WatchNode TODO
type WatchNode struct {
	Name   string
	Parent map[string]interface{}
}

// Value TODO
func (nd WatchNode) Value() interface{} {
	if nd.Name != "" {
		return nd.Parent[nd.Name]
	}
	return nd.Parent // Special case for the root node.
}

// Set TODO
func (nd WatchNode) Set(value interface{}) {
	// NOTE(rjeczalik): debug check, remove in final version
	if nd.Name == "" {
		panic("[DEBUG] notify: can't set root node")
	}
	nd.Parent[nd.Name] = value
}

// Del TODO
func (nd WatchNode) Del() {
	// NOTE(rjeczalik): debug check, remove in final version
	if nd.Name == "" {
		panic("[DEBUG] notify: can't delete root node")
	}
	delete(nd.Parent, nd.Name)
}

func mknode(nd WatchNode, names []string) WatchNode {
	for i := range names {
		// TODO(rjeczalik): node is a Watchpoint? (file)
		child, ok := nd.Value().(map[string]interface{})
		if !ok {
			child = make(map[string]interface{})
			nd.Set(child)
		}
		nd = WatchNode{Name: names[i], Parent: child}
	}
	return nd
}

func mknodes(nd WatchNode, names []string) []WatchNode {
	nodes := make([]WatchNode, len(names)+1)
	nodes[0] = nd
	for i := range names {
		// TODO(rjeczalik): node is a Watchpoint? (file)
		child, ok := nd.Value().(map[string]interface{})
		if !ok {
			child = make(map[string]interface{})
			nd.Set(child)
		}
		nd = WatchNode{Name: names[i], Parent: child}
		nodes[i+1] = nd
	}
	return nodes
}

// Point is a node which name is absolute path within the tree.
type Path WatchNode

func (p Path) WatchNode() WatchNode { return WatchNode{Name: filepath.Base(p.Name), Parent: p.Parent} }

func mkpath(p string, base Path) (nd WatchNode) {
	if p == base.Name {
		return base.WatchNode()
	}
	n := len(base.Name)
	if strings.HasPrefix(p, base.Name) && p[n] == os.PathSeparator {
		return mknode(base.WatchNode(), strings.Split(p[n+1:], sep))
	}
	panic("notify: invalid path")
}

type PathSlice []Path

func (p PathSlice) Len() int           { return len(p) }
func (p PathSlice) Less(i, j int) bool { return p[i].Name >= p[j].Name }
func (p PathSlice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p PathSlice) Sort()              { sort.Sort(p) }

// WatchNodeSet TODO
type WatchNodeSet []WatchNode

func (p WatchNodeSet) Len() int           { return len(p) }
func (p WatchNodeSet) Less(i, j int) bool { return p[i].Name < p[j].Name }
func (p WatchNodeSet) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

func (p WatchNodeSet) Search(nd WatchNode) int {
	return sort.Search(len(p), func(i int) bool { return p[i].Name >= nd.Name })
}

func (p *WatchNodeSet) Add(nd WatchNode) {
	switch i := p.Search(nd); {
	case i == len(*p):
		*p = append(*p, nd)
	case (*p)[i].Name == nd.Name:
	default:
		*p = append(*p, WatchNode{})
		copy((*p)[i+1:], (*p)[i:])
		(*p)[i] = nd
	}
}

func (p *WatchNodeSet) Del(nd WatchNode) {
	if i, n := p.Search(nd), len(*p); i != n && (*p)[i].Name == nd.Name {
		copy((*p)[i:], (*p)[i+1:])
		*p = (*p)[:n-1]
	}
}

// ChanWatchNodesMap TODO
type ChanWatchNodesMap map[chan<- EventInfo]*WatchNodeSet

func (m ChanWatchNodesMap) Add(c chan<- EventInfo, nd WatchNode) {
	if nds, ok := m[c]; ok {
		nds.Add(nd)
	} else {
		m[c] = &WatchNodeSet{nd}
	}
}

func (m ChanWatchNodesMap) Del(c chan<- EventInfo, nd WatchNode) {
	if nds, ok := m[c]; ok {
		if nds.Del(nd); len(*nds) == 0 {
			delete(m, c)
		}
	}
}

// WalkPathNodeFunc TODO
type WalkPathNodeFunc func(nd WatchNode, isbase bool) error

// WalkWatchNodeFunc TODO
type WalkWatchNodeFunc func(nd WatchNode, p string) error

// WatchpointTree TODO
type WatchpointTree struct {
	FS   fs.Filesystem          // TODO
	Cwd  Path                   // TODO
	Root map[string]interface{} // TODO

	cnd  ChanWatchNodesMap
	stop chan struct{}
	os   Interface
}

func (w *WatchpointTree) fs() fs.Filesystem {
	if w.FS != nil {
		return w.FS
	}
	return fs.Default
}

func (w *WatchpointTree) setos(wat Watcher) {
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

func (w *WatchpointTree) dispatch(c <-chan EventInfo) {
	for {
		select {
		case ei := <-c:
			fmt.Println(ei)
		case <-w.stop:
			return
		}
	}
}

// NewWatchpointTree TODO
func NewWatchpointTree(wat Watcher) *WatchpointTree {
	c := make(chan EventInfo, 128)
	w := &WatchpointTree{
		Root: make(map[string]interface{}),
		cnd:  make(ChanWatchNodesMap),
		stop: make(chan struct{}),
	}
	w.setos(wat)
	go w.dispatch(c)
	return w
}

func (w *WatchpointTree) isdir(p string) (bool, error) {
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
func (w *WatchpointTree) Watch(p string, c chan<- EventInfo, e ...Event) error {
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
func (w *WatchpointTree) Stop(c chan<- EventInfo) error {
	return errors.New("Stop not implemented")
}

// Close TODO
func (w *WatchpointTree) Close() error {
	close(w.stop)
	return nil
}

func (w *WatchpointTree) register(nd WatchNode, isdir bool, c chan<- EventInfo, e Event) EventDiff {
	var wp Watchpoint
	if isdir {
		dir, ok := nd.Value().(map[string]interface{})
		if !ok {
			wp = Watchpoint{}
			dir = map[string]interface{}{"": wp}
			nd.Set(dir)
		} else if wp, ok = dir[""].(Watchpoint); !ok {
			wp = Watchpoint{}
			dir[""] = wp
		}
	} else {
		var ok bool
		if wp, ok = nd.Value().(Watchpoint); !ok {
			wp = Watchpoint{}
			nd.Set(wp)
		}
	}
	w.cnd.Add(c, nd)
	return wp.Add(c, e)
}

func (w *WatchpointTree) unregister(nd WatchNode, c chan<- EventInfo, e Event) (diff EventDiff) {
	switch v := nd.Value().(type) {
	case Watchpoint:
		if diff = v.Del(c, e); diff != None && diff[1] == 0 {
			nd.Del()
		}
		// TODO(rjeczalik) if len(nd.Parent)==0 it should be removed from its parent
		// so the GC can collect empty nodes.
	case map[string]interface{}:
		if diff = v[""].(Watchpoint).Del(c, e); diff != None && diff[1] == 0 {
			if delete(v, ""); len(v) == 0 {
				nd.Del()
			}
		}
	default:
		panic("[DEBUG] notify: invalid node type")
	}
	w.cnd.Del(c, nd)
	return
}

func (w *WatchpointTree) watch(p string, isdir bool, c chan<- EventInfo, e Event) (err error) {
	nd := mknode(w.begin(p))
	// TODO(rjeczalik): check if any of the parents are being watched recursively
	// and the event set is sufficient.
	if diff := w.register(nd, isdir, c, e); diff != None {
		if diff[0] == 0 {
			err = w.os.Watch(p, diff[1])
		} else {
			err = w.os.Rewatch(p, diff[0], diff[1])
		}
	}
	if err != nil {
		w.unregister(nd, c, e)
		return err
	}
	return nil
}

func (w *WatchpointTree) watchrec(p string, isdir bool, c chan<- EventInfo, e Event) error {
	return errors.New("watch TODO(rjeczalik)")
}

// RecursiveWatch implements notify.RecursiveWatcher interface.
func (w *WatchpointTree) RecursiveWatch(p string, e Event) error {
	return errors.New("RecurisveWatch TODO(rjeczalik)")
}

// RecursiveUnwatch implements notify.RecursiveWatcher interface.
func (w *WatchpointTree) RecursiveUnwatch(p string) error {
	return errors.New("RecurisveUnwatch TODO(rjeczalik)")
}

// Rewatch implements notify.Rewatcher interface.
func (w *WatchpointTree) Rewatch(p string, olde, newe Event) error {
	if err := w.os.Unwatch(p); err != nil {
		return err
	}
	return w.os.Watch(p, newe)
}

// RecursiveRewatch implements notify.RecursiveRewatcher interface.
func (w *WatchpointTree) RecursiveRewatch(oldp, newp string, olde, newe Event) error {
	if err := w.os.RecursiveUnwatch(oldp); err != nil {
		return err
	}
	return w.os.RecursiveWatch(newp, newe)
}

func (w *WatchpointTree) begin(p string) (nd WatchNode, names []string) {
	nd.Parent, nd.Name = w.Root, filepath.VolumeName(p)
	if p = p[len(nd.Name)+1:]; p != "" {
		names = strings.Split(p, sep)
	}
	return
}

// WalkPath TODO
func (w *WatchpointTree) WalkPath(p string, fn WalkPathNodeFunc) (err error) {
	it, dirs := w.begin(p)
	ok := false
	for i := range dirs {
		if err = fn(it, false); err != nil {
			return
		}
		if it.Parent, ok = it.Value().(map[string]interface{}); !ok {
			return &PathError{Name: p}
		}
		it.Name = dirs[i]
	}
	if err = fn(it, true); err != nil {
		return
	}
	return
}

// WalkWatchpointFunc TODO
type WalkWatchpointFunc func(wp Watchpoint, base bool) error

// WalkWatchpoint TODO
//
// No, duplicated code here is ok.
func (w *WatchpointTree) WalkWatchpoint(p string, fn WalkWatchpointFunc) (err error) {
	it, dirs := w.begin(p)
	ok := false
	for i := range dirs {
		if wp, ok := it.Parent[""].(Watchpoint); ok {
			if err = fn(wp, false); err != nil {
				return
			}
		}
		if it.Parent, ok = it.Value().(map[string]interface{}); !ok {
			return &PathError{Name: p}
		}
		it.Name = dirs[i]
	}
	if wp, ok := it.Parent[""].(Watchpoint); ok {
		if err = fn(wp, true); err != nil {
			return
		}
	}
	switch v := it.Value().(type) {
	case Watchpoint:
		if err = fn(v, true); err != nil {
			return
		}
	case map[string]interface{}:
		if wp, ok := v[""].(Watchpoint); ok {
			if err = fn(wp, true); err != nil {
				return
			}
		}
	default:
		panic("[DEBUG] notify: invalid node type")
	}
	return nil
}

// MakePath TODO
func (w *WatchpointTree) MakePath(p string, fn WalkPathNodeFunc) error {
	nodes := mknodes(w.begin(p))
	n := len(nodes) - 1
	for i := range nodes {
		if err := fn(nodes[i], i == n); err != nil {
			return err
		}
	}
	return nil
}

// MakeTree TODO
func (w *WatchpointTree) MakeTree(p string, fn WalkWatchNodeFunc) error {
	base := Path{Name: p, Parent: mknode(w.begin(p)).Parent}
	if err := fn(base.WatchNode(), p); err != nil {
		return err
	}
	glob, dir := []string{p}, ""
	for n := len(glob); n != 0; n = len(glob) {
		dir, glob = glob[n-1], glob[:n-1]
		f, err := w.FS.Open(dir)
		if err != nil {
			return err
		}
		fis, err := f.Readdir(0)
		if err != nil {
			f.Close()
			return err
		}
		for _, fi := range fis {
			if fi.IsDir() {
				// TODO(rjeczalik): Fix memfs and get rid of filepath.Base.
				path := dir + sep + filepath.Base(fi.Name())
				glob = append(glob, path)
				// TODO(rjeczalik): revert nodes on failure
				if err := fn(mkpath(path, base), path); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// BFS TODO
//
// NOTE(rjeczalik): Used only for test/debugging purposes, should be move out
// from here.
func (w *WatchpointTree) BFS(fn func(interface{}) error) error {
	dir := (map[string]interface{})(nil)
	glob := []map[string]interface{}{w.Root}
	for n := len(glob); n != 0; n = len(glob) {
		dir, glob = glob[n-1], glob[:n-1]
		for _, v := range dir {
			if err := fn(v); err != nil {
				return err
			}
			dir, ok := v.(map[string]interface{})
			if ok {
				glob = append(glob, dir)
			}
		}
	}
	return nil
}
