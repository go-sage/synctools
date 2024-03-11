// Copyright © 2024 Timothy E. Peoples

package pipeline

import (
	"context"
)

func Send[T any](ctx context.Context, value T, ch chan<- any) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case ch <- value:
		return nil
	}
}

func Recv[T any](ctx context.Context, ch <-chan any) (T, bool, error) {
	var (
		out T
		val any
		ok  bool
	)

	select {
	case <-ctx.Done():
		return out, false, ctx.Err()

	case val, ok = <-ch:
		break
	}

	if !ok {
		return out, false, nil
	}

	if out, ok = val.(T); ok {
		return out, true, nil
	}

	return out, true, nil
}
