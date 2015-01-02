// +build ignore

// TODO(rjeczalik): rewrite with test_test.go

package notify_test

import (
	"os"
	"os/exec"
	"testing"

	"github.com/rjeczalik/notify"
	"github.com/rjeczalik/notify/test"
)

var kqueueActions = test.Actions{
	notify.NOTE_ATTRIB: func(p string) (err error) {
		_, err = exec.Command("touch", p).CombinedOutput()
		return
	},
	notify.NOTE_EXTEND: func(p string) (err error) {
		//TODO: Extend and no Write
		var f *os.File
		if f, err = os.OpenFile(p, os.O_RDWR|os.O_APPEND, 0755); err != nil {
			return
		}
		defer f.Close()
		if _, err = f.WriteString(" "); err != nil {
			return
		}
		return
	},
	notify.NOTE_LINK: func(p string) error {
		return os.Link(p, p+"link")
	},
	/*TODO: notify.NOTE_REVOKE: func(p string) error {
		if err := syscall.Revoke(p); err != nil {
			fmt.Printf("this is the error %v\n", err)
			return err
		}
		return nil
	},*/
}

func ExpectKqueueEvents(t *testing.T, wr notify.Watcher, e notify.Event,
	ei map[notify.EventInfo][]notify.Event) {
	w := test.W(t, kqueueActions)
	defer w.Close()
	if err := w.WatchAll(wr, e); err != nil {
		t.Fatal(err)
	}
	defer w.UnwatchAll(wr)
	w.ExpectEvents(wr, ei)
}

func TestKqueue(t *testing.T) {
	ei := map[notify.EventInfo][]notify.Event{
		test.EI("github.com/rjeczalik/fs/fs.go",
			notify.NOTE_ATTRIB): []notify.Event{notify.NOTE_ATTRIB},
		test.EI("github.com/rjeczalik/fs/fs.go",
			notify.NOTE_EXTEND): []notify.Event{notify.NOTE_EXTEND | notify.NOTE_WRITE},
		test.EI("github.com/rjeczalik/fs/fs.go",
			notify.NOTE_LINK): []notify.Event{notify.NOTE_LINK},
	}
	ExpectKqueueEvents(t, notify.NewWatcher(), notify.NOTE_DELETE|
		notify.NOTE_WRITE|notify.NOTE_EXTEND|notify.NOTE_ATTRIB|notify.NOTE_LINK|
		notify.NOTE_RENAME|notify.NOTE_REVOKE, ei)
}
