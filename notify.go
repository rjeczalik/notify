// Copyright (c) 2014-2015 The Notify Authors. All rights reserved.
// Use of this source code is governed by the MIT license that can be
// found in the LICENSE file.

package notify

import "sync"

type notifier interface {
	Watch(string, chan<- EventInfo, ...Event) error
	Stop(chan<- EventInfo)
}

func newNotifier(w watcher, c chan EventInfo) notifier {
	if rw, ok := w.(recursiveWatcher); ok {
		return newRecursiveTree(rw, c)
	}
	return newTree(w, c)
}

func initg() {
	if g == nil {
		c := make(chan EventInfo, 128)
		g = newNotifier(newWatcher(c), c)
	}
}

var once sync.Once
var m sync.Mutex
var g notifier

func tree() notifier {
	once.Do(initg)
	return g
}

// Watch TODO
func Watch(name string, c chan<- EventInfo, events ...Event) (err error) {
	m.Lock()
	err = tree().Watch(name, c, events...)
	m.Unlock()
	return
}

// Stop TODO
func Stop(c chan<- EventInfo) {
	m.Lock()
	tree().Stop(c)
	m.Unlock()
	return
}
