package notify

import "testing"

func TestRecursiveTreeWatch(t *testing.T) {
	n := NewRecursiveTreeTest(t, "testdata/vfs.txt")
	defer n.Close()

	ch := NewChans(1)

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
				C: ch[0],
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
				C: ch[0],
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
}
