// Copyright (c) 2014-2015 The Notify Authors. All rights reserved.
// Use of this source code is governed by the MIT license that can be
// found in the LICENSE file.

package notify

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// nonrecursiveTree TODO(rjeczalik)
type nonrecursiveTree struct {
	ops        sync.RWMutex // excludes public Watch/Stop from maintenance, without blocking dispatch
	rw         sync.RWMutex // protects root
	done       chan struct{}
	dispatched chan struct{}
	maintained chan struct{}
	closeOnce  sync.Once
	closeErr   error
	root       root
	w          watcher
	c          chan EventInfo
	rec        chan EventInfo
}

// newNonrecursiveTree TODO(rjeczalik)
func newNonrecursiveTree(w watcher, c, rec chan EventInfo) *nonrecursiveTree {
	if rec == nil {
		rec = make(chan EventInfo, buffer)
	}
	t := &nonrecursiveTree{
		root:       root{nd: newnode("")},
		done:       make(chan struct{}),
		dispatched: make(chan struct{}),
		maintained: make(chan struct{}),
		w:          w,
		c:          c,
		rec:        rec,
	}
	go t.dispatch(c)
	go t.internal(rec)
	return t
}

// dispatch TODO(rjeczalik)
func (t *nonrecursiveTree) dispatch(c <-chan EventInfo) {
	defer close(t.dispatched)
	for ei := range c {
		select {
		case <-t.done:
			continue // Drain backend senders until Close closes c.
		default:
		}
		dbgprintf("dispatching %v on %q", ei.Event(), ei.Path())
		// Keep directory lifecycle events ordered before sending them to rec.
		func(ei EventInfo) {
			var nd node
			var isrec bool
			dir, base := split(ei.Path())
			fn := func(it node, isbase bool) error {
				isrec = isrec || it.Watch.IsRecursive()
				if isbase {
					nd = it
				} else {
					it.Watch.Dispatch(ei, recursive)
				}
				return nil
			}
			t.rw.RLock()
			// Notify recursive watchpoints found on the path.
			if err := t.root.WalkPath(dir, fn); err != nil {
				dbgprint("dispatch did not reach leaf:", err)
				t.rw.RUnlock()
				return
			}
			// Notify parent watchpoint.
			nd.Watch.Dispatch(ei, 0)
			isrec = isrec || nd.Watch.IsRecursive()
			// If leaf watchpoint exists, notify it.
			if nd, ok := nd.Child[base]; ok {
				isrec = isrec || nd.Watch.IsRecursive()
				nd.Watch.Dispatch(ei, 0)
			}
			t.rw.RUnlock()
			// If the event describes newly leaf directory created within
			if !isrec || ei.Event()&(Create|Remove) == 0 {
				return
			}
			if ok, err := ei.(isDirer).isDir(); !ok || err != nil {
				return
			}
			select {
			case t.rec <- ei:
			case <-t.done:
			}
		}(ei)
	}
}

// maintenanceWorkers bounds concurrent filesystem work. The scheduler permits
// unrelated subtrees to progress together, but keeps conflicting lifecycle work
// in arrival order. Its pending queue is bounded by buffer, like the tree input.
const maintenanceWorkers = 8

func relatedPaths(a, b string) bool {
	return a == b || indexrel(a, b) != -1 || indexrel(b, a) != -1 ||
		(len(a) > 0 && a[len(a)-1] == os.PathSeparator && strings.HasPrefix(b, a)) ||
		(len(b) > 0 && b[len(b)-1] == os.PathSeparator && strings.HasPrefix(a, b))
}

func (t *nonrecursiveTree) internal(rec <-chan EventInfo) {
	defer close(t.maintained)
	work := make(chan EventInfo)
	finished := make(chan string, maintenanceWorkers)
	var workers sync.WaitGroup
	for j := 0; j < maintenanceWorkers; j++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for ei := range work {
				t.ops.RLock()
				t.maintain(ei)
				t.ops.RUnlock()
				finished <- ei.Path()
			}
		}()
	}
	defer func() { close(work); workers.Wait() }()
	pending := make([]EventInfo, 0, buffer)
	active := make([]string, 0, maintenanceWorkers)
	for {
		if t.stopping() {
			return
		}
		// A later job may bypass an earlier job only if their subtrees are disjoint.
		next := -1
		if len(active) < maintenanceWorkers {
			for j, ei := range pending {
				conflict := false
				for _, path := range active {
					if relatedPaths(path, ei.Path()) {
						conflict = true
						break
					}
				}
				if !conflict {
					for _, earlier := range pending[:j] {
						if relatedPaths(earlier.Path(), ei.Path()) {
							conflict = true
							break
						}
					}
				}
				if !conflict {
					next = j
					break
				}
			}
		}
		var out chan EventInfo
		var ei EventInfo
		if next != -1 {
			out = work
			ei = pending[next]
		}
		var in <-chan EventInfo
		if len(pending) < buffer {
			in = rec
		}
		select {
		case <-t.done:
			return
		case ei, ok := <-in:
			if !ok {
				return
			}
			pending = append(pending, ei)
		case out <- ei:
			active = append(active, ei.Path())
			copy(pending[next:], pending[next+1:])
			pending[len(pending)-1] = nil
			pending = pending[:len(pending)-1]
		case path := <-finished:
			for j, p := range active {
				if p == path {
					active = append(active[:j], active[j+1:]...)
					break
				}
			}
		}
	}
}

