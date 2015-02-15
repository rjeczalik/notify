// Copyright (c) 2014-2015 The Notify Authors. All rights reserved.
// Use of this source code is governed by the MIT license that can be
// found in the LICENSE file.

// +build !darwin,!linux,!freebsd,!dragonfly,!netbsd,!openbsd,!windows
// +build !kqueue

package notify

import "errors"

// Platform independent event values.
const (
	Create Event = iota
	Delete
	Write
	Move
	_ // reserved

	// recursive is used to distinguish recursive eventsets from non-recursive ones
	recursive
	// internal is used for watching for new directories within recursive subtrees
	// by non-recrusive watchers
	internal
)

var osestr = map[Event]string{}

var ekind = map[Event]Event{}

type event struct{}

func (e *event) Event() Event     { return Error }
func (e *event) Path() string     { return "" }
func (e *event) Sys() interface{} { return nil }

func isdir(EventInfo) (bool, error) {
	return false, errors.New("notify: not implemented")
}
