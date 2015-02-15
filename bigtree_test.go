// Copyright (c) 2014-2015 The Notify Authors. All rights reserved.
// Use of this source code is governed by the MIT license that can be
// found in the LICENSE file.

// +build ignore

package notify

// TODO(rjeczalik): Tree is currently broken under Windows

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestTreeLookPath(t *testing.T) {
	t.skip("TODO(rjeczalik)")
}

func TestTreeLook(t *testing.T) {
	t.skip("TODO(rjeczalik)")
}

func p(p string) string {
	return filepath.FromSlash(p)
}

func TestTreeDel(t *testing.T) {
	t.Parallel()
	cases := [...]struct {
		before node
		p      string
		after  node
	}{{
		node{Child: map[string]node{
			"a": {
				Name: p("/a"),
				Child: map[string]node{
					"b": {
						Name: p("/a/b"),
						Child: map[string]node{
							"c": {
								Name: p("/a/b/c"),
								Child: map[string]node{
									"d": {
										Name: p("/a/b/c/d"),
									},
								},
							},
						},
					},
					"x": {
						Name: p("/a/x"),
					},
				},
			},
		}},
		"/a/b/c/d",
		node{Child: map[string]node{
			"a": {
				Name: p("/a"),
				Child: map[string]node{
					"x": {
						Name: p("/a/x"),
					},
				},
			},
		}},
	}, {
		node{Child: map[string]node{
			"a": {
				Name: p("/a"),
				Child: map[string]node{
					"b": {
						Name: p("/a/b"),
						Child: map[string]node{
							"c": {
								Name: p("/a/b/c"),
							},
						},
					},
				},
			},
		}},
		"/a/b/c",
		node{Child: map[string]node{}},
	}}
	for i, cas := range cases {
		if (&bigTree{Root: cas.before}).Del(cas.p); !reflect.DeepEqual(cas.before, cas.after) {
			t.Errorf("want tree=%v; got %v (i=%d)", cas.after, cas.before, i)
		}
	}
}

func TestTreeWalkPath(t *testing.T) {
	t.skip("TODO(rjeczalik)")
}

func TestTreeWalkDir(t *testing.T) {
	t.skip("TODO(rjeczalik)")
}

func TestTreeWalk(t *testing.T) {
	t.skip("TODO(rjeczalik)")
}

func TestTreeWatch(t *testing.T) {
	t.Parallel()
	ch := NewChans(3)
	calls := [...]CallCase{{
		// i=0
		Call: Call{
			F: FuncWatch,
			C: ch[0],
			P: "/github.com/rjeczalik/fakerpc",
			E: Create | Delete | Move,
		},
		Record: Record{
			TreeAll: {{
				F: FuncWatch,
				P: "/github.com/rjeczalik/fakerpc",
				E: Delete | Create | Move,
			}}},
	}, { // i=1
		Call: Call{
			F: FuncWatch,
			C: ch[1],
			P: "/github.com/rjeczalik/fakerpc",
			E: Delete | Move,
		},
		Record: nil,
	}, { // i=2
		Call: Call{
			F: FuncWatch,
			C: ch[2],
			P: "/github.com/rjeczalik/fakerpc",
			E: Move,
		},
		Record: nil,
	}, { // i=3
		Call: Call{
			F: FuncWatch,
			C: ch[0],
			P: "/github.com/rjeczalik/fs",
			E: Create | Delete,
		},
		Record: Record{
			TreeAll: {{
				F: FuncWatch,
				P: "/github.com/rjeczalik/fs",
				E: Create | Delete,
			}}},
	}, { // i=4
		Call: Call{
			F: FuncWatch,
			C: ch[0],
			P: "/github.com/rjeczalik/fs",
			E: Create,
		},
		Record: nil,
	}, { // i=5
		Call: Call{
			F: FuncStop,
			C: ch[0],
		},
		Record: Record{
			TreeAll: {{
				F:  FuncRewatch,
				P:  "/github.com/rjeczalik/fakerpc",
				E:  Create | Delete | Move,
				NE: Delete | Move,
			}, {
				F: FuncUnwatch,
				P: "/github.com/rjeczalik/fs",
			}},
		},
	}, { // i=6
		Call: Call{
			F: FuncWatch,
			C: ch[0],
			P: "/github.com/rjeczalik/which",
			E: Create,
		},
		Record: Record{
			TreeAll: {{
				F: FuncWatch,
				P: "/github.com/rjeczalik/which",
				E: Create,
			}},
		},
	}, { // i=7
		Call: Call{
			F: FuncWatch,
			C: ch[1],
			P: "/github.com/rjeczalik/which",
			E: Delete,
		},
		Record: Record{
			TreeAll: {{
				F:  FuncRewatch,
				P:  "/github.com/rjeczalik/which",
				E:  Create,
				NE: Create | Delete,
			}},
		},
	}}
	events := [...]EventCase{{
		Event: TreeEvent{
			P: "/github.com/rjeczalik/fakerpc/.fakerpc.go.swp",
			E: Delete,
		},
		Receiver: Chans{ch[1]},
	}, {
		Event: TreeEvent{
			P: "/github.com/rjeczalik/fakerpc/.travis.yml",
			E: Move,
		},
		Receiver: Chans{ch[1], ch[2]},
	}, {
		Event: TreeEvent{
			P: "/github.com/rjeczalik/fakerpc/which",
			E: Delete,
		},
		Receiver: Chans{ch[1]},
	}}
	fixture := NewTreeFixture()
	fixture.TestCalls(t, calls[:])
	fixture.TestEvents(t, events[:])
	// Ensure no extra events were dispatched.
	if ei := ch.Drain(); len(ei) != 0 {
		t.Errorf("want ei=nil; got %v", ei)
	}
}

