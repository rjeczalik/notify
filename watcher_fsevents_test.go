// +build darwin,!kqueue

package notify

import (
	"reflect"
	"testing"
)

func TestSplitflags(t *testing.T) {
	cases := [...]struct {
		set   uint32
		flags []uint32
	}{
		{0, nil},
		{0xD, []uint32{0x1, 0x4, 0x8}},
		{0x0010 | 0x0040 | 0x0080 | 0x01000, []uint32{0x0010, 0x0040, 0x0080, 0x01000}},
		{0x40000 | 0x00100 | 0x00200, []uint32{0x00100, 0x00200, 0x40000}},
	}
	for i, cas := range cases {
		if flags := splitflags(cas.set); !reflect.DeepEqual(flags, cas.flags) {
			t.Errorf("want flags=%v; got %v (i=%d)", cas.flags, flags, i)
		}
	}
}

func TestFlagdiff(t *testing.T) {
	const (
		create = uint32(FSEventsCreated)
		remove = uint32(FSEventsRemoved)
		rename = uint32(FSEventsRenamed)
		write  = uint32(FSEventsModified)
		inode  = uint32(FSEventsInodeMetaMod)
	)
	fd := make(flagdiff)
	cases := [...]struct {
		flag uint32
		diff uint32
	}{
		{create | remove, create | remove},
		{create | remove | write, write},
		{create | remove, create | remove},
		{create | remove | write, write},
		{write, write},
		{create | remove | write, create | remove},
		{write, write},
		{write, write},
		{remove, remove},
		{create | write, create | write},
		{create | write, write},
		{write, write},
		{remove, remove},
		{create | remove, create},
		{write, write},
		{create | remove, create | remove},
		{create | remove | write, write},
	}
	for i, cas := range cases {
		if diff := fd.diff("", cas.flag); diff != cas.diff {
			t.Errorf("want diff=%v; got %v (i=%d)", Event(cas.diff), Event(diff), i)
		}
	}
}

// Test for cases 3) and 5) with shadowed write&create events.
//
// See comment for (flagdiff).diff method.
func TestWatcherShadowedWriteCreate(t *testing.T) {
	w := NewWatcherTest(t, "testdata/vfs.txt")
	defer w.Close()

	cases := [...]WCase{
		create(w, "src/github.com/rjeczalik/fs/.fs.go.swp"),
		write(w, "src/github.com/rjeczalik/fs/.fs.go.swp", []byte("XD")),
		write(w, "src/github.com/rjeczalik/fs/.fs.go.swp", []byte("XD")),
		remove(w, "src/github.com/rjeczalik/fs/.fs.go.swp"),
		create(w, "src/github.com/rjeczalik/fs/.fs.go.swp"),
		write(w, "src/github.com/rjeczalik/fs/.fs.go.swp", []byte("XD")),
	}

	w.ExpectAny(cases[:])
}