func (t *nonrecursiveTree) stopping() bool {
	select {
	case <-t.done:
		return true
	default:
		return false
	}
}

func (t *nonrecursiveTree) maintain(ei EventInfo) {
	if t.stopping() {
		return
	}
	if ei.Event() == Remove {
		t.rw.Lock()
		nd, err := t.root.Get(ei.Path())
		if err == nil {
			t.root.Del(ei.Path())
		}
		t.rw.Unlock()
		if err == nil {
			// This subtree is now detached. Related maintenance is serialized and
			// ops excludes Watch/Stop, so no other goroutine can mutate its nodes.
			t.walkWatchpoint(nd, func(_ Event, nd node) error {
				if t.stopping() {
					return errSkip
				}
				t.w.Unwatch(nd.Name)
				return nil
			})
		}
		return
	}
	// Keep ops for the whole traversal so Stop/Watch cannot change the inherited
	// mask between publishing a node and installing its backend watch. Release rw
	// before every backend call and directory read, allowing user delivery to run.
	stack := []string{ei.Path()}
	for len(stack) != 0 && !t.stopping() {
		path := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		t.rw.Lock()
		var eset = internal
		t.root.WalkPath(path, func(it node, _ bool) error {
			if e := it.Watch[t.rec]; e != 0 && e > eset {
				eset = e
			}
			return nil
		})
		if eset == internal {
			t.rw.Unlock()
			continue
		}
		nd := t.root.Add(path)
		diff := nd.Watch.Add(t.rec, eset|omit|Create)
		t.rw.Unlock()
		switch {
		case diff == none:
		case diff[0] == 0:
			t.w.Watch(path, diff[1])
		default:
			t.w.Rewatch(path, diff[0], diff[1])
		}
		if t.stopping() {
			return
		}
		entries, err := os.ReadDir(path)
		if err != nil {
			dbgprintf("internal(%p) error: %v", t.rec, err)
			continue
		}
		for _, entry := range entries {
			if entry.Type()&(fs.ModeSymlink|fs.ModeDir) == fs.ModeDir {
				stack = append(stack, filepath.Join(path, entry.Name()))
			}
		}
	}
}

// watchAdd TODO(rjeczalik)
func (t *nonrecursiveTree) watchAdd(nd node, c chan<- EventInfo, e Event) eventDiff {
	if e&recursive != 0 {
		diff := nd.Watch.Add(t.rec, e|Create|omit)
		nd.Watch.Add(c, e)
		return diff
	}
	return nd.Watch.Add(c, e)
}

// watchDelMin TODO(rjeczalik)
func (t *nonrecursiveTree) watchDelMin(min Event, nd node, c chan<- EventInfo, e Event) eventDiff {
	old, ok := nd.Watch[t.rec]
	if ok {
		nd.Watch[t.rec] = min
	}
	diff := nd.Watch.Del(c, e)
	if ok {
		switch old &^= diff[0] &^ diff[1]; {
		case old|internal == internal:
			delete(nd.Watch, t.rec)
			if set, ok := nd.Watch[nil]; ok && len(nd.Watch) == 1 && set == 0 {
				delete(nd.Watch, nil)
			}
		default:
			nd.Watch.Add(t.rec, old|Create)
			switch {
			case diff == none:
			case diff[1]|Create == diff[0]:
				diff = none
			default:
				diff[1] |= Create
			}
		}
	}
	return diff
}

// watchDel TODO(rjeczalik)
func (t *nonrecursiveTree) watchDel(nd node, c chan<- EventInfo, e Event) eventDiff {
	return t.watchDelMin(0, nd, c, e)
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
	eset := joinevents(events)
	t.ops.Lock()
	defer t.ops.Unlock()
	if t.stopping() {
		return errors.New("notify: tree is closed")
	}
	t.rw.Lock()
	defer t.rw.Unlock()
	nd := t.root.Add(path)
	if isrec {
		return t.watchrec(nd, c, eset|recursive)
	}
	return t.watch(nd, c, eset)
}

