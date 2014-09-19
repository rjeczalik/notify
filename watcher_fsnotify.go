package notify

import (
	"errors"

	old "gopkg.in/fsnotify.v1"
)

func init() {
	w, err := old.NewWatcher()
	if err != nil {
		panic(err)
	}
	global.Watcher = &fsnotify{
		w: w,
	}
}

type fsnotify struct {
	w *old.Watcher
}

// IsRecursive implements notify.Watcher interface.
func (fs fsnotify) IsRecursive() (nope bool) { return }

// Watch implements notify.Watcher interface.
func (fs fsnotify) Watch(p string, e Event) error {
	return errors.New("TODO")
}

// Unwatch implements notify.Watcher interface.
func (fs fsnotify) Unwatch(p string) error {
	return errors.New("TODO")
}

// Fanin implements notify.Watcher interface.
func (fs fsnotify) Fanin(c chan<- EventInfo) {
	panic("TODO")
}
