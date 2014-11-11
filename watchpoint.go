package notify

// None is an empty event diff, think null object.
var None EventDiff

// EventDiff describes a change to an event set - EventDiff[0] is an old state,
// while EventDiff[1] is a new state. If event set has not changed (old == new),
// functions typically return None value.
type EventDiff [2]Event

// WatchPoint TODO
//
// The nil key holds total event set - logical sum for all registered events.
// It speeds up computing EventDiff for Add method.
type WatchPoint map[chan<- EventInfo]Event

// Add TODO
//
// Add assumes neither c nor e are nil or zero values.
func (wp WatchPoint) Add(c chan<- EventInfo, e Event) (diff EventDiff) {
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
func (wp WatchPoint) Del(c chan<- EventInfo) (diff EventDiff) {
	delete(wp, c)
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
func (wp WatchPoint) Dispatch(ei EventInfo, isrec bool) {
	event := ei.Event()
	if isrec {
		event |= Recursive
	}
	if wp[nil]&event != event {
		return
	}
	for ch, e := range wp {
		if ch != nil && e&event == event {
			// Drop event is receiver is too slow.
			select {
			case ch <- ei:
			default:
			}
		}
	}
}

// Recursive TODO
func (wp WatchPoint) Recursive() bool {
	return wp[nil]&Recursive == Recursive
}
