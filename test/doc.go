// Package test provides utility functions and fixtures for testing notify package.
//
// The package consists of two major fixtures:
//
//   - w.go implementing W() which is a fixture for notify.Watcher
//   - r.go implementing R() which is a fixture for notify.Runtime
//
// Each fixture implements helper functions, which are responsible for testing
// single scenario. Each fixture instance defines a test life-time, which determines
// when test cleanup happens.
//
// Fixture scoping
//
// The following test creates two independant R fixtures, performs two independant
// tests and performs a test cleanup after each completed test:
//
//   func TestWatchIndependent(t *testing.T) {
//     test.ExpectCalls(t, cases1)
//     test.ExpectCalls(t, cases2)
//   }
//
// By creating a fixture instance explicitely within a test makes all test helpers
// executing within the same scope, thus making them depedant on each other:
//
//   func TestWatchDependent(t *testing.T) {
//     fixture := test.W(t)
//     fixture.ExpectCalls(cases1)
//     fixture.ExpectCalls(cases2)
//   }
//
// The idea behind fixture scoping is to make it possible to chain different
// test helpers, allowing for creating more complicated test scenarios with least
// effort.
package test
