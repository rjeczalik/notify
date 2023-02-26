// Copyright (c) 2014-2015 The Notify Authors. All rights reserved.
// Use of this source code is governed by the MIT license that can be
// found in the LICENSE file.

//go:build darwin || linux || freebsd || dragonfly || netbsd || openbsd || windows || solaris
// +build darwin linux freebsd dragonfly netbsd openbsd windows solaris

package notify

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNotifyExample(t *testing.T) {
	n := NewNotifyTest(t, "testdata/vfs.txt")

	ch := NewChans(3)

	// Watch-points can be set explicitly via Watch/Stop calls...
	n.Watch("src/github.com/rjeczalik/fs", ch[0], Write)
	n.Watch("src/github.com/pblaszczyk/qttu", ch[0], Write)
	n.Watch("src/github.com/pblaszczyk/qttu/...", ch[1], Create)
	n.Watch("src/github.com/rjeczalik/fs/cmd/...", ch[2], Remove)

	cases := []NCase{
		// i=0
		{
			Event:    write(n.W(), "src/github.com/rjeczalik/fs/fs.go", []byte("XD")),
			Receiver: Chans{ch[0]},
		},
		// TODO(rjeczalik): #62
		// i=1
		// {
		//	Event:    write(n.W(), "src/github.com/pblaszczyk/qttu/README.md", []byte("XD")),
		//	Receiver: Chans{ch[0]},
		// },
		// i=2
		{
			Event:    write(n.W(), "src/github.com/rjeczalik/fs/cmd/gotree/go.go", []byte("XD")),
			Receiver: nil,
		},
		// i=3
		{
			Event:    create(n.W(), "src/github.com/pblaszczyk/qttu/src/.main.cc.swp"),
			Receiver: Chans{ch[1]},
		},
		// i=4
		{
			Event:    create(n.W(), "src/github.com/pblaszczyk/qttu/src/.main.cc.swo"),
			Receiver: Chans{ch[1]},
		},
		// i=5
		{
			Event:    remove(n.W(), "src/github.com/rjeczalik/fs/cmd/gotree/go.go"),
			Receiver: Chans{ch[2]},
		},
	}

	n.ExpectNotifyEvents(cases, ch)

	// ...or using Call structures.
	stops := [...]Call{
		// i=0
		{
			F: FuncStop,
			C: ch[0],
		},
		// i=1
		{
			F: FuncStop,
			C: ch[1],
		},
	}

	n.Call(stops[:]...)

	cases = []NCase{
		// i=0
		{
			Event:    write(n.W(), "src/github.com/rjeczalik/fs/fs.go", []byte("XD")),
			Receiver: nil,
		},
		// i=1
		{
			Event:    write(n.W(), "src/github.com/pblaszczyk/qttu/README.md", []byte("XD")),
			Receiver: nil,
		},
		// i=2
		{
			Event:    create(n.W(), "src/github.com/pblaszczyk/qttu/src/.main.cc.swr"),
			Receiver: nil,
		},
		// i=3
		{
			Event:    remove(n.W(), "src/github.com/rjeczalik/fs/cmd/gotree/main.go"),
			Receiver: Chans{ch[2]},
		},
	}

	n.ExpectNotifyEvents(cases, ch)
}

func TestStop(t *testing.T) {
	t.Skip("TODO(rjeczalik)")
}

func TestRenameInRoot(t *testing.T) {
	tmpDir := t.TempDir()

	c := make(chan EventInfo, 100)
	first := filepath.Join(tmpDir, "foo")
	second := filepath.Join(tmpDir, "bar")
	file := filepath.Join(second, "file")

	mustT(t, os.Mkdir(first, 0777))

	if err := Watch(tmpDir+"/...", c, All); err != nil {
		t.Fatal(err)
	}
	defer Stop(c)

	mustT(t, os.Rename(first, second))
	time.Sleep(50 * time.Millisecond) // Need some time to process rename.
	fd, err := os.Create(file)
	mustT(t, err)
	fd.Close()

	timeout := time.After(time.Second)
	for {
		select {
		case ev := <-c:
			if samefile(t, ev.Path(), file) {
				return
			}
			t.Log(ev.Path())
		case <-timeout:
			t.Fatal("timed out before receiving event")
		}
	}
}

func prepareTestDir(t *testing.T) string {
	tmpDir, err := ioutil.TempDir("", "notify_test-")
	if err != nil {
		t.Fatal(err)
	}
	
	// resolve paths on OSX
	s, err := filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// create test dir
	err = os.MkdirAll(filepath.Join(s, "a/b/c"), 0755)
	if err != nil {
		t.Fatal(err)
	}
	
	return s
}

