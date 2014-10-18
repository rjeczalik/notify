package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/rjeczalik/notify"
)

const usage = "usage: notify path [EVENT...]"

var event = map[string]notify.Event{
	"all":    notify.All,
	"create": notify.Create,
	"delete": notify.Delete,
	"write":  notify.Write,
	"move":   notify.Move,
}

func parse(s []string) (e []notify.Event) {
	for _, s := range s {
		event, ok := event[strings.ToLower(s)]
		if !ok {
			die("invalid event: " + s)
		}
		e = append(e, event)
	}
	return
}

func die(v interface{}) {
	fmt.Fprintln(os.Stderr, v)
	os.Exit(1)
}

func main() {
	if len(os.Args) == 1 {
		die(usage)
	}
	ch := make(chan notify.EventInfo)
	if len(os.Args) > 1 {
		notify.Watch(os.Args[1], ch, parse(os.Args[2:])...)
	} else {
		notify.Watch(os.Args[1], ch, notify.All)
	}
	for ei := range ch {
		fmt.Printf("event: name=%s, type=%v\n", ei.Name(), ei.Event())
	}
}
