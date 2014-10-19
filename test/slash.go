package test

import (
	"fmt"
	"path/filepath"
)

// Slash TODO
//
// Slash panics if v is of different type than []CallCase.
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
	default:
		panic(fmt.Sprintf("notify/test: unexpected cases type: %T", cases))
	}
}
