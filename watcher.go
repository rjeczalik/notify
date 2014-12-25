package notify

// TODO(rjeczalik): Rework inline doc.

// NewWatcher gives new watcher, which is a layer on top of system-specific
// filesystem event notification functionalities.
//
// The newWatcher function must be implemented by each supported platform.
func NewWatcher(c chan<- EventInfo) Watcher {
	return newWatcher(c)
}

// Watcher is a temporary interface for wrapping inotify, ReadDirChangesW,
// FSEvents, kqueue, poller and fsnotify implementations.
//
// The watcher implementation is expected to do its own mapping between paths and
// create watchers if underlying event notification does not support it. For
// the ease of implementation it is guaranteed that paths provided via Watch and
// Unwatch methods are absolute and clean.
//
// It's used for development purposes, the finished packaged will switch between
// those using build tags.
type Watcher interface {
	// Watch requests a watcher creation for the given path and given event set.
	Watch(path string, event Event) error

	// Unwatch requests a watcher deletion for the given path and given event set.
	Unwatch(path string) error

	// Rewatch provides a functionality for modifying existing watch-points, like
	// expanding its event set.
	//
	// It is guaranteed that Runtime will not pass to Rewatch:
	//
	//   - a zero value for any of its arguments
	//   - old and new Events which are equal (which means nop)
	//
	// Rewatch modifies existing watch-point under for the given path. It passes
	// the existing event set currently registered for the given path, and the
	// new, requested event set.
	Rewatch(path string, old, new Event) error

	// Dispatch requests to fan in all events from all the created watchers into c.
	// It is guaranteed the c is non-nil. All unexpected events are ignored.
	//
	// The Dispatch method is called once on package init by the notify runtime.
	//
	// The stop channel is closed when the notify runtime is stopped and is no
	// longer receiving events sent to c.
	Dispatch(c chan<- EventInfo, stop <-chan struct{})
}

// RecursiveWatcher is an interface for a Watcher for those OS, which do support
// recursive watching over directories.
type RecursiveWatcher interface {
	// RecursiveWatch TODO
	RecursiveWatch(path string, event Event) error

	// RecursiveUnwatch removes a recursive watch-point given by the path. For
	// native recursive implementation there is no difference in functionality
	// between Unwatch and RecursiveUnwatch, however for those platforms, that
	// requires emulation for recursive watch-points, the implementation differs.
	RecursiveUnwatch(path string) error

	// RecursiveRewatcher provides a functionality for modifying and/or relocating
	// existing recursive watch-points.
	//
	// To relocate a watch-point means to unwatch oldpath and set a watch-point on
	// newpath.
	//
	// To modify a watch-point means either to expand or shrink its event set.
	//
	// Runtime can want to either relocate, modify or relocate and modify a watch-point
	// via single RecursiveRewatch call.
	//
	// If oldpath == newpath, the watch-point is expected to change its event set value
	// from oldevent to newevent.
	//
	// If oldevent == newevent, the watch-point is expected to relocate from oldpath
	// to the newpath.
	//
	// If oldpath != newpath and oldevent != newevent, the watch-point is expected
	// to relocate from oldpath to the newpath first and then change its event set
	// value from oldevent to the newevent. In other words the end result must be
	// a watch-point set on newpath with newevent value of its event set.
	//
	// It is guaranteed that Runtime will not pass to RecurisveRewatch:
	//
	//   - a zero value for any of its arguments
	//   - arguments which are simultaneously equal: oldpath == newpath and
	//     oldevent == newevent (which basically means a nop)
	RecursiveRewatch(oldpath, newpath string, oldevent, newevent Event) error
}

// TODO(rjeczalik): Remove after Runtime is replaced with Tree.

type Rewatcher interface {
	Rewatch(path string, old, new Event) error
}

type RecursiveRewatcher interface {
	RecursiveRewatch(oldpath, newpath string, oldevent, newevent Event) error
}
