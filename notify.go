package notify

type Event uint8

const (
	Create Event = 1 << iota
	Write
	Remove
	Rename
	Recursive
)

type EventInfo interface {
	Name() string
	Event() Event
	Sys() interface{}
}

func Notify(name string, c chan<- EventInfo, events ...Event) {
	Default.Notify(name, c, events...)
}

func Stop(c chan<- EventInfo) {
	Default.Stop(c)
}

type Notifier interface {
	Notify(string, chan<- EventInfo, ...Event)
	Stop(chan<- EventInfo)
}

var Default Notifier
