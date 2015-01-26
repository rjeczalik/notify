package notify

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

func stacktrace(max int) []string {
	pc, stack := make([]uintptr, max), make([]string, 0, max)
	runtime.Callers(2, pc)
	for _, pc := range pc {
		if f := runtime.FuncForPC(pc); f != nil {
			fname := f.Name()
			idx := strings.LastIndex(fname, string(os.PathSeparator))
			if idx != -1 {
				stack = append(stack, fname[idx+1:])
			} else {
				stack = append(stack, fname)
			}
		}
	}
	return stack
}

type debug bool

func (d debug) Print(v ...interface{}) {
	if d {
		fmt.Printf("[D] ")
		fmt.Print(v...)
		fmt.Printf(" (callstack=%v)\n", stacktrace(3))
	}
}

func (d debug) Printf(format string, v ...interface{}) {
	if d {
		fmt.Printf("[D] ")
		fmt.Printf(format, v...)
		fmt.Printf("\n\n")
	}
}

var dbg = func() debug {
	if os.Getenv("NOTIFY_DEBUG") != "" {
		return true
	}
	return false
}()
