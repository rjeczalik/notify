// +build fsnotify

package notify

import (
	"os"
	"runtime"

	fsnotifyv1 "gopkg.in/fsnotify.v1"
)

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
