// +build linux,!fsnotify

package notify

import "testing"

// TODO(rjeczalik): Sorry Knaps for splitting this, but acceptance tests must
// be in separate notify_test package. Eventually after everything is done, I'd
// move all Watchers and Runtime into /internal/ so the decodemask function can
// be exported and this way this test file can be merged back with the original
// one.

func TestEventMaskEvent(t *testing.T) {
	tests := []struct {
		passed   Event
		got      Event
		received Event
	}{
		{Create, IN_CREATE, Create},
		{Create, IN_MOVED_TO, Create},
		{Create | IN_CREATE, IN_CREATE, IN_CREATE},
		{Create | IN_MOVED_TO, IN_MOVED_TO, IN_MOVED_TO},
		{Create | IN_CREATE, IN_MOVED_TO, Create},
		{Create | IN_MOVED_TO, IN_CREATE, Create},
		{Create | IN_MOVED_TO | IN_CREATE, IN_CREATE, IN_CREATE},

		{Delete, IN_DELETE, Delete},
		{Delete, IN_DELETE_SELF, Delete},
		{Delete | IN_DELETE, IN_DELETE, IN_DELETE},
		{Delete | IN_DELETE_SELF, IN_DELETE_SELF, IN_DELETE_SELF},
		{Delete | IN_DELETE, IN_DELETE_SELF, Delete},
		{Delete | IN_DELETE_SELF, IN_DELETE, Delete},
		{Delete | IN_DELETE_SELF | IN_DELETE, IN_DELETE, IN_DELETE},

		{Write, IN_MODIFY, Write},
		{Write | IN_MODIFY, IN_MODIFY, IN_MODIFY},

		{Move, IN_MOVED_FROM, Move},
		{Move, IN_MOVE_SELF, Move},
		{Move | IN_MOVED_FROM, IN_MOVED_FROM, IN_MOVED_FROM},
		{Move | IN_MOVE_SELF, IN_MOVE_SELF, IN_MOVE_SELF},
		{Move | IN_MOVED_FROM, IN_MOVE_SELF, Move},
		{Move | IN_MOVE_SELF, IN_MOVED_FROM, Move},
		{Move | IN_MOVE_SELF | IN_MOVED_FROM, IN_MOVED_FROM, IN_MOVED_FROM},

		{IN_MOVE, IN_MOVED_FROM, IN_MOVED_FROM},
		{IN_MOVE, IN_MOVED_TO, IN_MOVED_TO},
		{Move | IN_MOVE, IN_MOVED_FROM, IN_MOVED_FROM},
		{Create | IN_MOVE, IN_MOVED_TO, IN_MOVED_TO},

		{IN_CLOSE, IN_CLOSE_WRITE, IN_CLOSE_WRITE},
		{IN_CLOSE, IN_CLOSE_NOWRITE, IN_CLOSE_NOWRITE},
		{Move | IN_CLOSE, IN_CLOSE_WRITE, IN_CLOSE_WRITE},
		{Create | IN_CLOSE, IN_CLOSE_NOWRITE, IN_CLOSE_NOWRITE},

		{All, IN_CREATE, Create},
		{All, IN_MOVED_TO, Create},
		{All, IN_DELETE, Delete},
		{All, IN_DELETE_SELF, Delete},
		{All, IN_MODIFY, Write},
		{All, IN_MOVED_FROM, Move},
		{All, IN_MOVE_SELF, Move},

		{IN_ALL_EVENTS, IN_ACCESS, IN_ACCESS},
		{IN_ALL_EVENTS, IN_MODIFY, IN_MODIFY},
		{IN_ALL_EVENTS, IN_ATTRIB, IN_ATTRIB},
		{IN_ALL_EVENTS, IN_CLOSE_WRITE, IN_CLOSE_WRITE},
		{IN_ALL_EVENTS, IN_CLOSE_NOWRITE, IN_CLOSE_NOWRITE},
		{IN_ALL_EVENTS, IN_OPEN, IN_OPEN},
		{IN_ALL_EVENTS, IN_MOVED_FROM, IN_MOVED_FROM},
		{IN_ALL_EVENTS, IN_MOVED_TO, IN_MOVED_TO},
		{IN_ALL_EVENTS, IN_CREATE, IN_CREATE},
		{IN_ALL_EVENTS, IN_DELETE, IN_DELETE},
		{IN_ALL_EVENTS, IN_DELETE_SELF, IN_DELETE_SELF},
		{IN_ALL_EVENTS, IN_MOVE_SELF, IN_MOVE_SELF},

		{IN_ACCESS, IN_ACCESS, IN_ACCESS},
		{IN_MODIFY, IN_MODIFY, IN_MODIFY},
		{IN_ATTRIB, IN_ATTRIB, IN_ATTRIB},
		{IN_CLOSE_WRITE, IN_CLOSE_WRITE, IN_CLOSE_WRITE},
		{IN_CLOSE_NOWRITE, IN_CLOSE_NOWRITE, IN_CLOSE_NOWRITE},
		{IN_OPEN, IN_OPEN, IN_OPEN},
		{IN_MOVED_FROM, IN_MOVED_FROM, IN_MOVED_FROM},
		{IN_MOVED_TO, IN_MOVED_TO, IN_MOVED_TO},
		{IN_CREATE, IN_CREATE, IN_CREATE},
		{IN_DELETE, IN_DELETE, IN_DELETE},
		{IN_DELETE_SELF, IN_DELETE_SELF, IN_DELETE_SELF},
		{IN_MOVE_SELF, IN_MOVE_SELF, IN_MOVE_SELF},
	}

	for i, test := range tests {
		if e := decodemask(uint32(test.passed), uint32(test.got)); e != test.received {
			t.Errorf("want event=%v; got %v (i=%d)", test.received, e, i)
		}
	}
}
