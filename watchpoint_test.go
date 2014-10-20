package notify_test

import (
	"fmt"
	"reflect"
	"testing"

	. "github.com/rjeczalik/notify"
	"github.com/rjeczalik/notify/test"
)

func call(wp WatchPoint, fn interface{}, args []interface{}) EventDiff {
	vals := []reflect.Value{reflect.ValueOf(wp)}
	for _, arg := range args {
		vals = append(vals, reflect.ValueOf(arg))
	}
	res := reflect.ValueOf(fn).Call(vals)
	if n := len(res); n != 1 {
		panic(fmt.Sprintf("unexpected len(res)=%d", n))
	}
	diff, ok := res[0].Interface().(EventDiff)
	if !ok {
		panic(fmt.Sprintf("want typeof(diff)=EventDiff; got %T", res[0].Interface()))
	}
	return diff
}

func TestWatchPoint(t *testing.T) {
	ch := test.Chans(5)
	cases := [...]struct {
		fn    interface{}
		args  []interface{}
		diff  EventDiff
		total Event
	}{{
		WatchPoint.Add,
		[]interface{}{ch[0], Delete},
		EventDiff{0, Delete},
		Delete,
	}, {
		WatchPoint.Add,
		[]interface{}{ch[1], Create | Delete | Recursive},
		EventDiff{Delete, Delete | Create},
		Create | Delete | Recursive,
	}, {
		WatchPoint.Add,
		[]interface{}{ch[2], Create | Move},
		EventDiff{Create | Delete, Create | Delete | Move},
		Create | Delete | Move | Recursive,
	}, {
		WatchPoint.Add,
		[]interface{}{ch[0], Write | Recursive},
		EventDiff{Create | Delete | Move, Create | Delete | Move | Write},
		Create | Delete | Move | Write | Recursive,
	}, {
		WatchPoint.Add,
		[]interface{}{ch[2], Delete | Recursive},
		None,
		Create | Delete | Move | Write | Recursive,
	}, {
		WatchPoint.Del,
		[]interface{}{ch[0]},
		EventDiff{Create | Delete | Move | Write, Create | Delete | Move},
		Create | Delete | Move | Recursive,
	}, {
		WatchPoint.Del,
		[]interface{}{ch[2]},
		EventDiff{Create | Delete | Move, Create | Delete},
		Create | Delete | Recursive,
	}, {
		WatchPoint.Add,
		[]interface{}{ch[3], Create | Delete},
		None,
		Create | Delete | Recursive,
	}, {
		WatchPoint.Del,
		[]interface{}{ch[1]},
		None,
		Create | Delete,
	}, {
		WatchPoint.Add,
		[]interface{}{ch[3], Recursive | Write},
		EventDiff{Create | Delete, Create | Delete | Write},
		Create | Delete | Write | Recursive,
	}}
	wp := WatchPoint{}
	for i, cas := range cases {
		if diff := call(wp, cas.fn, cas.args); diff != cas.diff {
			t.Errorf("want diff=%v; got %v (i=%d)", cas.diff, diff, i)
			continue
		}
		if total := wp[nil]; total != cas.total {
			t.Errorf("want total=%v; got %v (i=%d)", cas.total, total, i)
			continue
		}
	}
	if n := len(wp); n != 2 {
		t.Errorf("want len(wp)=2; got %d", n)
	}
	e := Create | Delete | Write | Recursive
	if total := wp[nil]; e != total {
		t.Errorf("want total=%v; got %v", e, total)
	}
	if ch3 := wp[ch[3]]; ch3 != e {
		t.Errorf("want ch3=%v; got %v", e, ch3)
	}
}