func mustWatch(t *testing.T, path string) chan EventInfo {
	c := make(chan EventInfo, 1)
	err := Watch(path+"...", c, All)
	if err != nil {
		t.Fatal(err)
	}

	return c
}

func TestAddParentAfterStop(t *testing.T) {
	tmpDir := prepareTestDir(t)
	defer os.RemoveAll(tmpDir)

	// watch a child and parent path across multiple channels.
	// this can happen in any order.
	ch1 := mustWatch(t, filepath.Join(tmpDir, "a/b"))
	ch2 := mustWatch(t, filepath.Join(tmpDir, "a/b/c"))
	defer Stop(ch2)

	// unwatch ./a/b -- this is what causes the panic on the next line.
	// note that this also fails if we notify.Stop(ch1) instead.
	Stop(ch1)
	
	// add parent watchpoint
	ch3 := mustWatch(t, filepath.Join(tmpDir, "a"))
	defer Stop(ch3)

	// fire an event
	filePath := filepath.Join(tmpDir, "a/b/c/d")
	go func() { _ = ioutil.WriteFile(filePath, []byte("X"), 0664) }()

	timeout := time.After(5 * time.Second)
	for {
		select {
		case ev := <-ch2:
			t.Log(ev.Path(), ev.Event())
			if ev.Path() == filePath && (ev.Event() == Create || ev.Event() == Write) {
				return
			}
		case <-timeout:
			t.Fatal("timed out before receiving event")
		}
	}
}

func TestStopChild(t *testing.T) {
	tmpDir := prepareTestDir(t)
	defer os.RemoveAll(tmpDir)

	// watch a child and parent path across multiple channels.
	// this can happen in any order.
	ch1 := mustWatch(t, filepath.Join(tmpDir, "a"))
	defer Stop(ch1)
	ch2 := mustWatch(t, filepath.Join(tmpDir, "a/b/c"))

	// this leads to tmpDir/a being unwatched
	Stop(ch2)

	// fire an event
	filePath := filepath.Join(tmpDir, "a/b/c/d")
	go func() { _ = ioutil.WriteFile(filePath, []byte("X"), 0664) }()

	timeout := time.After(5 * time.Second)
	for {
		select {
		case ev := <-ch1:
			t.Log(ev.Path(), ev.Event())
			if ev.Path() == filePath && (ev.Event() == Create || ev.Event() == Write) {
				return
			}
		case <-timeout:
			t.Fatal("timed out before receiving event")
		}
	}
}

func TestRecreated(t *testing.T) {
	tmpDir := t.TempDir()

	dir := filepath.Join(tmpDir, "folder")
	file := filepath.Join(dir, "file")

	// Start watching
	eventChan := make(chan EventInfo, 1000)
	mustT(t, Watch(tmpDir+"/...", eventChan, All))
	defer Stop(eventChan)

	recreateFolder := func() {
		// Give the sync some time to process events
		mustT(t, os.RemoveAll(dir))
		mustT(t, os.Mkdir(dir, 0777))
		time.Sleep(100 * time.Millisecond)

		// Create a file
		mustT(t, os.WriteFile(file, []byte("abc"), 0666))
	}
	timeout := time.After(5 * time.Second)
	checkCreated := func() {
		for {
			select {
			case ev := <-eventChan:
				t.Log(ev.Path(), ev.Event())
				if samefile(t, ev.Path(), file) && ev.Event() == Create {
					return
				}
			case <-timeout:
				t.Fatal("timed out before receiving event")
			}
		}
	}

	// 1. Create a folder and a file within it
	// This will create a node in the internal tree for the subfolder test/folder
	// Will create a new inotify watch for the folder
	t.Log("######## First ########")
	recreateFolder()
	checkCreated()

	// 2. Create a folder and a file within it again
	// This will set the events for the subfolder test/folder in the internal tree
	// Will create a new inotify watch for the folder because events differ
	t.Log("######## Second ########")
	recreateFolder()
	checkCreated()

	// 3. Create a folder and a file within it yet again
	// This time no new inotify watch will be created, because the events
	// and node already exist in the internal tree and all subsequent events
	// are lost, hence there is no event for the created file here anymore
	t.Log("######## Third ########")
	recreateFolder()
	checkCreated()
}

func mustT(t testing.TB, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func samefile(t *testing.T, p1, p2 string) bool {
	// The tests sometimes delete files shortly after creating them.
	// That's expected; ignore stat failures.
	fi1, err := os.Stat(p1)
	if errors.Is(err, os.ErrNotExist) {
		return false
	}
	mustT(t, err)
	fi2, err := os.Stat(p2)
	if errors.Is(err, os.ErrNotExist) {
		return false
	}
	mustT(t, err)
	return os.SameFile(fi1, fi2)
}
