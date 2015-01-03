// +build linux
// +build !fsnotify

package notify

import (
	"fmt"
	"os"
	"testing"
)

func iopen(w *W, path string) WCase {
	return WCase{
		Action: func() {
			f, err := os.OpenFile(path, os.O_RDWR, 0644)
			if err != nil {
				w.Fatalf("OpenFile(%q)=%v", path, err)
			}
			if err := f.Close(); err != nil {
				w.Fatalf("Close(%q)=%v", path, err)
			}
			w.Debug(fmt.Sprintf("open %q", path))
		},
		Events: []EventInfo{
			eventinfo{path: path, event: IN_ACCESS},
			eventinfo{path: path, event: IN_OPEN},
			eventinfo{path: path, event: IN_CLOSE_NOWRITE},
		},
	}
}

func iread(w *W, path string, p []byte) WCase {
	return WCase{
		Action: func() {
			f, err := os.OpenFile(path, os.O_RDWR, 0644)
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
			eventinfo{path: path, event: IN_ACCESS},
			eventinfo{path: path, event: IN_OPEN},
			eventinfo{path: path, event: IN_CLOSE_NOWRITE},
		},
	}
}

func iwrite(w *W, path string, p []byte) WCase {
	cas := write(w, path, p)
	cas.Events = append(cas.Events,
		eventinfo{path: path, event: IN_ACCESS},
		eventinfo{path: path, event: IN_OPEN},
		eventinfo{path: path, event: IN_MODIFY},
		eventinfo{path: path, event: IN_CLOSE_WRITE},
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
	w := NewWatcherTest(t, "testdata/gopath.txt", events...)
	defer w.Stop()

	cases := [...]WCase{
		iopen(w, "github.com/rjeczalik/fs/fs.go"),
		iwrite(w, "github.com/rjeczalik/fs/fs.go", []byte("XD")),
		iread(w, "github.com/rjeczalik/fs/fs.go", []byte("XD")),
	}

	w.ExpectAny(cases[:])
}
