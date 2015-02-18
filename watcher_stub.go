// Copyright (c) 2014-2015 The Notify Authors. All rights reserved.
// Use of this source code is governed by the MIT license that can be
// found in the LICENSE file.

// +build !darwin,!linux,!freebsd,!dragonfly,!netbsd,!openbsd,!windows
// +build !kqueue

package notify

type stub struct{ error }

// NewWatcher stub.
func newWatcher(chan<- EventInfo) stub {
	return stub{errors.New("notify: not implemented")}
}

// Following methods implement notify.watcher interface.
func (s stub) Watch(string, Event) error          { return s }
func (s stub) Rewatch(string, Event, Event) error { return s }
func (s stub) Unwatch(string) (err error)         { return s }
func (s stub) Close() error                       { return s }
