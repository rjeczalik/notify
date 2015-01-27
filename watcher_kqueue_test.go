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
	notify.NoteAttrib: func(p string) (err error) {
		_, err = exec.Command("touch", p).CombinedOutput()
		return
	},
	notify.NoteExtend: func(p string) (err error) {
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
	notify.NoteLink: func(p string) error {
		return os.Link(p, p+"link")
	},
	/*TODO: notify.NoteRevoke: func(p string) error {
		if err := syscall.Revoke(p); err != nil {
			fmt.Printf("this is the error %v\n", err)
			return err
		}
		return nil
	},*/
}

func ExpectKqueueEvents(t *testing.T, wr notify.watcher, e notify.Event,
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
			notify.NoteAttrib): {notify.NoteAttrib},
		test.EI("github.com/rjeczalik/fs/fs.go",
			notify.NoteExtend): {notify.NoteExtend | notify.NoteWrite},
		test.EI("github.com/rjeczalik/fs/fs.go",
			notify.NoteLink): {notify.NoteLink},
	}
	ExpectKqueueEvents(t, notify.NewWatcher(), notify.NoteDelete|
		notify.NoteWrite|notify.NoteExtend|notify.NoteAttrib|notify.NoteLink|
		notify.NoteRename|notify.NoteRevoke, ei)
}
