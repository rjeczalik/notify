// +build windows

package notify

func matches(set Event, event Event) bool {
	return set&event == event
}
