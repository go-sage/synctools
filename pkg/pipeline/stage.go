// Copyright © 2024 Timothy E. Peoples

package pipeline

import (
	"context"

	"github.com/go-sage/synctools/pkg/errgroupx"
	"github.com/go-sage/synctools/pkg/waypoint"
)

type stage struct {
	name     string
	capacity int
	pfunc    PipelineFunc
	waypt    *waypoint.Waypoint
}

// runner is an errgroupx.GoFunc as expected by the GoContext method of
// (*github.com/go-sage/synctools/pkg/errgroupx).Group.
func (s *stage) runner(inch <-chan any, outch chan<- any) errgroupx.GoFunc {
	return func(ctx context.Context) error {
		defer close(outch)

		s.waypt = waypoint.New(s.capacity)
		eg, ctx, cancel := errgroupx.New(ctx)
		defer cancel()

		const errInputDone = errstr("no more input")

		runloop := func() error {
			for {
				in, ok, err := Recv[any](ctx, inch)
				if err != nil {
					return err
				} else if !ok {
					return errInputDone
				}

				w, err := s.waypt.Wait(ctx)
				if err != nil {
					return err
				}

				eg.Go(func() (err error) {
					defer w.Done()
					var out any

					if out, err = s.pfunc(ctx, in); err != nil {
						return err
					}

					return Send(ctx, out, outch)
				})
			}
		}

		if err := runloop(); err != nil && err != errInputDone {
			return err
		}

		return eg.Wait()
	}
}
