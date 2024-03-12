
# github.com/go-sage/synctools

The `synctools` module provides a set of packages implementing an opinionated
but well-oiled mechanism for cleanly managing high concurrency workloads with
Go.

## Provided Packages

### `errgroupx`

Package errgroupx provides an opinionated convenience wrapper around package
`golang.org/x/sync/errgroup` that makes it easier to deal with Groups and
Contexts in a consistent manner.


### `waypoint`

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

Package pipeline provides logic for processing a pipeline of data elements
using a coordinated concurrency model. A Pipeline is made up of one or more
stages each executing a finite (but resizable) set of concurrent goroutines
that are coordinated using this module's [waypoint] package.

