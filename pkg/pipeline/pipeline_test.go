// Copyright © 2024 Timothy E. Peoples

package pipeline

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestFoo(t *testing.T) {
	start := time.Now()
	ctx := context.Background()
	tt := &testThing{t.Logf}
	p := New(tt)

	p.Add("plus1", 50, tt.stage1)

	if err := p.Run(ctx); err != nil {
		t.Fatal(err)
	}
	t.Logf("Total time: %v", time.Since(start).Round(time.Microsecond))
}

type testThing struct {
	logf func(format string, args ...any)
}

func (tt *testThing) Feed(ctx context.Context, ch chan<- any) error {
	defer close(ch)

	values := []int{
		96, 30, 51, 97, 30, 33, 72, 79, 32, 22,
		80, 88, 35, 46, 55, 93, 95, 64, 40, 10,
		37, 47, 55, 45, 42, 18, 16, 49, 33, 48,
		95, 30, 68, 31, 28, 73, 54, 74, 53, 60,
		71, 18, 49, 96, 54, 94, 74, 35, 44, 88,
	}

	for _, v := range values {
		if err := Send[int](ctx, v, ch); err != nil {
			return err
		}
	}

	return nil
}

func (tt *testThing) Collect(ctx context.Context, ch <-chan any) error {
	var min, max time.Time

	for {
		r, ok, err := Recv[record](ctx, ch)
		switch {
		case err != nil:
			return err
		case !ok:
			tt.logf("min=%q max=%q diff=%v", min.Format(tf), max.Format(tf), max.Sub(min).Round(time.Microsecond))
			return nil
		}

		// r := v.(record)
		tt.logf("%v", r)

		if min.IsZero() || r.start.Before(min) {
			min = r.start
		}

		if max.IsZero() || r.stop.After(max) {
			max = r.stop
		}
	}
}

func (tt *testThing) stage1(ctx context.Context, input any) (any, error) {
	r := record{start: time.Now(), orig: input.(int)}
	r.value = r.orig + 1
	time.Sleep(20 * time.Millisecond)
	r.stop = time.Now()
	return r, nil
}

type record struct {
	start time.Time
	stop  time.Time
	orig  int
	value int
}

const tf = "15:04:05.000000"

func (r record) String() string {
	return fmt.Sprintf("%s %3d - %-3d %v", r.start.Format(tf), r.orig, r.value, r.stop.Sub(r.start).Round(time.Microsecond))
}
