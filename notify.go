package notify

import "strings"

type Event uint8

const (
	Create Event = 1 << iota
	Delete
	Write
	Move
	Recursive
)

const All Event = Create | Delete | Write | Move | Recursive

var s = map[Event]string{
	Create:    "Create",
	Delete:    "Delete",
	Write:     "Write",
	Move:      "Move",
	Recursive: "Recursive",
}

func (e Event) String() string {
	var z []string
	for _, event := range splitevents(e) {
		z = append(z, s[event])
	}
	return strings.Join(z, ",")
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
