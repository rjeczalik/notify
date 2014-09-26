package notify

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// Runtime TODO
type Runtime struct {
	// Watcher implements the OS filesystem event notification.
	Watcher RecursiveWatcher

	// Tree TODO
	Tree map[string]interface{}

	native bool // whether RecursiveWatch is native or emulated
}

// NewRuntime TODO
func NewRuntime() *Runtime {
	w, ch := NewWatcher(), make(chan EventInfo)
	r := &Runtime{
		Tree: make(map[string]interface{}),
	}
	rw, ok := w.(RecursiveWatcher)
	if ok {
		r.Watcher, r.native = rw, ok
	} else {
		r.Watcher = Recursive{
			Watcher: w,
			Runtime: r,
		}
	}
	// TODO(rjeczalik): Uncomment after #5.
	// r.Watcher.Fanin(ch)
	go r.loop()
	return r
}

func (r Runtime) watchFile(s string, dir map[string]interface{},
	ch chan<- EventInfo, e Event) (err error) {
	return errors.New("TODO")
}

func (r Runtime) watchDir(s string, dir map[string]interface{},
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
func (r Runtime) Watch(p string, c chan<- EventInfo, events ...Event) (err error) {
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
	dir, s := r.Tree, filepath.Base(p)
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
		return r.watchDir(s, dir, c, e)
	}
	// TODO
	return r.watchFile(s, dir, c, e)
}

// Stop TODO
func (r Runtime) Stop(c chan<- EventInfo) {
	panic("TODO(rjeczalik)")
}
