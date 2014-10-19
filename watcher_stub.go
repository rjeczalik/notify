// +build !darwin,!linux,!freebsd,!dragonfly,!netbsd,!openbsd
// +build !fsnotify

package notify

// NewWatcher stub.
func newWatcher() Watcher {
	return nil
}
