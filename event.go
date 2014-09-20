package notify

import "strings"

// Event TODO
type Event int

// String implements fmt.Stringer interface.
func (e Event) String() string {
	var s []string
	for ev, str := range estr {
		if e&ev == ev {
			s = append(s, str)
		}
	}
	return strings.Join(s, "|")
}

// EventInfo TODO
type EventInfo interface {
	Event() Event     // TODO
	IsDir() bool      // TODO
	Name() string     // TODO
	Sys() interface{} // TODO
}
