# go-oryx

<a href="https://godoc.org/github.com/ossrs/go-oryx">
    <img src="https://godoc.org/github.com/ossrs/go-oryx?status.svg" alt="GoDoc">
</a>

Focus on real-time live streaming cluster, advanced [srs][srs].

## Usage

For linux/unix-like os:

```
go get github.com/ossrs/go-oryx &&
cd $GOPATH/src/github.com/ossrs/go-oryx &&
$GOPATH/bin/go-oryx -c conf/srs.json
```

Or, for windows:

```
go get github.com/ossrs/go-oryx &&
cd %GOPATH%\src\github.com\ossrs\go-oryx &&
%GOPATH%\bin\go-oryx.exe -c conf\srs.json
```

About how to build and run at current directory:

```
cd $GOPATH/src/github.com/ossrs/go-oryx &&
go build . && ./go-oryx -c conf/srs.json
```

About how to set $GOPATH, read [prepare go][go-prepare].

## IDE

GO SDK: [download][go-download]

JetBrains IntelliJ IDEA: [download][go-ide]

IntelliJ IDEA Golang Plugin: [repository][go-ide-plugin], [download][go-ide-plugin-download]

### Features

1. v0.1.0 Supports Multiple Processes.
1. v0.1.0 Supports Linux, Unix-like and Windows.
1. v0.1.1 Supports JSON style config file.
1. v0.1.2 Supports Reload config file.
1. v0.1.3 Standard godoc, gofmt, gotest and TravisCI.
1. v0.1.4 Support daemon over [ossrs/go-daemon][go-daemon](fork from [sevlyar/go-daemon][fork-go-daemon]).
1. v0.1.5 Extend JSON with c++ style comments.
1. v0.1.6 Support heartbeat to report for ARM.
1. v0.1.7 Use agent(source+channel+sink) to build complex stream river.
1. v0.1.8 Supports Publish and Play VP6+Speex RTMP stream.
1. [dev] Supports Delivery VP6/H.264 and Speex/AAC/MP3/Nellymoser codec.
1. [dev] Supports Connection Oriented Traceable log.

Winlin 2015.10

[srs]: https://github.com/ossrs/srs

[go-download]: http://www.golangtc.com/download
[go-prepare]: http://blog.csdn.net/win_lin/article/details/40618671
[go-ide]: http://www.jetbrains.com/idea/download
[go-ide-plugin]: https://github.com/go-lang-plugin-org/go-lang-idea-plugin
[go-ide-plugin-download]: https://plugins.jetbrains.com/plugin/5047
[go-daemon]: http://github.com/ossrs/go-daemon
[fork-go-daemon]: http://github.com/sevlyar/go-daemon
