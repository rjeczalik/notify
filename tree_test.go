// +build !windows

package notify

// TODO(rjeczalik): Tree is currently broken under Windows

import (
	"path/filepath"
	"reflect"
	"testing"
)

func (nd Node) Path(s ...string) Node {
	for _, s := range s {
		nd = nd.Child[s]
	}
	return nd
}

func TestTreeLookPath(t *testing.T) {
	t.Skip("TODO(rjeczalik)")
}

func TestTreeLook(t *testing.T) {
	t.Skip("TODO(rjeczalik)")
}

func p(p string) string {
	return filepath.FromSlash(p)
}

func TestTreeDel(t *testing.T) {
	cases := [...]struct {
		before Node
		p      string
		after  Node
	}{{
		Node{Child: map[string]Node{
			"a": {
				Name: p("/a"),
				Child: map[string]Node{
					"b": {
						Name: p("/a/b"),
						Child: map[string]Node{
							"c": {
								Name: p("/a/b/c"),
								Child: map[string]Node{
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
		Node{Child: map[string]Node{
			"a": {
				Name: p("/a"),
				Child: map[string]Node{
					"x": {
						Name: p("/a/x"),
					},
				},
			},
		}},
	}, {
		Node{Child: map[string]Node{
			"a": {
				Name: p("/a"),
				Child: map[string]Node{
					"b": {
						Name: p("/a/b"),
						Child: map[string]Node{
							"c": {
								Name: p("/a/b/c"),
							},
						},
					},
				},
			},
		}},
		"/a/b/c",
		Node{Child: map[string]Node{}},
	}}
	for i, cas := range cases {
		if (&Tree{Root: cas.before}).Del(cas.p); !reflect.DeepEqual(cas.before, cas.after) {
			t.Errorf("want tree=%v; got %v (i=%d)", cas.after, cas.before, i)
		}
	}
}

func TestTreeWalkPath(t *testing.T) {
	t.Skip("TODO(rjeczalik)")
}

func TestTreeWalkDir(t *testing.T) {
	t.Skip("TODO(rjeczalik)")
}

func TestTreeWalk(t *testing.T) {
	t.Skip("TODO(rjeczalik)")
}

func TestTreeDir(t *testing.T) {
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
	}}
	fixture := NewTreeFixture()
	fixture.TestCalls(t, calls[:])
	fixture.TestEvents(t, events[:])
}

func TestTreeRecursiveDir(t *testing.T) {
	ch := NewChans(6)
	calls := [...]CallCase{{ // i=0 create new watchpoint
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
	}, { // i=4 rewatch oldp!=newp subtree (optimization: rewatch newp only?)
		Call: Call{
			F: FuncWatch,
			C: ch[4],
			P: "/github.com/rjeczalik/fakerpc/cmd/...",
			E: Delete | Move,
		},
		Record: Record{
			TreeFakeRecursive: {{
				F:  FuncRewatch,
				P:  "/github.com/rjeczalik/fakerpc",
				E:  Create | Delete | Write,
				NE: Create | Write | Move | Delete,
			}, {
				F:  FuncRewatch,
				P:  "/github.com/rjeczalik/fakerpc/cli",
				E:  Create | Delete | Write,
				NE: Create | Write | Move | Delete,
			}, {
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
		// }, { // i=4
		// TODO(rjeczalik): merge watchpoints
		// Call: Call{
		//	F: FuncWatch,
		//	C: ch[4],
		//	P: "/github.com/rjeczalik/...",
		//	E: Create,
		// },
		// Record: Record{
		// // TODO
		// },
		// }, { // i=5
		// TODO(rjeczalik): rewatch subtree
		// Call: Call{
		// 	F: FuncWatch,
		//	C: ch[5],
		//	P: "/github.com/rjeczalik/...",
		//	E: Move,
		// },
		// Record: Record{
		// // TODO
		// },
	}}
	fixture := NewTreeFixture()
	fixture.TestCalls(t, calls[:])
}
