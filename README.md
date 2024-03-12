
# [![Mit License][mit-img]][mit] [![GitHub Release][release-img]][release]

``` go
  import "github.com/go-sage/synctools"
```

This module provides a set of packages implementing an opinionated but
well-oiled mechanism for cleanly and efficiently managing high concurrency
workloads with Go.

## Provided Packages

### `errgroupx`

[![GoDoc][errgroupx-godoc-img]][errgroupx-godoc]

Package errgroupx provides an opinionated convenience wrapper around package
[`golang.org/x/sync/errgroup`](https://pkg.go.dev/golang.org/x/sync/errgroup)
that makes it easier to deal with Groups and Contexts in a consistent manner.


### `waypoint`

[![GoDoc][waypoint-godoc-img]][waypoint-godoc]

Package waypoint implements an opinionated scheme for coordinating limited
concurrency over a variably-sized set of cooperating callers. This is
accomplished primarily through the provided types Worker and Waypoint.

A Worker represents a cooperating unit of work (most often a goroutine)
intended for concurrent execution along with other Workers. The lifecycle
for each Worker transitions through three states: Waiting, Active, and
Finished (albeit, Waiting workers are never visible to the caller).

A Waypoint acts as a coordination point over a group of Workers by ensuring
only a set number may be in the Active state at any given time. The number
of Workers that a Waypoint allows in the Active state is referred to as its
capacity. Workers wishing to become Active while a Waypoint is at (or above)
its set capacity are blocked until one or more currently Active Workers are
transitioned to the Finished state.

### `pipeline`

[![GoDoc][pipeline-godoc-img]][pipeline-godoc]

Package pipeline provides logic for processing a pipeline of data elements
using a coordinated concurrency model. A Pipeline is made up of one or more
stages each executing a finite (but resizable) set of concurrent goroutines
that are coordinated using this module's waypoint package.

[mit-img]: http://img.shields.io/badge/License-MIT-c41e3a.svg
[mit]: https://github.com/go-sage/synctools/blob/main/LICENSE

[release-img]: https://img.shields.io/github/release/go-sage/synctools/all.svg
[release]: https://github.com/go-sage/synctools/releases

[reportcard-img]: https://goreportcard.com/badge/github.com/go-sage/synctools
[reportcard]: https://goreportcard.com/report/github.com/go-sage/synctools

[errgroupx-godoc-img]: https://godoc.org/github.com/go-sage/synctools/pkg/errgroupx?status.svg
[errgroupx-godoc]: https://godoc.org/github.com/go-sage/synctools/pkg/errgroupx

[waypoint-godoc-img]: https://godoc.org/github.com/go-sage/synctools/pkg/waypoint?status.svg
[waypoint-godoc]: https://godoc.org/github.com/go-sage/synctools/pkg/waypoint

[pipeline-godoc-img]: https://godoc.org/github.com/go-sage/synctools/pkg/pipeline?status.svg
[pipeline-godoc]: https://godoc.org/github.com/go-sage/synctools/pkg/pipeline
