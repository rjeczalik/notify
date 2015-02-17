// Copyright (c) 2014-2015 The Notify Authors. All rights reserved.
// Use of this source code is governed by the MIT license that can be
// found in the LICENSE file.

// Command notify listens on filesystem changes and forwards received mapping to
// user-defined handlers.
//
// Usage
//
//    usage: notify [-c command] [-f script file] [path]...
//
// The -c flag registers a command handler, which uses the syntax
// of package template. Notify passes struct to the template,
// splits produced string into command and args, and runs it using
// exec.Command(). Additionaly the path and event type values are
// accesible to the process via NOTIFY_PATH and NOTIFY_EVENT
// environment variables.
//
// The struct being passed to the template is:
//
//   type Event struct {
//       Path  string
//       Event string
//   }
//
// Values for the Event field are:
//
//   - create
//   - remove
//   - rename
//   - write
//
// The -t flag registers a file handler, which works similary
// to the -c handler. The only difference the template is read from
// the given file instead of the command line.
//
// The path argument tells notify which director or directories to
// listen on. By default notify listens recursively in current working
// directory.
//
// If no handler is specified notify prints each event to os.Stdout.
//
// Example usage
//
// Executing event handler from command line:
//
//   ~ $ notify -c 'echo "Hello from handler! (event={{.Event}}, path={{.Path}})"'
//   2015/02/17 01:17:40 received notify.Create: "/Users/rjeczalik/notify.tmp"
//   Hello from handler! (event=create, path=/Users/rjeczalik/notify.tmp)
//  ...
//
// Executing event handler from file:
//
//   ~ $ cat > handler <<EOF
//   > echo "Hello from handler! (event={{.Event}}, path={{.Path}})"
//   > EOF
//
//   ~ $ notify -f handler
//   2015/02/17 01:22:26 received notify.Create: "/Users/rjeczalik/notify.tmp"
//   Hello from handler! (event=create, path=/Users/rjeczalik/notify.tmp)
//   ...
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"text/template"

	"github.com/rjeczalik/notify"
)

const usage = `usage: notify [-c command] [-f script file] [path]...

Listens on filesystem changes and forwards received mapping to
user-defined handlers.

The -c flag registers a command handler, which uses the syntax
of package template. Notify passes struct to the template,
splits produced string into command and args, and runs it using
exec.Command(). Additionaly the path and event type values are
accesible to the process via NOTIFY_PATH and NOTIFY_EVENT
environment variables.

The struct being passed to the template is:

	type Event struct {
		Path  string
		Event string
	}

Values for the Event field are:

	- create
	- remove
	- rename
	- write

The -t flag registers a file handler, which works similary
to the -c handler. The only difference the template is read from
the given file instead of the command line.

The path argument tells notify which director or directories to
listen on. By default notify listens recursively in current working
directory.

If no handler is specified notify prints each event to os.Stdout.`

var (
	file    string
	command string
	paths   = []string{"." + string(os.PathSeparator) + "..."}
	env     = newenv()
)

var mapping = map[notify.Event]string{
	notify.Create: "create",
	notify.Remove: "remove",
	notify.Rename: "rename",
	notify.Write:  "write",
}

func newenv() func(Event) []string {
	env := os.Environ()
	for i, s := range env {
		s = strings.ToLower(s)
		if strings.Contains(s, "NOTIFY_PATH=") || strings.Contains(s, "NOTIFY_EVENT=") {
			env[i], env = env[len(env)-1], env[:len(env)-1]
		}
	}
	env = append(env, "", "")
	return func(e Event) []string {
		s := make([]string, len(env))
		copy(s, env)
		s[len(s)-1] = "NOTIFY_EVENT=" + e.Event
		s[len(s)-2] = "NOTIFY_PATH=" + e.Path
		return s
	}
}

// Handler TODO(rjeczalik)
type Handler struct {
	tmpl *template.Template
	env  []string
}

// NewHandler TODO(rjeczalik)
func NewHandler(text string) (*Handler, error) {
	tmpl, err := template.New("main.Handler").Parse(text)
	if err != nil {
		return nil, err
	}
	h := &Handler{
		tmpl: tmpl,
		env:  env(Event{}),
	}
	return h, nil
}

// Run TODO(rjeczalik)
func (h *Handler) Run(e Event) error {
	var buf bytes.Buffer
	if err := h.tmpl.Execute(&buf, e); err != nil {
		return err
	}
	s := buf.String()
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", s)
	default:
		cmd = exec.Command("/bin/sh", "-c", s)
	}
	h.env[len(h.env)-1] = "NOTIFY_EVENT=" + e.Event
	h.env[len(h.env)-2] = "NOTIFY_PATH=" + e.Path
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = h.env
	return cmd.Run()
}

// Daemon TODO(rjeczalik)
func (h *Handler) Daemon() chan<- Event {
	c := make(chan Event)
	go func() {
		for e := range c {
			if err := h.Run(e); err != nil {
				log.Println("handler error:", err)
			}
		}
	}()
	return c
}

// Event TODO(rjeczalik)
type Event struct {
	Path  string
	Event string
}

// NewEvent TODO(rjeczalik)
func NewEvent(ei notify.EventInfo) Event {
	return Event{
		Path:  ei.Path(),
		Event: mapping[ei.Event()],
	}
}

func die(v interface{}) {
	fmt.Fprintln(os.Stderr, v)
	os.Exit(1)
}

func init() {
	flag.CommandLine.Usage = func() {
		fmt.Fprintln(os.Stderr, usage)
	}
	flag.StringVar(&file, "f", "", "script file to execute on received event")
	flag.StringVar(&command, "c", "", "command to run on received event")
	flag.Parse()
	if flag.NArg() != 0 {
		paths = flag.Args()
	}
}

func main() {
	var handlers []*Handler
	if command != "" {
		h, err := NewHandler(command)
		if err != nil {
			die(err)
		}
		handlers = append(handlers, h)
	}
	if file != "" {
		p, err := ioutil.ReadFile(file)
		if err != nil {
			die(err)
		}
		h, err := NewHandler(string(p))
		if err != nil {
			die(err)
		}
		handlers = append(handlers, h)
	}
	var run []chan<- Event
	for _, h := range handlers {
		run = append(run, h.Daemon())
	}
	c := make(chan notify.EventInfo, 1)
	for _, path := range paths {
		if err := notify.Watch(path, c, notify.All); err != nil {
			die(err)
		}
	}
	for ei := range c {
		log.Println("received", ei)
		e := NewEvent(ei)
		for _, run := range run {
			select {
			case run <- e:
			default:
				log.Println("event dropped due to slow handler")
			}
		}
	}
}
