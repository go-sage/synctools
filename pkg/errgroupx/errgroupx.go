// Copyright © 2024 Timothy E. Peoples

// Package errgroupx provides an opinionated convenience wrapper around
// package golang.org/x/sync/errgroup that provides a consistent manner
// of dealing with Contexts and (error) Groups.
//
// A typical usecase might be:
//
//	// The run methods executes its receiver's doTHIS and doTHAT methods
//	// concurrently until either both succeed or one of them fails.
//	func (t *thing) run(ctx context.Context) error {
//		eg, ctx, cancel := errgroupx.WithCancel(ctx)
//		defer cancel()
//
//		eg.GoContext(ctx, t.doTHIS)
//		eg.GoContext(ctx, t.doTHAT)
//
//		return eg.Wait()
//	}
//
//	func (t *thing) doTHIS(ctx context.Context) error {
//		//...do THIS with t and ctx returning an error as appropriate.
//	}
//
//	func (t *thing) doTHAT(ctx context.Context) error {
//		//...do THAT with t and ctx returning an error as appropriate.
//	}
//
// Note that if you make no use of the added constructor functions
// ('WithCancel', 'WithDeadline' or 'WithTimeout') or the Context aware
// methods ('GoContext' or 'TryGoContext'), then this package can be
// used as a drop-in replacement for golang.org/x/sync/errgroup.
package errgroupx

import (
	"context"
	"time"

	"golang.org/x/sync/errgroup"
)

type (
	// Group is a convenience wrapper around an embedded Group from package
	// golang.org/x/sync/errgroup which adds two new methods: GoContext and
	// TryGoContext. All of the other methods on the embedded Group type are
	// available here as well.
	Group struct {
		*group
	}

	group = errgroup.Group
)

// New is equivalent to WithCancel.
//
// Deprecated: use WithCancel instead.
func New(ctx context.Context) (*Group, context.Context, context.CancelFunc) {
	return newGroup(context.WithCancel(ctx))
}

// WithCancel is a wrapper around errgroup.WithContext and context.WithCancel
// returning a new Group, a derived Context, and a CancelFunc. The derived
// Context is canceled the first time a function passed to GoContext (or
// similar) returns a non-nil error, or the first time Wait returns, whichever
// occurs first.
//
// See package 'context' about what to do with the CancelFunc.
func WithCancel(ctx context.Context) (*Group, context.Context, context.CancelFunc) {
	return newGroup(context.WithCancel(ctx))
}

// WithDeadline is a similar to WithCancel but wraps context.WithDeadline
// instead of context.WithCancel.
func WithDeadline(ctx context.Context, d time.Time) (*Group, context.Context, context.CancelFunc) {
	return newGroup(context.WithDeadline(ctx, d))
}

// WithTimeout is a similar to WithCancel but wraps context.WithTimeout
// instead of context.WithCancel.
func WithTimeout(ctx context.Context, timeout time.Duration) (*Group, context.Context, context.CancelFunc) {
	return newGroup(context.WithTimeout(ctx, timeout))
}

// newGroup provides common logic for the constructor functions WithCancel,
// WithDeadline, and WithTimeout.
func newGroup(ctx context.Context, cancel context.CancelFunc) (*Group, context.Context, context.CancelFunc) {
	group, ctx := errgroup.WithContext(ctx)
	return &Group{group}, ctx, cancel
}

// ContextFunc is the function type passed to GoContext or TryGoContext.
type ContextFunc func(context.Context) error

func (cf ContextFunc) do(ctx context.Context) func() error {
	return func() error { return cf(ctx) }
}

// GoContext is a wrapper around the (*Group).Go method from package
// golang.org/x/sync/errgroup that accepts an anonymous function with
// a Context parameter. The Context provided here is passed to the
// ContextFunc unchanged.
func (g *Group) GoContext(ctx context.Context, cfunc ContextFunc) {
	g.group.Go(cfunc.do(ctx))
}

// TryGoContext is a wrapper around the (*Group).TryGo method from package
// golang.org/x/sync/errgroup that accepts an anonymous function with a
// Context parameter. The Context provided here is passed to the ContextFunc
// unchanged.
func (g *Group) TryGoContext(ctx context.Context, cfunc ContextFunc) bool {
	return g.group.TryGo(cfunc.do(ctx))
}
