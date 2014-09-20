package notify

import "errors"

// Recursive TODO
type Recursive struct {
	Watcher

	// Tree reads and writes to Dispatch.Tree.
	Tree map[string]interface{}
}

// RecursiveWatch TODO
func (r Recursive) RecursiveWatch(p string, e Event) error {
	return errors.New("TODO(rjeczalik)")
}
