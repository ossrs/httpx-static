# go-srs

Golang implementation for [srs][srs].

## Usage

For linux/unix-like os:

```
go get github.com/simple-rtmp-server/go-srs/srs &&
cd $GOPATH/src/github.com/simple-rtmp-server/go-srs &&
$GOPATH/bin/srs -c conf/srs.json
```

Or, for windows:

```
go get github.com/simple-rtmp-server/go-srs/srs &&
cd %GOPATH%\src\github.com\simple-rtmp-server\go-srs &&
%GOPATH%\bin\srs.exe -c conf\srs.json
```

About how to build and run at current directory:

```
cd $GOPATH/src/github.com/simple-rtmp-server/go-srs/srs &&
go build . && ./srs -c ../conf/srs.json
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
1. [dev] Supports Publish and Play RTMP stream.
1. [dev] Supports Delivery VP6/H.264 and Speex/AAC/MP3/Nellymoser codec.

Winlin 2015.10

[srs]: https://github.com/simple-rtmp-server/srs

[go-download]: http://www.golangtc.com/download
[go-prepare]: http://blog.csdn.net/win_lin/article/details/40618671
[go-ide]: http://www.jetbrains.com/idea/download
[go-ide-plugin]: https://github.com/go-lang-plugin-org/go-lang-idea-plugin
[go-ide-plugin-download]: https://plugins.jetbrains.com/plugin/5047
