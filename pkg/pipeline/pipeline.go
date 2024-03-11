// Copyright © 2024 Timothy E. Peoples

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

// Interface should be implemented by types intended for integration with this
// package by feeding values to and collecting values from a Pipeline.
type Interface interface {
	Feed(ctx context.Context, wchan chan<- any) error
	Collect(ctx context.Context, rchan <-chan any) error
}

// New creates and returns a new Pipeline for the given Interface.
func New(impl Interface) *Pipeline {
	return &Pipeline{
		impl:   impl,
		byname: make(map[string]int),
	}
}

// A PipelineFunc is a function that processes an individual piece of
// data by accepting in input value and returning an output value. These
// functions are used when registering a pipeline stage using Add method
// on type Pipeline.
type PipelineFunc func(ctx context.Context, input any) (any, error)

// Add registers a single pipeline stage with the receiver having the
// given name (which must be unique amongst all stages for the Pipline),
// to use at most capacity goroutines executing PipelineFunc. Multiple
// stages may be registered with a Pipeline, each having a different
// function and/or capacity, but the Pipeline will fail without at least
// one stage.
//
// If the receiver has already been started (by calling its Run method)
// Add will return ErrIsStarted. If a stage has already been registered
// using the given name, ErrNameConflict is returned. Otherwise, the
// new stage is registered and a nil error is returned.
//
// The name parameter may be used with the Resize method in order to alter
// the capacity of this particular state. See the waypoint package for
// more details.
//
// [waypoint package](github.com/go-sage/synctools/pkg/waypoint)
func (p *Pipeline) Add(name string, capacity int, pfunc PipelineFunc) error {
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
		pfunc:    pfunc,
	})

	p.byname[name] = idx

	return nil
}

// Resize updates the capacity of the pipeline stage having the given name to
// the provided newcap value and returns that stage's previous capacity value.
// If name is not a known Pipeline stage name then zero and ErrNameUnknown
// will be returned.
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
