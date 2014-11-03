package notify

import "sort"

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
	diff[0] &= ^Recursive
	diff[1] &= ^Recursive
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
	diff[0] &= ^Recursive
	diff[1] &= ^Recursive
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

// Point TODO
type Point struct {
	Name   string
	Parent map[string]interface{}
}

// PointSet TODO
type PointSet []Point

func (p PointSet) Len() int           { return len(p) }
func (p PointSet) Less(i, j int) bool { return p[i].Name < p[j].Name }
func (p PointSet) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

func (p PointSet) Search(pt Point) int {
	return sort.Search(len(p), func(i int) bool { return p[i].Name >= pt.Name })
}

func (p *PointSet) Add(pt Point) {
	switch i := p.Search(pt); {
	case i == len(*p):
		*p = append(*p, pt)
	case (*p)[i].Name == pt.Name:
		*p = append(*p, Point{})
		copy((*p)[i+1:], (*p)[i:])
		(*p)[i] = pt
	}
}

func (p *PointSet) Del(pt Point) {
	if i, n := p.Search(pt), len(*p); i != n && (*p)[i].Name == pt.Name {
		copy((*p)[i:], (*p)[i+1:])
		*p = (*p)[:n-1]
	}
}
