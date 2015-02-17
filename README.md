[![Build Status](https://img.shields.io/travis/rjeczalik/notify/master.svg)](https://travis-ci.org/rjeczalik/notify "inotify") [![Build Status](https://img.shields.io/travis/rjeczalik/notify/fsevents.svg)](https://travis-ci.org/rjeczalik/notify "FSEvents") [![Build Status](https://img.shields.io/travis/rjeczalik/notify/kqueue.svg)](https://travis-ci.org/rjeczalik/notify "kqueue") [![Build status](https://img.shields.io/appveyor/ci/rjeczalik/notify-246.svg)](https://ci.appveyor.com/project/rjeczalik/notify-246 "ReadDirectoryChangesW") [![Coverage Status](https://img.shields.io/coveralls/rjeczalik/notify/master.svg)](https://coveralls.io/r/rjeczalik/notify?branch=master)
======

### notify [![GoDoc](https://godoc.org/github.com/rjeczalik/notify?status.svg)](https://godoc.org/github.com/rjeczalik/notify)

Filesystem event notification library on steroids. (under active development)

*Installation*

```
~ $ go get -u github.com/rjeczalik/notify
```

*Documentation* 

[godoc.org/github.com/rjeczalik/notify](https://godoc.org/github.com/rjeczalik/notify)

### cmd/notify [![GoDoc](https://godoc.org/github.com/rjeczalik/notify?status.svg)](https://godoc.org/github.com/rjeczalik/notify)

Listens on filesystem changes and forwards received events to user-defined handlers.

*Installation*

```
~ $ go get -u github.com/rjeczalik/notify/cmd/notify
```

*Documentation*

[godoc.org/github.com/rjeczalik/notify/cmd/notify](https://godoc.org/github.com/rjeczalik/notify/cmd/notify)
