// +build !darwin,!linux,!freebsd,!dragonfly,!netbsd,!openbsd,!windows
// +build !kqueue

package notify

// NewWatcher stub.
func newWatcher() Watcher {
	return nil
}
