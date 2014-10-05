package notify

import "errors"

// Recursive TODO
type Recursive struct {
	Watcher Watcher
	Runtime *Runtime
}

// RecursiveWatch TODO
func (r Recursive) RecursiveWatch(p string, e Event) error {
	return errors.New("RecursiveWatch TODO(rjeczalik)")
}

// RecursiveUnwatch TODO
func (r Recursive) RecursiveUnwatch(p string) error {
	return errors.New("RecursiveUnwatch TODO(rjeczalik)")
}

// Re TODO
type Re struct {
	Watcher Watcher
}

// Rewatch TODO
func (r Re) Rewatch(p string, old, new Event) error {
	if err := r.Watcher.Unwatch(p); err != nil {
		return err
	}
	return r.Watcher.Watch(p, new)
}
