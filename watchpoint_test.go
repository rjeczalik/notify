package notify

import (
	"fmt"
	"reflect"
	"testing"
)

func call(wp Watchpoint, fn interface{}, args []interface{}) EventDiff {
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

func TestWatchpoint(t *testing.T) {
	ch := NewChans(5)
	all := All | Recursive
	cases := [...]struct {
		fn    interface{}
		args  []interface{}
		diff  EventDiff
		total Event
	}{{ // i=0
		Watchpoint.Add,
		[]interface{}{ch[0], Delete},
		EventDiff{0, Delete},
		Delete,
	}, { // i=1
		Watchpoint.Add,
		[]interface{}{ch[1], Create | Delete | Recursive},
		EventDiff{Delete, Delete | Create},
		Create | Delete | Recursive,
	}, { // i=2
		Watchpoint.Add,
		[]interface{}{ch[2], Create | Move},
		EventDiff{Create | Delete, Create | Delete | Move},
		Create | Delete | Move | Recursive,
	}, { // i=3
		Watchpoint.Add,
		[]interface{}{ch[0], Write | Recursive},
		EventDiff{Create | Delete | Move, Create | Delete | Move | Write},
		Create | Delete | Move | Write | Recursive,
	}, { // i=4
		Watchpoint.Add,
		[]interface{}{ch[2], Delete | Recursive},
		None,
		Create | Delete | Move | Write | Recursive,
	}, { // i=5
		Watchpoint.Del,
		[]interface{}{ch[0], all},
		EventDiff{Create | Delete | Move | Write, Create | Delete | Move},
		Create | Delete | Move | Recursive,
	}, { // i=6
		Watchpoint.Del,
		[]interface{}{ch[2], all},
		EventDiff{Create | Delete | Move, Create | Delete},
		Create | Delete | Recursive,
	}, { // i=7
		Watchpoint.Add,
		[]interface{}{ch[3], Create | Delete},
		None,
		Create | Delete | Recursive,
	}, { // i=8
		Watchpoint.Del,
		[]interface{}{ch[1], all},
		None,
		Create | Delete,
	}, { // i=9
		Watchpoint.Add,
		[]interface{}{ch[3], Recursive | Write},
		EventDiff{Create | Delete, Create | Delete | Write},
		Create | Delete | Write | Recursive,
	}, { // i=10
		Watchpoint.Del,
		[]interface{}{ch[3], Create},
		EventDiff{Create | Delete | Write, Delete | Write},
		Delete | Write | Recursive,
	}, { // i=11
		Watchpoint.Add,
		[]interface{}{ch[3], Create | Move},
		EventDiff{Delete | Write, Create | Delete | Move | Write},
		Create | Delete | Move | Write | Recursive,
	}, { // i=12
		Watchpoint.Add,
		[]interface{}{ch[2], Delete | Write},
		None,
		Create | Delete | Move | Write | Recursive,
	}, { // i=13
		Watchpoint.Del,
		[]interface{}{ch[3], Create | Delete | Write},
		EventDiff{Create | Delete | Move | Write, Delete | Move | Write},
		Delete | Move | Write | Recursive,
	}, { // i=14
		Watchpoint.Del,
		[]interface{}{ch[2], Delete},
		EventDiff{Delete | Move | Write, Move | Write},
		Move | Write | Recursive,
	}}
	wp := Watchpoint{}
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
}
