package notify

import (
	"path/filepath"
	"strings"
)

const all = ^Event(0)

// must panics if err is non-nil.
func must(err error) {
	if err != nil {
		panic(err)
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

// watchAdd TODO(rjeczalik)
func watchAdd(nd node, c chan<- EventInfo, e Event) eventDiff {
	diff := nd.Watch.Add(c, e)
	if wp := nd.Child[""].Watch; len(wp) != 0 {
		e = wp.Total()
		diff[0] |= e
		diff[1] |= e
		if diff[0] == diff[1] {
			return none
		}
	}
	return diff
}

// watchAddInactive TODO(rjeczalik)
func watchAddInactive(nd node, c chan<- EventInfo, e Event) eventDiff {
	wp := nd.Child[""].Watch
	if wp == nil {
		wp = make(watchpoint)
		nd.Child[""] = node{Watch: wp}
	}
	diff := wp.Add(c, e)
	e = nd.Watch.Total()
	diff[0] |= e
	diff[1] |= e
	if diff[0] == diff[1] {
		return none
	}
	return diff
}

// watchCopy TODO(rjeczalik)
func watchCopy(src, dst node) {
	for c, e := range src.Watch {
		if c == nil {
			continue
		}
		watchAddInactive(dst, c, e)
	}
	if wpsrc := src.Child[""].Watch; len(wpsrc) != 0 {
		wpdst := dst.Child[""].Watch
		for c, e := range wpsrc {
			if c == nil {
				continue
			}
			wpdst.Add(c, e)
		}
	}
}

// watchDel TODO(rjeczalik)
func watchDel(nd node, c chan<- EventInfo, e Event) eventDiff {
	diff := nd.Watch.Del(c, e)
	if wp := nd.Child[""].Watch; len(wp) != 0 {
		diffInactive := wp.Del(c, e)
		e = wp.Total()
		diff[0] |= diffInactive[0] | e
		diff[1] |= diffInactive[1] | e
		if diff[0] == diff[1] {
			return none
		}
	}
	return diff
}

// watchTotal TODO(rjeczalik)
func watchTotal(nd node) Event {
	e := nd.Watch.Total()
	if wp := nd.Child[""].Watch; len(wp) != 0 {
		e |= wp.Total()
	}
	return e
}

// watchIsRecursive TODO(rjeczalik)
func watchIsRecursive(nd node) bool {
	ok := nd.Watch.IsRecursive()
	if wp := nd.Child[""].Watch; len(wp) != 0 {
		ok = ok || wp.IsRecursive()
	}
	return ok
}

// recursiveTree TODO(rjeczalik): lock root
type recursiveTree struct {
	root root
	// TODO(rjeczalik): merge watcher + recursiveWatcher after #5 and #6
	w interface {
		watcher
		recursiveWatcher
	}
	c chan EventInfo
}

// newRecursiveTree TODO(rjeczalik)
func newRecursiveTree(w recursiveWatcher, c chan EventInfo) *recursiveTree {
	t := &recursiveTree{
		root: root{nd: newnode("")},
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
		dbg.Printf("dispatching %v on %q", ei.Event(), ei.Path())
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
			dbg.Print("dispatch did not reach leaf:", err)
			continue
		}
		nd.Watch.Dispatch(ei, false) // notify parent watchpoint
		if nd, ok = nd.Child[base]; ok {
			nd.Watch.Dispatch(ei, false) // notify leaf watchpoint
		}
	}
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
		if watchTotal(nd) != 0 {
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
		switch diff := watchAddInactive(parent, c, eventset); {
		case diff == none:
			return nil
		case diff[0] == 0:
			panic("dangling watchpoint: " + parent.Name)
		default:
			if isrec || watchIsRecursive(parent) {
				err = t.w.RecursiveRewatch(parent.Name, parent.Name, diff[0], diff[1])
			} else {
				err = t.w.Rewatch(parent.Name, diff[0], diff[1])
			}
			if err != nil {
				// TODO(rjeczalik): it should be possible to keep all watchpoints
				// only in the actual watch node; currently it's split between
				// parent watch node (inactive watchpoints) and the children nodes
				// (active ones).
				watchDel(parent, c, diff.Event())
				return err
			}
			watchAdd(cur, c, eventset)
			// TODO(rjeczalik): traverse tree instead; eventually track minimum
			// subtree root per each chan
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
		// no child watches, cur holds a new watch
	case 1:
		watchAdd(cur, c, eventset) // TODO(rjeczalik): update cache c subtree root?
		watchCopy(children[0], cur)
		err = t.w.RecursiveRewatch(children[0].Name, cur.Name, watchTotal(children[0]),
			watchTotal(cur))
		if err != nil {
			// Clean inactive watchpoint. The c chan did not exist before.
			cur.Child[""] = node{}
			delete(cur.Watch, c)
			return err
		}
		return nil
	default:
		watchAdd(cur, c, eventset)
		// Copy children inactive watchpoints to the new parent.
		for _, nd := range children {
			watchCopy(nd, cur)
		}
		// Watch parent subtree.
		if err = t.w.RecursiveWatch(cur.Name, watchTotal(cur)); err != nil {
			// Clean inactive watchpoint. The c chan did not exist before.
			cur.Child[""] = node{}
			delete(cur.Watch, c)
			return err
		}
		// Unwatch children subtrees.
		var e error
		for _, nd := range children {
			if watchIsRecursive(nd) {
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
	if diff := watchAdd(cur, c, eventset); diff[0] == 0 {
		if isrec {
			err = t.w.RecursiveWatch(cur.Name, diff[1])
		} else {
			err = t.w.Watch(cur.Name, diff[1])
		}
		if err != nil {
			watchDel(cur, c, diff.Event())
			return err
		}
	}
	return nil
}

// Stop TODO(rjeczalik)
func (t *recursiveTree) Stop(c chan<- EventInfo) {
	var err error
	fn := func(nd node) (e error) {
		switch diff := watchDel(nd, c, all); {
		case diff == none:
			return nil
		case diff[1] == 0:
			if watchIsRecursive(nd) {
				e = t.w.RecursiveUnwatch(nd.Name)
			} else {
				e = t.w.Unwatch(nd.Name)
			}
		default:
			if watchIsRecursive(nd) {
				e = t.w.RecursiveRewatch(nd.Name, nd.Name, diff[0], diff[1])
			} else {
				e = t.w.Rewatch(nd.Name, diff[0], diff[1])
			}
		}
		fn := func(nd node) error {
			watchDel(nd, c, all)
			return nil
		}
		err = nonil(err, e, nd.Walk(fn))
		return skip
	}
	// TODO(rjeczalik): use max root per c
	if e := t.root.Walk("", fn); e != nil {
		err = nonil(err, e)
	}
	dbg.Printf("Stop(%p) error: %v\n", c, err)
}

// Close TODO(rjeczalik)
func (t *recursiveTree) Close() error {
	err := t.w.Close()
	close(t.c)
	return err
}
