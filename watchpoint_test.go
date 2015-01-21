package notify

import (
	"fmt"
	"reflect"
	"testing"
)

func call(wp watchpoint, fn interface{}, args []interface{}) eventDiff {
	vals := []reflect.Value{reflect.ValueOf(wp)}
	for _, arg := range args {
		vals = append(vals, reflect.ValueOf(arg))
	}
	res := reflect.ValueOf(fn).Call(vals)
	if n := len(res); n != 1 {
		panic(fmt.Sprintf("unexpected len(res)=%d", n))
	}
	diff, ok := res[0].Interface().(eventDiff)
	if !ok {
		panic(fmt.Sprintf("want typeof(diff)=EventDiff; got %T", res[0].Interface()))
	}
	return diff
}

func TestWatchpoint(t *testing.T) {
	ch := NewChans(5)
	all := All | recursive
	cases := [...]struct {
		fn    interface{}
		args  []interface{}
		diff  eventDiff
		total Event
	}{{ // i=0
		watchpoint.Add,
		[]interface{}{ch[0], Delete},
		eventDiff{0, Delete},
		Delete,
	}, { // i=1
		watchpoint.Add,
		[]interface{}{ch[1], Create | Delete | recursive},
		eventDiff{Delete, Delete | Create},
		Create | Delete | recursive,
	}, { // i=2
		watchpoint.Add,
		[]interface{}{ch[2], Create | Move},
		eventDiff{Create | Delete, Create | Delete | Move},
		Create | Delete | Move | recursive,
	}, { // i=3
		watchpoint.Add,
		[]interface{}{ch[0], Write | recursive},
		eventDiff{Create | Delete | Move, Create | Delete | Move | Write},
		Create | Delete | Move | Write | recursive,
	}, { // i=4
		watchpoint.Add,
		[]interface{}{ch[2], Delete | recursive},
		none,
		Create | Delete | Move | Write | recursive,
	}, { // i=5
		watchpoint.Del,
		[]interface{}{ch[0], all},
		eventDiff{Create | Delete | Move | Write, Create | Delete | Move},
		Create | Delete | Move | recursive,
	}, { // i=6
		watchpoint.Del,
		[]interface{}{ch[2], all},
		eventDiff{Create | Delete | Move, Create | Delete},
		Create | Delete | recursive,
	}, { // i=7
		watchpoint.Add,
		[]interface{}{ch[3], Create | Delete},
		none,
		Create | Delete | recursive,
	}, { // i=8
		watchpoint.Del,
		[]interface{}{ch[1], all},
		none,
		Create | Delete,
	}, { // i=9
		watchpoint.Add,
		[]interface{}{ch[3], recursive | Write},
		eventDiff{Create | Delete, Create | Delete | Write},
		Create | Delete | Write | recursive,
	}, { // i=10
		watchpoint.Del,
		[]interface{}{ch[3], Create},
		eventDiff{Create | Delete | Write, Delete | Write},
		Delete | Write | recursive,
	}, { // i=11
		watchpoint.Add,
		[]interface{}{ch[3], Create | Move},
		eventDiff{Delete | Write, Create | Delete | Move | Write},
		Create | Delete | Move | Write | recursive,
	}, { // i=12
		watchpoint.Add,
		[]interface{}{ch[2], Delete | Write},
		none,
		Create | Delete | Move | Write | recursive,
	}, { // i=13
		watchpoint.Del,
		[]interface{}{ch[3], Create | Delete | Write},
		eventDiff{Create | Delete | Move | Write, Delete | Move | Write},
		Delete | Move | Write | recursive,
	}, { // i=14
		watchpoint.Del,
		[]interface{}{ch[2], Delete},
		eventDiff{Delete | Move | Write, Move | Write},
		Move | Write | recursive,
	}, { // i=15
		watchpoint.Del,
		[]interface{}{ch[3], Move | recursive},
		eventDiff{Move | Write, Write},
		Write,
	}, { // i=16
		watchpoint.AddRecursive,
		[]interface{}{Create | Delete},
		eventDiff{Write, Create | Delete | Write},
		Create | Delete | Write | recursive,
	}, { // i=17
		watchpoint.AddRecursive,
		[]interface{}{Move},
		eventDiff{Create | Delete | Write, Create | Delete | Move | Write},
		Create | Delete | Move | Write | recursive,
	}, { // i=18
		watchpoint.DelRecursive,
		[]interface{}{Create | Delete | Write},
		eventDiff{Create | Delete | Move | Write, Move | Write},
		Move | Write | recursive,
	}, { // i=19
		watchpoint.DelRecursive,
		[]interface{}{Move},
		eventDiff{Move | Write, Write},
		Write,
	}, { // i=20
		watchpoint.Del,
		[]interface{}{ch[2], Write},
		eventDiff{Write, 0},
		0,
	}}
	wp := watchpoint{}
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
