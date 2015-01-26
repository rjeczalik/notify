package notify

import (
	"path/filepath"
	"strings"
)

// must panics if err is non-nil.
func must(err error) {
	if err != nil {
		panic(err)
	}
}

// recursiveTree TODO(rjeczalik)
type recursiveTree struct {
	root root
	// TODO(rjeczalik): merge watcher + recursiveWatcher after #5 and #6
	w interface {
		watcher
		recursiveWatcher
	}
	c   chan EventInfo
	cnd chanNodesMap // TODO(rjeczalik): replace with subtree scanning
}

// newRecursiveTree TODO(rjeczalik)
func newRecursiveTree(w recursiveWatcher, c chan EventInfo) *recursiveTree {
	t := &recursiveTree{
		root: root{nd: newnode("")},
		cnd:  make(chanNodesMap),
		w: struct {
			watcher
			recursiveWatcher
		}{w.(watcher), w},
		c: c,
	}
	go t.dispatch(c)
	return t
}

// dispatch TODO(rjeczalik)
func (t *recursiveTree) dispatch(c <-chan EventInfo) {
	nd, ok := node{}, false
	for ei := range c {
		dir, base := split(ei.Path())
		fn := func(it node, isbase bool) error {
			if isbase {
				nd = it
			} else {
				it.Watch.Dispatch(ei, true) // notify recursively nodes on the path
			}
			return nil
		}
		if err := t.root.WalkPath(dir, fn); err != nil {
			continue
		}
		nd.Watch.Dispatch(ei, false) // notify parent watchpoint
		if nd, ok = nd.Child[base]; ok {
			nd.Watch.Dispatch(ei, false) // notify leaf watchpoint
		}
	}
}

// cleanpath TODO(rjeczalik): move to util.go
func cleanpath(path string) (realpath string, isrec bool, err error) {
	if strings.HasSuffix(path, "...") {
		isrec = true
		path = path[:len(path)-3]
	}
	if path, err = filepath.Abs(path); err != nil {
		return "", false, err
	}
	if path, err = canonical(path); err != nil {
		return "", false, err
	}
	return path, isrec, nil
}

// Watch TODO(rjeczalik)
func (t *recursiveTree) Watch(path string, c chan<- EventInfo, events ...Event) error {
	if c == nil {
		panic("notify: Watch using nil channel")
	}
	// Expanding with empty event set is a nop.
	if len(events) == 0 {
		return nil
	}
	path, isrec, err := cleanpath(path)
	if err != nil {
		return err
	}
	eventset := joinevents(events)
	if isrec {
		eventset |= recursive
	}
	// case 1: cur is a child
	//
	// Look for parent watch which already covers the given path.
	parent := node{}
	err = t.root.WalkPath(path, func(nd node, _ bool) error {
		if nd.Watch.Total() != 0 {
			parent = nd
			return skip
		}
		return nil
	})
	cur := t.root.Add(path) // add after the walk, so it's less to traverse
	if err == nil && parent.Watch != nil {
		// Parent watch found. Register inactive watchpoint, so we have enough
		// information to shrink the eventset on eventual Stop.
		// return t.resetwatchpoint(parent, parent, c, eventset|inactive)
		switch diff := parent.Watch.Add(c, eventset|inactive); {
		case diff == none:
			return nil
		case diff[0] == 0:
			panic("dangling watchpoint: " + parent.Name)
		default:
			if parent.Watch.IsRecursive() || isrec {
				err = t.w.RecursiveRewatch(parent.Name, parent.Name, diff[0], diff[1])
			} else {
				err = t.w.Rewatch(parent.Name, diff[0], diff[1])
			}
			if err != nil {
				// TODO(rjeczalik): it should be possible to keep all watchpoints
				// only in the actual watch node; currently it's split between
				// parent watch node (inactive watchpoints) and the children nodes
				// (active ones).
				parent.Watch.Del(c, diff.Event())
				return err
			}
			cur.Watch.Add(c, eventset)
			// TODO(rjeczalik): traverse tree instead; eventually track minimum
			// subtree root per each chan
			t.cnd.Add(c, cur)    // rm
			t.cnd.Add(c, parent) // rm
			return nil
		}
	}
	// case 2: cur is new parent
	//
	// Look for children nodes, unwatch n-1 of them and rewatch the last one.
	children := nodeSet{}
	fn := func(nd node) error {
		if len(nd.Watch) == 0 {
			return nil
		}
		children.Add(nd)
		return skip
	}
	switch must(cur.Walk(fn)); len(children) {
	case 0:
	case 1:
		cur.Watch.Add(c, eventset) // TODO(rjeczalik): update cache c subtree root?
		for c, e := range children[0].Watch {
			if c == nil {
				continue
			}
			cur.Watch.Add(c, e|inactive)
		}
		err = t.w.RecursiveRewatch(children[0].Name, cur.Name, children[0].Watch.Total(),
			cur.Watch.Total())
		if err != nil {
			for c := range cur.Watch {
				delete(cur.Watch, c)
			}
			return err
		}
		return nil
	default:
		cur.Watch.Add(c, eventset)
		// Copy children inactive watchpoints to new parent.
		for _, nd := range children {
			for c, e := range nd.Watch {
				// TODO(rjeczalik): remove rec chan
				if c == nil {
					continue
				}
				cur.Watch.Add(c, e|inactive)
			}
		}
		// Watch parent subtree.
		if err = t.w.RecursiveWatch(cur.Name, cur.Watch.Total()); err != nil {
			for c := range cur.Watch {
				delete(cur.Watch, c)
			}
			return err
		}
		// Unwatch children subtrees.
		var e error
		for _, nd := range children {
			if nd.Watch.IsRecursive() {
				e = t.w.RecursiveUnwatch(nd.Name)
			} else {
				e = t.w.Unwatch(nd.Name)
			}
			if e != nil {
				err = nonil(err, e)
				// TODO(rjeczalik): child is still watched, warn all its watchpoints
				// about possible duplicate events via Error event
			}
		}
		return err
	}
	// case 3: cur is new, alone node
	if diff := cur.Watch.Add(c, eventset); diff[0] == 0 {
		if isrec {
			err = t.w.RecursiveWatch(cur.Name, diff[1])
		} else {
			err = t.w.Watch(cur.Name, diff[1])
		}
		if err != nil {
			cur.Watch.Del(c, diff.Event())
			return err
		}
	}
	return nil
}

// Stop TODO(rjeczalik)
func (t *recursiveTree) Stop(c chan<- EventInfo) {
	// TODO

}

// Close TODO(rjeczalik)
func (t *recursiveTree) Close() error {
	err := t.w.Close()
	close(t.c)
	return err
}
