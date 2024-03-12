// Copyright © 2024 Timothy E. Peoples

// Package pipeline provides logic for processing a pipeline of data elements
// using a coordinated concurrency model. A Pipeline is made up of one or more
// stages each executing a finite (but resizable) set of concurrent goroutines
// that are coordinated using this module's [waypoint] package.
package pipeline

import (
	"context"
	"sync"
)

type (
	Pipeline struct {
		impl    Interface
		stages  []stage
		byname  map[string]int
		started bool

		mutex
	}

	mutex = sync.Mutex
)

// Interface should be implemented by types written to provide the data source
// and sink for a given Pipeline.
type Interface interface {
	// Feed provides the data source for a Pipeline by sending data elements
	// into the provided channel. It is the responsibility of the implementor
	// to close this channel once all data has been provided, otherwise the
	// Pipeline will never end.
	Feed(ctx context.Context, wchan chan<- any) error

	// A Collect method acts as the data sink for the Pipeline by receiving
	// data elements from the provided channel. The Pipeline will close this
	// channel when no more data is forthcoming.
	Collect(ctx context.Context, rchan <-chan any) error
}

// New creates and returns a new Pipeline using the provided Interface.
func New(impl Interface) *Pipeline {
	return &Pipeline{
		impl:   impl,
		byname: make(map[string]int),
	}
}

// A StageFunc is the function called to process each piece of data
// for a stage registered using the (*Pipeline).Add method.
type StageFunc func(ctx context.Context, input any) (any, error)

// Add registers a named Pipeline stage that will execute the provided
// StageFunc using an initial [waypoint] capacity.  The given name must be
// unique among all stages for this Pipeline. Add may be called multiple
// times, to register multiple stages, and data will flow through each stage
// of the Pipeline in the order they are registered. Note however that the
// receiver's Run method will fail if no stages have been registered.
//
// Once the receiver has been started (by calling its Run method) no more
// stages may be registered. Add returns ErrIsStarted if it is called after
// Run.  ErrNameConflict is returned if Add is called using a previously
// registered name.  Otherwise, the new stage is registered and a nil error
// is returned.
//
// The name parameter may be used with the Resize method in order to alter
// the capacity of this particular stage. For more details, see this
// module's [waypoint] package.
func (p *Pipeline) Add(name string, capacity int, pfunc StageFunc) error {
	if p == nil {
		return ErrNilReceiver
	}

	p.Lock()
	defer p.Unlock()

	if p.started {
		return ErrIsStarted
	}

	if _, ok := p.byname[name]; ok {
		return ErrNameConflict
	}

	idx := len(p.stages)
	p.stages = append(p.stages, stage{
		name:     name,
		capacity: capacity,
		sfunc:    pfunc,
	})

	p.byname[name] = idx

	return nil
}

// Resize updates the capacity of the pipeline stage with the given name to the
// provided newcap value and returns that stage's previous capacity value.  If
// name is not a registered stage name then zero and ErrNameUnknown will be
// returned.
func (p *Pipeline) Resize(name string, newcap int) (int, error) {
	if p == nil {
		return 0, ErrNilReceiver
	}

	p.Lock()
	defer p.Unlock()

	ndx, ok := p.byname[name]
	if !ok {
		return 0, ErrNameUnknown
	}

	if ndx < 0 || ndx >= len(p.stages) {
		return 0, ErrCorrupted
	}

	return p.stages[ndx].waypt.Resize(newcap), nil

}

// [waypoint]: https://pkg.go.dev/github.com/go-sage/synctools/pkg/waypoint
