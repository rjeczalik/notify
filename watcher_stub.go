// +build !darwin,!linux,!freebsd,!dragonfly,!netbsd,!openbsd
// +build !fsnotify

package notify

// NewWatcher stub.
func newWatcher() Watcher {
	panic("notify: no watcher implementation available on this platform")
}
