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
)

// File creation before inotify_add_watch is inherently racy. Record that gap,
// and require every probe created after installation to be delivered.
func TestInotifyFilesAroundInstallation(t *testing.T) {
	const count = 100
	root := t.TempDir()
	tr := newTree().(*nonrecursiveTree)
	defer tr.Close()
	user := make(chan EventInfo, 4*count)
	mustT(t, tr.Watch(filepath.Join(root, "..."), user, Create))
	for j := 0; j < count; j++ {
		dir := filepath.Join(root, fmt.Sprint(j))
		mustT(t, os.Mkdir(dir, 0755))
		mustT(t, os.WriteFile(filepath.Join(dir, "early"), nil, 0644))
	}
	backend := tr.w.(*inotify)
	deadline := time.Now().Add(3 * time.Second)
	for {
		backend.RLock()
		n := len(backend.m)
		backend.RUnlock()
		if n == count+1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("watch installation did not finish")
		}
		time.Sleep(time.Millisecond)
	}
	for j := 0; j < count; j++ {
		mustT(t, os.WriteFile(filepath.Join(root, fmt.Sprint(j), "probe"), nil, 0644))
	}
	early, probe := make(map[string]bool), make(map[string]bool)
	timeout := time.After(3 * time.Second)
	for len(probe) < count {
		select {
		case ei := <-user:
			switch filepath.Base(ei.Path()) {
			case "early":
				early[ei.Path()] = true
			case "probe":
				probe[ei.Path()] = true
			}
		case <-timeout:
			t.Fatalf("only %d/%d installed-watch probes delivered", len(probe), count)
		}
	}
	t.Logf("early nested creates: %d/%d; after-installation probes: %d/%d", len(early), count, len(probe), count)
}
