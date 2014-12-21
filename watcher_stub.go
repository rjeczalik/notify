// +build !darwin,!linux,!freebsd,!dragonfly,!netbsd,!openbsd,!windows
// +build !fsnotify !kqueue

package notify

// NewWatcher stub.
func newWatcher() Watcher {
	return nil
}
