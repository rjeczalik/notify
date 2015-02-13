package notify

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

// NOTE(rjeczalik): some useful environment variables:
//
//   - NOTIFY_DEBUG gives some extra information about generated events
//   - NOTIFY_TIMEOUT allows for changing default wait time for watcher's
//     events
//   - NOTIFY_TMP allows for changing location of temporary directory trees
//     created for test purpose

// wd TODO(rjeczalik)
var wd = func() string {
	s, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return s
}()

// timeout TODO(rjeczalik)
func timeout() time.Duration {
	if s := os.Getenv("NOTIFY_TIMEOUT"); s != "" {
		if t, err := time.ParseDuration(s); err == nil {
			return t
		}
	}
	return 2 * time.Second
}

// vfs TODO(rjeczalik)
func vfs() (string, string) {
	if s := os.Getenv("NOTIFY_TMP"); s != "" {
		return filepath.Split(s)
	}
	return "testdata", ""
}

// isDir TODO(rjeczalik)
func isDir(path string) bool {
	r := path[len(path)-1]
	return r == '\\' || r == '/'
}

// tmpcreateall TODO(rjeczalik)
func tmpcreateall(tmp string, path string) error {
	isdir := isDir(path)
	path = filepath.Join(tmp, filepath.FromSlash(path))
	if isdir {
		if err := os.MkdirAll(path, 0755); err != nil {
			return err
		}
	} else {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		f, err := os.Create(path)
		if err != nil {
			return err
		}
		if err := nonil(f.Sync(), f.Close()); err != nil {
			return err
		}
	}
	return nil
}

// tmpcreate TODO(rjeczalik)
func tmpcreate(root, path string) (bool, error) {
	isdir := isDir(path)
	path = filepath.Join(root, filepath.FromSlash(path))
	if isdir {
		if err := os.Mkdir(path, 0755); err != nil {
			return false, err
		}
	} else {
		f, err := os.Create(path)
		if err != nil {
			return false, err
		}
		if err := nonil(f.Sync(), f.Close()); err != nil {
			return false, err
		}
	}
	return isdir, nil
}

