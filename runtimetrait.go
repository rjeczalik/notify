package notify

import "errors"

// WatcherWithTraits TODO
type WatcherWithTraits interface {
	// Watcher provides a minimum functionality required, that must be implemented
	// for each supported platform.
	Watcher

	// Rewatcher provides an interface for modyfing existing watch-points, like
	// expanding its event set.
	//
	// If the native Watcher does not implement it, it is emulated by the notify
	// Runtime.
	Rewatcher

	// RecusiveWatcher provides an interface for watching directories recursively
	// for events.
	//
	// If the native Watcher does not implement it, it is emulated by the notify
	// Runtime.
	RecursiveWatcher

	// RecursiveRewatcher provides an interface for relocating and/or modifying
	// existing watch-points.
	//
	// If the native Watcher does not implement it, it is emulated by the notify
	// Runtime.
	RecursiveRewatcher
}

// TraitUp TODO
func TraitUp(r *Runtime, w Watcher) WatcherWithTraits {
	if wwb, ok := w.(WatcherWithTraits); ok {
		return wwb
	}
	wwb := struct {
		Watcher
		Rewatcher
		RecursiveWatcher
		RecursiveRewatcher
	}{Watcher: w}
	if re, ok := w.(Rewatcher); ok {
		wwb.Rewatcher = re
	} else {
		wwb.Rewatcher = rewatch{w}
	}
	if rw, ok := w.(RecursiveWatcher); ok {
		wwb.RecursiveWatcher = rw
	} else {
		wwb.RecursiveWatcher = recursive{W: w, R: r}
	}
	if rr, ok := w.(RecursiveRewatcher); ok {
		wwb.RecursiveRewatcher = rr
	} else {
		wwb.RecursiveWatcher = recrew{wwb.RecursiveWatcher}
	}
	return wwb
}

// Recursive TODO
type recursive struct {
	W Watcher
	R *Runtime
}

// RecursiveWatch implements notify.RecursiveWatcher interface.
func (r recursive) RecursiveWatch(p string, e Event) error {
	return errors.New("RecurisveWatch TODO(rjeczalik)")
}

// RecursiveUnwatch implements notify.RecursiveWatcher interface.
func (r recursive) RecursiveUnwatch(p string) error {
	return errors.New("RecurisveUnwatch TODO(rjeczalik)")
}

// Rewatch TODO
type rewatch struct {
	Watcher
}

// Rewatch implements notify.Rewatcher interface.
func (r rewatch) Rewatch(p string, old, new Event) error {
	if err := r.Watcher.Unwatch(p); err != nil {
		return err
	}
	return r.Watcher.Watch(p, new)
}

// Recrew TODO
type recrew struct {
	RecursiveWatcher
}

// RecursiveRewatch implements notify.RecursiveRewatcher interface.
func (r recrew) RecursiveRewatch(oldp, newp string, olde, newe Event) error {
	if err := r.RecursiveWatcher.RecursiveUnwatch(oldp); err != nil {
		return err
	}
	return r.RecursiveWatcher.RecursiveWatch(newp, newe)
}
