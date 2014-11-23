package notify

import "os"

const sep = string(os.PathSeparator)

// Joinevents TODO
func joinevents(events []Event) (e Event) {
	if len(events) == 0 {
		e = All
	} else {
		for _, event := range events {
			e |= event
		}
	}
	return
}

func Split(s string) (string, string) {
	if i := LastIndexSep(s); i != -1 {
		return s[:i], s[i+1:]
	}
	return "", s
}

func Base(s string) string {
	if i := LastIndexSep(s); i != -1 {
		return s[i+1:]
	}
	return s
}

// IndexSep TODO
func IndexSep(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] == os.PathSeparator {
			return i
		}
	}
	return -1
}

// LastIndexSep TODO
func LastIndexSep(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == os.PathSeparator {
			return i
		}
	}
	return -1
}