// tmptree TODO(rjeczalik)
func tmptree(root, list string) (string, error) {
	f, err := os.Open(list)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if root == "" {
		if root, err = ioutil.TempDir(vfs()); err != nil {
			return "", err
		}
	}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if err := tmpcreateall(root, scanner.Text()); err != nil {
			return "", err
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return root, nil
}

// callern TODO(rjeczalik)
func callern(n int) string {
	_, file, line, ok := runtime.Caller(n)
	if !ok {
		return "<unknown>"
	}
	return filepath.Base(file) + ":" + strconv.Itoa(line)
}

// caller TODO(rjeczalik)
func caller() string {
	return callern(3)
}

// WCase TODO(rjeczalik)
type WCase struct {
	Action func()
	Events []EventInfo
}

// String TODO(rjeczalik)
func (cas WCase) String() string {
	s := make([]string, 0, len(cas.Events))
	for _, ei := range cas.Events {
		s = append(s, "Event("+ei.Event().String()+")@"+filepath.FromSlash(ei.Path()))
	}
	return strings.Join(s, ", ")
}

// W TODO(rjeczalik)
type W struct {
	Watcher watcher
	C       chan EventInfo
	Timeout time.Duration

	t    *testing.T
	root string
}

// newWatcherTest TODO(rjeczalik)
func newWatcherTest(t *testing.T, tree string) *W {
	root, err := tmptree("", filepath.FromSlash(tree))
	if err != nil {
		t.Fatalf(`tmptree("", %q)=%v`, tree, err)
	}
	Sync()
	return &W{
		t:    t,
		root: root,
	}
}

// NewWatcherTest TODO
func NewWatcherTest(t *testing.T, tree string, events ...Event) *W {
	w := newWatcherTest(t, tree)
	if len(events) == 0 {
		events = []Event{Create, Delete, Write, Move}
	}
	if rw, ok := w.watcher().(recursiveWatcher); ok {
		if err := rw.RecursiveWatch(w.root, joinevents(events)); err != nil {
			t.Fatalf("RecursiveWatch(%q, All)=%v", w.root, err)
		}
	} else {
		fn := func(path string, fi os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if fi.IsDir() {
				if err := w.watcher().Watch(path, joinevents(events)); err != nil {
					return err
				}
			}
			return nil
		}
		if err := filepath.Walk(w.root, fn); err != nil {
			t.Fatalf("Walk(%q, fn)=%v", w.root, err)
		}
	}
	drainall(w.C)
	return w
}

func (w *W) clean(path string) string {
	path, isrec, err := cleanpath(filepath.Join(w.root, path))
	if err != nil {
		w.Fatalf("cleanpath(%q)=%v", path, err)
	}
	if isrec {
		path = path + "..."
	}
	return path
}

func (w *W) Fatal(v interface{}) {
	w.t.Fatalf("%s: %v", caller(), v)
}

func (w *W) Fatalf(format string, v ...interface{}) {
	w.t.Fatalf("%s: %s", caller(), fmt.Sprintf(format, v...))
}

func (w *W) Watch(path string, e Event) {
	if err := w.watcher().Watch(w.clean(path), e); err != nil {
		w.Fatalf("Watch(%s, %v)=%v", path, e, err)
	}
}

func (w *W) Rewatch(path string, olde, newe Event) {
	if err := w.watcher().Rewatch(w.clean(path), olde, newe); err != nil {
		w.Fatalf("Rewatch(%s, %v, %v)=%v", path, olde, newe, err)
	}
}

func (w *W) Unwatch(path string) {
	if err := w.watcher().Unwatch(w.clean(path)); err != nil {
		w.Fatalf("Unwatch(%s)=%v", path, err)
	}
}

func (w *W) RecursiveWatch(path string, e Event) {
	rw, ok := w.watcher().(recursiveWatcher)
	if !ok {
		w.Fatal("watcher does not implement recursive watching on this platform")
	}
	if err := rw.RecursiveWatch(w.clean(path), e); err != nil {
		w.Fatalf("RecursiveWatch(%s, %v)=%v", path, e, err)
	}
}

func (w *W) RecursiveRewatch(oldp, newp string, olde, newe Event) {
	rw, ok := w.watcher().(recursiveWatcher)
	if !ok {
		w.Fatal("watcher does not implement recursive watching on this platform")
	}
	if err := rw.RecursiveRewatch(w.clean(oldp), w.clean(newp), olde, newe); err != nil {
		w.Fatalf("RecursiveRewatch(%s, %s, %v, %v)=%v", oldp, newp, olde, newe, err)
	}
}

func (w *W) RecursiveUnwatch(path string) {
	rw, ok := w.watcher().(recursiveWatcher)
	if !ok {
		w.Fatal("watcher does not implement recursive watching on this platform")
	}
	if err := rw.RecursiveUnwatch(w.clean(path)); err != nil {
		w.Fatalf("RecursiveUnwatch(%s)=%v", path, err)
	}
}

func (w *W) initwatcher(buffer int) {
	c := make(chan EventInfo, buffer)
	w.Watcher = newWatcher(c)
	w.C = c
}

func (w *W) watcher() watcher {
	if w.Watcher == nil {
		w.initwatcher(512)
	}
	return w.Watcher
}

func (w *W) c() chan EventInfo {
	if w.C == nil {
		w.initwatcher(512)
	}
	return w.C
}

func (w *W) timeout() time.Duration {
	if w.Timeout != 0 {
		return w.Timeout
	}
	return timeout()
}

// Close TODO
func (w *W) Close() error {
	defer os.RemoveAll(w.root)
	if err := w.watcher().Close(); err != nil {
		w.Fatalf("w.Watcher.Close()=%v", err)
	}
	return nil
}

// Equal TODO
func EqualEventInfo(want, got EventInfo) error {
	if got.Event() != want.Event() {
		return fmt.Errorf("want Event()=%v; got %v (path=%s)", want.Event(),
			got.Event(), want.Path())
	}
	path := strings.TrimRight(filepath.FromSlash(want.Path()), `/\`)
	if !strings.HasSuffix(got.Path(), path) {
		return fmt.Errorf("want Path()=%s; got %s (event=%v)", path, got.Path(),
			want.Event())
	}
	return nil
}

// EqualCall TODO(rjeczalik)
func EqualCall(want, got Call) error {
	if want.F != got.F {
		return fmt.Errorf("want F=%v; got %v (P=%q)", want.F, got.F, want.P)
	}
	if got.E != want.E {
		return fmt.Errorf("want E=%v; got %v (P=%q)", want.E, got.E, want.P)
	}
	if got.NE != want.NE {
		return fmt.Errorf("want NE=%v; got %v (P=%q)", want.NE, got.NE, want.P)
	}
	if want.C != got.C {
		return fmt.Errorf("want C=%p; got %p (P=%q)", want.C, got.C, want.P)
	}
	if want := filepath.FromSlash(want.P); !strings.HasSuffix(got.P, want) {
		return fmt.Errorf("want P=%s; got %s", want, got.P)
	}
	if want := filepath.FromSlash(want.NP); !strings.HasSuffix(got.NP, want) {
		return fmt.Errorf("want NP=%s; got %s", want, got.NP)
	}
	return nil
}

// create TODO
func create(w *W, path string) WCase {
	return WCase{
		Action: func() {
			isdir, err := tmpcreate(w.root, filepath.FromSlash(path))
			if err != nil {
				w.Fatalf("tmpcreate(%q, %q)=%v", w.root, path, err)
			}
			if isdir {
				dbg.Printf("[FS] os.Mkdir(%q)\n", path)
			} else {
				dbg.Printf("[FS] os.Create(%q)\n", path)
			}
		},
		Events: []EventInfo{
			&Call{P: path, E: Create},
		},
	}
}

// remove TODO
func remove(w *W, path string) WCase {
	return WCase{
		Action: func() {
			if err := os.RemoveAll(filepath.Join(w.root, filepath.FromSlash(path))); err != nil {
				w.Fatal(err)
			}
			dbg.Printf("[FS] os.Remove(%q)\n", path)
		},
		Events: []EventInfo{
			&Call{P: path, E: Delete},
		},
	}
}

// rename TODO
func rename(w *W, oldpath, newpath string) WCase {
	return WCase{
		Action: func() {
			err := os.Rename(filepath.Join(w.root, filepath.FromSlash(oldpath)),
				filepath.Join(w.root, filepath.FromSlash(newpath)))
			if err != nil {
				w.Fatal(err)
			}
			dbg.Printf("[FS] os.Rename(%q, %q)\n", oldpath, newpath)
		},
		Events: []EventInfo{
			&Call{P: newpath, E: Move},
		},
	}
}

// write TODO
func write(w *W, path string, p []byte) WCase {
	return WCase{
		Action: func() {
			f, err := os.OpenFile(filepath.Join(w.root, filepath.FromSlash(path)),
				os.O_WRONLY, 0644)
			if err != nil {
				w.Fatalf("OpenFile(%q)=%v", path, err)
			}
			if _, err := f.Write(p); err != nil {
				w.Fatalf("Write(%q)=%v", path, err)
			}
			if err := nonil(f.Sync(), f.Close()); err != nil {
				w.Fatalf("Sync(%q)/Close(%q)=%v", path, path, err)
			}
			dbg.Printf("[FS] Write(%q)\n", path)
		},
		Events: []EventInfo{
			&Call{P: path, E: Write},
		},
	}
}

func drainall(c chan EventInfo) (ei []EventInfo) {
	// TODO(rjeczalik): remove
	time.Sleep(50 * time.Millisecond)
	for {
		select {
		case e := <-c:
			ei = append(ei, e)
			runtime.Gosched()
		default:
			return
		}
	}
}

type WCaseFunc func(i int, cas WCase, ei EventInfo) error

// ExpectAnyFunc TODO(rjeczalik)
func (w *W) ExpectAnyFunc(cases []WCase, fn WCaseFunc) {
	UpdateWait() // Wait some time before starting the test.
Test:
	for i, cas := range cases {
		dbg.Printf("ExpectAny: i=%d\n", i)
		cas.Action()
		Sync()
		switch cas.Events {
		case nil:
			if ei := drainall(w.C); len(ei) != 0 {
				w.Fatalf("unexpected dangling events: %v (i=%d)", ei, i)
			}
		default:
			select {
			case ei := <-w.C:
				dbg.Printf("received: path=%q, event=%v, sys=%v (i=%d)", ei.Path(),
					ei.Event(), ei.Sys(), i)
				for j, want := range cas.Events {
					if err := EqualEventInfo(want, ei); err != nil {
						dbg.Print(err, j)
						continue
					}
					if fn != nil {
						if err := fn(i, cas, ei); err != nil {
							w.Fatalf("ExpectAnyFunc(%d, %v)=%v", i, ei, err)
						}
					}
					drainall(w.C) // TODO(rjeczalik): revisit
					continue Test
				}
				w.Fatalf("ExpectAny received an event which does not match any of "+
					"the expected ones (i=%d): want one of %v; got %v", i, cas.Events, ei)
			case <-time.After(w.timeout()):
				w.Fatalf("timed out after %v waiting for one of %v (i=%d)", w.timeout(),
					cas.Events, i)
			}
			drainall(w.C) // TODO(rjeczalik): revisit
		}
	}
}

// ExpectAny TODO(rjeczalik)
func (w *W) ExpectAny(cases []WCase) {
	w.ExpectAnyFunc(cases, nil)
}

// FuncType represents enums for Watcher interface.
type FuncType string

const (
	FuncWatch            = FuncType("Watch")
	FuncUnwatch          = FuncType("Unwatch")
	FuncRewatch          = FuncType("Rewatch")
	FuncRecursiveWatch   = FuncType("RecursiveWatch")
	FuncRecursiveUnwatch = FuncType("RecursiveUnwatch")
	FuncRecursiveRewatch = FuncType("RecursiveRewatch")
	FuncStop             = FuncType("Stop")
)

// Chans TODO
type Chans []chan EventInfo

// Foreach TODO
func (c Chans) Foreach(fn func(chan<- EventInfo, node)) {
	for i, ch := range c {
		fn(ch, node{Name: strconv.Itoa(i)})
	}
}

// Drain TODO
func (c Chans) Drain() (ei []EventInfo) {
	n := len(c)
	stop := make(chan struct{})
	eich := make(chan EventInfo, n*buffer)
	go func() {
		defer close(eich)
		cases := make([]reflect.SelectCase, n+1)
		for i := range c {
			cases[i].Chan = reflect.ValueOf(c[i])
			cases[i].Dir = reflect.SelectRecv
		}
		cases[n].Chan = reflect.ValueOf(stop)
		cases[n].Dir = reflect.SelectRecv
		for {
			i, v, ok := reflect.Select(cases)
			if i == n {
				return
			}
			if !ok {
				panic("(Chans).Drain(): unexpected chan close")
			}
			eich <- v.Interface().(EventInfo)
		}
	}()
	<-time.After(50 * time.Duration(n) * time.Millisecond)
	close(stop)
	for e := range eich {
		ei = append(ei, e)
	}
	return
}

func (c Chans) Debug() {
	for i, c := range c {
		fmt.Printf("ch[%d] = %p\n", i, c)
	}
}

// NewChans TODO
func NewChans(n int) Chans {
	ch := make([]chan EventInfo, n)
	for i := range ch {
		ch[i] = make(chan EventInfo, buffer)
	}
	return ch
}

// Call represents single call to Watcher issued by the Tree
// and recorded by a spy Watcher mock.
type Call struct {
	F  FuncType       //
	C  chan EventInfo //
	P  string         // regular Path argument and old path from RecursiveRewatch call
	NP string         // new Path argument from RecursiveRewatch call
	E  Event          // regular Event argument and old Event from a Rewatch call
	NE Event          // new Event argument from Rewatch call
	S  interface{}    // when Call is used as EventInfo, S is a value of Sys()
}

// Call implements an EventInfo interface.
func (c *Call) Event() Event     { return c.E }
func (c *Call) Path() string     { return c.P }
func (c *Call) String() string   { return fmt.Sprintf("%#v", c) }
func (c *Call) Sys() interface{} { return c.S }

// Spy is a mock for Watcher interface, which records every call.
type Spy []Call

func (s Spy) Close() (_ error) { return }

func (s *Spy) Watch(p string, e Event) (_ error) {
	dbg.Printf("%s: (*Spy).Watch(%q, %v)", caller(), p, e)
	*s = append(*s, Call{F: FuncWatch, P: p, E: e})
	return
}

func (s *Spy) Unwatch(p string) (_ error) {
	dbg.Printf("%s: (*Spy).Unwatch(%q)", caller(), p)
	*s = append(*s, Call{F: FuncUnwatch, P: p})
	return
}

func (s *Spy) Rewatch(p string, olde, newe Event) (_ error) {
	dbg.Printf("%s: (*Spy).Rewatch(%q, %v, %v)", caller(), p, olde, newe)
	*s = append(*s, Call{F: FuncRewatch, P: p, E: olde, NE: newe})
	return
}

func (s *Spy) RecursiveWatch(p string, e Event) (_ error) {
	dbg.Printf("%s: (*Spy).RecursiveWatch(%q, %v)", caller(), p, e)
	*s = append(*s, Call{F: FuncRecursiveWatch, P: p, E: e})
	return
}

func (s *Spy) RecursiveUnwatch(p string) (_ error) {
	dbg.Printf("%s: (*Spy).RecursiveUnwatch(%q)", caller(), p)
	*s = append(*s, Call{F: FuncRecursiveUnwatch, P: p})
	return
}

func (s *Spy) RecursiveRewatch(oldp, newp string, olde, newe Event) (_ error) {
	dbg.Printf("%s: (*Spy).RecursiveRewatch(%q, %q, %v, %v)", caller(), oldp, newp, olde, newe)
	*s = append(*s, Call{F: FuncRecursiveRewatch, P: oldp, NP: newp, E: olde, NE: newe})
	return
}

// RCase TODO(rjeczalik)
type RCase struct {
	Call   Call
	Record []Call
}

// TCase TODO(rjeczalik)
type TCase struct {
	Event    Call
	Receiver Chans
}

// NCase TODO(rjeczalik)
type NCase struct {
	Event    WCase
	Receiver Chans
}

// N TODO(rjeczalik)
type N struct {
	// Notifier TODO(rjeczalik)
	//
	// TODO(rjeczalik): unexport
	notifier notifier

	// Timeout TODO(rjeczalik)
	Timeout time.Duration

	t   *testing.T
	w   *W
	spy *Spy
	c   chan<- EventInfo
	j   int // spy offset

	realroot string
}

func newN(t *testing.T, tree string) *N {
	n := &N{
		t: t,
		w: newWatcherTest(t, tree),
	}
	realroot, err := canonical(n.w.root)
	if err != nil {
		t.Fatalf("%s: unexpected fixture failure: %v", caller(), err)
	}
	n.realroot = realroot
	return n
}

func newTreeN(t *testing.T, tree string, fn func(spy *Spy) watcher) *N {
	n := newN(t, tree)
	n.spy = &Spy{}
	c := make(chan EventInfo, 512)
	n.w.Watcher = fn(n.spy)
	n.w.C = c
	n.c = c
	n.notifier = newNotifier(n.w.watcher(), n.w.c())
	return n
}

// NewNotifyTest TODO(rjeczalik)
func NewNotifyTest(t *testing.T, tree string) *N {
	n := newN(t, tree)
	n.notifier = newNotifier(n.w.watcher(), n.w.c())
	return n
}

// NewTreeTest TODO(rjeczalik)
func NewTreeTest(t *testing.T, tree string) *N {
	fn := func(spy *Spy) watcher {
		return struct {
			watcher
		}{spy}
	}
	return newTreeN(t, tree, fn)
}

// NewRecursiveTreeTest TODO(rjeczalik)
func NewRecursiveTreeTest(t *testing.T, tree string) *N {
	fn := func(spy *Spy) watcher { return spy }
	return newTreeN(t, tree, fn)
}

func (n *N) timeout() time.Duration {
	if n.Timeout != 0 {
		return n.Timeout
	}
	return n.w.timeout()
}

// W TODO(rjeczalik)
func (n *N) W() *W {
	return n.w
}

// Close TODO
func (n *N) Close() (err error) {
	if cl, ok := n.notifier.(io.Closer); ok {
		os.RemoveAll(n.w.root)
		return cl.Close()
	}
	return n.w.Close()
}

// Watch TODO(rjeczalik)
func (n *N) Watch(path string, c chan<- EventInfo, events ...Event) {
	path = filepath.Join(n.w.root, path)
	if err := n.notifier.Watch(path, c, events...); err != nil {
		n.t.Errorf("Watch(%s, %p, %v)=%v", path, c, events, err)
	}
}

// WatchErr TODO(ppknap)
func (n *N) WatchErr(path string, c chan<- EventInfo, err error, events ...Event) {
	path = filepath.Join(n.w.root, path)
	switch e := n.notifier.Watch(path, c, events...); {
	case err == nil && e == nil:
		n.t.Errorf("Watch(%s, %p, %v)=nil", path, c, events)
	case err != nil && e != err:
		n.t.Errorf("Watch(%s, %p, %v)=%v != %v", path, c, events, e, err)
	}
}

// Stop TODO(rjeczalik)
func (n *N) Stop(c chan<- EventInfo) {
	n.notifier.Stop(c)
}

// Call TODO(rjeczalik)
func (n *N) Call(calls ...Call) {
	for i := range calls {
		switch calls[i].F {
		case FuncWatch:
			n.Watch(calls[i].P, calls[i].C, calls[i].E)
		case FuncStop:
			n.Stop(calls[i].C)
		default:
			panic("unsupported call type: " + string(calls[i].F))
		}
	}
}

// expectDry TODO(rjeczalik)
func (n *N) expectDry(ch Chans, i int) {
	if ei := ch.Drain(); len(ei) != 0 {
		n.w.Fatalf("unexpected dangling events: %v (i=%d)", ei, i)
	}
}

// ExpectRecordedCalls TODO(rjeczalik)
func (n *N) ExpectRecordedCalls(cases []RCase) {
	for i, cas := range cases {
		dbg.Printf("ExpectRecordedCalls: i=%d\n", i)
		n.Call(cas.Call)
		record := (*n.spy)[n.j:]
		if len(cas.Record) == 0 && len(record) == 0 {
			continue
		}
		n.j = len(*n.spy)
		if len(record) != len(cas.Record) {
			n.t.Fatalf("%s: want len(record)=%d; got %d [%+v] (i=%d)", caller(),
				len(cas.Record), len(record), record, i)
		}
		for k := range cas.Record {
			if err := EqualCall(cas.Record[k], record[k]); err != nil {
				n.t.Fatalf("%s: %v (i=%d)", caller(), err, i)
			}
		}
	}
}

func (n *N) collect(ch Chans) <-chan []EventInfo {
	done := make(chan []EventInfo)
	go func() {
		cases := make([]reflect.SelectCase, len(ch))
		unique := make(map[<-chan EventInfo]EventInfo, len(ch))
		for i := range ch {
			cases[i].Chan = reflect.ValueOf(ch[i])
			cases[i].Dir = reflect.SelectRecv
		}
		for i := len(cases); i != 0; i = len(cases) {
			j, v, ok := reflect.Select(cases)
			if !ok {
				n.t.Fatal("unexpected chan close")
			}
			ch := cases[j].Chan.Interface().(chan EventInfo)
			got := v.Interface().(EventInfo)
			if ei, ok := unique[ch]; ok {
				n.t.Fatalf("duplicated event %v (previous=%v) received on collect", got, ei)
			}
			unique[ch] = got
			cases[j], cases = cases[i-1], cases[:i-1]
		}
		collected := make([]EventInfo, 0, len(ch))
		for _, ch := range unique {
			collected = append(collected, ch)
		}
		done <- collected
	}()
	return done
}

// abs TODO(rjeczalik)
func (n *N) abs(rel Call) *Call {
	rel.P = filepath.Join(n.realroot, filepath.FromSlash(rel.P))
	if !filepath.IsAbs(rel.P) {
		rel.P = filepath.Join(wd, rel.P)
	}
	return &rel
}

// ExpectTreeEvents TODO(rjeczalik)
func (n *N) ExpectTreeEvents(cases []TCase, all Chans) {
	for i, cas := range cases {
		dbg.Printf("ExpectTreeEvents: i=%d\n", i)
		// Ensure there're no dangling event left by previous test-case.
		n.expectDry(all, i)
		n.c <- n.abs(cas.Event)
		switch cas.Receiver {
		case nil:
			n.expectDry(all, i)
		default:
			ch := n.collect(cas.Receiver)
			select {
			case collected := <-ch:
				for _, got := range collected {
					if err := EqualEventInfo(&cas.Event, got); err != nil {
						n.w.Fatalf("%s: %s (i=%d)", caller(), err, i)
					}
				}
			case <-time.After(n.timeout()):
				n.w.Fatalf("ExpectTreeEvents has timed out after %v waiting for"+
					" %v on %s (i=%d)", n.timeout(), cas.Event.E, cas.Event.P, i)
			}

		}
	}
	n.expectDry(all, -1)
}

// ExpectNotifyEvents TODO(rjeczalik)
func (n *N) ExpectNotifyEvents(cases []NCase, all Chans) {
	UpdateWait() // Wait some time before starting the test.
	for i, cas := range cases {
		dbg.Printf("ExpectNotifyEvents: i=%d\n", i)
		cas.Event.Action()
		Sync()
		switch cas.Receiver {
		case nil:
			n.expectDry(all, i)
		default:
			ch := n.collect(cas.Receiver)
			select {
			case collected := <-ch:
			Compare:
				for j, ei := range collected {
					dbg.Printf("received: path=%q, event=%v, sys=%v (i=%d, j=%d)", ei.Path(),
						ei.Event(), ei.Sys(), i, j)
					for _, want := range cas.Event.Events {
						if err := EqualEventInfo(want, ei); err != nil {
							dbg.Print(err, j)
							continue
						}
						continue Compare
					}
					n.w.Fatalf("ExpectNotifyEvents received an event which does not"+
						" match any of the expected ones (i=%d): want one of %v; got %v", i,
						cas.Event.Events, ei)
				}
			case <-time.After(n.timeout()):
				n.w.Fatalf("ExpectNotifyEvents did not receive any of the expected events [%v] "+
					"after %v (i=%d)", cas.Event, n.timeout(), i)
			}
		}
	}
	n.expectDry(all, -1)
}
