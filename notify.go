package notify

type Event uint8

const (
	Create Event = 1 << iota
	Write
	Remove
	Rename
	Recursive
)

const All Event = Create | Write | Remove | Rename | Recursive

type EventInfo interface {
	Name() string
	Event() Event
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
