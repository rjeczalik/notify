// Copyright (c) 2014-2015 The Notify Authors. All rights reserved.
// Use of this source code is governed by the MIT license that can be
// found in the LICENSE file.

// +build !darwin,!linux,!freebsd,!dragonfly,!netbsd,!openbsd,!windows
// +build !kqueue

package notify

// NewWatcher stub.
func newWatcher() watcher {
	return nil
}
