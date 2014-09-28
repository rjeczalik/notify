// +build !linux,!fsnotify

package notify

// TODO(ppknap)
var madev = map[Event][]Event{
	Create: {Create},
	Delete: {Delete},
	Write:  {Write},
	Move:   {Move},
}
