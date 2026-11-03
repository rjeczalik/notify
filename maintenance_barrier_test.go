// Copyright (c) 2026 The Notify Authors. All rights reserved.
// Use of this source code is governed by the MIT license that can be
// found in the LICENSE file.

package notify

import (
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// A fence's parent path conflicts with all preceding maintenance in this test,
// but has no recursive watch of its own, so processing it needs no filesystem
// traversal. A dispatch fence ensures the last user delivery has also enqueued
// its lifecycle work before we submit the maintenance fence.
// Upstream's goroutine-per-event dispatcher cannot provide that dispatch
// ordering: its fence timing is only an observation, not proof of a full drain.
type benchmarkFence struct {
	path  string
	kind  Event
	once  sync.Once
	ready chan struct{}
}

func (f *benchmarkFence) Path() string     { return f.path }
func (f *benchmarkFence) Event() Event     { f.once.Do(func() { close(f.ready) }); return f.kind }
func (f *benchmarkFence) Sys() interface{} { return nil }

func waitTreeMaintenance(tb testing.TB, tr *nonrecursiveTree, root string) {
	tb.Helper()
	dispatched := &benchmarkFence{path: filepath.Join(root, "fence"), kind: Write, ready: make(chan struct{})}
	select {
	case tr.c <- dispatched:
	case <-time.After(3 * time.Second):
		tb.Fatal("dispatch fence send stalled")
	}
	select {
	case <-dispatched.ready:
	case <-time.After(3 * time.Second):
		tb.Fatal("dispatch fence stalled")
	}
	maintained := &benchmarkFence{path: filepath.Dir(root), kind: Create, ready: make(chan struct{})}
	select {
	case tr.rec <- maintained:
	case <-time.After(3 * time.Second):
		tb.Fatal("maintenance fence send stalled")
	}
	select {
	case <-maintained.ready:
	case <-time.After(3 * time.Second):
		tb.Fatal("maintenance fence stalled")
	}
}
