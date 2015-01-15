// +build linux

package notify

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func iopen(w *W, path string) WCase {
	return WCase{
		Action: func() {
			f, err := os.OpenFile(filepath.Join(w.root, path), os.O_RDWR, 0644)
			if err != nil {
				w.Fatalf("OpenFile(%q)=%v", path, err)
			}
			if err := f.Close(); err != nil {
				w.Fatalf("Close(%q)=%v", path, err)
			}
			w.Debug(fmt.Sprintf("open %q", path))
		},
		Events: []EventInfo{
			&Call{P: path, E: IN_ACCESS},
			&Call{P: path, E: IN_OPEN},
			&Call{P: path, E: IN_CLOSE_NOWRITE},
		},
	}
}

func iread(w *W, path string, p []byte) WCase {
	return WCase{
		Action: func() {
			f, err := os.OpenFile(filepath.Join(w.root, path), os.O_RDWR, 0644)
			if err != nil {
				w.Fatalf("OpenFile(%q)=%v", path, err)
			}
			if _, err := f.Read(p); err != nil {
				w.Fatalf("Read(%q)=%v", path, err)
			}
			if err := f.Close(); err != nil {
				w.Fatalf("Close(%q)=%v", path, err)
			}
			w.Debug(fmt.Sprintf("read %q", path))
		},
		Events: []EventInfo{
			&Call{P: path, E: IN_ACCESS},
			&Call{P: path, E: IN_OPEN},
			&Call{P: path, E: IN_CLOSE_NOWRITE},
		},
	}
}

func iwrite(w *W, path string, p []byte) WCase {
	cas := write(w, path, p)
	path = cas.Events[0].Path()
	cas.Events = append(cas.Events,
		&Call{P: path, E: IN_ACCESS},
		&Call{P: path, E: IN_OPEN},
		&Call{P: path, E: IN_MODIFY},
		&Call{P: path, E: IN_CLOSE_WRITE},
	)
	return cas
}

var events = []Event{
	Create,
	Delete,
	Write,
	Move,
	IN_OPEN,
	IN_MODIFY,
	IN_CLOSE_WRITE,
	IN_OPEN,
	IN_ACCESS,
	IN_CLOSE_NOWRITE,
}

func TestWatcherInotify(t *testing.T) {
	w := NewWatcherTest(t, "testdata/vfs.txt", events...)
	defer w.Close()

	cases := [...]WCase{
		iopen(w, "src/github.com/rjeczalik/fs/fs.go"),
		iwrite(w, "src/github.com/rjeczalik/fs/fs.go", []byte("XD")),
		iread(w, "src/github.com/rjeczalik/fs/fs.go", []byte("XD")),
	}

	w.ExpectAny(cases[:])
}
