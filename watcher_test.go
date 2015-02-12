package notify

import (
	"fmt"
	"testing"
)

// NOTE Set DEBUG env var for extra debugging info.

func TestWatcher(t *testing.T) {
	w := NewWatcherTest(t, "testdata/vfs.txt")
	defer w.Close()

	cases := [...]WCase{
		create(w, "src/github.com/ppknap/link/include/coost/.link.hpp.swp"),
		create(w, "src/github.com/rjeczalik/fs/fs_test.go"),
		create(w, "src/github.com/rjeczalik/fs/binfs/"),
		create(w, "src/github.com/rjeczalik/fs/binfs.go"),
		create(w, "src/github.com/rjeczalik/fs/binfs_test.go"),
		remove(w, "src/github.com/rjeczalik/fs/binfs/"),
		create(w, "src/github.com/rjeczalik/fs/binfs/"),
		create(w, "src/github.com/rjeczalik/fs/virfs"),
		remove(w, "src/github.com/rjeczalik/fs/virfs"),
		create(w, "file"),
		create(w, "dir/"),
	}

	w.ExpectAny(cases[:])
}

func TestIsDirCreateEvent(t *testing.T) {
	w := NewWatcherTest(t, "testdata/vfs.txt")
	defer w.Close()

	cases := [...]WCase{
		// i=0
		create(w, "src/github.com/jszwec/"),
		// i=1
		create(w, "src/github.com/jszwec/gojunitxml/"),
		// i=2
		create(w, "src/github.com/jszwec/gojunitxml/README.md"),
		// i=3
		create(w, "src/github.com/jszwec/gojunitxml/LICENSE"),
		// i=4
		create(w, "src/github.com/jszwec/gojunitxml/cmd/"),
	}

	dirs := [...]bool{
		true,  // i=0
		true,  // i=1
		false, // i=2
		false, // i=3
		true,  // i=4
	}

	fn := func(i int, _ WCase, ei EventInfo) error {
		switch ok, err := isdir(ei); {
		case err != nil:
			return err
		case ok != dirs[i]:
			return fmt.Errorf("want ok=%v; got %v", dirs[i], ok)
		default:
			return nil
		}
	}

	w.ExpectAnyFunc(cases[:], fn)
}
