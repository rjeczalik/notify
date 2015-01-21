notify
======

Filesystem event notification library on steroids.

*Installation*

```
~ $ go get -u github.com/rjeczalik/notify
```

*Documentation*

[godoc.org/github.com/rjeczalik/notify](https://godoc.org/github.com/rjeczalik/notify)

*Usage*

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

	go func() {
		for ei := range c {
			switch ei.Event() {
			case notify.Create:
				log.Println("create", ei)
			case notify.Write:
				log.Println("write", ei)
			}
		}
	}()

	select {}
}
```

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

	if err := notify.Watch(".", write, notify.Write); err != nil {
		log.Fatal(err)
	}

	go func() {
		for ei := range create {
			log.Println("create", ei)
		}
	}()

	go func() {
		for ei := range write {
			log.Println("write", ei)
		}
	}()

	select {}
}
```
