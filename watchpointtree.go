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

// Node TODO
type Node struct {
	Name   string
	Parent map[string]interface{}
}

// Point is a node which name is absolute path within the tree.
type Path Node

func (p Path) Node() Node { return Node{Name: filepath.Base(p.Name), Parent: p.Parent} }

type PathSlice []Path

func (p PathSlice) Len() int           { return len(p) }
func (p PathSlice) Less(i, j int) bool { return p[i].Name >= p[j].Name }
func (p PathSlice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p PathSlice) Sort()              { sort.Sort(p) }

// NodeSet TODO
type NodeSet []Node

func (p NodeSet) Len() int           { return len(p) }
func (p NodeSet) Less(i, j int) bool { return p[i].Name < p[j].Name }
func (p NodeSet) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

func (p NodeSet) Search(nd Node) int {
	return sort.Search(len(p), func(i int) bool { return p[i].Name >= nd.Name })
}

func (p *NodeSet) Add(nd Node) {
	switch i := p.Search(nd); {
	case i == len(*p):
		*p = append(*p, nd)
	case (*p)[i].Name == nd.Name:
	default:
		*p = append(*p, Node{})
		copy((*p)[i+1:], (*p)[i:])
		(*p)[i] = nd
	}
}

func (p *NodeSet) Del(nd Node) {
	if i, n := p.Search(nd), len(*p); i != n && (*p)[i].Name == nd.Name {
		copy((*p)[i:], (*p)[i+1:])
		*p = (*p)[:n-1]
	}
}

// ChanNodesMap TODO
type ChanNodesMap map[chan<- EventInfo]*NodeSet

func (m ChanNodesMap) Add(c chan<- EventInfo, nd Node) {
	if nds, ok := m[c]; ok {
		nds.Add(nd)
	} else {
		m[c] = &NodeSet{nd}
	}
}

func (m ChanNodesMap) Del(c chan<- EventInfo, nd Node) {
	if nds, ok := m[c]; ok {
		if nds.Del(nd); len(*nds) == 0 {
			delete(m, c)
		}
	}
}

