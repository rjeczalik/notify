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

type EventInfo interface {
	Event() Event
	IsDir() bool
	Name() string
	Sys() interface{}
}

func Watch(name string, c chan<- EventInfo, events ...Event) {
	global.Watch(name, c, events...)
}

func Stop(c chan<- EventInfo) {
	global.Stop(c)
}

var global = dispatch{
	tree: make(map[string]interface{}),
}
