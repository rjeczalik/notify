package notify

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

type dispatch struct {
	// Watcher implements the OS filesystem event notification.
	Watcher Watcher

	// Tree TODO
	Tree map[string]interface{}

	rw    RecursiveWatcher // underlying implementation
	isrec bool             // whether Watcher implements RecursiveWatcher
}

func (d dispatch) watchFile(s string, dir map[string]interface{},
	ch chan<- EventInfo, e Event) (err error) {
	return errors.New("TODO")
}

func (d dispatch) watchDir(s string, dir map[string]interface{},
	ch chan<- EventInfo, e Event) (err error) {
	return errors.New("TODO")
}

func isdir(p string) (bool, error) {
	fi, err := os.Stat(p)
	if err != nil {
		return false, err
	}
	return fi.IsDir(), nil
}

// TODO(rjeczalik): Move to init? Ensure the d.Watcher was set before that init?
func (d dispatch) checkinit() {
	if d.rw == nil {
		if d.Watcher == nil {
			panic("notify: no implementation found")
		}
		rw, ok := d.Watcher.(RecursiveWatcher)
		if ok {
			d.rw = rw
		} else {
			d.rw = Recursive{
				Watcher: d.Watcher,
				Tree:    d.Tree,
			}
		}
	}
}

// Watch TODO
func (d dispatch) Watch(p string, c chan<- EventInfo, events ...Event) (err error) {
	d.checkinit()
	var isrec bool
	if strings.HasSuffix(p, "...") {
		p, isrec = p[:len(p)-3], true
	}
	isdir, err := isdir(p)
	if err != nil {
		return
	}
	if isrec && !isdir {
		return &os.PathError{
			Op:   "notify.Watch",
			Path: p,
			Err:  os.ErrInvalid,
		}
	}
	dir, s := d.Tree, filepath.Base(p)
	fn := func(s string) bool {
		d, ok := dir[s]
		if !ok {
			d := make(map[string]interface{})
			dir[s], dir = d, d
			return true
		}
		if d, ok := d.(map[string]interface{}); ok {
			dir = d
			return true
		}
		return false
	}
	if !walkpath(p, fn) {
		return &os.PathError{
			Op:   "notify.Watch",
			Path: p,
			Err:  os.ErrInvalid,
		}
	}
	e := joinevents(events)
	if isdir {
		// TODO
		return d.watchDir(s, dir, c, e)
	}
	// TODO
	return d.watchFile(s, dir, c, e)
}

// Stop TODO
func (d dispatch) Stop(c chan<- EventInfo) {
	d.checkinit()
	panic("TODO(rjeczalik)")
}
