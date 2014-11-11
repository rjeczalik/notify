package test

import (
	"fmt"
	"path/filepath"
)

// Slash TODO
//
// Slash panics if v is of different type than:
//
//   - []CallCase
//   - []EventCase
//   - PathCase
//   - TreeCase
func SlashCases(cases interface{}) {
	if filepath.Separator == '/' {
		return
	}
	switch cases := cases.(type) {
	case []CallCase:
		for i := range cases {
			cases[i].Call.P = filepath.FromSlash(cases[i].Call.P)
			for _, calls := range cases[i].Record {
				for i := range calls {
					calls[i].P = filepath.FromSlash(calls[i].P)
				}
			}
		}
	case []EventCase:
		for i := range cases {
			cases[i].Event.P = filepath.FromSlash(cases[i].Event.P)
		}
	case PathCase:
		keys := make([]string, 0, len(cases))
		for k := range cases {
			keys = append(keys, k)
		}
		for _, k := range keys {
			cases[filepath.FromSlash(k)] = cases[k]
			delete(cases, k)
		}
	case TreeCase:
		keys := make([]string, 0, len(cases))
		for k := range cases {
			keys = append(keys, k)
		}
		for _, k := range keys {
			vals := make(map[string]struct{}, len(cases[k]))
			for v := range cases[k] {
				vals[filepath.FromSlash(v)] = struct{}{}
			}
			delete(cases, k)
			cases[filepath.FromSlash(k)] = vals
		}
	default:
		panic(fmt.Sprintf("notify/test: unexpected cases type: %T", cases))
	}
}
