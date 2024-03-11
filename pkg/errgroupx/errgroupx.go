// Copyright © 2024 Timothy E. Peoples

// Package errgroupx provides an opinionated convenience wrapper around
// package golang.org/x/sync/errgroup that makes it easier to deal with
// Groups and Contexts in a consistent manner.
//
// A typical usecase would be:
//
//	// run executes the receiver's doTHIS and doTHAT methods
//	// concurrently until both succeed or one fails.
//	func (t *thing) run(ctx context.Context) error {
//		eg, ctx, cancel := errgroupx.New(ctx)
//		defer cancel()
//
//		for _, gf := range[]errgroupx.GoFunc{t.doTHIS, t.doTHAT} {
//			eg.GoContext(ctx, gf)
//		}
//
//		return eg.Wait()
//	}
//
//	func (t *thing) doTHIS(ctx context.Context) error {
//		//...do THIS with t and ctx returning an error if need be.
//	}
//
//	func (t *thing) doTHAT(ctx context.Context) error {
//		//...do THAT with t and ctx returning an error if need be.
//	}
//
// Note that if you do not use the added constructor function (`New`) or
// the Context-aware methods, this package is a drop-in replacement for
// golang.org/x/sync/errgroup.
package errgroupx

import (
	"context"

	"golang.org/x/sync/errgroup"
)

type (
	// Group is a convenience wrapper around an embedded Group from package
	// golang.org/x/sync/errgroup that adds two new methods: GoContext and
	// TryGoContext. All of the other methods on the embedded Group type are
	// available here as well.
	Group struct {
		*group
	}

	group = errgroup.Group
)

// New is a simple wrapper around context.WithCancel and errgroup.WithContext
// returning a new Group, a derived Context, and a CancelFunc. The derived
// Context is canceled the first time a function passed to GoContext returns
// a non-nil error or the first time Wait returns, whichever occurs first.
func New(ctx context.Context) (*Group, context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(ctx)
	group, ctx := errgroup.WithContext(ctx)
	return &Group{group}, ctx, cancel
}

// GoFunc is the function type passed to GoContext or TryGoContext.
type GoFunc func(context.Context) error

func (gf GoFunc) egFunc(ctx context.Context) func() error {
	return func() error { return gf(ctx) }
}

// GoContext is a wrapper around the Go method from golang.org/x/sync/errgroup
// that accepts an anonymous function with a Context parameter. The Context
// provided here is passed to the GoFunc unchanged.
func (g *Group) GoContext(ctx context.Context, f GoFunc) {
	g.group.Go(f.egFunc(ctx))
}

// TryGoContext is a wrapper around the similarly named TryGo method from
// golang.org/x/sync/errgroup that accepts an anonymous function with a
// Context parameter. The Context provided here is passed to the GoFunc
// unchanged.
func (g *Group) TryGoContext(ctx context.Context, f GoFunc) bool {
	return g.group.TryGo(f.egFunc(ctx))
}
