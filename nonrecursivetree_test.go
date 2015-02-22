package notify

import "testing"

func TestNonrecursiveTree(t *testing.T) {
	n := NewNonrecursiveTreeTest(t, "testdata/vfs.txt")
	defer n.Close()

	ch := NewChans(5)

	watches := [...]RCase{
		// i=0
		{
			Call: Call{
				F: FuncWatch,
				P: "src/github.com/rjeczalik/fs/fs.go",
				C: ch[0],
				E: Rename,
			},
			Record: []Call{
				{
					F: FuncWatch,
					P: "src/github.com/rjeczalik/fs/fs.go",
					E: Rename,
				},
			},
		},
		// i=1
		{
			Call: Call{
				F: FuncWatch,
				P: "src/github.com/rjeczalik/fs/cmd/...",
				C: ch[1],
				E: Remove,
			},
			Record: []Call{
				{
					F: FuncWatch,
					P: "src/github.com/rjeczalik/fs/cmd",
					E: Create | Remove,
				},
				{
					F: FuncWatch,
					P: "src/github.com/rjeczalik/fs/cmd/mktree",
					E: Create | Remove,
				},
				{
					F: FuncWatch,
					P: "src/github.com/rjeczalik/fs/cmd/gotree",
					E: Create | Remove,
				},
			},
		},
		// i=2
		{
			Call: Call{
				F: FuncWatch,
				P: "src/github.com/rjeczalik/fs/cmd/...",
				C: ch[2],
				E: Rename,
			},
			Record: []Call{
				{
					F:  FuncRewatch,
					P:  "src/github.com/rjeczalik/fs/cmd",
					E:  Create | Remove,
					NE: Create | Remove | Rename,
				},
				{
					F:  FuncRewatch,
					P:  "src/github.com/rjeczalik/fs/cmd/mktree",
					E:  Create | Remove,
					NE: Create | Remove | Rename,
				},
				{
					F:  FuncRewatch,
					P:  "src/github.com/rjeczalik/fs/cmd/gotree",
					E:  Create | Remove,
					NE: Create | Remove | Rename,
				},
			},
		},
		// i=3
		{
			Call: Call{
				F: FuncWatch,
				P: "src/github.com/rjeczalik/fs/cmd/mktree/...",
				C: ch[2],
				E: Write,
			},
			Record: []Call{
				{
					F:  FuncRewatch,
					P:  "src/github.com/rjeczalik/fs/cmd/mktree",
					E:  Create | Remove | Rename,
					NE: Create | Remove | Rename | Write,
				},
			},
		},
		// i=4
		{
			Call: Call{
				F: FuncWatch,
				P: "src/github.com/pblaszczyk/qttu/include",
				C: ch[3],
				E: Create,
			},
			Record: []Call{
				{
					F: FuncWatch,
					P: "src/github.com/pblaszczyk/qttu/include",
					E: Create,
				},
			},
		},
		// i=5
		{
			Call: Call{
				F: FuncWatch,
				P: "src/github.com/pblaszczyk/qttu/include/qttu/detail/...",
				C: ch[3],
				E: Write,
			},
			Record: []Call{
				{
					F: FuncWatch,
					P: "src/github.com/pblaszczyk/qttu/include/qttu/detail",
					E: Create | Write,
				},
			},
		},
		// i=6
		{
			Call: Call{
				F: FuncWatch,
				P: "src/github.com/pblaszczyk/qttu/include/...",
				C: ch[0],
				E: Rename,
			},
			Record: []Call{
				{
					F:  FuncRewatch,
					P:  "src/github.com/pblaszczyk/qttu/include",
					E:  Create,
					NE: Create | Rename,
				},
				{
					F: FuncWatch,
					P: "src/github.com/pblaszczyk/qttu/include/qttu",
					E: Create | Rename,
				},
				{
					F:  FuncRewatch,
					P:  "src/github.com/pblaszczyk/qttu/include/qttu/detail",
					E:  Create | Write,
					NE: Create | Write | Rename,
				},
			},
		},
		// i=7
		{
			Call: Call{
				F: FuncWatch,
				P: "src/github.com/pblaszczyk/...",
				C: ch[1],
				E: Write,
			},
			Record: []Call{
				{
					F: FuncWatch,
					P: "src/github.com/pblaszczyk",
					E: Create | Write,
				},
				{
					F: FuncWatch,
					P: "src/github.com/pblaszczyk/qttu",
					E: Create | Write,
				},
				{
					F: FuncWatch,
					P: "src/github.com/pblaszczyk/qttu/src",
					E: Create | Write,
				},
				{
					F:  FuncRewatch,
					P:  "src/github.com/pblaszczyk/qttu/include",
					E:  Create | Rename,
					NE: Create | Rename | Write,
				},
				{
					F:  FuncRewatch,
					P:  "src/github.com/pblaszczyk/qttu/include/qttu",
					E:  Create | Rename,
					NE: Create | Rename | Write,
				},
			},
		},
		// i=8
		{
			Call: Call{
				F: FuncWatch,
				P: "src/github.com/pblaszczyk/qttu/include/...",
				C: ch[4],
				E: Write,
			},
			Record: nil,
		},
		// i=9
		{
			Call: Call{
				F: FuncWatch,
				P: "src/github.com/pblaszczyk/qttu",
				C: ch[3],
				E: Remove,
			},
			Record: []Call{
				{
					F:  FuncRewatch,
					P:  "src/github.com/pblaszczyk/qttu",
					E:  Create | Write,
					NE: Create | Write | Remove,
				},
			},
		},
	}

	n.ExpectRecordedCalls(watches[:])

	events := [...]TCase{
		// i=0
		{
			Event:    Call{P: "src/github.com/rjeczalik/fs/fs.go", E: Rename},
			Receiver: Chans{ch[0]},
		},
		// i=1
		{
			Event:    Call{P: "src/github.com/rjeczalik/fs/fs.go", E: Create},
			Receiver: nil,
		},
		// i=2
		{
			Event:    Call{P: "src/github.com/rjeczalik/fs/cmd/cmd.go", E: Remove},
			Receiver: Chans{ch[1]},
		},
		// i=3
		{
			Event:    Call{P: "src/github.com/rjeczalik/fs/cmd/doc.go", E: Write},
			Receiver: nil,
		},
		// i=4
		{
			Event:    Call{P: "src/github.com/rjeczalik/fs/cmd/mktree/main.go", E: Write},
			Receiver: Chans{ch[2]},
		},
		// i=5
		{
			Event:    Call{P: "src/github.com/rjeczalik/fs/cmd/mktree/tree.go", E: Create},
			Receiver: nil,
		},
		// i=6
		{
			Event:    Call{P: "src/github.com/pblaszczyk/qttu/include/.lock", E: Create},
			Receiver: Chans{ch[3]},
		},
		// i=7
		{
			Event:    Call{P: "src/github.com/pblaszczyk/qttu/include/qttu/detail/registry.hh", E: Write},
			Receiver: Chans{ch[3], ch[1], ch[4]},
		},
		// i=8
		{
			Event:    Call{P: "src/github.com/pblaszczyk/qttu/include/qttu", E: Remove},
			Receiver: nil,
		},
		// i=9
		{
			Event:    Call{P: "src/github.com/pblaszczyk/qttu/include", E: Remove},
			Receiver: Chans{ch[3]},
		},
	}

	n.ExpectTreeEvents(events[:], ch)
}
