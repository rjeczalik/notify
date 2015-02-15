// Copyright (c) 2014-2015 The Notify Authors. All rights reserved.
// Use of this source code is governed by the MIT license that can be
// found in the LICENSE file.

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
	for _, path := range strings.Split(os.Args[1], string(os.PathListSeparator)) {
		ch := make(chan notify.EventInfo, 10)
		if err := notify.Watch(path, ch, parse(os.Args[2:])...); err != nil {
			die(err)
		}
		go func(path string) {
			for ei := range ch {
				fmt.Printf("[%v] [%s] Event: %v\n", time.Now().Format(tformat), path, ei)
			}
		}(path)
	}
	select {}
}
