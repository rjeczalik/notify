// +build darwin,!kqueue

package notify

import "testing"

// TODO(rjeczalik): build it under Windows

// noevent stripts test-case from expected event list, used when action is not
// expected to trigger any events.
func noevent(cas WCase) WCase {
	return WCase{Action: cas.Action}
}

func TestWatcherRecursiveRewatch(t *testing.T) {
	w := newWatcherTest(t, "testdata/vfs.txt")
	defer w.Close()

	cases := []WCase{
		create(w, "src/github.com/rjeczalik/file"),
		create(w, "src/github.com/rjeczalik/dir/"),
		noevent(create(w, "src/github.com/rjeczalik/fs/dir/")),
		noevent(create(w, "src/github.com/dir/")),
		noevent(write(w, "src/github.com/rjeczalik/file", []byte("XD"))),
		noevent(rename(w, "src/github.com/rjeczalik/fs/LICENSE", "src/LICENSE")),
	}

	w.Watch("src/github.com/rjeczalik", Create)
	w.ExpectAny(cases)

	cases = []WCase{
		create(w, "src/github.com/rjeczalik/fs/file"),
		create(w, "src/github.com/rjeczalik/fs/cmd/gotree/file"),
		create(w, "src/github.com/rjeczalik/fs/cmd/dir/"),
		create(w, "src/github.com/rjeczalik/fs/cmd/gotree/dir/"),
		noevent(write(w, "src/github.com/rjeczalik/fs/file", []byte("XD"))),
		noevent(create(w, "src/github.com/anotherdir/")),
	}

	w.RecursiveRewatch("src/github.com/rjeczalik", "src/github.com/rjeczalik", Create, Create)
	w.ExpectAny(cases)

	cases = []WCase{
		create(w, "src/github.com/rjeczalik/1"),
		create(w, "src/github.com/rjeczalik/2/"),
		noevent(create(w, "src/github.com/rjeczalik/fs/cmd/1")),
		noevent(create(w, "src/github.com/rjeczalik/fs/1/")),
		noevent(write(w, "src/github.com/rjeczalik/fs/file", []byte("XD"))),
	}

	w.Rewatch("src/github.com/rjeczalik", Create, Create)
	w.ExpectAny(cases)
}
