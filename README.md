# go-oryx

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
1. `httplb` http flv/hls+ frontend of oryx, prox to backend streaming workers.
1. `srs` the streaming worker, other stream worker is also ok.

```
                         +----------+
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
