package notify

import "errors"

// TODO(rjeczalik): Rework inline doc.

// NewWatcher gives new watcher, which is a layer on top of system-specific
// filesystem event notification functionalities.
//
// The newWatcher function must be implemented by each supported platform.
func NewWatcher() Watcher {
	return newWatcher()
}

// Watcher is a temporary interface for wrapping inotify, ReadDirChangesW,
// FSEvents, kqueue, poller and fsnotify implementations.
//
// The watcher implementation is expected to do its own mapping between paths and
// create watchers if underlying event notification does not support it. For
// the ease of implementation it is guaranteed that paths provided via Watch and
// Unwatch methods are absolure and clean.
//
// It's used for development purposes, the finished packaged will switch between
// those using build tags.
type Watcher interface {
	// Watch requests a watcher creation for the given path and given event set.
	//
	// NOTE(rjeczalik): For now notify runtime will call Watch method in a thread-safe
	// manner, so you may want to not bother with synchronization from the beginning,
	// it may be added later, e.g. when notify runtime  is going to be changed to some
	// producer-consumer model.
	Watch(path string, event Event) error

	// Unwatch requests a watcher deletion for the given path and given event set.
	//
	// NOTE(rjeczalik): For now notify runtime will call Unwatch method in a thread-safe
	// manner, so you may want to not bother with synchronization from the beginning,
	// it may be added later, e.g. when notify runtime is going to be changed to some
	// 1:M producer-consumer model.
	Unwatch(path string) error

	// Fanin requests to fan in all events from all the created watchers into c.
	// It is guaranteed the c is non-nil. All unexpected events are ignored.
	//
	// The Fanin method is called once on package init by the notify runtime.
	//
	// The stop channel is closed when the notify runtime is stopped and is no
	// longer receiving events sent to c.
	Fanin(c chan<- EventInfo, stop <-chan struct{})
}

// Rewatcher provides an interface for modyfing existing watch-points, like
// expanding its event set.
type Rewatcher interface {
	// Rewatch modifies exisiting watch-point under for the given path. It passes
	// the existing event set currently registered for the given path, and the
	// new, requested event set.
	Rewatch(path string, old, new Event) error
}

// RecursiveWatcher is an interface for a Watcher for those OS, which do support
// recursive watching over directories.
type RecursiveWatcher interface {
	// RecursiveWatch watches the path passed recursively for changes. It is
	// guaranteed that the given path points to a valid directory and the path
	// is stripped from "/...".
	//
	//   notify.Watch("/home/notify/...",notify.Create)
	//
	// whole "/home/notify" for either file or directory create events.
	//
	// Implementations that do not support recursive watchers will get that feature
	// emulated by notify runtime - it means that more Watch and Unwatch methods
	// are going to be called, e.g. for the following Watch:
	//
	//   notify.Watch("/home/notify", notify.Recursive, notify.Create)
	//
	// The following methods may be called on the Watcher:
	//
	//   notify.r.i.Watch("/home/notify", notify.Create)
	//   notify.r.i.Watch("/home/notify/Music", notify.Create)
	//   notify.r.i.Watch("/home/notify/Documents", notify.Create)
	//   notify.r.i.Watch("/home/notify/Downloads", notify.Create)
	//   ...
	//
	RecursiveWatch(path string, event Event) error

	// RecursiveUnwatch removes a recursive watch-point given by the path. For
	// native recursive implementation there is no difference in functionality
	// between Unwatch and RecursiveUnwatch, however for those platforms, that
	// requires emulation for recursive watch-points, the implementation differs.
	RecursiveUnwatch(path string) error
}

// recursive TODO
type recursive struct {
	Watcher Watcher
	Runtime *Runtime
}

// recursiveWatch TODO
func (r recursive) RecursiveWatch(p string, e Event) error {
	return errors.New("RecurisveWatch TODO(rjeczalik)")
}

// recursiveUnwatch TODO
func (r recursive) RecursiveUnwatch(p string) error {
	return errors.New("RecurisveUnwatch TODO(rjeczalik)")
}

// rewatch TODO
type rewatch struct {
	Watcher Watcher
}

// rewatchwatch TODO
func (r rewatch) Rewatch(p string, old, new Event) error {
	if err := r.Watcher.Unwatch(p); err != nil {
		return err
	}
	return r.Watcher.Watch(p, new)
}
