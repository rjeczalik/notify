// Copyright (c) 2014-2015 The Notify Authors. All rights reserved.
// Use of this source code is governed by the MIT license that can be
// found in the LICENSE file.

package notify

// EventDiff describes a change to an event set - EventDiff[0] is an old state,
// while EventDiff[1] is a new state. If event set has not changed (old == new),
// functions typically return the None value.
type eventDiff [2]Event

// Event TODO(rjeczalik)
func (diff eventDiff) Event() Event {
	return diff[1] &^ diff[0]
}

// Watchpoint TODO(rjeczalik)
//
// The nil key holds total event set - logical sum for all registered events.
// It speeds up computing EventDiff for Add method.
//
// The rec key holds an event set for a watchpoints created by RecursiveWatch
// for a Watcher implementation which is not natively recursive.
type watchpoint map[chan<- EventInfo]Event

// None is an empty event diff, think null object.
var none eventDiff

// rec is just a placeholder
var rec = func() (ch chan<- EventInfo) {
	ch = make(chan<- EventInfo)
	close(ch)
	return
}()

// Diff TODO(rjeczalik)
func (wp watchpoint) Diff(e Event) eventDiff {
	e &^= recursive
	if wp[nil]&e == e {
		return none
	}
	total := wp[nil] &^ recursive
	return eventDiff{total, total | e}
}

// Add TODO(rjeczalik)
//
// Add assumes neither c nor e are nil or zero values.
func (wp watchpoint) Add(c chan<- EventInfo, e Event) (diff eventDiff) {
	wp[c] |= e
	diff[0] = wp[nil]
	diff[1] = diff[0] | e
	wp[nil] = diff[1]
	// Strip diff from recursive events.
	diff[0] &^= recursive
	diff[1] &^= recursive
	if diff[0] == diff[1] {
		return none
	}
	return
}

// Del TODO(rjeczalik)
func (wp watchpoint) Del(c chan<- EventInfo, e Event) (diff eventDiff) {
	wp[c] &^= e
	if wp[c] == 0 {
		delete(wp, c)
	}
	diff[0] = wp[nil]
	delete(wp, nil)
	if len(wp) != 0 {
		// Recalculate total event set.
		for _, e := range wp {
			diff[1] |= e
		}
		wp[nil] = diff[1]
	}
	// Strip diff from recursive events.
	diff[0] &^= recursive
	diff[1] &^= recursive
	if diff[0] == diff[1] {
		return none
	}
	return
}

// Dispatch TODO(rjeczalik)
func (wp watchpoint) Dispatch(ei EventInfo, ctrl Event) {
	event := ei.Event() | ctrl
	if !matches(wp[nil], event) {
		return
	}
	for ch, e := range wp {
		// TODO(rjeczalik): rm rec
		if ch != nil && ch != rec && matches(e, event) {
			select {
			case ch <- ei:
			default:
				// Drop event if receiver is too slow
			}
		}
	}
}

// AddRecursive TODO(rjeczalik): rm
func (wp watchpoint) AddRecursive(e Event) eventDiff {
	return wp.Add(rec, e|recursive)
}

// DelRecursive TODO(rjeczalik): rm
func (wp watchpoint) DelRecursive(e Event) eventDiff {
	// If this delete would remove all events from rec event set, ensure Recursive
	// is also gone.
	if wp[rec] == e|recursive {
		e |= recursive
	}
	return wp.Del(rec, e)
}

// Recursive TODO(rjeczalik): rm
func (wp watchpoint) recursive() Event {
	return wp[rec]
}

// Total TODO(rjeczalik)
func (wp watchpoint) Total() Event {
	return wp[nil] &^ recursive
}

// IsRecursive TODO(rjeczalik)
func (wp watchpoint) IsRecursive() bool {
	return wp[nil]&recursive != 0
}
