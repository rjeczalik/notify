package notify

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
	Watch(string, Event) error

	// Unwatch requests a watcher deletion for the given path and given event set.
	//
	// NOTE(rjeczalik): For now notify runtime will call Unwatch method in a thread-safe
	// manner, so you may want to not bother with synchronization from the beginning,
	// it may be added later, e.g. when notify runtime is going to be changed to some
	// 1:M producer-consumer model.
	Unwatch(string) error

	// Fanin requests to fan in all events from all the created watchers into ch.
	// It is guaranteed the ch is non-nil. All unexpected events are ignored.
	//
	// The Fanin method is called once on package init by the notify runtime.
	Fanin(ch chan<- EventInfo)
}

// RecursiveWatcher is an interface for a Watcher for those OS, which do support
// recursive watching over directories.
type RecursiveWatcher interface {
	Watcher

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
	//   notify.global.Watcher.Watch("/home/notify", notify.Create)
	//   notify.global.Watcher.Watch("/home/notify/Music", notify.Create)
	//   notify.global.Watcher.Watch("/home/notify/Documents", notify.Create)
	//   notify.global.Watcher.Watch("/home/notify/Downloads", notify.Create)
	//   ...
	//
	// TODO(rjeczalik): Rework.
	RecursiveWatch(string, Event) error
}
