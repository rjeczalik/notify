package notify

type Event uint8

const (
	Create Event = 1 << iota
	Delete
	Write
	Move
	Recursive
)

const All Event = Create | Delete | Write | Move | Recursive

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
