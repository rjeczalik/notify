// +build fsnotify

package notify

import (
	"os"
	"runtime"

	fsnotifyv1 "gopkg.in/fsnotify.v1"
)

type event struct {
	name  string
	ev    Event
	isdir bool // tribool - yes, no, can't tell?
}

func (e event) Event() Event     { return e.ev }
func (e event) IsDir() bool      { return e.isdir }
func (e event) Name() string     { return e.name }
func (e event) Sys() interface{} { return nil } // no-one cares about fsnotify.Event

func newEvent(ev fsnotifyv1.Event) EventInfo {
	e := event{
		name: ev.Name,
		ev:   Event(ev.Op),
	}
	// TODO(rjeczalik): Temporary, to be improved. If it's delete event, notify
	// runtime would know whether the subject was a file or directory.
	if e.ev&Delete == 0 {
		if fi, err := os.Stat(ev.Name); err == nil {
			e.isdir = fi.IsDir()
		}
	}
	return e
}

// Fsnotify implements notify.Watcher interface by wrapping fsnotifyv1 package.
type fsnotify struct {
	w *fsnotifyv1.Watcher
}

// NewWatcher creates new non-recursive watcher backed by fsnotifyv1 package.
func newWatcher() Watcher {
	w, err := fsnotifyv1.NewWatcher()
	if err != nil {
		panic(err)
	}
	fs := &fsnotify{w: w}
	runtime.SetFinalizer(fs, (*fsnotify).stop)
	return fs
}

// Watch implements notify.Watcher interface.
func (fs fsnotify) Watch(p string, _ Event) error {
	return fs.w.Add(p)
}

// Unwatch implements notify.Watcher interface.
func (fs fsnotify) Unwatch(p string) error {
	return fs.w.Remove(p)
}

// Dispatch implements notify.Watcher interface.
func (fs fsnotify) Dispatch(c chan<- EventInfo, stop <-chan struct{}) {
	go func() {
		for {
			select {
			case e := <-fs.w.Events:
				c <- newEvent(e)
			case <-stop:
				fs.stop()
				return
			}
		}
	}()
}

func (fs *fsnotify) stop() {
	if fs.w != nil {
		fs.w.Close()
		fs.w = nil
	}
}
