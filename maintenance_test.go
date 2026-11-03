// Copyright (c) 2026 The Notify Authors. All rights reserved.
// Use of this source code is governed by the MIT license that can be
// found in the LICENSE file.

package notify

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

type maintenanceWatcher struct {
	watch   func(string)
	unwatch func(string)
	close   func()
}

func (w maintenanceWatcher) Watch(p string, _ Event) error {
	if w.watch != nil {
		w.watch(p)
	}
	return nil
}
func (w maintenanceWatcher) Rewatch(p string, _, _ Event) error { return w.Watch(p, 0) }
func (w maintenanceWatcher) Unwatch(p string) error {
	if w.unwatch != nil {
		w.unwatch(p)
	}
	return nil
}
func (w maintenanceWatcher) Close() error {
	if w.close != nil {
		w.close()
	}
	return nil
}

func awaitMaintenance(t *testing.T, c <-chan struct{}) {
	t.Helper()
	select {
	case <-c:
	case <-time.After(3 * time.Second):
		t.Fatal("maintenance did not progress")
	}
}

func TestMaintenanceSubtreeOrder(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "child")
	sibling := filepath.Join(root, "sibling")
	mustT(t, os.Mkdir(child, 0755))
	input := make(chan EventInfo, buffer)
	entered, release, installed, independent := make(chan struct{}), make(chan struct{}), make(chan struct{}), make(chan struct{})
	var mu sync.Mutex
	removalStarted := false
	w := maintenanceWatcher{
		watch: func(p string) {
			mu.Lock()
			defer mu.Unlock()
			if p == child && removalStarted {
				close(installed)
			}
			if p == sibling {
				close(independent)
			}
		},
		unwatch: func(p string) {
			if p == child {
				mu.Lock()
				removalStarted = true
				mu.Unlock()
				close(entered)
				<-release
			}
		},
	}
	tr := newNonrecursiveTree(w, input, nil)
	var releaseOnce sync.Once
	defer tr.Close()
	defer releaseOnce.Do(func() { close(release) })
	user := make(chan EventInfo, 10)
	mustT(t, tr.Watch(filepath.Join(root, "..."), user, All))
	input <- &Call{P: child, E: Remove, Dir: true}
	awaitMaintenance(t, entered)
	input <- &Call{P: child, E: Create, Dir: true}
	input <- &Call{P: sibling, E: Create, Dir: true}
	awaitMaintenance(t, independent)
	select {
	case <-installed:
		t.Fatal("replacement watch overtook removal")
	default:
	}
	releaseOnce.Do(func() { close(release) })
	awaitMaintenance(t, installed)
}

