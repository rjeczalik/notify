package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rjeczalik/notify"
)

const usage = "usage: notify path [EVENT...]"

const tformat = "2006-01-02 15:04:05.0000"

var event = map[string]notify.Event{
	"all":    notify.All,
	"create": notify.Create,
	"delete": notify.Delete,
	"write":  notify.Write,
	"move":   notify.Move,
}

func parse(s []string) (e []notify.Event) {
	if len(s) == 0 {
		return []notify.Event{notify.All}
	}
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
	ch := make(chan notify.EventInfo, 10)
	notify.Watch(os.Args[1], ch, parse(os.Args[2:])...)
	for ei := range ch {
		fmt.Printf("[%v] Event: %v\n", time.Now().Format(tformat), ei)
	}
}