func (t *nonrecursiveTree) watch(nd node, c chan<- EventInfo, e Event) (err error) {
	diff := nd.Watch.Add(c, e)
	switch {
	case diff == none:
		return nil
	case diff[1] == 0:
		// TODO(rjeczalik): cleanup this panic after implementation is stable
		panic("eset is empty: " + nd.Name)
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

func (t *nonrecursiveTree) recFunc(e Event) walkFunc {
	return func(nd node) error {
		switch diff := nd.Watch.Add(t.rec, e|omit|Create); {
		case diff == none:
		case diff[1] == 0:
			// TODO(rjeczalik): cleanup this panic after implementation is stable
			panic("eset is empty: " + nd.Name)
		case diff[0] == 0:
			t.w.Watch(nd.Name, diff[1])
		default:
			t.w.Rewatch(nd.Name, diff[0], diff[1])
		}
		return nil
	}
}

func (t *nonrecursiveTree) watchrec(nd node, c chan<- EventInfo, e Event) error {
	var traverse func(walkFunc) error
	// Non-recursive tree listens on Create event for every recursive
	// watchpoint in order to automagically set a watch for every
	// created directory.
	switch diff := nd.Watch.dryAdd(t.rec, e|Create); {
	case diff == none:
		t.watchAdd(nd, c, e)
		nd.Watch.Add(t.rec, e|omit|Create)
		return nil
	case diff[1] == 0:
		// TODO(rjeczalik): cleanup this panic after implementation is stable
		panic("eset is empty: " + nd.Name)
	case diff[0] == 0:
		// TODO(rjeczalik): BFS into directories and skip subtree as soon as first
		// recursive watchpoint is encountered.
		traverse = nd.AddDir
	default:
		traverse = nd.Walk
	}
	// TODO(rjeczalik): account every path that failed to be (re)watched
	// and retry.
	if err := traverse(t.recFunc(e)); err != nil {
		return err
	}
	t.watchAdd(nd, c, e)
	return nil
}

type walkWatchpointFunc func(Event, node) error

func (t *nonrecursiveTree) walkWatchpoint(nd node, fn walkWatchpointFunc) error {
	type minode struct {
		min Event
		nd  node
	}
	mnd := minode{nd: nd}
	stack := []minode{mnd}
Traverse:
	for n := len(stack); n != 0; n = len(stack) {
		mnd, stack = stack[n-1], stack[:n-1]
		// There must be no recursive watchpoints if the node has no watchpoints
		// itself (every node in subtree rooted at recursive watchpoints must
		// have at least nil (total) and t.rec watchpoints).
		if len(mnd.nd.Watch) != 0 {
			switch err := fn(mnd.min, mnd.nd); err {
			case nil:
			case errSkip:
				continue Traverse
			default:
				return err
			}
		}
		for _, nd := range mnd.nd.Child {
			stack = append(stack, minode{mnd.nd.Watch[t.rec], nd})
		}
	}
	return nil
}

// Stop TODO(rjeczalik)
func (t *nonrecursiveTree) Stop(c chan<- EventInfo) {
	t.ops.Lock()
	defer t.ops.Unlock()
	fn := func(min Event, nd node) error {
		// TODO(rjeczalik): aggregate watcher errors and retry; in worst case
		// forward to the user.
		switch diff := t.watchDelMin(min, nd, c, all); {
		case diff == none:
			return nil
		case diff[1] == 0:
			t.w.Unwatch(nd.Name)
		default:
			t.w.Rewatch(nd.Name, diff[0], diff[1])
		}
		return nil
	}
	t.rw.Lock()
	err := t.walkWatchpoint(t.root.nd, fn) // TODO(rjeczalik): store max root per c
	t.rw.Unlock()
	dbgprintf("Stop(%p) error: %v\n", c, err)
}

// Close cancels queued maintenance, waits for active filesystem operations, and
// drains input until the backend has stopped sending. An in-flight filesystem
// syscall cannot be cancelled, but queue backpressure cannot prevent shutdown.
func (t *nonrecursiveTree) Close() error {
	t.closeOnce.Do(func() {
		close(t.done)
		<-t.maintained
		t.ops.Lock()
		t.closeErr = t.w.Close()
		close(t.c)
		t.ops.Unlock()
		<-t.dispatched
	})
	return t.closeErr
}
