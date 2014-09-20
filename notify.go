package notify

import (
	"strconv"
	"strings"
)

type Event int

func (e Event) Kind() Event {
	if ek, ok := evkinds[int(e)]; ok {
		return ek
	}
	panic("notify: invalid event type - " + e.String())
}

func (e Event) String() string {
	var z []string
	for _, event := range splitevents(e) {
		if en := evnames[int(event)]; en != "" {
			z = append(z, en)
		} else {
			z = append(z, "event "+strconv.Itoa(int(event)))
		}

	}
	return strings.Join(z, ",")
}

// TODO(someone): splitevents supports only generic events.
func splitevents(e Event) (events []Event) {
	for _, event := range []Event{Create, Delete, Write, Move} {
		if e&event != 0 {
			events = append(events, event)
		}
	}
	return
}

// EventInfo TODO
type EventInfo interface {
	Event() Event     // TODO
	IsDir() bool      // TODO
	Name() string     // TODO
	Sys() interface{} // TODO
}

// Watch TODO
func Watch(name string, c chan<- EventInfo, events ...Event) {
	global.Watch(name, c, events...)
}

// Stop TODO
func Stop(c chan<- EventInfo) {
	global.Stop(c)
}

var global = dispatch{
	Tree: make(map[string]interface{}),
}
