package notify

import (
	"errors"
	"os"
	"path/filepath"
)

type dispatch struct {
	// Watcher implements the OS filesystem event notification.
	Watcher Watcher

	tree map[string]interface{}
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

// Watch TODO
func (d dispatch) Watch(p string, c chan<- EventInfo, events ...Event) (err error) {
	isdir, err := isdir(p)
	if err != nil {
		return
	}
	dir, s := d.tree, filepath.Base(p)
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
	e := joinevents(events, isdir)
	if isdir {
		// TODO
		return d.watchDir(s, dir, c, e)
	}
	// TODO
	return d.watchFile(s, dir, c, e)
}

// Stop TODO
func (d dispatch) Stop(c chan<- EventInfo) {
	panic("TODO")
}
