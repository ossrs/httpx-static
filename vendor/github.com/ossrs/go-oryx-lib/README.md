# go-oryx-lib

[![Build Status](https://travis-ci.org/ossrs/go-oryx-lib.svg?branch=master)](https://travis-ci.org/ossrs/go-oryx-lib)
[![codecov](https://codecov.io/gh/ossrs/go-oryx-lib/branch/master/graph/badge.svg)](https://codecov.io/gh/ossrs/go-oryx-lib)

This library is exported by server [go-oryx](https://github.com/ossrs/go-oryx).

## Requires

[GO1.4](https://golang.org/dl/)+

## Packages

The library provides packages about network and multiple media processing:

- [x] [logger](logger/example_test.go): Connection-Oriented logger for application server.
- [x] [json](json/example_test.go): Json+ supports c and c++ style comments.
- [x] [options](options/example_test.go): Frequently used service options with config file.
- [x] [http](http/example_test.go): For http response with error, jsonp and std reponse.
- [x] [asprocess](asprocess/example_test.go): The associate-process, for SRS/BMS to work with external process.
- [x] [kxps](kxps/example_test.go): The k-some-ps, for example, kbps, krps.
- [x] [https](https/example_test.go): For https server over [lego/acme](https://github.com/xenolf/lego/tree/master/acme) of [letsencrypt](https://letsencrypt.org/).
- [x] [gmoryx](gmoryx/README.md): A [gomobile](https://github.com/golang/mobile) API for go-oryx-lib.
- [x] [flv](flv/example_test.go): The FLV muxer and demuxer, for oryx.
- [x] [errors](errors/example_test.go): Fork from [pkg/errors](https://github.com/pkg/errors), a complex error with message and stack, read [article](https://gocn.io/article/348).
- [x] [aac](aac/example_test.go): The AAC utilities to demux and mux AAC RAW data, for oryx.
- [x] [websocket](https://golang.org/x/net/websocket): Fork from [websocket](https://github.com/gorilla/websocket/tree/v1.2.0).
- [ ] [sip](sip/example_test.go): A [sip RFC3261](https://tools.ietf.org/html/rfc3261) library for WebRTC signaling.
- [ ] [turn](turn/example_test.go): A [turn RFC5766](https://tools.ietf.org/html/rfc5766) library for WebRTC and SFU.
- [ ] [avc](avc/example_test.go): The AVC utilities to demux and mux AVC RAW data, for oryx.
- [ ] [rtmp](rtmp/example_test.go): The RTMP protocol stack, for oryx.

> Remark: For library, please never use `logger`, use `errors` instead.

Other multiple media libraries in golang:

- [x] [go-speex](https://github.com/winlinvip/go-speex): A go binding for [speex](https://speex.org/).
- [x] [go-fdkaac](https://github.com/winlinvip/go-fdkaac): A go binding for [fdk-aac](https://github.com/mstorsjo/fdk-aac).
- [x] [go-aresample](https://github.com/winlinvip/go-aresample): Resample the audio PCM.

## License

This library just depends on golang standard library,
we do this by copying the code of other libraries,
while all the licenses are liberal:

1. [go-oryx-lib](LICENSE) uses [MIT License](https://github.com/ossrs/go-oryx-lib/blob/master/LICENSE).
1. [pkg/errors](errors/LICENSE) uses [BSD 2-clause "Simplified" License](https://github.com/pkg/errors/blob/master/LICENSE).
1. [acme](https/acme/LICENSE) uses [MIT License](https://github.com/xenolf/lego/blob/master/LICENSE).
1. [jose](https/jose/LICENSE) uses [Apache License 2.0](https://github.com/square/go-jose/blob/v1.1.0/LICENSE).
1. [letsencrypt](https/letsencrypt/LICENSE) uses [BSD 3-clause "New" or "Revised" License](https://github.com/rsc/letsencrypt/blob/master/LICENSE).
1. [websocket](https://github.com/gorilla/websocket) uses [BSD 2-clause "Simplified" License](https://github.com/gorilla/websocket/blob/master/LICENSE).

Winlin 2016
