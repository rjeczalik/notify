package notify

import (
	"os"
	"runtime"

	old "gopkg.in/fsnotify.v1"
)

func init() {
	w, err := old.NewWatcher()
	if err != nil {
		panic(err)
	}
	fs := &fsnotify{w: w}
	// TODO(rjeczalik): Not really going to happen?
	runtime.SetFinalizer(fs, func(fs *fsnotify) { fs.w.Close() })
	global.Watcher = fs
}

var m = map[old.Op]Event{
	old.Create: Create,
	old.Remove: Delete,
	old.Write:  Write,
	old.Rename: Move,
	// TODO(rjeczalik): Find out who cares about chmod.
	// old.Chmod: Write,
}

type event struct {
	name  string
	ev    Event
	isdir bool // tribool - yes, no, can't tell?
}

func (e event) Event() Event     { return e.ev }
func (e event) IsDir() bool      { return e.isdir }
func (e event) Name() string     { return e.name }
func (e event) Sys() interface{} { return nil } // no-one cares about fsnotify.Event

func newEvent(ev old.Event) EventInfo {
	e := event{
		name: ev.Name,
		ev:   m[ev.Op],
	}
	// TODO(rjeczalik): Temporary, to be improved. If it's delete event, Dispatch
	// would know whether the subject was a file or directory.
	if e.ev&Delete == 0 {
		if fi, err := os.Stat(ev.Name); err == nil {
			e.isdir = fi.IsDir()
		}
	}
	return e
}

// Fsnotify implements notify.Watcher interface by wrapping fsnotify.v1 package.
type fsnotify struct {
	w *old.Watcher
}

// IsRecursive implements notify.Watcher interface.
func (fs fsnotify) IsRecursive() (nope bool) { return }

// Watch implements notify.Watcher interface.
func (fs fsnotify) Watch(p string, _ Event) error {
	return fs.w.Add(p)
}

// Unwatch implements notify.Watcher interface.
func (fs fsnotify) Unwatch(p string) error {
	return fs.w.Remove(p)
}

// Fanin implements notify.Watcher interface.
func (fs fsnotify) Fanin(c chan<- EventInfo) {
	go func() {
		for e := range fs.w.Events {
			if e.Op&old.Chmod != 0 {
				continue // currently we don't care about chmod
			}
			c <- newEvent(e)
		}
	}()
}
