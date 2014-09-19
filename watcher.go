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
	// IsRecursive is used to tell Dispatch that it supports, or not, recursive
	// watching. When it returns true, it is able to watch via the following call:
	//
	//   notify.Watch("/home/notify", notify.Recursive, notify.Create)
	//
	// whole "/home/notify" for either file or directory create events.
	//
	// Implementations that do not support recursive watchers will get that feature
	// emulated by Dispatch - it means that more Watch and Unwatch methods
	// are going to be called. Moreover it is guranteed that no notify.Recursive
	// event is going to be passed to the Watch method, e.g. for the following
	// call:
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
	// All of them stripped from notify.Recursive.
	//
	// The IsRecurisve method is called once on package init by the Dispatch.
	IsRecursive() bool

	// Watch requests a watcher creation for the given path and given event set.
	//
	// NOTE(rjeczalik): For now Dispatch will call Watch method in a thread-safe
	// manner, so you may want to not bother with synchronization from the beginning,
	// it may be added later, e.g. when Dispatch is going to be changed to some
	// producer-consumer model.
	Watch(string, Event) error

	// Unwatch requests a watcher deletion for the given path and given event set.
	//
	// NOTE(rjeczalik): For now Dispatch will call Unwatch method in a thread-safe
	// manner, so you may want to not bother with synchronization from the beginning,
	// it may be added later, e.g. when Dispatch is going to be changed to some
	// 1:M producer-consumer model.
	Unwatch(string) error

	// Fanin requests to fan in all events from all the created watchers into ch.
	// It is guaranteed the ch is non-nil.
	//
	// The Fanin method is called once on package init by the Dispatch.
	Fanin(ch chan<- EventInfo)
}
