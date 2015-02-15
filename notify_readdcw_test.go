// Copyright (c) 2014-2015 The Notify Authors. All rights reserved.
// Use of this source code is governed by the MIT license that can be
// found in the LICENSE file.

// +build windows

package notify

import "testing"

func TestNotifySystemAndGlobalMix(t *testing.T) {
	t.Skip("TODO(ppknap)")
	n := NewNotifyTest(t, "testdata/vfs.txt")
	defer n.Close()

	ch := NewChans(2)

	n.Watch("src/github.com/rjeczalik/fs", ch[0], Create)
	n.Watch("src/github.com/rjeczalik/fs", ch[1], FileActionAdded)

	cases := []NCase{
		{
			Event:    rcreate(n.W(), "src/github.com/rjeczalik/fs/.main.cc.swr"),
			Receiver: Chans{ch[0], ch[1]},
		},
	}

	n.ExpectNotifyEvents(cases, ch)
}
