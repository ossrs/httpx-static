# go-oryx

[![CircleCI](https://circleci.com/gh/ossrs/go-oryx/tree/develop.svg?style=svg)](https://circleci.com/gh/ossrs/go-oryx/tree/develop)
[![Gitter](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/ossrs/go-oryx?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge)

Oryx is next generation media streaming server, extract service to processes 
which communicates over http with each other, to get more flexible, low latency,
programmable and high maintainable server.

Oryx will implement most of features of [srs](https://github.com/ossrs/srs), 
which is industrial-strength live streaming cluster, for the best conceptual 
integrity and the simplest implementation. In another words, oryx is next-generation 
srs, the srs-ng.

## Architecture

The architecture of oryx is a group of isolate processes:

1. `shell` which exec other processes, the frontend of oryx.
1. `rtmplb` rtmp frontend of oryx, proxy to backend streaming workers.
1. `httplb` http flv/hls+ frontend of oryx, proxy to backend streaming workers.
1. `apilb` http api frontend of oryx, proxy to backend api.
1. `srs` the streaming worker, other stream worker is also ok.

```
                         +----------+
                    +----+  API LB  |
                    |    +----------+
                    |
                    |    +----------+
                    +----+  HTTP LB |
                    |    +----------+
                    |
+---------+         |    +----------+
|  Shell  +--exec->-+----+  RTMP LB |
+---------+         |    +----------+
                    |
                    |    +------------+
                    +----+ SRS Worker |
                         +------------+
```

Winlin 2016.07.09
