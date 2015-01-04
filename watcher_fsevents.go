// +build darwin,!kqueue

package notify

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
)

const (
	failure = uint32(FSEventsMustScanSubDirs | FSEventsUserDropped | FSEventsKernelDropped)
	filter  = uint32(FSEventsCreated | FSEventsRemoved | FSEventsRenamed |
		FSEventsModified | FSEventsInodeMetaMod)
)

var (
	errAlreadyWatched  = errors.New("path is already watched")
	errNotWatched      = errors.New("path is not being watched")
	errInvalidEventSet = errors.New("invalid event set provided")
	errDepth           = errors.New("exceeded allowed iteration count (circular symlink?)")
)

// FSEvent represents single file event.
type FSEvent struct {
	Path  string
	ID    uint64
	Flags uint32
}

// canonical resolves any symlink in the given path and returns it in a clean form.
// It expects the path to be absolute. It fails to resolve circular symlinks by
// maintaining a simple iteration limit.
//
// TODO(rjeczalik): replace with realpath?
func canonical(p string) (string, error) {
	for i, depth := 1, 1; i < len(p); i, depth = i+1, depth+1 {
		if depth > 128 {
			return "", &os.PathError{Op: "canonical", Path: p, Err: errDepth}
		}
		if j := strings.IndexRune(p[i:], '/'); j == -1 {
			i = len(p)
		} else {
			i = i + j
		}
		fi, err := os.Lstat(p[:i])
		if err != nil {
			return "", err
		}
		if fi.Mode()&os.ModeSymlink == os.ModeSymlink {
			s, err := os.Readlink(p[:i])
			if err != nil {
				return "", err
			}
			p = "/" + s + p[i:]
			i = 1 // no guarantee s is canonical, start all over
		}
	}
	return filepath.Clean(p), nil
}

// splitflags separates event flags from single set into slice of flags.
func splitflags(set uint32) (e []uint32) {
	for i := uint32(1); set != 0; i, set = i<<1, set>>1 {
		if (set & 1) != 0 {
			e = append(e, i)
		}
	}
	return
}

// flagdiff stores last event flags per path in order to filter out old flags
// for new events, which appratenly FSEvents likes to retain. It's a disgusting
// hack, it should be researched how to get rid of it.
type flagdiff map[string]uint32

// diff filters out redundant (old) flags from the given set. It does so by using
// half-ass heuristics invented during manual testing. They're most likely broken
// and/or not covering all the use-cases. They should be rewritten or at least fully
// tested to ensure proper accuracy.
func (fd flagdiff) diff(name string, set uint32) uint32 {
	old, ok := fd[name]
	fd[name] = set & filter
	if !ok || set&old != old {
		return set
	}
	switch old = set &^ old; {
	case old == set:
		delete(fd, name)
	// For the following series of events:
	//
	//   ~ $ touch /tmp/XD
	//   ~ $ echo XD > /tmp/XD
	//   ~ $ echo XD > /tmp/XD
	//   ~ $ rm /tmp/XD
	//   ~ $ touch /tmp/XD
	//
	// FSEvents yields:
	//
	//   1) /tmp/XD Create
	//   2) /tmp/XD Create|Write
	//   3) /tmp/XD Create|Write
	//   4) /tmp/XD Create|Delete|Write
	//   5) /tmp/XD Create|Delete|Write
	//
	// This case is to ensure Create from 5) is not missed.
	case old&FSEventsRemoved != 0:
		fd[name] &^= FSEventsCreated
	// For the before-mentioned series of events this case is to ensure Write
	// from 3) is not missed.
	case old&filter == 0:
		return set & (FSEventsModified | FSEventsInodeMetaMod)
	}
	return old
}

// watch represents a filesystem watchpoint. It is a higher level abstraction
// over FSEvents' stream, which implements filtering of file events based
// on path and event set. It emulates non-recursive watch-point by filtering out
// events which paths are more than 1 level deeper than the watched path.
type watch struct {
	fd     flagdiff // TODO(rjeczalik): find nicer hack
	c      chan<- EventInfo
	stream *Stream
	path   string
	events uint32
	isrec  int32
}

// Dispatch is a stream function which forwards given file events for the watched
// path to underlying FileInfo channel.
func (w *watch) Dispatch(ev []FSEvent) {
	events := atomic.LoadUint32(&w.events)
	isrec := (atomic.LoadInt32(&w.isrec) == 1)
	for i := range ev {
		if ev[i].Flags&failure != 0 {
			// TODO(rjeczalik): missing error handling
			panic("unhandled error: " + Event(ev[i].Flags).String())
		}
		if !strings.HasPrefix(ev[i].Path, w.path) {
			continue
		}
		n, base := len(w.path), ""
		if len(ev[i].Path) > n {
			if ev[i].Path[n] != '/' {
				continue
			}
			base = ev[i].Path[n+1:]
			if !isrec && strings.IndexByte(base, '/') != -1 {
				continue
			}
		}
		// TODO(rjeczalik): get diff only from filtered events?
		e := w.fd.diff(string(base), ev[i].Flags) & events
		if e == 0 {
			continue
		}
		for _, e := range splitflags(e) {
			w.c <- &event{
				fse:   ev[i],
				event: Event(e),
			}
		}
	}
}

