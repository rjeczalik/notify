// +build !linux,!fsnotify,!freebsd,!dragonfly,!netbsd,!openbsd

package notify

// NewWatcher stub.
func newWatcher() Watcher {
	panic("notify: no watcher implementation available on this platform")
}
