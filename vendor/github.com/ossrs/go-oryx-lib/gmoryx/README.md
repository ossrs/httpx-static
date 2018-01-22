# GMORYX

The GMORYX(GOMOBILE ORYX) is a API adapter to use [go-oryx-lib](https://github.com/ossrs/go-oryx-lib)
in Android or iOS.

- [x] AndroidHTTPServer, The HTTP server example for Android.

## GOMOBILE

To setup the [gomobile](https://github.com/golang/go/wiki/Mobile),
please read [blog post](http://blog.csdn.net/win_lin/article/details/60956485).

The Go mobile subrepository adds support for mobile platforms (Android and iOS)
and provides tools to build mobile applications.

There are two strategies you can follow to include Go into your mobile stack:

- Writing all-Go native mobile applications.
- Writing SDK applications by generating bindings from a Go package and invoke them
from Java (on Android) and Objective-C (on iOS).

For more information, please read [wiki](https://github.com/golang/go/wiki/Mobile)
and [repository](https://github.com/golang/mobile).

## AndroidHTTPServer

First of all, please build the library `gmoryx.aar` by:

```
cd $GOPATH/src/github.com/ossrs/go-oryx-lib/gmoryx &&
mkdir -p AndroidHTTPServer/app/libs &&
gomobile bind -target=android -o AndroidHTTPServer/app/libs/gmoryx.aar
```

> Remark: Read [GOMOBILE](#gomobile) to setup environment.

Open this project in AndroidStudio, run in Android phone, which will start a web server:

![GMOryx on Android](https://cloud.githubusercontent.com/assets/2777660/23847853/4abcce20-080f-11e7-83e3-3e12cae4dda3.png)

Access the web server:

![Firefox Client](https://cloud.githubusercontent.com/assets/2777660/23847860/52d54010-080f-11e7-8c97-4f8901aa4b35.png)

Winlin, 2017


