package notify

import "strings"

// Event TODO
type Event uint8

const (
	Create Event = 1 << iota
	Delete
	Write
	Move
)

const All Event = Create | Delete | Write | Move

var s = map[Event]string{
	Create: "Create",
	Delete: "Delete",
	Write:  "Write",
	Move:   "Move",
}

// String TODO
func (e Event) String() string {
	var z []string
	for _, event := range splitevents(e) {
		z = append(z, s[event])
	}
	return strings.Join(z, ",")
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