func TestTreeStop(t *testing.T) {
	t.Parallel()
	ch := NewChans(3)
	// Watchpoints:
	//
	// x ch[0] -> {"/github.com/rjeczalik",         Create|Delete}
	// x ch[1] -> {"/github.com/rjeczalik",         Create|Delete|Move}
	// x ch[2] -> {"/github.com/rjeczalik",         Move|Write}
	// x ch[0] -> {"/github.com/rjeczalik/which",   Write|Move}
	// x ch[1] -> {"/github.com/rjeczalik/which",   Create|Delete}
	// x ch[2] -> {"/github.com/rjeczalik/which",   Delete|Move}
	// x ch[0] -> {"/github.com/rjeczalik/fs/fs.go, Write}
	// x ch[1] -> {"/github.com/rjeczalik/fs/fs.go, Move|Delete}
	// x ch[2] -> {"/github.com/rjeczalik/fs/fs.go, Create|Delete}
	//
	setup := [...]CallCase{{
		// i=0
		Call: Call{
			F: FuncWatch,
			C: ch[0],
			P: "/github.com/rjeczalik",
			E: Create | Delete,
		},
		Record: Record{
			TreeAll: {{
				F: FuncWatch,
				P: "/github.com/rjeczalik",
				E: Create | Delete,
			}},
		},
	}, {
		// i=1
		Call: Call{
			F: FuncWatch,
			C: ch[1],
			P: "/github.com/rjeczalik", E: Create | Delete | Move,
		},
		Record: Record{
			TreeAll: {{
				F:  FuncRewatch,
				P:  "/github.com/rjeczalik",
				E:  Create | Delete,
				NE: Create | Delete | Move,
			}},
		},
	}, {
		// i=2
		Call: Call{
			F: FuncWatch,
			C: ch[2],
			P: "/github.com/rjeczalik", E: Move | Write,
		},
		Record: Record{
			TreeAll: {{
				F:  FuncRewatch,
				P:  "/github.com/rjeczalik",
				E:  Create | Delete | Move,
				NE: Create | Delete | Move | Write,
			}},
		},
	}, {
		// i=3
		Call: Call{
			F: FuncWatch,
			C: ch[0],
			P: "/github.com/rjeczalik/which",
			E: Write | Move,
		},
		Record: Record{
			TreeAll: {{
				F: FuncWatch,
				P: "/github.com/rjeczalik/which",
				E: Write | Move,
			}},
		},
	}, {
		// i=4
		Call: Call{
			F: FuncWatch,
			C: ch[1],
			P: "/github.com/rjeczalik/which",
			E: Create | Delete,
		},
		Record: Record{
			TreeAll: {{
				F:  FuncRewatch,
				P:  "/github.com/rjeczalik/which",
				E:  Write | Move,
				NE: Write | Move | Create | Delete,
			}},
		},
	}, {
		// i=5
		Call: Call{
			F: FuncWatch,
			C: ch[2],
			P: "/github.com/rjeczalik/which",
			E: Delete | Move,
		},
		Record: nil,
	}, {
		// i=6
		Call: Call{
			F: FuncWatch,
			C: ch[0],
			P: "/github.com/rjeczalik/fs/fs.go",
			E: Write,
		},
		Record: Record{
			TreeAll: {{
				F: FuncWatch,
				P: "/github.com/rjeczalik/fs/fs.go",
				E: Write,
			}},
		},
	}, {
		// i=7
		Call: Call{
			F: FuncWatch,
			C: ch[1],
			P: "/github.com/rjeczalik/fs/fs.go",
			E: Move | Delete,
		},
		Record: Record{
			TreeAll: {{
				F:  FuncRewatch,
				P:  "/github.com/rjeczalik/fs/fs.go",
				E:  Write,
				NE: Write | Move | Delete,
			}},
		},
	}, {
		// i=8
		Call: Call{
			F: FuncWatch,
			C: ch[2],
			P: "/github.com/rjeczalik/fs/fs.go",
			E: Create | Delete,
		},
		Record: Record{
			TreeAll: {{
				F:  FuncRewatch,
				P:  "/github.com/rjeczalik/fs/fs.go",
				E:  Write | Move | Delete,
				NE: Write | Move | Delete | Create,
			}},
		},
	}}
	events := [...]EventCase{{
		// i=0
		Event: TreeEvent{
			P: "/github.com/rjeczalik/.thumbs",
			E: Create,
		},
		Receiver: Chans{ch[0], ch[1]},
	}, {
		// i=1
		Event: TreeEvent{
			P: "/github.com/rjeczalik/fs/fs.go",
			E: Delete,
		},
		Receiver: Chans{ch[1], ch[2]},
	}, {
		// i=2
		Event: TreeEvent{
			P: "/github.com/rjeczalik/fs/fs.go",
			E: Delete,
		},
		Receiver: Chans{ch[1], ch[2]},
	}, {
		// i=3
		Event: TreeEvent{
			P: "/github.com/rjeczalik",
			E: Create,
		},
		Receiver: Chans{ch[0], ch[1]},
	}, {
		// i=4
		Event: TreeEvent{
			P: "/github.com/rjeczalik",
			E: Write,
		},
		Receiver: Chans{ch[2]},
	}, {
		// i=5
		Event: TreeEvent{
			P: "/github.com/rjeczalik/fs/fs.go",
			E: Write,
		},
		Receiver: Chans{ch[0]},
	}}
	cases := [...]CallCase{{
		Call: Call{
			F: FuncStop,
			C: ch[0],
		},
		Record: Record{
			TreeAll: {{
				F:  FuncRewatch,
				P:  "/github.com/rjeczalik/fs/fs.go",
				E:  Create | Delete | Move | Write,
				NE: Create | Delete | Move,
			}, {
				F:  FuncRewatch,
				P:  "/github.com/rjeczalik/which",
				E:  Create | Delete | Write | Move,
				NE: Create | Delete | Move,
			}},
		},
	}, {
		Call: Call{
			F: FuncStop,
			C: ch[1],
		},
		Record: Record{
			TreeAll: {{
				F:  FuncRewatch,
				P:  "/github.com/rjeczalik",
				E:  Create | Delete | Write | Move,
				NE: Move | Write,
			}, {
				F:  FuncRewatch,
				P:  "/github.com/rjeczalik/fs/fs.go",
				E:  Create | Delete | Move,
				NE: Create | Delete,
			}, {
				F:  FuncRewatch,
				P:  "/github.com/rjeczalik/which",
				E:  Create | Delete | Move,
				NE: Delete | Move,
			}},
		},
	}, {
		Call: Call{
			F: FuncStop,
			C: ch[2],
		},
		Record: Record{
			TreeAll: {{
				F: FuncUnwatch,
				P: "/github.com/rjeczalik",
			}, {
				F: FuncUnwatch,
				P: "/github.com/rjeczalik/fs/fs.go",
			}, {
				F: FuncUnwatch,
				P: "/github.com/rjeczalik/which",
			}},
		},
	}}
	fixture := NewTreeFixture()
	fixture.TestCalls(t, setup[:])
	fixture.TestEvents(t, events[:])
	fixture.TestCalls(t, cases[:])
	// Ensure no extra events were dispatched.
	if ei := ch.Drain(); len(ei) != 0 {
		t.Errorf("want ei=nil; got %v", ei)
	}
}

