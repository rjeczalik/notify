// Copyright (c) 2026 The Notify Authors. All rights reserved.
// Use of this source code is governed by the MIT license that can be
// found in the LICENSE file.

//go:build linux

package notify

import (
	"path/filepath"
	"testing"
	"time"
)

type orderingWatcher struct{}

func (orderingWatcher) Watch(string, Event) error          { return nil }
func (orderingWatcher) Rewatch(string, Event, Event) error { return nil }
func (orderingWatcher) Unwatch(string) error               { return nil }
func (orderingWatcher) Close() error                       { return nil }

type orderingEvent struct {
	path    string
	kind    Event
	entered chan struct{}
	release <-chan struct{}
}

func (e orderingEvent) Path() string     { return e.path }
func (e orderingEvent) Event() Event     { return e.kind }
func (e orderingEvent) Sys() interface{} { return nil }
func (e orderingEvent) isDir() (bool, error) {
	close(e.entered)
	if e.release != nil {
		<-e.release
	}
	return true, nil
}

// A delayed directory removal must not be overtaken by a later create. Such
// reordering can remove the replacement directory's recursive watch.
func TestRecursiveDirectoryEventsKeepOrder(t *testing.T) {
	dir := t.TempDir()
	input := make(chan EventInfo, 2)
	tree := newNonrecursiveTree(orderingWatcher{}, input, nil)
	defer tree.Close()
	user := make(chan EventInfo, 10)
	if err := tree.Watch(filepath.Join(dir, "..."), user, All); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "child")
	enteredRemove, enteredCreate, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
	input <- orderingEvent{path: path, kind: Remove, entered: enteredRemove, release: release}
	select {
	case <-enteredRemove:
	case <-time.After(time.Second):
		t.Fatal("remove was not processed")
	}
	input <- orderingEvent{path: path, kind: Create, entered: enteredCreate}
	select {
	case <-enteredCreate:
		close(release)
		t.Fatal("create overtook the preceding directory removal")
	case <-time.After(100 * time.Millisecond):
	}
	close(release)
	select {
	case <-enteredCreate:
	case <-time.After(time.Second):
		t.Fatal("create was not processed")
	}
}
