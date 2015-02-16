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

*Usage*

```bash
~ $ notify -c 'echo "Hello from handler! (event={{.Event}}, path={{.Path}})"'
2015/02/17 01:17:40 received notify.Create: "/Users/rjeczalik/notify.tmp"
Hello from handler! (event=create, path=/Users/rjeczalik/notify.tmp)
2015/02/17 01:18:13 received notify.Write: "/Users/rjeczalik/notify.tmp"
Hello from handler! (event=write, path=/Users/rjeczalik/notify.tmp)
```
```bash
~ $ cat > handler <<EOF
> echo "Hello from handler! (event={{.Event}}, path={{.Path}})"
> EOF
~ $ notify -f handler
2015/02/17 01:22:26 received notify.Create: "/Users/rjeczalik/notify.tmp"
Hello from handler! (event=create, path=/Users/rjeczalik/notify.tmp)
2015/02/17 01:22:26 received notify.Remove: "/Users/rjeczalik/notify.tmp"
Hello from handler! (event=remove, path=/Users/rjeczalik/notify.tmp)
```