// WatchPointTree TODO
type WatchPointTree struct {
	FS   fs.Filesystem          // TODO
	Cwd  Path                   // TODO
	Root map[string]interface{} // TODO

	cnd  ChanNodesMap
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
		cnd:  make(ChanNodesMap),
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

func (w *WatchPointTree) register(nd Node, isdir bool, c chan<- EventInfo, e Event) EventDiff {
	var wp WatchPoint
	if isdir {
		dir, ok := nd.Parent[nd.Name].(map[string]interface{})
		if !ok {
			wp = WatchPoint{}
			dir = map[string]interface{}{"": wp}
			nd.Parent[nd.Name] = dir
		} else if wp, ok = dir[""].(WatchPoint); !ok {
			wp = WatchPoint{}
			dir[""] = wp
		}
	} else {
		var ok bool
		if wp, ok = nd.Parent[nd.Name].(WatchPoint); !ok {
			wp = WatchPoint{}
			nd.Parent[nd.Name] = wp
		}
	}
	w.cnd.Add(c, nd)
	return wp.Add(c, e)
}

func (w *WatchPointTree) unregister(nd Node, c chan<- EventInfo) (diff EventDiff) {
	switch v := nd.Parent[nd.Name].(type) {
	case WatchPoint:
		if diff = v.Del(c); diff != None && diff[1] == 0 {
			delete(nd.Parent, nd.Name)
		}
		// TODO(rjeczalik) if len(nd.Parent)==0 it should be removed from its parent
		// so the GC can collect empty nodes.
	case map[string]interface{}:
		if diff = v[""].(WatchPoint).Del(c); diff != None && diff[1] == 0 {
			if delete(v, ""); len(v) == 0 {
				delete(nd.Parent, nd.Name)
			}
		}
	}
	w.cnd.Del(c, nd)
	return
}

func (w *WatchPointTree) watch(p string, isdir bool, c chan<- EventInfo, e Event) (err error) {
	nd := w.lookup(p)
	if diff := w.register(nd, isdir, c, e); diff != None {
		if diff[0] == 0 {
			err = w.os.Watch(p, diff[1])
		} else {
			err = w.os.Rewatch(p, diff[0], diff[1])
		}
	}
	if err != nil {
		w.unregister(nd, c)
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

func issubpath(path, sub string) bool {
	return strings.HasPrefix(path, sub) && len(path) > len(sub) &&
		path[len(sub)] == os.PathSeparator
}

func (w *WatchPointTree) cwd(p string, nds ...Node) (map[string]interface{}, int) {
	for _, nd := range nds {
		if n := len(nd.Name); n != 0 {
			if p == nd.Name {
				return nd.Parent, n
			}
			if issubpath(p, nd.Name) {
				return nd.Parent, n + 1
			}
		}
	}
	return nil, 0
}

func (w *WatchPointTree) root(p string) (d map[string]interface{}, n int) {
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

func mknode(nd Node, names []string) Node {
	for i := range names {
		// TODO(rjeczalik): node is a WatchPoint? (file)
		child, ok := nd.Parent[nd.Name].(map[string]interface{})
		if !ok {
			child = make(map[string]interface{})
			nd.Parent[nd.Name] = child
		}
		nd = Node{Name: names[i], Parent: child}
	}
	return nd
}

func mknodes(nd Node, names []string) []Node {
	nodes := make([]Node, len(names)+1)
	nodes[0] = nd
	for i := range names {
		// TODO(rjeczalik): node is a WatchPoint? (file)
		child, ok := nd.Parent[nd.Name].(map[string]interface{})
		if !ok {
			child = make(map[string]interface{})
			nd.Parent[nd.Name] = child
		}
		nd = Node{Name: names[i], Parent: child}
		nodes[i+1] = nd
	}
	return nodes
}

func (w *WatchPointTree) begin(p string) (Node, []string) {
	vol := filepath.VolumeName(p)
	names := ([]string)(nil)
	if p = p[len(vol)+1:]; p != "" {
		names = strings.Split(p, sep)
	}
	root := w.Root
	if vol != "" {
		ok := false
		if root, ok = w.Root[vol].(map[string]interface{}); !ok {
			root = make(map[string]interface{})
			w.Root[vol] = root
		}
	}
	switch n := len(names); {
	case n == 0 && vol != "":
		return Node{Parent: root, Name: vol}, nil
	case n == 0:
		return Node{Parent: root, Name: sep}, nil
	case n == 1:
		return Node{Parent: root, Name: names[0]}, nil
	default:
		return Node{Parent: root, Name: names[0]}, names[1:]
	}
}

// BFS TODO
//
// NOTE(rjeczalik): Used only for test/debugging purposes, should be move out
// from here.
func (w *WatchPointTree) BFS(fn func(interface{}) error) error {
	// TODO(rjeczalik): get rid of VolumeName
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

// WalkNodeFunc TODO
type WalkNodeFunc func(nd Node, last bool) error

// WalkNode TODO
//
// WalkNode expectes the `p` path to be clean.
func (w *WatchPointTree) WalkNode(p string, fn WalkNodeFunc) (err error) {
	nodes := mknodes(w.begin(p))
	n := len(nodes) - 1
	for i := range nodes {
		if err = fn(nodes[i], i == n); err != nil {
			return
		}
	}
	return
}

func subindex(path, subpath string) int {
	// TODO
	return -1
}

func (w *WatchPointTree) lookup(p string, paths ...Path) Node {
	// TODO(rjeczalik): measure if using waypoints makes sense
	if len(paths) != 0 {
		PathSlice(paths).Sort()
		for _, path := range paths {
			if n := len(path.Name); n != 0 {
				if p == path.Name {
					return path.Node()
				}
				if i := subindex(p, path.Name); i != -1 {
					return mknode(path.Node(), strings.Split(p[i:], sep))
				}
			}

		}
	}
	return mknode(w.begin(p))
}
