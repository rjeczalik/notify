package notify

import "testing"

func TestRecursiveTree(t *testing.T) {
	n := NewRecursiveTreeTest(t, "testdata/vfs.txt")
	defer n.Close()

	ch := NewChans(3)

	calls := [...]RCase{
		// i=0
		{
			Call: Call{
				F: FuncWatch,
				P: "src/github.com/rjeczalik/fs/fs.go",
				C: ch[0],
				E: Create,
			},
			Record: []Call{
				{
					F: FuncWatch,
					P: "src/github.com/rjeczalik/fs/fs.go",
					E: Create,
				},
			},
		},
		// i=1
		{
			Call: Call{
				F: FuncWatch,
				P: "src/github.com/rjeczalik/fs/cmd/...",
				C: ch[1],
				E: Delete,
			},
			Record: []Call{
				{
					F: FuncRecursiveWatch,
					P: "src/github.com/rjeczalik/fs/cmd",
					E: Delete,
				},
			},
		},
		// i=2
		{
			Call: Call{
				F: FuncWatch,
				P: "src/github.com/rjeczalik/fs",
				C: ch[2],
				E: Move,
			},
			Record: []Call{
				{
					F: FuncRecursiveWatch,
					P: "src/github.com/rjeczalik/fs",
					E: Create | Delete | Move,
				},
				{
					F: FuncRecursiveUnwatch,
					P: "src/github.com/rjeczalik/fs/cmd",
				},
				{
					F: FuncUnwatch,
					P: "src/github.com/rjeczalik/fs/fs.go",
				},
			},
		},
		// i=3
		{
			Call: Call{
				F: FuncWatch,
				P: "src/github.com/ppknap/link/README.md",
				C: ch[0],
				E: Create,
			},
			Record: []Call{
				{
					F: FuncWatch,
					P: "src/github.com/ppknap/link/README.md",
					E: Create,
				},
			},
		},
		// i=4
		{
			Call: Call{
				F: FuncWatch,
				P: "src/github.com/ppknap/link/include/...",
				C: ch[0],
				E: Delete,
			},
			Record: []Call{
				{
					F: FuncRecursiveWatch,
					P: "src/github.com/ppknap/link/include",
					E: Delete,
				},
			},
		},
		// i=5
		{
			Call: Call{
				F: FuncWatch,
				P: "src/github.com/ppknap/link/include",
				C: ch[0],
				E: Write,
			},
			Record: []Call{
				{
					F:  FuncRecursiveRewatch,
					P:  "src/github.com/ppknap/link/include",
					NP: "src/github.com/ppknap/link/include",
					E:  Delete,
					NE: Delete | Write,
				},
			},
		},
		// i=6
		{
			Call: Call{
				F: FuncWatch,
				P: "src/github.com/ppknap/link/test/Jamfile.jam",
				C: ch[0],
				E: Move,
			},
			Record: []Call{
				{
					F: FuncWatch,
					P: "src/github.com/ppknap/link/test/Jamfile.jam",
					E: Move,
				},
			},
		},
		// i=7
		{
			Call: Call{
				F: FuncWatch,
				P: "src/github.com/ppknap/link/test/Jamfile.jam",
				C: ch[0],
				E: Create,
			},
			Record: []Call{
				{
					F:  FuncRewatch,
					P:  "src/github.com/ppknap/link/test/Jamfile.jam",
					E:  Move,
					NE: Move | Create,
				},
			},
		},
		// i=8
		{
			Call: Call{
				F: FuncWatch,
				P: "src/github.com/ppknap/...",
				C: ch[0],
				E: Create,
			},
			Record: []Call{
				{
					F: FuncRecursiveWatch,
					P: "src/github.com/ppknap",
					E: Create | Delete | Write | Move,
				},
				{
					F: FuncUnwatch,
					P: "src/github.com/ppknap/link/README.md",
				},
				{
					F: FuncRecursiveUnwatch,
					P: "src/github.com/ppknap/link/include",
				},
				{
					F: FuncUnwatch,
					P: "src/github.com/ppknap/link/test/Jamfile.jam",
				},
			},
		},
		// i=9
		{
			Call: Call{
				F: FuncWatch,
				P: "src/github.com/rjeczalik/fs/README.md",
				C: ch[0],
				E: Move,
			},
			Record: nil,
		},
		// i=10
		{
			Call: Call{
				F: FuncWatch,
				P: "src/github.com/rjeczalik/fs/cmd/gotree",
				C: ch[0],
				E: Create | Delete,
			},
			Record: nil,
		},
		// i=11
		{
			Call: Call{
				F: FuncWatch,
				P: "src/github.com/pblaszczyk/qttu/src/main.cc",
				C: ch[0],
				E: Create,
			},
			Record: []Call{
				{
					F: FuncWatch,
					P: "src/github.com/pblaszczyk/qttu/src/main.cc",
					E: Create,
				},
			},
		},
		// i=12
		{
			Call: Call{
				F: FuncWatch,
				P: "src/github.com/pblaszczyk/qttu/src/main.cc",
				C: ch[0],
				E: Delete,
			},
			Record: []Call{
				{
					F:  FuncRewatch,
					P:  "src/github.com/pblaszczyk/qttu/src/main.cc",
					E:  Create,
					NE: Create | Delete,
				},
			},
		},
		// i=13
		{
			Call: Call{
				F: FuncWatch,
				P: "src/github.com/pblaszczyk/qttu/src/main.cc",
				C: ch[0],
				E: Create | Delete,
			},
			Record: nil,
		},
		// i=14
		{
			Call: Call{
				F: FuncWatch,
				P: "src/github.com/pblaszczyk/qttu/src",
				C: ch[0],
				E: Create,
			},
			Record: []Call{
				{
					F:  FuncRecursiveRewatch,
					P:  "src/github.com/pblaszczyk/qttu/src/main.cc",
					NP: "src/github.com/pblaszczyk/qttu/src",
					E:  Create | Delete,
					NE: Create | Delete,
				},
			},
		},
		// i=15
		{
			Call: Call{
				F: FuncWatch,
				P: "src/github.com/pblaszczyk/qttu",
				C: ch[0],
				E: Write,
			},
			Record: []Call{
				{
					F:  FuncRecursiveRewatch,
					P:  "src/github.com/pblaszczyk/qttu/src",
					NP: "src/github.com/pblaszczyk/qttu",
					E:  Create | Delete,
					NE: Create | Delete | Write,
				},
			},
		},
	}

	n.ExpectRecordedCalls(calls[:])

	events := [...]TCase{
		// i=0
		{
			Event:    Call{P: "src/github.com/rjeczalik/fs/fs.go", E: Move},
			Receiver: Chans{ch[2]},
		},
		// i=1
		{
			Event:    Call{P: "src/github.com/rjeczalik/fs/fs.go", E: Create},
			Receiver: Chans{ch[0]},
		},
		// i=2
		{
			Event:    Call{P: "src/github.com/rjeczalik/fs/fs.go/file", E: Create},
			Receiver: Chans{ch[0]},
		},
		// i=3
		{
			Event:    Call{P: "src/github.com/rjeczalik/fs", E: Move},
			Receiver: Chans{ch[2]},
		},
		// i=4
		{
			Event:    Call{P: "src/github.com/rjeczalik/fs/fs_test.go", E: Move},
			Receiver: Chans{ch[2]},
		},
		// i=5
		{
			Event:    Call{P: "src/github.com/rjeczalik/fs/cmd/mktree/main.go", E: Delete},
			Receiver: Chans{ch[1]},
		},
		// i=6
		{
			Event:    Call{P: "src/github.com/rjeczalik/fs/cmd/gotree", E: Delete},
			Receiver: Chans{ch[1]},
		},
		// i=7
		{
			Event:    Call{P: "src/github.com/rjeczalik/fs/cmd", E: Delete},
			Receiver: Chans{ch[1]},
		},
		// i=8
		{
			Event:    Call{P: "src/github.com/rjeczalik/fs/fs.go/file", E: Write},
			Receiver: nil,
		},
		// i=9
		{
			Event:    Call{P: "src/github.com/rjeczalik/fs/fs.go/file", E: Write},
			Receiver: nil,
		},
		// i=10
		{
			Event:    Call{P: "src/github.com/rjeczalik/fs", E: Delete},
			Receiver: nil,
		},
		// i=11
		{
			Event:    Call{P: "src/github.com/rjeczalik/fs/cmd", E: Move},
			Receiver: Chans{ch[2]},
		},
		// i=12
		{
			Event:    Call{P: "src/github.com/rjeczalik/fs/cmd/mktree/main.go", E: Write},
			Receiver: nil,
		},
		// i=13
		{
			Event:    Call{P: "src/github.com/rjeczalik/fs/cmd/gotree", E: Move},
			Receiver: nil,
		},
		// i=14
		{
			Event:    Call{P: "src/github.com/rjeczalik/fs/cmd/file", E: Move},
			Receiver: nil,
		},
	}

	n.ExpectTreeEvents(events[:], ch)
}
