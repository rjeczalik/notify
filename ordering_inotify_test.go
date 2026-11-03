// Copyright (c) 2026 The Notify Authors. All rights reserved.
// Use of this source code is governed by the MIT license that can be
// found in the LICENSE file.

//go:build linux && !(android && amd64)

package notify

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestInotifyBatchOrder(t *testing.T) {
	// Multiple senders can overtake a blocked earlier batch. Keep the production
	// sender count covered as well as the ordering within and between batches.
	if consumersCount != 1 {
		t.Fatal("inotify requires one ordered batch sender")
	}
	out := make(chan EventInfo)
	i := newWatcher(out).(*inotify)
	i.m[1] = &watched{path: "/watched", mask: uint32(Create)}
	batches := make(chan []*event, 2)
	for batch := 0; batch < 2; batch++ {
		es := make([]*event, 200)
		for j := range es {
			es[j] = &event{path: fmt.Sprint(batch*len(es) + j), sys: unix.InotifyEvent{Wd: 1, Mask: unix.IN_CREATE | unix.IN_ISDIR}}
		}
		batches <- es
	}
	close(batches)
	i.wg.Add(1)
	go i.send(batches)
	for j := 0; j < 400; j++ {
		select {
		case e := <-out:
			if e.Path() != filepath.Join("/watched", fmt.Sprint(j)) {
				t.Fatalf("event %d: %s", j, e.Path())
			}
		case <-time.After(time.Second):
			t.Fatal("sender stalled")
		}
	}
	i.wg.Wait()
}

func TestInotifyRecreationBurst(t *testing.T) {
	const count = 100
	root := t.TempDir()
	tr := newTree().(*nonrecursiveTree)
	defer tr.Close()
	user := make(chan EventInfo, 16*count)
	mustT(t, tr.Watch(filepath.Join(root, "..."), user, All))
	backend := tr.w.(*inotify)
	waitParents := func(kind Event) {
		t.Helper()
		seen := make(map[string]bool, count)
		deadline := time.After(3 * time.Second)
		for len(seen) < count {
			select {
			case ei := <-user:
				if ei.Event() == kind && filepath.Dir(ei.Path()) == root {
					seen[ei.Path()] = true
				}
			case <-deadline:
				t.Fatalf("got %d/%d parent %v events", len(seen), count, kind)
			}
		}
		waitTreeMaintenance(t, tr, root)
	}

	for round := 0; round < 5; round++ {
		for j := 0; j < count; j++ {
			mustT(t, os.MkdirAll(filepath.Join(root, fmt.Sprint(j), "nested"), 0755))
		}
		waitParents(Create)
		backend.RLock()
		for j := 0; j < count; j++ {
			path := filepath.Join(root, fmt.Sprint(j), "nested")
			if _, ok := backend.paths[path]; !ok {
				backend.RUnlock()
				t.Fatalf("nested watch missing: %s", path)
			}
		}
		backend.RUnlock()
		// Drain parent notifications; files created before installation may be missed
		// by inotify. Here every file is created after its directory is watched.
	drain:
		for {
			select {
			case <-user:
			default:
				break drain
			}
		}
		for j := 0; j < count; j++ {
			mustT(t, os.WriteFile(filepath.Join(root, fmt.Sprint(j), "nested", "file"), nil, 0644))
		}
		seen := make(map[string]bool)
		deadline := time.After(3 * time.Second)
		for len(seen) < count {
			select {
			case e := <-user:
				if e.Event() == Create && filepath.Base(e.Path()) == "file" {
					seen[e.Path()] = true
				}
			case <-deadline:
				t.Fatalf("round %d: got %d/%d nested file creates", round, len(seen), count)
			}
		}
		for j := 0; j < count; j++ {
			mustT(t, os.RemoveAll(filepath.Join(root, fmt.Sprint(j))))
		}
		waitParents(Remove)
	}
}

func TestInotifyPathIndex(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "dir")
	mustT(t, os.Mkdir(path, 0755))
	output := make(chan EventInfo, buffer)
	backend := newWatcher(output).(*inotify)
	defer backend.Close()
	mustT(t, backend.Watch(path, All))
	mustT(t, backend.Rewatch(path, All, Create|Remove))
	// Replacement without an explicit Unwatch leaves a dead kernel descriptor
	// until the next registration. The index must identify only the live watch.
	mustT(t, os.Remove(path))
	mustT(t, os.Mkdir(path, 0755))
	mustT(t, backend.Watch(path, All))
	backend.RLock()
	wd, ok := backend.paths[path]
	consistent := ok && len(backend.paths) == 1 && len(backend.m) == 1 && backend.m[wd] != nil && backend.m[wd].path == path
	backend.RUnlock()
	if !consistent {
		t.Fatal("replacement left inconsistent path/descriptor indices")
	}
	file := filepath.Join(path, "probe")
	mustT(t, os.WriteFile(file, nil, 0644))
	timeout := time.After(time.Second)
	received := false
	for !received {
		select {
		case e := <-output:
			received = e.Path() == file && e.Event() == Create
		case <-timeout:
			t.Fatal("replacement watch missed probe")
		}
	}
	mustT(t, backend.Unwatch(path))
	backend.RLock()
	empty := len(backend.paths) == 0 && len(backend.m) == 0
	backend.RUnlock()
	if !empty {
		t.Fatal("Unwatch did not clear both indices")
	}
}
