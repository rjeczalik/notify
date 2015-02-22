package notify

import (
	"runtime"
	"sync"
)

// nonrecursiveTree TODO(rjeczalik)
type nonrecursiveTree struct {
	rw   sync.RWMutex // protects root
	root root
	w    watcher
	c    chan EventInfo
	rec  chan EventInfo
}

// newNonrecursiveTree TODO(rjeczalik)
func newNonrecursiveTree(w watcher, c chan EventInfo) *nonrecursiveTree {
	t := &nonrecursiveTree{
		root: root{nd: newnode("")},
		w:    w,
		c:    c,
		rec:  make(chan EventInfo, max(cap(c), 2*runtime.NumCPU())),
	}
	go t.dispatch()
	go t.internal()
	return t
}

// dispatch TODO(rjeczalik)
func (t *nonrecursiveTree) dispatch() {
	for ei := range t.c {
		dbg.Printf("dispatching %v on %q", ei.Event(), ei.Path())
		go func(ei EventInfo) {
			var parent node
			dir, base := split(ei.Path())
			fn := func(nd node, isbase bool) error {
				if isbase {
					parent = nd
				} else {
					nd.Watch.Dispatch(ei, recursive)
				}
				return nil
			}
			t.rw.RLock()
			defer t.rw.RUnlock()
			// Notify recursive watchpoints found on the path.
			if err := t.root.WalkPath(dir, fn); err != nil {
				dbg.Print("dispatch did not reach leaf:", err)
				return
			}
			// Notify parent watchpoint.
			parent.Watch.Dispatch(ei, 0)
			// If leaf watchpoint exists, notify it.
			if nd, ok := parent.Child[base]; ok {
				nd.Watch.Dispatch(ei, 0)
				return
			}
			// TODO rm
			if ei.Event() != Create {
				return
			}
			if isdir, err := ei.(isDirer).isDir(); err != nil || !isdir {
				return
			}
			parent.Watch.Dispatch(ei, internal)
			dbg.Printf("watching ad-hoc newly created directory: %s", ei.Path())
		}(ei)
	}
}

// internal TODO(rjeczalik)
func (t *nonrecursiveTree) internal() {
	for ei := range t.rec {
		t.rw.Lock()
		dbg.Printf("(*nonrecursiveTree).internal()=%v\n", ei)
		// TODO
		t.rw.Unlock()
	}
}

// watchAdd TODO(rjeczalik)
func (t *nonrecursiveTree) watchAdd(nd node, c chan<- EventInfo, e Event) eventDiff {
	if e&recursive != 0 {
		diff := nd.Watch.Add(t.rec, e|Create)
		nd.Watch.Add(c, e)
		return diff
	}
	return nd.Watch.Add(c, e)
}

// watchDel TODO(rjeczalik)
func (t *nonrecursiveTree) watchDel(nd node, c chan<- EventInfo, e Event) eventDiff {
	rec, ok := nd.Watch[t.rec]
	if ok {
		delete(nd.Watch, t.rec)
	}
	diff := nd.Watch.Del(c, e)
	if ok {
		switch rec &^= diff.Event(); {
		case !nd.Watch.IsRecursive():
			// no recursive watchpoint anymore
		case rec|recursive == recursive:
			// no more events, only internal ones
		default:
			nd.Watch.Add(t.rec, rec)
		}
	}
	return diff
}

// watchRec TODO(rjeczalik)
func (t *nonrecursiveTree) watchRec(nd node) Event {
	return nd.Watch[t.rec]
}

// Watch TODO(rjeczalik)
func (t *nonrecursiveTree) Watch(path string, c chan<- EventInfo, events ...Event) error {
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
	t.rw.Lock()
	defer t.rw.Unlock()
	nd := t.root.Add(path)
	if isrec {
		return t.watchrec(nd, c, eventset|recursive)
	}
	return t.watch(nd, c, eventset)
}

func (t *nonrecursiveTree) watch(nd node, c chan<- EventInfo, e Event) (err error) {
	diff := nd.Watch.Add(c, e)
	switch {
	case diff == none:
		return nil
	case diff[1] == 0:
		// TODO(rjeczalik): cleanup this panic after implementation is stable
		panic("eventset is empty: " + nd.Name)
	case diff[0] == 0:
		err = t.w.Watch(nd.Name, diff[1])
	default:
		err = t.w.Rewatch(nd.Name, diff[0], diff[1])
	}
	if err != nil {
		nd.Watch.Del(c, diff.Event())
		return err
	}
	return nil
}

func (t *nonrecursiveTree) watchrec(nd node, c chan<- EventInfo, e Event) error {
	var traverse func(walkFunc) error
	// Non-recursive tree listens on Create event for every recursive
	// watchpoint in order to automagically set a watch for every
	// created directory.
	switch diff := nd.Watch.dryAdd(t.rec, e|Create); {
	case diff == none:
		t.watchAdd(nd, c, e)
		return nil
	case diff[1] == 0:
		// TODO(rjeczalik): cleanup this panic after implementation is stable
		panic("eventset is empty: " + nd.Name)
	case diff[0] == 0:
		// TODO(rjeczalik): BFS into directories and skip subtree as soon as first
		// recursive watchpoint is encountered.
		traverse = nd.AddDir
	default:
		traverse = nd.Walk
	}
	// TODO(rjeczalik): account every path that failed to be (re)watched
	// and retry.
	fn := func(nd node) error {
		switch diff := nd.Watch.Add(t.rec, e|Create); {
		case diff == none:
		case diff[1] == 0:
			// TODO(rjeczalik): cleanup this panic after implementation is stable
			panic("eventset is empty: " + nd.Name)
		case diff[0] == 0:
			t.w.Watch(nd.Name, diff[1])
		default:
			t.w.Rewatch(nd.Name, diff[0], diff[1])
		}
		return nil
	}
	if err := traverse(fn); err != nil {
		return err
	}
	t.watchAdd(nd, c, e)
	return nil
}

// Stop TODO(rjeczalik)
func (t *nonrecursiveTree) Stop(c chan<- EventInfo) {
	// TODO
}

// Close TODO(rjeczalik)
func (t *nonrecursiveTree) Close() error {
	err := t.w.Close()
	close(t.c)
	return err
}
