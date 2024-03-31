// Copyright © 2024 Timothy E. Peoples

package pipeline

import (
	"context"

	"github.com/go-sage/synctools/pkg/errgroupx"
)

// Run executes the Pipeline defined for the receiver as at least three
// separate goroutines: one each for the Feed and Collect methods implemented
// by the Interface provided to the constructor plus one or more additional
// goroutines for the registered stages. Channels are interleaved between the
// Feed stage, each of the individually registered stages (in the order each
// was added), and the final Collect stage.
//
// Run blocks until all of its goroutines have completed -- either successfully
// or until any one of them returns a non-nil error. If the provided context is
// canceled that cancelation will be propagated to all running goroutines (note
// that an err returned by a goroutine will cancel the context provided to all
// of the others).
//
// If the receiver has no stages registered then ErrNoStages is returned.
// Otherwise, any error returned will be one returned from one of the
// underlying goroutines.
func (p *Pipeline) Run(ctx context.Context) error {
	if p == nil {
		return ErrNilReceiver
	}

	eg, err := p.run(ctx)
	if err != nil {
		return err
	}
	return eg.Wait()
}

// run exists as a separate method so we can Lock the receiver, set things
// up, Unlock the reciever, then return the *errgroupx.Group so that Run can
// call its Wait method without holding the receiver's lock for way too long.
func (p *Pipeline) run(ctx context.Context) (*errgroupx.Group, error) {
	p.Lock()
	defer p.Unlock()

	if len(p.stages) == 0 {
		p.Unlock()
		return nil, ErrNoStages
	}

	p.started = true

	eg, ctx, cancel := errgroupx.New(ctx)
	defer cancel()

	for _, cf := range p.funcs {
		eg.GoContext(ctx, cf)
	}

	inch := make(chan any)
	eg.GoContext(ctx, p.feedFunc(inch))

	prev := inch
	var last chan any

	for _, s := range p.stages {
		ch := make(chan any)
		eg.GoContext(ctx, s.runner(prev, ch))
		prev = ch
		last = ch
	}

	eg.GoContext(ctx, p.collectFunc(last))

	return eg, nil
}

// feedFunc returns an errgroupx.ContextFunc that executes the receiver's
// Interface.Feed method in order to send data to the given channel.
func (p *Pipeline) feedFunc(ch chan<- any) errgroupx.ContextFunc {
	return func(ctx context.Context) error {
		defer close(ch)
		return p.impl.Feed(ctx, ch)
	}
}

// collectFunc returns an errgroupx.ContextFunc that executes the receiver's
// Interface.Collect method in order to receive data from the given channel.
func (p *Pipeline) collectFunc(ch <-chan any) errgroupx.ContextFunc {
	return func(ctx context.Context) error {
		return p.impl.Collect(ctx, ch)
	}
}
