# go-srs

<a href="https://godoc.org/github.com/simple-rtmp-server/go-srs">
    <img src="https://godoc.org/github.com/simple-rtmp-server/go-srs?status.svg" alt="GoDoc">
</a>

Golang implementation for [srs][srs], focus on real-time live streaming cluster.

## Usage

For linux/unix-like os:

```
go get github.com/simple-rtmp-server/go-srs &&
cd $GOPATH/src/github.com/simple-rtmp-server/go-srs &&
$GOPATH/bin/go-srs -c conf/srs.json
```

Or, for windows:

```
go get github.com/simple-rtmp-server/go-srs &&
cd %GOPATH%\src\github.com\simple-rtmp-server\go-srs &&
%GOPATH%\bin\go-srs.exe -c conf\srs.json
```

About how to build and run at current directory:

```
cd $GOPATH/src/github.com/simple-rtmp-server/go-srs &&
go build . && ./go-srs -c conf/srs.json
```

About how to set $GOPATH, read [prepare go][go-prepare].

## IDE

GO SDK: [download][go-download]

JetBrains IntelliJ IDEA: [download][go-ide]

IntelliJ IDEA Golang Plugin: [repository][go-ide-plugin], [download][go-ide-plugin-download]

### Features

1. Supports Multiple Processes.
1. Supports Linux, Unix-like and Windows.
1. Supports JSON style config file.
1. Supports Reload config file.
1. Standard godoc, gofmt, gotest and TravisCI.
1. Supports daemon for unix-like os.
1. [dev] Extend JSON with c++ style comments.
1. [dev] Support heartbeat to report.
1. [dev] Supports Publish and Play RTMP stream.
1. [dev] Supports Delivery VP6/H.264 and Speex/AAC/MP3/Nellymoser codec.

Winlin 2015.10

[srs]: https://github.com/simple-rtmp-server/srs

[go-download]: http://www.golangtc.com/download
[go-prepare]: http://blog.csdn.net/win_lin/article/details/40618671
[go-ide]: http://www.jetbrains.com/idea/download
[go-ide-plugin]: https://github.com/go-lang-plugin-org/go-lang-idea-plugin
[go-ide-plugin-download]: https://plugins.jetbrains.com/plugin/5047
