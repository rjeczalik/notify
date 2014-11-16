package notify

// EventDiff describes a change to an event set - EventDiff[0] is an old state,
// while EventDiff[1] is a new state. If event set has not changed (old == new),
// functions typically return the None value.
type EventDiff [2]Event

// Watchpoint TODO
//
// The nil key holds total event set - logical sum for all registered events.
// It speeds up computing EventDiff for Add method.
//
// The rec key holds an event set for a watchpoints created by RecursiveWatch
// for a Watcher implementation which is not natively recursive.
type Watchpoint map[chan<- EventInfo]Event

// None is an empty event diff, think null object.
var None EventDiff

// rec is just a placeholder
var rec = func() (ch chan<- EventInfo) {
	ch = make(chan<- EventInfo)
	close(ch)
	return
}()

// Add TODO
//
// Add assumes neither c nor e are nil or zero values.
func (wp Watchpoint) Add(c chan<- EventInfo, e Event) (diff EventDiff) {
	wp[c] |= e
	diff[0] = wp[nil]
	diff[1] = diff[0] | e
	wp[nil] = diff[1]
	// Strip diff from internal events.
	diff[0] &^= Recursive
	diff[1] &^= Recursive
	if diff[0] == diff[1] {
		return None
	}
	return
}

// Del TODO
func (wp Watchpoint) Del(c chan<- EventInfo, e Event) (diff EventDiff) {
	wp[c] &^= e
	if wp[c] == 0 {
		delete(wp, c)
	}
	diff[0] = wp[nil]
	// Recalculate total event set.
	delete(wp, nil)
	for _, e := range wp {
		diff[1] |= e
	}
	wp[nil] = diff[1]
	// Strip diff from internal events.
	diff[0] &^= Recursive
	diff[1] &^= Recursive
	if diff[0] == diff[1] {
		return None
	}
	return
}

// Dispatch TODO
func (wp Watchpoint) Dispatch(ei EventInfo, recursiveonly bool) {
	event := ei.Event()
	if recursiveonly {
		event |= Recursive
	}
	if wp[nil]&event != event {
		return
	}
	for ch, e := range wp {
		if ch != nil && ch != rec && e&event == event {
			select {
			case ch <- ei:
			default:
				// Drop event is receiver is too slow.
			}
		}
	}
}

// AddRecursive TODO
func (wp Watchpoint) AddRecursive(e Event) EventDiff {
	return wp.Add(rec, e|Recursive)
}

// DelRecursive TODO
func (wp Watchpoint) DelRecursive(e Event) EventDiff {
	// If this delete would remove all events from rec event set, ensure Recursive
	// is also gone.
	if wp[rec] == e|Recursive {
		e |= Recursive
	}
	return wp.Del(rec, e)
}

// Recursive TODO
func (wp Watchpoint) Recursive() Event {
	return wp[rec]
}

// IsRecursive TODO
func (wp Watchpoint) IsRecursive() bool {
	return wp[nil]&Recursive == Recursive
}
