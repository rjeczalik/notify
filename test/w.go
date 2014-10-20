package test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rjeczalik/fs/fsutil"
	"github.com/rjeczalik/notify"
)

// Actions TODO
type Actions map[notify.Event]func(path string) error

// DefaultActions TODO
var defaultActions = Actions{
	notify.Create: func(p string) error {
		if isDir(p) {
			return os.MkdirAll(p, 0755)
		}
		f, err := os.Create(p)
		if err != nil {
			return err
		}
		return f.Close()

	},
	notify.Delete: func(p string) error {
		return os.RemoveAll(p)
	},
	notify.Write: func(p string) error {
		f, err := os.OpenFile(p, os.O_RDWR, 0755)
		if err != nil {
			return err
		}
		fi, err := f.Stat()
		if err != nil {
			f.Close()
			return err
		}
		if fi.IsDir() {
			panic("invalid EventInfo exec: " + p)
		}
		_, err = f.WriteString(p)
		if err != nil {
			f.Close()
			return err
		}
		return f.Close()
	},
	notify.Move: func(p string) error {
		return os.Rename(p, p+".moved")
	},
}

// Timeout is a default timeout for ExpectEvent and ExpectEvents tests.
var Timeout = time.Second

// W TODO
type w struct {
	actions Actions
	path    string
	t       *testing.T
	iswatch uint32
}

// W TODO
func W(t *testing.T, actions Actions) *w {
	for s, fn := range defaultActions {
		if _, ok := actions[s]; !ok {
			actions[s] = fn
		}
	}
	path, err := FS.Dump()
	if err != nil {
		t.Fatal(err)
	}
	return &w{
		actions: actions,
		path:    path,
		t:       t,
	}
}

func (w w) equal(want, got notify.EventInfo) error {
	wante, wantp, wantb := want.Event(), want.Name(), want.IsDir()
	gote, gotp, gotb := got.Event(), got.Name(), got.IsDir()
	if !strings.HasPrefix(gotp, w.path) {
		return fmt.Errorf("want EventInfo.Name()=%q to be rooted at %q", gotp,
			w.path)
	}
	// Strip the temp path from the event's origin and convert to slashes.
	gotp = filepath.ToSlash(gotp[len(w.path)+1:])
	// Strip trailing slash from expected path.
	if n := len(wantp) - 1; wantp[n] == '/' {
		wantp = wantp[:n]
	}
	// Take into account wantb, gotb (not taken because of fsnotify for delete).
	if wante != gote || wantp != gotp {
		return fmt.Errorf("want EventInfo{Event: %v, Name: %s, IsDir: %v}; "+
			"got EventInfo{Event: %v, Name: %s, IsDir: %v}", wante, wantp, wantb,
			gote, gotp, gotb)
	}
	return nil

}

func (w w) exec(ei notify.EventInfo) error {
	fn, ok := w.actions[ei.Event()]
	if !ok {
		return fmt.Errorf("unexpected fixture failure: invalid Event=%v", ei.Event())
	}
	if err := fn(join(w.path, ei.Name())); err != nil {
		return fmt.Errorf("want err=nil; got %v (ei=%+v)", err, ei)
	}
	return nil
}

func (w w) walk(fn filepath.WalkFunc) error {
	return fsutil.Rel(FS, w.path).Walk(sep, fn)
}

// ErrAlreadyWatched is returned when WatchAll is called more than once on
// a single instance of the w fixture.
var ErrAlreadyWatched = errors.New("notify/test: path already being watched")

// ErrNotWatched is returned when UnwatchAll is called more than once on
// a single instance of the w fixture.
var ErrNotWatched = errors.New("notify/test: path is not being watched")

// WatchAll is a temporary implementation for RecursiveWatch.
//
// TODO(rjeczalik): Replace with Watcher.RecursiveWatch.
func (w *w) WatchAll(wr notify.Watcher, e notify.Event) error {
	if !atomic.CompareAndSwapUint32(&w.iswatch, 0, 1) {
		return ErrAlreadyWatched
	}
	return w.walk(watch(wr, e))
}

// UnwatchAll is a temporary implementation for RecursiveUnwatch.
func (w *w) UnwatchAll(wr notify.Watcher) error {
	if !atomic.CompareAndSwapUint32(&w.iswatch, 1, 0) {
		return ErrNotWatched
	}
	return w.walk(unwatch(wr))
}

// Close TODO
//
// TODO(rjeczalik): Some safety checks?
func (w *w) Close() error {
	return os.RemoveAll(w.path)
}

// ExpectEvent watches events described by e within Watcher given by the w and
// executes in order events described by ei.
//
// It immadiately fails and stops if either expected event was not received or
// the time test took has exceeded default global timeout.
func (w *w) ExpectEvent(wr notify.Watcher, ei []notify.EventInfo) {
	if wr == nil {
		w.t.Skip("TODO: ExpectEvent on nil Watcher")
	}
	done, c, stop := make(chan error), make(chan notify.EventInfo, len(ei)), make(chan struct{})
	wr.Dispatch(c, stop)
	defer close(stop)
	var i int
	go func() {
		for i = range ei {
			if err := w.exec(ei[i]); err != nil {
				done <- err
				return
			}
			if err := w.equal(ei[i], <-c); err != nil {
				done <- err
				return
			}
		}
		done <- nil
	}()
	select {
	case <-time.After(Timeout):
		w.t.Fatalf("ExpectEvent test has timed out after %v for %v (id:%d)",
			Timeout, ei[i], i)
	case err := <-done:
		if err != nil {
			w.t.Error(err)
		}
	}
}

