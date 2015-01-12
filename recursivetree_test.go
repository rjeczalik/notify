package notify

import "testing"

func TestRecursiveTreeWatch(t *testing.T) {
	n := NewTreeTest(t, "testdata/gopath.txt")
	defer n.Close()

	ch := NewChans(3)

	cases := [...]RCase{{
		// i=0
		Call: Call{
			F: FuncWatch,
			C: ch[0],
			P: "src/github.com/rjeczalik/fakerpc",
			E: Create | Delete | Move,
		},
		Record: []Call{{
			F: FuncWatch,
			P: "src/github.com/rjeczalik/fakerpc",
			E: Delete | Create | Move,
		}},
	}, { // i=1
		Call: Call{
			F: FuncWatch,
			C: ch[1],
			P: "src/github.com/rjeczalik/fakerpc",
			E: Delete | Move,
		},
		Record: nil,
	}, { // i=2
		Call: Call{
			F: FuncWatch,
			C: ch[2],
			P: "src/github.com/rjeczalik/fakerpc",
			E: Move,
		},
		Record: nil,
	}, { // i=3
		Call: Call{
			F: FuncWatch,
			C: ch[0],
			P: "src/github.com/rjeczalik/fs",
			E: Create | Delete,
		},
		Record: []Call{{
			F: FuncWatch,
			P: "src/github.com/rjeczalik/fs",
			E: Create | Delete,
		}},
	}, { // i=4
		Call: Call{
			F: FuncWatch,
			C: ch[0],
			P: "src/github.com/rjeczalik/fs",
			E: Create,
		},
		Record: nil,
	}, { // i=5
		Call: Call{
			F: FuncStop,
			C: ch[0],
		},
		Record: []Call{{
			F:  FuncRewatch,
			P:  "src/github.com/rjeczalik/fakerpc",
			E:  Create | Delete | Move,
			NE: Delete | Move,
		}, {
			F: FuncUnwatch,
			P: "src/github.com/rjeczalik/fs",
		}},
	}, { // i=6
		Call: Call{
			F: FuncWatch,
			C: ch[0],
			P: "src/github.com/rjeczalik/which",
			E: Create,
		},
		Record: []Call{{
			F: FuncWatch,
			P: "src/github.com/rjeczalik/which",
			E: Create,
		}},
	}, { // i=7
		Call: Call{
			F: FuncWatch,
			C: ch[1],
			P: "src/github.com/rjeczalik/which",
			E: Delete,
		},
		Record: []Call{{
			F:  FuncRewatch,
			P:  "src/github.com/rjeczalik/which",
			E:  Create,
			NE: Create | Delete,
		}},
	}}
	events := [...]TCase{{
		Event: Call{
			P: "src/github.com/rjeczalik/fakerpc/.fakerpc.go.swp",
			E: Delete,
		},
		Receiver: Chans{ch[1]},
	}, {
		Event: Call{
			P: "src/github.com/rjeczalik/fakerpc/.travis.yml",
			E: Move,
		},
		Receiver: Chans{ch[1], ch[2]},
	}, {
		Event: Call{
			P: "src/github.com/rjeczalik/fakerpc/which",
			E: Delete,
		},
		Receiver: Chans{ch[1]},
	}}

	n.ExpectRecordedCalls(cases[:])
	n.ExpectTreeEvents(events[:])

	// Ensure no extra events were dispatched.
	if ei := ch.Drain(); len(ei) != 0 {
		t.Errorf("want ei=nil; got %v", ei)
	}
}
