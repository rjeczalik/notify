// +build windows

package notify

// matches reports a match only when:
//
//   - for user events, when event is present in the given set
//   - for internal events, when additionaly both event and set have omit bit set
//
// Internal events must not be sent to user channels and vice versa.
func matches(set, event Event) bool {
	if (set&omit)^(event&omit) == omit {
		// No match - one of set or event contains omit bit; meaning either
		// user event is being dispatched to internal channel or internal
		// event is being dispatched to user channel. This situation is always
		// invalid.
		return false
	}
	// TODO(ppknap): Here be the dragons.
	return set&event == event
}
