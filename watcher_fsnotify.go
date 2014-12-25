// +build fsnotify

package notify

import (
	"runtime"

	fsnotifyv1 "gopkg.in/fsnotify.v1"
)

// Fsnotify implements notify.Watcher interface by wrapping fsnotifyv1 package.
type fsnotify struct {
	w    *fsnotifyv1.Watcher
	stop chan struct{}
}

// NewWatcher creates new non-recursive watcher backed by fsnotifyv1 package.
func newWatcher() Watcher {
	w, err := fsnotifyv1.NewWatcher()
	if err != nil {
		panic(err)
	}
	fs := &fsnotify{w: w, stop: make(chan struct{})}
	go fs.loop()
	runtime.SetFinalizer(fs, func(fs fsnotify) { fs.Close() })
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
//
// TODO(rjeczalik): remove
func (fs fsnotify) Dispatch(c chan<- EventInfo, stop <-chan struct{}) {
}

func (fs fsnotify) loop() {
	for {
		select {
		case e := <-fs.w.Events:
			c <- newEvent(e)
		case <-w.stop:
			return
		}
	}
}

func (fs *fsnotify) Close() error {
	if fs.w != nil {
		fs.w.Close()
		fs.w = nil
		close(fs.stop)
	}
	return nil
}
