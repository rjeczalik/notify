// Copyright (c) 2014-2015 The Notify Authors. All rights reserved.
// Use of this source code is governed by the MIT license that can be
// found in the LICENSE file.

// +build darwin linux freebsd dragonfly netbsd openbsd windows

package notify

import "testing"

// NOTE Set DEBUG env var for extra debugging info.

func TestWatcher(t *testing.T) {
	w := NewWatcherTest(t, "testdata/vfs.txt")
	defer w.Close()

	cases := [...]WCase{
		// i=0
		create(w, "src/github.com/ppknap/link/include/coost/.link.hpp.swp"),
		// i=1
		create(w, "src/github.com/rjeczalik/fs/fs_test.go"),
		// i=2
		create(w, "src/github.com/rjeczalik/fs/binfs/"),
		// i=3
		create(w, "src/github.com/rjeczalik/fs/binfs.go"),
		// i=4
		create(w, "src/github.com/rjeczalik/fs/binfs_test.go"),
		// i=5
		remove(w, "src/github.com/rjeczalik/fs/binfs/"),
		// i=6
		create(w, "src/github.com/rjeczalik/fs/binfs/"),
		// i=7
		create(w, "src/github.com/rjeczalik/fs/virfs"),
		// i=8
		remove(w, "src/github.com/rjeczalik/fs/virfs"),
		// i=9
		create(w, "file"),
		// i=10
		create(w, "dir/"),
	}

	w.ExpectAny(cases[:])
}
