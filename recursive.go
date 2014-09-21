package notify

import "errors"

// Recursive TODO
type Recursive struct {
	Watcher

	// Runtime TODO
	Runtime *Runtime
}

// RecursiveWatch TODO
func (r Recursive) RecursiveWatch(p string, e Event) error {
	return errors.New("TODO(rjeczalik)")
}
