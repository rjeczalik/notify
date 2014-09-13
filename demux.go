package notify

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// false - early exit
func pathwalk(p string, fn func(string) bool) bool {
	if p == "" || p == "." {
		return false
	}
	i, n := strings.Index(p, sep)+1, len(p)
	if i == 0 || i == n {
		return false
	}
	for i < n {
		j := strings.Index(p[i:], sep)
		if j == -1 {
			j = n - i
		}
		if !fn(p[i : i+j]) {
			return i+i+j+2 > n
		}
		i += j + 1
	}
	return true
}

type demux struct {
	// Watcher
	Watcher watcher

	tree map[string]interface{}
}

func (d demux) watchFile(s string, dir map[string]interface{},
	ch chan<- EventInfo, e Event) (err error) {
	return errors.New("TODO")
}

func (d demux) watchDir(s string, dir map[string]interface{},
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

// Watch
func (d demux) Watch(p string, c chan<- EventInfo, events ...Event) (err error) {
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
	if !pathwalk(p, fn) {
		return &os.PathError{
			Op:   "notify.Watch",
			Path: p,
			Err:  os.ErrInvalid,
		}
	}
	e := joinevents(events, isdir)
	if isdir {
		return d.watchDir(s, dir, c, e)
	}
	return d.watchFile(s, dir, c, e)
}

// Stop
func (d demux) Stop(c chan<- EventInfo) {
	panic("TODO")
}
