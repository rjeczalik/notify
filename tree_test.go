package notify

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
	//t.Skip("TODO(rjeczalik)")
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
			TreeWatcher: {{
				F: FuncUnwatch,
				P: "/github.com/rjeczalik/fakerpc",
			}, {
				F: FuncWatch,
				P: "/github.com/rjeczalik/fakerpc",
				E: Delete | Move,
			}, {
				F: FuncUnwatch,
				P: "/github.com/rjeczalik/fs",
			}},
			TreeRewatcher | TreeRecursive: {{
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
			TreeRewatcher: {{
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
			TreeRecursive: {{
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
			TreeRewatcher: {{
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
			TreeRecursive: {{
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
			TreeRewatcher: {{
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
			TreeRecursive: {{
				F:  FuncRecursiveRewatch,
				P:  "/github.com/rjeczalik/fakerpc",
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
			TreeRewatcher: {{
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
			TreeRecursive: {{
				F:  FuncRecursiveRewatch,
				P:  "/github.com/rjeczalik/fakerpc",
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
	fixture := NewTreeFixture(TreeRewatcher) //, TreeRecursive)
	fixture.TestCalls(t, calls[:])
}

func TestTreeSplit(t *testing.T) {
	cases := [...]struct {
		path string
		dir  string
		base string
	}{
		{"/github.com/rjeczalik/fakerpc", "/github.com/rjeczalik", "fakerpc"},
		{"/home/rjeczalik/src", "/home/rjeczalik", "src"},
		{"/Users/pknap/porn/gopher.avi", "/Users/pknap/porn", "gopher.avi"},
		{"C:/Documents and Users", "C:", "Documents and Users"},
		{"C:/Documents and Users/pblaszczyk/wiertarka.exe", "C:/Documents and Users/pblaszczyk", "wiertarka.exe"},
		{"/home/(╯°□°）╯︵ ┻━┻", "/home", "(╯°□°）╯︵ ┻━┻"},
	}
	for i, cas := range cases {
		dir, base := Split(filepath.FromSlash(cas.path))
		if dir != cas.dir {
			t.Errorf("want dir=%s; got %s (i=%d)", cas.dir, dir, i)
		}
		if base != cas.base {
			t.Errorf("want base=%s; got %s (i=%d)", cas.base, base, i)
		}
	}
}

func TestTreeBase(t *testing.T) {
	cases := [...]struct {
		path string
		base string
	}{
		{"/github.com/rjeczalik/fakerpc", "fakerpc"},
		{"/home/rjeczalik/src", "src"},
		{"/Users/pknap/porn/gopher.avi", "gopher.avi"},
		{"C:/Documents and Users", "Documents and Users"},
		{"C:/Documents and Users/pblaszczyk/wiertarka.exe", "wiertarka.exe"},
		{"/home/(╯°□°）╯︵ ┻━┻", "(╯°□°）╯︵ ┻━┻"},
	}
	for i, cas := range cases {
		if base := Base(filepath.FromSlash(cas.path)); base != cas.base {
			t.Errorf("want base=%s; got %s (i=%d)", cas.base, base, i)
		}
	}
}

func TestTreeIndexSep(t *testing.T) {
	cases := [...]struct {
		path string
		n    int
	}{
		{"github.com/rjeczalik/fakerpc", 10},
		{"home/rjeczalik/src", 4},
		{"Users/pknap/porn/gopher.avi", 5},
		{"C:/Documents and Users", 2},
		{"Documents and Users/pblaszczyk/wiertarka.exe", 19},
		{"(╯°□°）╯︵ ┻━┻/Downloads", 30},
	}
	for i, cas := range cases {
		if n := IndexSep(filepath.FromSlash(cas.path)); n != cas.n {
			t.Errorf("want n=%d; got %d (i=%d)", cas.n, n, i)
		}
	}
}

func TestTreeLastIndexSep(t *testing.T) {
	cases := [...]struct {
		path string
		n    int
	}{
		{"github.com/rjeczalik/fakerpc", 20},
		{"home/rjeczalik/src", 14},
		{"Users/pknap/porn/gopher.avi", 16},
		{"C:/Documents and Users", 2},
		{"Documents and Users/pblaszczyk/wiertarka.exe", 30},
		{"/home/(╯°□°）╯︵ ┻━┻", 5},
	}
	for i, cas := range cases {
		if n := LastIndexSep(filepath.FromSlash(cas.path)); n != cas.n {
			t.Errorf("want n=%d; got %d (i=%d)", cas.n, n, i)
		}
	}
}

func TestTreeNodeSet(t *testing.T) {
	cases := [...]struct {
		nd  []Node
		nds NodeSet
	}{{
		[]Node{{Name: "g"}, {Name: "t"}, {Name: "u"}, {Name: "a"}, {Name: "b"}},
		NodeSet{{Name: "a"}, {Name: "b"}, {Name: "g"}, {Name: "t"}, {Name: "u"}},
	}, {
		[]Node{{Name: "aA"}, {Name: "aA"}, {Name: "aa"}, {Name: "AA"}},
		NodeSet{{Name: "AA"}, {Name: "aA"}, {Name: "aa"}},
	}, {
		[]Node{{Name: "b"}, {Name: "b"}, {Name: "a"}, {Name: "Y"}, {Name: ""}, {Name: "a"}},
		NodeSet{{Name: ""}, {Name: "Y"}, {Name: "a"}, {Name: "b"}},
	}}
Test:
	for i, cas := range cases {
		nds := NodeSet{}
		for _, nd := range cas.nd {
			nds.Add(nd)
		}
		if !reflect.DeepEqual(nds, cas.nds) {
			t.Errorf("want nds=%v; got %v (i=%d)", cas.nds, nds, i)
			continue Test
		}
		for _, nd := range cas.nd {
			if j := nds.Search(nd); nds[j].Name != nd.Name {
				t.Errorf("want nds[%d]=%v; got %v (i=%d)", j, nd, nds[j], i)
				continue Test
			}
		}
		for _, nd := range cas.nd {
			nds.Del(nd)
		}
		if n := len(nds); n != 0 {
			t.Errorf("want len(nds)=0; got %d (i=%d)", n, i)
			continue Test
		}
	}
}

func TestTreeChanNodesMap(t *testing.T) {
	ch := NewChans(10)
	cases := [...]struct {
		ch  Chans
		cnd ChanNodesMap
	}{{
		Chans{ch[0]},
		ChanNodesMap{ch[0]: {{Name: "0"}}},
	}, {
		Chans{ch[0], ch[0], ch[0]},
		ChanNodesMap{
			ch[0]: {{Name: "0"}, {Name: "1"}, {Name: "2"}},
		},
	}, {
		Chans{ch[0], ch[3], ch[2], ch[1]},
		ChanNodesMap{
			ch[0]: {{Name: "0"}},
			ch[1]: {{Name: "3"}},
			ch[2]: {{Name: "2"}},
			ch[3]: {{Name: "1"}},
		},
	}, {
		Chans{ch[0], ch[0], ch[2], ch[1], ch[3], ch[3], ch[2], ch[2], ch[4], ch[0]},
		ChanNodesMap{
			ch[0]: {{Name: "0"}, {Name: "1"}, {Name: "9"}},
			ch[1]: {{Name: "3"}},
			ch[2]: {{Name: "2"}, {Name: "6"}, {Name: "7"}},
			ch[3]: {{Name: "4"}, {Name: "5"}},
			ch[4]: {{Name: "8"}},
		},
	}}
	for i, cas := range cases {
		cnd := make(ChanNodesMap)
		cas.ch.Foreach(cnd.Add)
		if !reflect.DeepEqual(cnd, cas.cnd) {
			t.Errorf("want cnd=%v; got %v (i=%d)", cas.cnd, cnd, i)
			continue
		}
		cas.ch.Foreach(cnd.Del)
		if n := len(cnd); n != 0 {
			t.Errorf("want len(cnd)=0; got %d (i=%d)", n, i)
			continue
		}
	}
}
