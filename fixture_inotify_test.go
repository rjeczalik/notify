// +build linux,!fsnotify

package notify

import (
	"io"
	"os"
)

// TODO(rjeczalik): This file can be merged with watcher_inotify_test as long as inotify
// has only one test file. This is not Java XD

// TODO(pknap)
var inotify = map[Event][]Event{
	IN_ACCESS: {IN_OPEN, IN_MODIFY, IN_CLOSE_WRITE, IN_OPEN, IN_ACCESS, IN_CLOSE_NOWRITE},
}

var fixtureos = Fixture(FixtureFunc{
	IN_ACCESS: func(p string) error {
		f, err := os.OpenFile(p, os.O_RDWR, 0755)
		if err != nil {
			return err
		}
		if _, err := f.WriteString(p); err != nil {
			f.Close()
			return err
		}
		f.Close()
		f, err = os.Open(p)
		if err != nil {
			return err
		}
		if _, err = f.Read([]byte{0x00}); err != nil && err != io.EOF {
			f.Close()
			return err
		}
		return f.Close()
	},
})
