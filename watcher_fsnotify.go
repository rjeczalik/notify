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

// Watch
func (fs fsnotify) Watch(p string, e Event) error {
	return errors.New("TODO")
}

// Unwatch
func (fs fsnotify) Unwatch(p string) error {
	return errors.New("TODO")
}

// Mux
func (fs fsnotify) Mux(c chan<- EventInfo) {
	panic("TODO")
}