func TestTreeStopRecursive(t *testing.T) {
	t.Parallel()
	ch := NewChans(7)
	// Watchpoints:
	//
	//   ch[0] -> {"/github.com/rjeczalik/fakerpc/...", Create|Delete}
	// x ch[1] -> {"/github.com/rjeczalik/fakerpc/...", Create}
	// x ch[2] -> {"/github.com/rjeczalik/fakerpc/...", Create|Delete}
	// x ch[3] -> {"/github.com/rjeczalik/fakerpc", Create|Delete|Write}
	// x ch[4] -> {"/github.com/rjeczalik/fakerpc/cli", Delete|Move}
	//   ch[5] -> {"/github.com/rjeczalik/fakerpc/cmd/...", Create|Delete}
	//   ch[6] -> {"/github.com/rjeczalik/fakerpc/cmd/fakerpc/...", Move|Delete}
	//
	setup := [...]CallCase{{
		// i=0
		Call: Call{
			F: FuncWatch,
			C: ch[0],
			P: "/github.com/rjeczalik/fakerpc/...",
			E: Create | Delete,
		},
		Record: Record{
			TreeFakeRecursive: {{
				F: FuncWatch,
				P: "/github.com/rjeczalik/fakerpc",
				E: Create | Delete,
			}, {
				F: FuncWatch,
				P: "/github.com/rjeczalik/fakerpc/cli",
				E: Create | Delete,
			}, {
				F: FuncWatch,
				P: "/github.com/rjeczalik/fakerpc/cmd",
				E: Create | Delete,
			}, {
				F: FuncWatch,
				P: "/github.com/rjeczalik/fakerpc/cmd/fakerpc",
				E: Create | Delete,
			}},
			TreeNativeRecursive: {{
				F: FuncRecursiveWatch,
				P: "/github.com/rjeczalik/fakerpc",
				E: Create | Delete,
			}},
		},
	}, {
		// i=1
		Call: Call{
			F: FuncWatch,
			C: ch[1],
			P: "/github.com/rjeczalik/fakerpc/...",
			E: Create,
		},
		Record: nil,
	}, {
		// i=2
		Call: Call{
			F: FuncWatch,
			C: ch[2],
			P: "/github.com/rjeczalik/fakerpc/...",
			E: Create | Delete,
		},
		Record: nil,
	}, {
		// i=3
		Call: Call{
			F: FuncWatch,
			C: ch[3],
			P: "/github.com/rjeczalik/fakerpc",
			E: Create | Delete | Write,
		},
		Record: Record{
			TreeFakeRecursive: {{
				F:  FuncRewatch,
				P:  "/github.com/rjeczalik/fakerpc",
				E:  Create | Delete,
				NE: Create | Delete | Write,
			}},
			TreeNativeRecursive: {{
				F:  FuncRecursiveRewatch,
				P:  "/github.com/rjeczalik/fakerpc",
				NP: "/github.com/rjeczalik/fakerpc",
				E:  Create | Delete,
				NE: Create | Delete | Write,
			}},
		},
	}, {
		// i=4
		Call: Call{
			F: FuncWatch,
			C: ch[4],
			P: "/github.com/rjeczalik/fakerpc/cli",
			E: Delete | Move,
		},
		Record: Record{
			TreeFakeRecursive: {{
				F:  FuncRewatch,
				P:  "/github.com/rjeczalik/fakerpc/cli",
				E:  Create | Delete,
				NE: Create | Delete | Move,
			}},
			TreeNativeRecursive: {{
				F:  FuncRecursiveRewatch,
				P:  "/github.com/rjeczalik/fakerpc",
				NP: "/github.com/rjeczalik/fakerpc",
				E:  Create | Delete | Write,
				NE: Create | Delete | Write | Move,
			}},
		},
	}, {
		// i=5
		Call: Call{
			F: FuncWatch,
			C: ch[5],
			P: "/github.com/rjeczalik/fakerpc/cmd/...",
			E: Create | Delete,
		},
		Record: nil,
	}, {
		// i=6
		Call: Call{
			F: FuncWatch,
			C: ch[6],
			P: "/github.com/rjeczalik/fakerpc/cmd/fakerpc/...",
			E: Move | Delete,
		},
		Record: Record{
			TreeFakeRecursive: {{
				F:  FuncRewatch,
				P:  "/github.com/rjeczalik/fakerpc/cmd/fakerpc",
				E:  Create | Delete,
				NE: Create | Delete | Move,
			}},
			TreeNativeRecursive: nil,
		},
	}}
	events := [...]EventCase{{
		// i=0
		Event: TreeEvent{
			P: "/github.com/rjeczalik/fakerpc/cmd/fakerpc/.main.go.swp",
			E: Create,
		},
		Receiver: Chans{ch[0], ch[1], ch[2], ch[5]},
	}, {
		// i=1
		Event: TreeEvent{
			P: "/github.com/rjeczalik/fakerpc/cmd/fakerpc/.main.go.swp",
			E: Delete,
		},
		Receiver: Chans{ch[0], ch[2], ch[5], ch[6]},
	}, {
		// i=2
		Event: TreeEvent{
			P: "/github.com/rjeczalik/fakerpc/fakerpc.go",
			E: Write,
		},
		Receiver: Chans{ch[3]},
	}, {
		// i=3
		Event: TreeEvent{
			P: "/github.com/rjeczalik/fakerpc/fakerpc.go",
			E: Delete,
		},
		Receiver: Chans{ch[0], ch[2], ch[3]},
	}, {
		// i=4
		Event: TreeEvent{
			P: "/github.com/rjeczalik/fakerpc/cli/cli_test.go",
			E: Delete,
		},
		Receiver: Chans{ch[0], ch[2], ch[4]},
	}, {
		// i=5
		Event: TreeEvent{
			P: "/github.com/rjeczalik/fakerpc/cli/cli_test.go",
			E: Move,
		},
		Receiver: Chans{ch[4]},
	}, {
		// i=6
		Event: TreeEvent{
			P: "/github.com/rjeczalik/fakerpc/.DS_Store",
			E: Write,
		},
		Receiver: Chans{ch[3]},
	}}
	// BUG(rjeczalik): Bummer, it's broken - Stop does not take path but channel,
	// which is nil for each of nop cases - nil channel is nop by default. Extend
	// fixture to be able to "inject" bad paths into Stop command.
	nop := [...]CallCase{{
		// i=0
		Call: Call{
			F: FuncStop,
			P: "/github.com/rjeczalik",
		},
		Record: nil,
	}, {
		// i=1
		Call: Call{
			F: FuncStop,
			P: "/github.com/rjeczalik/fakerpc/cmd",
		},
		Record: nil,
	}, {
		// i=2
		Call: Call{
			F: FuncStop,
			P: "/github.com",
		},
		Record: nil,
	}, {
		// i=3
		Call: Call{
			F: FuncStop,
			P: "/",
		},
		Record: nil,
	}, {
		// i=4
		Call: Call{
			F: FuncStop,
			P: "/github.com/rjeczalik/fakerpc/cmd/fakerpc",
		},
		Record: nil,
	}, {
		// i=5
		Call: Call{
			F: FuncStop,
			P: "/github.com/rjeczalik/which",
		},
		Record: nil,
	}, {
		// i=6
		Call: Call{
			F: FuncStop,
			P: "/github.com/rjeczalik/fakerpc/LICENSE",
		},
		Record: nil,
	}, {
		// i=7
		Call: Call{
			F: FuncStop,
			P: "/github.com/rjeczalik/fakerpc/cli/cli.go",
		},
		Record: nil,
	}, {
		// i=8
		Call: Call{
			F: FuncStop,
			P: "/github.com/rjeczalik/fakerpc/DOESNOTEXIST",
		},
		Record: nil,
	}, {
		// i=9
		Call: Call{
			F: FuncStop,
			P: "/DOES/NOT/EXIST",
		},
		Record: nil,
	}, {
		// i=10
		Call: Call{
			F: FuncStop,
			P: "https://4037702efda8a44467ef8931cc22168fb441ca6f:x-oauth-basic@github.com/rjeczalik/notify.git",
		},
		Record: nil,
	}}
	cases := [...]CallCase{{
		// i=0
		Call: Call{
			F: FuncStop,
			C: ch[1],
		},
		Record: nil,
	}, {
		// i=1
		Call: Call{
			F: FuncStop,
			C: ch[4],
		},
		Record: Record{
			TreeFakeRecursive: {{
				F:  FuncRewatch,
				P:  "/github.com/rjeczalik/fakerpc/cli",
				E:  Create | Delete | Move,
				NE: Create | Delete,
			}},
			TreeNativeRecursive: nil,
		},
	}, {
		// i=2
		Call: Call{
			F: FuncStop,
			C: ch[2],
		},
		Record: nil,
	}, {
		// i=3
		Call: Call{
			F: FuncStop,
			C: ch[3],
		},
		Record: Record{
			TreeFakeRecursive: {{
				F:  FuncRewatch,
				P:  "/github.com/rjeczalik/fakerpc",
				E:  Create | Delete | Write,
				NE: Create | Delete,
			}},
			TreeNativeRecursive: {{
				F:  FuncRecursiveRewatch,
				P:  "/github.com/rjeczalik/fakerpc",
				NP: "/github.com/rjeczalik/fakerpc",
				E:  Create | Delete | Write,
				NE: Create | Delete,
			}},
		},
	}}
	fixture := NewTreeFixture()
	// 1. Setup fixture tree with watches.
	fixture.TestCalls(t, setup[:])
	// 2. Test the fixture.
	fixture.TestEvents(t, events[:])
	// 3. Call Stop on unwatched paths, which should be a no-op to the Tree.
	fixture.TestCalls(t, nop[:])
	// 4. Call no-ops again, because we can.
	fixture.TestCalls(t, nop[:])
	// 6. Test the tree again.
	fixture.TestEvents(t, events[:])
	// 7. The cherry - test Stop on recursive watchpoints.
	fixture.TestCalls(t, cases[:3])
	// 8. ???
	// 9. Ensure no extra events were dispatched (and there was no 5).
	if ei := ch.Drain(); len(ei) != 0 {
		t.Errorf("want ei=nil; got %v", ei)
	}
}

