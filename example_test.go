// Copyright (c) 2014-2015 The Notify Authors. All rights reserved.
// Use of this source code is governed by the MIT license that can be
// found in the LICENSE file.

package notify_test

import (
	"log"

	"github.com/rjeczalik/notify"
)

func ExampleWatch() {
	// Create a channel which will be used to receive file system notifications.
	// The channel must be buffered since the notify package does not block when
	// sending events. Thus, if an event is created and there is no one who can
	// receive it, the event will be dropped.
	c := make(chan notify.EventInfo, 1)
	// We are going to watch our working directory recursively trying to catch
	// either Create or Write event.
	if err := notify.Watch("./...", c, notify.Create, notify.Write); err != nil {
		log.Fatal(err)
	}

	// Block until an event is received.
	ei := <-c
	log.Println("Got event:", ei)
}
