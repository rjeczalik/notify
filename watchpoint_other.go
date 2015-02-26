// Copyright (c) 2014-2015 The Notify Authors. All rights reserved.
// Use of this source code is governed by the MIT license that can be
// found in the LICENSE file.

// +build !windows

package notify

// matches reports a match only when:
//
//   - for user events, when event is present in the given set
//   - for internal events, when additionaly both event and set have omit bit set
//
// Internal events must not be sent to user channels and vice versa.
func matches(set, event Event) bool {
	return (set&omit)^(event&omit) == 0 && set&event == event
}
