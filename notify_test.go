package notify

import "testing"

func TestNotifyExample(t *testing.T) {
	n := NewNotifyTest(t, "testdata/vfs.txt")
	defer n.Close()

	ch := NewChans(3)

	// Watch-points can be set explicitely via Watch/Stop calls...
	n.Watch("src/github.com/rjeczalik/fs", ch[0], Write)
	n.Watch("src/github.com/pblaszczyk/qttu", ch[0], Write)
	n.Watch("src/github.com/pblaszczyk/qttu/...", ch[1], Create)
	n.Watch("src/github.com/rjeczalik/fs/cmd/...", ch[2], Delete)

	cases := []NCase{
		{
			Event:    create(n.W(), "src/github.com/rjeczalik/fs/fs.go"),
			Receiver: nil,
		},
		{
			Event:    create(n.W(), "src/github.com/pblaszczyk/qttu/LICENSE"),
			Receiver: Chans{ch[1]},
		},
		{
			Event:    create(n.W(), "src/github.com/rjeczalik/fs/cmd/gotree/go.go"),
			Receiver: nil,
		},
		{
			Event:    write(n.W(), "src/github.com/rjeczalik/fs/fs.go", []byte("XD")),
			Receiver: Chans{ch[0]},
		},
		{
			Event:    write(n.W(), "src/github.com/pblaszczyk/qttu/LICENSE", []byte("XD")),
			Receiver: Chans{ch[0]},
		},
		{
			Event:    write(n.W(), "src/github.com/rjeczalik/fs/cmd/gotree/go.go", []byte("XD")),
			Receiver: nil,
		},
		{
			Event:    create(n.W(), "src/github.com/pblaszczyk/qttu/src/.main.cc.swp"),
			Receiver: Chans{ch[1]},
		},
		{
			Event:    create(n.W(), "src/github.com/pblaszczyk/qttu/src/.main.cc.swo"),
			Receiver: Chans{ch[1]},
		},
		{
			Event:    remove(n.W(), "src/github.com/rjeczalik/fs/cmd/gotree/go.go"),
			Receiver: Chans{ch[2]},
		},
	}

	n.ExpectNotifyEvents(cases, ch)

	// ...or using Call structures.
	stops := [...]Call{
		{
			F: FuncStop,
			C: ch[0],
		},
		{
			F: FuncStop,
			C: ch[1],
		},
	}

	n.Call(stops[:]...)

	cases = []NCase{
		{
			Event:    write(n.W(), "src/github.com/rjeczalik/fs/fs.go", []byte("XD")),
			Receiver: nil,
		},
		{
			Event:    write(n.W(), "src/github.com/pblaszczyk/qttu/README.md", []byte("XD")),
			Receiver: nil,
		},
		{
			Event:    create(n.W(), "src/github.com/pblaszczyk/qttu/src/.main.cc.swr"),
			Receiver: nil,
		},
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
