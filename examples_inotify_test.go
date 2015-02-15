// Copyright (c) 2014-2015 The Notify Authors. All rights reserved.
// Use of this source code is governed by the MIT license that can be
// found in the LICENSE file.

// +build linux

package notify_test

import (
	"log"
	"syscall"

	"github.com/rjeczalik/notify"
)

func ExampleWatch_system_specific_events() {
	// Make the channel buffered to ensure no event is dropped. Notify will drop
	// an event if the receiver is not able to keep up the sending pace.
	c := make(chan notify.EventInfo, 1)

	// Set up a watchpoint listening on inotify system-specific events within a
	// current working directory. Relay each IN_CLOSE_WRITE and IN_MOVED_TO
	// events separately to c.
	if err := notify.Watch(".", c, notify.InCloseWrite, notify.InMovedTo); err != nil {
		log.Fatal(err)
	}
	defer notify.Stop(c)

	// Block until an event is received.
	switch ei := <-c; ei.Event() {
	case notify.InCloseWrite:
		log.Println("Editing of", ei.Path(), "file is done.")
	case notify.InMovedTo:
		log.Println("File", ei.Path(), "was swapped/moved into the watched directory.")
	}
}

func ExampleWatch_move_file_on_linux() {
	// Make the channel buffered to ensure no event is dropped. Notify will drop
	// an event if the receiver is not able to keep up the sending pace.
	c := make(chan notify.EventInfo, 2)

	// Set up a watchpoint listening on inotify system-specific events within a
	// current working directory. Relay each IN_MOVED_FROM and IN_MOVED_TO
	// events separately to c.
	if err := notify.Watch(".", c, notify.InMovedFrom, notify.InMovedTo); err != nil {
		log.Fatal(err)
	}
	defer notify.Stop(c)

	// We create a simple map which connects two events that have equal cookie values.
	moves := make(map[uint32]struct {
		From string
		To   string
	})

	// Wait for moves.
	for ei := range c {
		cookie := ei.Sys().(*syscall.InotifyEvent).Cookie

		info := moves[cookie]
		switch ei.Event() {
		case notify.InMovedFrom:
			info.From = ei.Path()
		case notify.InMovedTo:
			info.To = ei.Path()
		}
		moves[cookie] = info

		if cookie != 0 && info.From != "" && info.To != "" {
			log.Println("File:", info.From, "was renamed to", info.To)
			delete(moves, cookie)
		}
		// From time to time we should clear our map.
	}
}
