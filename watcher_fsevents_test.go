// +build darwin,!kqueue
// +build !fsnotify

package notify

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func TestDebug(t *testing.T) {
	if os.Getenv("DEBUG") == "" {
		t.Skip("TODO(rjeczalik)")
	}
	c, stop := make(chan EventInfo, 16), make(chan struct{})
	fs := &fsevents{
		watches: make(map[string]*watch),
	}
	fs.Dispatch(c, stop)
	go func() {
		for ei := range c {
			fmt.Println(ei)
		}
	}()
	if err := fs.RecursiveWatch("/private/tmp/wut", Create|Delete); err != nil {
		t.Fatalf("want err=nil; got %v", err)
	}
	fmt.Println("listening...")
	time.Sleep(20 * time.Second)
	fmt.Println("unwatching")
	fs.Unwatch("/private/tmp/wut")
	select {}
}