func TestMaintenancePendingAncestorOrder(t *testing.T) {
	root := t.TempDir()
	parent := filepath.Join(root, "parent")
	child := filepath.Join(parent, "child")
	sibling := filepath.Join(parent, "sibling")
	marker := filepath.Join(root, "marker")
	marked := make(chan struct{})
	mustT(t, os.MkdirAll(child, 0755))
	entered, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	var mu sync.Mutex
	blocked := false
	var calls []string
	w := maintenanceWatcher{
		watch: func(p string) {
			mu.Lock()
			calls = append(calls, "watch:"+p)
			mu.Unlock()
			if p == marker {
				close(marked)
			}
		},
		unwatch: func(p string) {
			mu.Lock()
			calls = append(calls, "unwatch:"+p)
			block := p == child && !blocked
			if block {
				blocked = true
			}
			mu.Unlock()
			if block {
				close(entered)
				<-release
			}
		},
	}
	input := make(chan EventInfo, buffer)
	tr := newNonrecursiveTree(w, input, nil)
	defer tr.Close()
	defer releaseOnce.Do(func() { close(release) })
	mustT(t, tr.Watch(filepath.Join(root, "..."), make(chan EventInfo, 10), All))
	input <- &Call{P: child, E: Remove, Dir: true}
	awaitMaintenance(t, entered)
	input <- &Call{P: parent, E: Remove, Dir: true}
	input <- &Call{P: sibling, E: Create, Dir: true}
	// A disjoint marker proves both pending events reached the scheduler while
	// the child is stalled. Sibling must remain behind the pending parent remove.
	input <- &Call{P: marker, E: Create, Dir: true}
	awaitMaintenance(t, marked)
	mu.Lock()
	for _, c := range calls {
		if c == "watch:"+sibling {
			mu.Unlock()
			t.Fatal("pending ancestor was bypassed")
		}
	}
	mu.Unlock()
	releaseOnce.Do(func() { close(release) })
	// Wait for sibling installation; it must happen after parent's removal.
	deadline := time.After(3 * time.Second)
	for {
		mu.Lock()
		parentAt, siblingAt := -1, -1
		for j, c := range calls {
			if c == "unwatch:"+parent {
				parentAt = j
			}
			if c == "watch:"+sibling {
				siblingAt = j
			}
		}
		mu.Unlock()
		if siblingAt >= 0 {
			if parentAt < 0 || parentAt > siblingAt {
				t.Fatal("pending ancestor was bypassed")
			}
			break
		}
		select {
		case <-deadline:
			t.Fatal("sibling was not installed")
		case <-time.After(time.Millisecond):
		}
	}
}

func TestMaintenanceCloseWithBackpressure(t *testing.T) {
	root := t.TempDir()
	entered, release, backendClosed := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var once sync.Once
	w := maintenanceWatcher{watch: func(p string) {
		if p != root {
			once.Do(func() { close(entered) })
			<-release
		}
	}, close: func() { close(backendClosed) }}
	input := make(chan EventInfo, buffer)
	tr := newNonrecursiveTree(w, input, nil)
	var releaseOnce sync.Once
	defer tr.Close()
	defer releaseOnce.Do(func() { close(release) })
	mustT(t, tr.Watch(filepath.Join(root, "..."), make(chan EventInfo), All))
	input <- &Call{P: filepath.Join(root, "0"), E: Create, Dir: true}
	awaitMaintenance(t, entered)
	sent := make(chan struct{})
	go func() {
		for j := 1; j < 4*buffer; j++ {
			input <- &Call{P: filepath.Join(root, fmt.Sprint(j)), E: Create, Dir: true}
		}
		close(sent)
	}()
	// Wait for both bounded tree channels to fill before initiating shutdown.
	deadline := time.After(3 * time.Second)
	for len(input) != cap(input) || len(tr.rec) != cap(tr.rec) {
		select {
		case <-deadline:
			t.Fatal("backpressure did not fill the bounded channels")
		case <-time.After(time.Millisecond):
		}
	}
	closed := make(chan struct{})
	go func() { tr.Close(); close(closed) }()
	// Close must unblock queued backend sends even while a filesystem call is
	// stalled. It waits for already-running calls, which cannot be cancelled.
	awaitMaintenance(t, sent)
	select {
	case <-backendClosed:
		t.Fatal("backend closed before active maintenance completed")
	default:
	}
	releaseOnce.Do(func() { close(release) })
	awaitMaintenance(t, closed)
	mustT(t, tr.Close())
	if err := tr.Watch(root, make(chan EventInfo), All); err == nil {
		t.Fatal("Watch after Close succeeded")
	}
}

func TestMaintenanceRootAndPrefixConflicts(t *testing.T) {
	root := filepath.VolumeName(t.TempDir()) + string(os.PathSeparator)
	a, b := filepath.Join(root, "a"), filepath.Join(root, "ab")
	if !relatedPaths(root, a) || !relatedPaths(a, root) {
		t.Fatal("volume root must conflict with its descendants")
	}
	if relatedPaths(a, b) {
		t.Fatal("sibling name prefixes must not conflict")
	}
}
