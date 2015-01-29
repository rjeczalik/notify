notify [![Build Status](https://img.shields.io/travis/rjeczalik/notify/wip.svg)](https://travis-ci.org/rjeczalik/notify "inotify") [![Build Status](https://img.shields.io/travis/rjeczalik/notify/fsevents.svg)](https://travis-ci.org/rjeczalik/notify "FSEvents") [![Build Status](https://img.shields.io/travis/rjeczalik/notify/kqueue.svg)](https://travis-ci.org/rjeczalik/notify "kqueue") [![Build status](https://img.shields.io/appveyor/ci/rjeczalik/notify-246.svg)](https://ci.appveyor.com/project/rjeczalik/notify-246 "ReadDirectoryChangesW") [![Coverage Status](https://img.shields.io/coveralls/rjeczalik/notify/wip.svg)](https://coveralls.io/r/rjeczalik/notify?branch=wip)
======

Filesystem event notification library on steroids. (under active development)

*Installation*

```
~ $ go get -u github.com/rjeczalik/notify
```

*Documentation*

[godoc.org/github.com/rjeczalik/notify](https://godoc.org/github.com/rjeczalik/notify)

*Usage*

Watching recursively all events on a single chan.

```go
package main

import (
	"log"

	"github.com/rjeczalik/notify"
)

func main() {
	c := make(chan notify.EventInfo, 16)

	if err := notify.Watch("./...", c, notify.All); err != nil {
		log.Fatal(err)
	}
	defer notify.Stop(c)

	for ei := range c {
		log.Println(ei.Event(), ei.Path())
	}
}
```

Event error handling.

```go
package main

import (
	"log"

	"github.com/rjeczalik/notify"
)

func main() {
	c := make(chan notify.EventInfo, 1)

	if err := notify.Watch(".", c, notify.Create); err != nil {
		log.Fatal(err)
	}
	defer notify.Stop(c)

	for ei := range c {
		switch ei.Event() {
		case notify.Error:
			log.Fatal(ei.Sys().(error))
		default:
			log.Println(ei)
		}
	}
}
```

Watching multiple events on a single chan.

```go
package main

import (
	"log"

	"github.com/rjeczalik/notify"
)

func main() {
	c := make(chan notify.EventInfo, 1)

	if err := notify.Watch(".", c, notify.Create, notify.Write); err != nil {
		log.Fatal(err)
	}
	defer notify.Stop(c)

	for ei := range c {
		switch ei.Event() {
		case notify.Create:
			log.Println("create", ei)
		case notify.Write:
			log.Println("write", ei)
		}
	}
}
```

Watching multiple events on multiple chans.

```go
package main

import (
	"log"

	"github.com/rjeczalik/notify"
)

func main() {
	create := make(chan notify.EventInfo, 1)
	write := make(chan notify.EventInfo, 1)

	if err := notify.Watch(".", create, notify.Create); err != nil {
		log.Fatal(err)
	}
	defer notify.Stop(create)

	if err := notify.Watch(".", write, notify.Write); err != nil {
		log.Fatal(err)
	}
	defer notify.Stop(write)

	for {
		select {
		case ei := <-create:
			log.Println("create", ei)
		case ei := <-write:
			log.Println("write", ei)
		}
	}
}
```