func TestTreeRecursiveWatch(t *testing.T) {
	t.Parallel()
	ch := NewChans(6)
	calls := [...]CallCase{{
		// i=0 create new watchpoint
		Call: Call{
			F: FuncWatch,
			C: ch[0],
			P: "/github.com/rjeczalik/fakerpc/...",
			E: Create | Delete,
		},
		Record: Record{
			TreeFakeRecursive: {{
				F: FuncWatch,
				P: "/github.com/rjeczalik/fakerpc",
				E: Create | Delete,
			}, {
				F: FuncWatch,
				P: "/github.com/rjeczalik/fakerpc/cli",
				E: Create | Delete,
			}, {
				F: FuncWatch,
				P: "/github.com/rjeczalik/fakerpc/cmd",
				E: Create | Delete,
			}, {
				F: FuncWatch,
				P: "/github.com/rjeczalik/fakerpc/cmd/fakerpc",
				E: Create | Delete,
			}},
			TreeNativeRecursive: {{
				F: FuncRecursiveWatch,
				P: "/github.com/rjeczalik/fakerpc",
				E: Create | Delete,
			}},
		},
	}, { // i=1 create new watchpoint
		Call: Call{
			F: FuncWatch,
			C: ch[1],
			P: "/github.com/rjeczalik/fs/...",
			E: Create | Write,
		},
		Record: Record{
			TreeFakeRecursive: {{
				F: FuncWatch,
				P: "/github.com/rjeczalik/fs",
				E: Create | Write,
			}, {
				F: FuncWatch,
				P: "/github.com/rjeczalik/fs/cmd",
				E: Create | Write,
			}, {
				F: FuncWatch,
				P: "/github.com/rjeczalik/fs/fsutil",
				E: Create | Write,
			}, {
				F: FuncWatch,
				P: "/github.com/rjeczalik/fs/memfs",
				E: Create | Write,
			}, {
				F: FuncWatch,
				P: "/github.com/rjeczalik/fs/cmd/gotree",
				E: Create | Write,
			}, {
				F: FuncWatch,
				P: "/github.com/rjeczalik/fs/cmd/mktree",
				E: Create | Write,
			}},
			TreeNativeRecursive: {{
				F: FuncRecursiveWatch,
				P: "/github.com/rjeczalik/fs",
				E: Create | Write,
			}},
		},
	}, { // i=2 use existing watchpoint (from i=1)
		Call: Call{
			F: FuncWatch,
			C: ch[2],
			P: "/github.com/rjeczalik/fs/cmd/...",
			E: Create | Write,
		},
		Record: nil,
	}, { // i=3 rewatch oldp==newp subtree
		Call: Call{
			F: FuncWatch,
			C: ch[3],
			P: "/github.com/rjeczalik/fakerpc/...",
			E: Create | Write,
		},
		Record: Record{
			TreeFakeRecursive: {{
				F:  FuncRewatch,
				P:  "/github.com/rjeczalik/fakerpc",
				E:  Create | Delete,
				NE: Create | Delete | Write,
			}, {
				F:  FuncRewatch,
				P:  "/github.com/rjeczalik/fakerpc/cli",
				E:  Create | Delete,
				NE: Create | Delete | Write,
			}, {
				F:  FuncRewatch,
				P:  "/github.com/rjeczalik/fakerpc/cmd",
				E:  Create | Delete,
				NE: Create | Delete | Write,
			}, {
				F:  FuncRewatch,
				P:  "/github.com/rjeczalik/fakerpc/cmd/fakerpc",
				E:  Create | Delete,
				NE: Create | Delete | Write,
			}},
			TreeNativeRecursive: {{
				F:  FuncRecursiveRewatch,
				P:  "/github.com/rjeczalik/fakerpc",
				NP: "/github.com/rjeczalik/fakerpc",
				E:  Create | Delete,
				NE: Create | Delete | Write,
			}},
		},
	}, { // i=4 rewatch oldp!=newp subtree
		Call: Call{
			F: FuncWatch,
			C: ch[4],
			P: "/github.com/rjeczalik/fakerpc/cmd/...",
			E: Delete | Move,
		},
		Record: Record{
			TreeFakeRecursive: {{
				F:  FuncRewatch,
				P:  "/github.com/rjeczalik/fakerpc/cmd",
				E:  Create | Delete | Write,
				NE: Create | Write | Move | Delete,
			}, {
				F:  FuncRewatch,
				P:  "/github.com/rjeczalik/fakerpc/cmd/fakerpc",
				E:  Create | Delete | Write,
				NE: Create | Write | Move | Delete,
			}},
			TreeNativeRecursive: {{
				F:  FuncRecursiveRewatch,
				P:  "/github.com/rjeczalik/fakerpc",
				NP: "/github.com/rjeczalik/fakerpc",
				E:  Create | Delete | Write,
				NE: Create | Write | Move | Delete,
			}},
		},
	}, { // i=5 merge two subtree watchpoints into one subtree watchpoint
		Call: Call{
			F: FuncWatch,
			C: ch[4],
			P: "/github.com/rjeczalik/...",
			E: Create,
		},
		Record: Record{
			TreeFakeRecursive: {{
				F: FuncWatch,
				P: "/github.com/rjeczalik",
				E: Create,
			}, {
				F: FuncWatch,
				P: "/github.com/rjeczalik/which",
				E: Create,
			}, {
				F: FuncWatch,
				P: "/github.com/rjeczalik/which/cmd",
				E: Create,
			}, {
				F: FuncWatch,
				P: "/github.com/rjeczalik/which/testdata",
				E: Create,
			}, {
				F: FuncWatch,
				P: "/github.com/rjeczalik/which/testdata/cmd",
				E: Create,
			}, {
				F: FuncWatch,
				P: "/github.com/rjeczalik/which/testdata/darwin_386",
				E: Create,
			}, {
				F: FuncWatch,
				P: "/github.com/rjeczalik/which/testdata/darwin_amd64",
				E: Create,
			}, {
				F: FuncWatch,
				P: "/github.com/rjeczalik/which/testdata/freebsd_386",
				E: Create,
			}, {
				F: FuncWatch,
				P: "/github.com/rjeczalik/which/testdata/freebsd_amd64",
				E: Create,
			}, {
				F: FuncWatch,
				P: "/github.com/rjeczalik/which/testdata/linux_386",
				E: Create,
			}, {
				F: FuncWatch,
				P: "/github.com/rjeczalik/which/testdata/linux_amd64",
				E: Create,
			}, {
				F: FuncWatch,
				P: "/github.com/rjeczalik/which/testdata/windows_386",
				E: Create,
			}, {
				F: FuncWatch,
				P: "/github.com/rjeczalik/which/testdata/windows_amd64",
				E: Create,
			}, {
				F: FuncWatch,
				P: "/github.com/rjeczalik/which/testdata/cmd/echo",
				E: Create,
			}, {
				F: FuncWatch,
				P: "/github.com/rjeczalik/which/cmd/gofile",
				E: Create,
			}, {
				F: FuncWatch,
				P: "/github.com/rjeczalik/which/cmd/gowhich",
				E: Create,
			}},
			TreeNativeRecursive: {{
				F: FuncRecursiveUnwatch,
				P: "/github.com/rjeczalik/fs",
			}, {
				F:  FuncRecursiveRewatch,
				P:  "/github.com/rjeczalik/fakerpc",
				NP: "/github.com/rjeczalik",
				NE: Create | Delete | Move | Write,
			}},
		},
	}, { // i=6 plant new recursive watchpoint in already watched subtree
		Call: Call{
			F: FuncWatch,
			C: ch[5],
			P: "/github.com/rjeczalik/which/cmd/...",
			E: Delete | Write,
		},
		Record: Record{
			TreeFakeRecursive: {{
				F:  FuncRewatch,
				P:  "/github.com/rjeczalik/which/cmd",
				E:  Create,
				NE: Create | Delete | Write,
			}, {
				F:  FuncRewatch,
				P:  "/github.com/rjeczalik/which/cmd/gofile",
				E:  Create,
				NE: Create | Delete | Write,
			}, {
				F:  FuncRewatch,
				P:  "/github.com/rjeczalik/which/cmd/gowhich",
				E:  Create,
				NE: Create | Delete | Write,
			}},
			TreeNativeRecursive: nil,
		},
	}}
	events := [...]EventCase{{
		// i=0
		Event: TreeEvent{
			P: "/github.com/rjeczalik/fakerpc/cmd/fakerpc/.main.go.swp",
			E: Create,
		},
		Receiver: Chans{ch[0], ch[3], ch[4]},
	}, { // i=1
		Event: TreeEvent{
			P: "/github.com/rjeczalik/fakerpc/cmd/fakerpc/.main.go.swp",
			E: Delete,
		},
		Receiver: Chans{ch[0], ch[4]},
	}, { // i=2
		Event: TreeEvent{
			P: "/github.com/rjeczalik/which/cmd/gowhich/.main.go.swp",
			E: Create,
		},
		Receiver: Chans{ch[4]},
	}, { // i=3
		Event: TreeEvent{
			P: "/github.com/rjeczalik/fs/cmd/gofs",
			E: Create,
		},
		Receiver: Chans{ch[1], ch[2], ch[4]},
	}}
	fixture := NewTreeFixture()
	fixture.TestCalls(t, calls[:])
	fixture.TestEvents(t, events[:])
	// Ensure no extra events were dispatched.
	if ei := ch.Drain(); len(ei) != 0 {
		t.Errorf("want ei=nil; got %v", ei)
	}
}

func TestTreeRecursiveUnwatch(t *testing.T) {
	t.skip("TODO(rjeczalik)")
}