// fsevents implements Watcher and RecursiveWatcher interfaces backed by FSEvents
// framework.
type fsevents struct {
	watches map[string]*watch
	c       chan<- EventInfo
}

func newWatcher(c chan<- EventInfo) Watcher {
	return &fsevents{
		watches: make(map[string]*watch),
		c:       c,
	}
}

func (fse *fsevents) watch(path string, event Event, isrec int32) (err error) {
	if path, err = canonical(path); err != nil {
		return
	}
	if _, ok := fse.watches[path]; ok {
		return errAlreadyWatched
	}
	w := &watch{
		c:      fse.c,
		path:   path,
		events: uint32(event),
		isrec:  isrec,
		fd:     make(flagdiff),
	}
	w.stream = NewStream(path, w.Dispatch)
	if err = w.stream.Start(); err != nil {
		return
	}
	fse.watches[path] = w
	return nil
}

func (fse *fsevents) unwatch(path string) (err error) {
	if path, err = canonical(path); err != nil {
		return
	}
	w, ok := fse.watches[path]
	if !ok {
		return errNotWatched
	}
	w.stream.Stop()
	delete(fse.watches, path)
	return nil
}

// Watch implements Watcher interface. It fails with non-nil error when setting
// the watch-point by FSEvents fails or with errAlreadyWatched error when
// the given path is already watched.
func (fse *fsevents) Watch(path string, event Event) error {
	return fse.watch(path, event, 0)
}

// Unwatch implements Watcher interface. It fails with errNotWatched when
// the given path is not being watched.
func (fse *fsevents) Unwatch(path string) error {
	return fse.unwatch(path)
}

// Rewatch implements Watcher interface. It fails with errNotWatched when
// the given path is not being watched or with errInvalidEventSet when oldevent
// does not match event set the watch-point currently holds.
func (fse *fsevents) Rewatch(path string, oldevent, newevent Event) error {
	w, ok := fse.watches[path]
	if !ok {
		return errNotWatched
	}
	if !atomic.CompareAndSwapUint32(&w.events, uint32(oldevent), uint32(newevent)) {
		return errInvalidEventSet
	}
	return nil
}

// RecursiveWatch implements RecursiveWatcher interface. It fails with non-nil
// error when setting the watch-point by FSEvents fails or with errAlreadyWatched
// error when the given path is already watched.
func (fse *fsevents) RecursiveWatch(path string, event Event) error {
	return fse.watch(path, event, 1)
}

// RecursiveUnwatch implements RecursiveWatcher interface. It fails with
// errNotWatched when the given path is not being watched.
//
// TODO(rjeczalik): fail if w.isrec == 0?
func (fse *fsevents) RecursiveUnwatch(path string) error {
	return fse.unwatch(path)
}

// RecrusiveRewatch implements RecursiveWatcher interface. It fails:
//
//   * with errNotWatched when the given path is not being watched
//   * with errInvalidEventSet when oldevent does not match the current event set
//   * with errAlreadyWatched when watch-point given by the oldpath was meant to
//     be relocated to newpath, but the newpath is already watched
//   * a non-nil error when setting the watch-point with FSEvents fails
//
// TODO(rjeczalik): Improve handling of watch-point relocation? See two TODOs
// that follows.
func (fse *fsevents) RecursiveRewatch(oldpath, newpath string, oldevent, newevent Event) error {
	switch [2]bool{oldpath == newpath, oldevent == newevent} {
	case [2]bool{true, true}:
		w, ok := fse.watches[oldpath]
		if !ok {
			return errNotWatched
		}
		atomic.CompareAndSwapInt32(&w.isrec, 0, 1)
		return nil
	case [2]bool{true, false}:
		w, ok := fse.watches[oldpath]
		if !ok {
			return errNotWatched
		}
		if !atomic.CompareAndSwapUint32(&w.events, uint32(oldevent), uint32(newevent)) {
			return errors.New("invalid event state diff")
		}
		atomic.CompareAndSwapInt32(&w.isrec, 0, 1)
		return nil
	default:
		// TODO(rjeczalik): rewatch newpath only if exists?
		if _, ok := fse.watches[newpath]; ok {
			return errAlreadyWatched
		}
		if err := fse.Unwatch(oldpath); err != nil {
			return err
		}
		// TODO(rjeczalik): revert unwatch if watch fails?
		return fse.watch(newpath, newevent, 1)
	}
}

// Close unwatches all watch-points.
func (fse *fsevents) Close() error {
	for _, w := range fse.watches {
		w.stream.Stop()
	}
	fse.watches = make(map[string]*watch)
	return nil
}
