// +build darwin,!kqueue

package notify

import (
	"io/ioutil"
	"os"
	"reflect"
	"testing"
)

func tmpfile(s string) (string, error) {
	f, err := ioutil.TempFile("/tmp", s)
	if err != nil {
		return "", err
	}
	if err = nonil(f.Sync(), f.Close()); err != nil {
		return "", err
	}
	return f.Name(), nil
}

func symlink(s string) (string, error) {
	name, err := tmpfile("symlink")
	if err != nil {
		return "", err
	}
	if err = nonil(os.Remove(name), os.Symlink(s, name)); err != nil {
		return "", err
	}
	return name, nil
}

func removeall(s ...string) {
	for _, s := range s {
		os.Remove(s)
	}
}

type caseCanonical struct {
	path string
	full string
}

func testCanonical(t *testing.T, cases []caseCanonical) {
	for i, cas := range cases {
		full, err := canonical(cas.path)
		if err != nil {
			t.Errorf("want err=nil; got %v (i=%d)", err, i)
			continue
		}
		if full != cas.full {
			t.Errorf("want full=%q; got %q (i=%d)", cas.full, full, i)
			continue
		}
	}
}

func TestCanonicalize(t *testing.T) {
	cases := [...]caseCanonical{
		{"/etc", "/private/etc"},
		{"/etc/defaults", "/private/etc/defaults"},
		{"/etc/hosts", "/private/etc/hosts"},
		{"/tmp", "/private/tmp"},
		{"/var", "/private/var"},
	}
	testCanonical(t, cases[:])
}

func TestCanonicalizeMultiple(t *testing.T) {
	link1, err := symlink("/etc")
	if err != nil {
		t.Fatal(err)
	}
	link2, err := symlink("/tmp")
	if err != nil {
		t.Fatal(nonil(err, os.Remove(link1)))
	}
	defer removeall(link1, link2)
	cases := [...]caseCanonical{
		{link1, "/private/etc"},
		{link1 + "/hosts", "/private/etc/hosts"},
		{link2, "/private/tmp"},
	}
	testCanonical(t, cases[:])
}

func TestCanonicalizeCircular(t *testing.T) {
	tmp1, err := tmpfile("circular")
	if err != nil {
		t.Fatal(err)
	}
	tmp2, err := tmpfile("circular")
	if err != nil {
		t.Fatal(nonil(err, os.Remove(tmp1)))
	}
	defer removeall(tmp1, tmp2)
	// Symlink tmp1 -> tmp2.
	if err = nonil(os.Remove(tmp1), os.Symlink(tmp2, tmp1)); err != nil {
		t.Fatal(err)
	}
	// Symlnik tmp2 -> tmp1.
	if err = nonil(os.Remove(tmp2), os.Symlink(tmp1, tmp2)); err != nil {
		t.Fatal(err)
	}
	if _, err = canonical(tmp1); err == nil {
		t.Fatalf("want canonical(%q)!=nil", tmp1)
	}
	if _, ok := err.(*os.PathError); !ok {
		t.Fatalf("want canonical(%q)=os.PathError; got %T", tmp1, err)
	}
}

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
		create(w, "src/github.com/rjeczalik/fs/fs.go"),
		write(w, "src/github.com/rjeczalik/fs/fs.go", []byte("XD")),
		write(w, "src/github.com/rjeczalik/fs/fs.go", []byte("XD")),
		remove(w, "src/github.com/rjeczalik/fs/fs.go"),
		create(w, "src/github.com/rjeczalik/fs/fs.go"),
		write(w, "src/github.com/rjeczalik/fs/fs.go", []byte("XD")),
	}

	w.ExpectAny(cases[:])
}