// ExpectEvents watches events described by e within Watcher given by the w and
// executes in order events described by ei.
//
// It immadiately fails and stops if either received event was not amongst the
// expected ones or the time test took has exceeded default global timeout.
//
// Eventhough cases is described by a map, events are executed in the
// order they were either defined or assigned to the cases.
func (w *w) ExpectEvents(wr notify.Watcher, cases map[notify.EventInfo][]notify.Event) {
	if wr == nil {
		w.t.Skip("TODO: ExpectEvent on nil Watcher")
	}
	done, c, stop := make(chan error), make(chan notify.EventInfo, len(cases)), make(chan struct{})
	wr.Dispatch(c, stop)
	defer close(stop)
	go func() {
		// Sort keys to ensure cases are executed in chronological order.
		for _, ei := range SortKeys(cases) {
			if err := w.exec(ei); err != nil {
				done <- err
				return
			}
			for _, event := range cases[ei] {
				if got := <-c; got.Event() == ei.Event() {
					if err := w.equal(ei, got); err != nil {
						done <- err
						return
					}
				} else {
					if got.Event() != event {
						done <- fmt.Errorf("want %v; got %v (ei=%v)",
							event, got.Event(), ei)
						return
					}
				}
			}
		}
		done <- nil
	}()
	select {
	case <-time.After(Timeout):
		w.t.Fatalf("ExpectEvents test has timed out after %v", Timeout)
	case err := <-done:
		if err != nil {
			w.t.Fatal(err)
		}
	}
}

// ExpectGroupEvents TODO
func (w *w) ExpectGroupEvents(wr notify.Watcher, ei [][]notify.EventInfo) {
	if wr == nil {
		w.t.Skip("TODO: ExpectGroupEvents on nil Watcher")
	}
	done, c, stop := make(chan error), make(chan notify.EventInfo, len(ei)), make(chan struct{})
	wr.Dispatch(c, stop)
	defer close(stop)
	var i int
	go func() {
		for i = range ei {
			if len(ei[i]) == 0 {
				w.t.Fatalf("len(ei[%d])=0", i)
			}
			if err := w.exec(ei[i][0]); err != nil {
				done <- err
				return
			}
			got := make([]notify.EventInfo, 0, len(ei[i]))
			for j := 0; j < len(ei[i]); j++ {
				got = append(got, <-c)
			}
			if len(got) != len(ei[i]) {
				done <- fmt.Errorf("want len(got)=len(ei[i]); got %d!=%d (id:%d)",
					len(got), len(ei[i]), i)
				return
			}
		loop:
			for j := range got {
				for k := range ei[i] {
					if err := w.equal(ei[i][k], got[j]); err == nil {
						continue loop
					}
				}
				done <- fmt.Errorf("%v not present in %v (id:%d)", got[j], ei[i], i)
				return
			}
		}
		done <- nil
	}()
	select {
	case <-time.After(Timeout):
		w.t.Fatalf("ExpecGrouptEvents test has timed out after %v for %v (id:%d)",
			Timeout, ei[i], i)
	case err := <-done:
		if err != nil {
			w.t.Error(err)
		}
	}
}

// TODO(rjeczalik): Create helper method which will implement running global test
// methods using reflect package (aim is to remove duplication in ExpectEvent and
// ExpectEvents via generic generator function).

// ExpectEvent TODO
func ExpectEvent(t *testing.T, wr notify.Watcher, e notify.Event, ei []notify.EventInfo) {
	if wr == nil {
		t.Skip("TODO: ExpectEvent on nil Watcher")
	}
	w := W(t, defaultActions)
	defer w.Close()
	if err := w.WatchAll(wr, e); err != nil {
		t.Fatal(err)
	}
	defer w.UnwatchAll(wr)
	w.ExpectEvent(wr, ei)
}

// ExpectEvents TODO
func ExpectEvents(t *testing.T, wr notify.Watcher, e notify.Event, ei map[notify.EventInfo][]notify.Event) {
	if wr == nil {
		t.Skip("TODO: ExpectEvents on nil Watcher")
	}
	w := W(t, defaultActions)
	defer w.Close()
	if err := w.WatchAll(wr, e); err != nil {
		t.Fatal(err)
	}
	defer w.UnwatchAll(wr)
	w.ExpectEvents(wr, ei)
}

// ExpectGroupEvents watches for event e. Test is configured with ei structure.
// ei[i][0], where i 0..len(ei)-1 is an expected event and is executed
// as requested action. Remaining events are expected to be triggered.
// There is no order requirement of elements of ei[i].
func ExpectGroupEvents(t *testing.T, wr notify.Watcher, e notify.Event,
	ei [][]notify.EventInfo) {
	if wr == nil {
		t.Skip("TODO: ExepctGroupEvents on nil Watcher")
	}
	w := W(t, defaultActions)
	defer w.Close()
	if err := w.WatchAll(wr, e); err != nil {
		t.Fatal(err)
	}
	defer w.UnwatchAll(wr)
	w.ExpectGroupEvents(wr, ei)
}
