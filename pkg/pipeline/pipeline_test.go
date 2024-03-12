// Copyright © 2024 Timothy E. Peoples

package pipeline

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func TestPipeline(t *testing.T) {
	ctx := context.Background()
	tt := mkTestThing(25, t)
	p := New(tt)

	p.Add("stage1", 50, tt.stage1)

	if err := p.Run(ctx); err != nil {
		t.Fatal(err)
	}

	var smin, smax, emin, emax time.Time

	for i, r := range tt.output {
		if got, want := r.value, i+1; got != want {
			t.Errorf("record %d beget %d; wanted %d", i, got, want)
		}

		if smin.IsZero() || r.start.Before(smin) {
			smin = r.start
		}

		if emin.IsZero() || r.stop.Before(emin) {
			emin = r.stop
		}

		if smax.IsZero() || r.start.After(smax) {
			smax = r.start
		}

		if emax.IsZero() || r.stop.After(emax) {
			emax = r.stop
		}
	}

	want := tt.tolerance(12.5)

	if srange := smax.Sub(smin); srange > want {
		t.Errorf("Start range too wide: got %v; wanted %v", srange, want)
	} else {
		t.Logf("All workers began between %s and %s: %v", smin.Format(tf), smax.Format(tf), srange)
	}

	if erange := emax.Sub(emin); erange > want {
		t.Errorf("End range too wide: got %v; wanted %v", erange, want)
	} else {
		t.Logf("All workers ended between %s and %s: %v", emin.Format(tf), emax.Format(tf), erange)
	}
}

//╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴

type testThing struct {
	delay  time.Duration
	input  []int
	output map[int]record
	logf   func(format string, args ...any)
}

func mkTestThing(millis int, t *testing.T) *testThing {
	dm := make(map[int]bool)

	var ints []int
	for len(ints) != 50 {
		i := 10 + rand.Intn(89)
		if !dm[i] {
			ints = append(ints, i)
			dm[i] = true
		}
	}

	return &testThing{
		delay:  time.Duration(millis) * time.Millisecond,
		input:  ints,
		output: make(map[int]record),
		logf:   t.Logf,
	}
}

func (tt *testThing) tolerance(pct float64) time.Duration {
	return time.Duration(float64(tt.delay) * (pct / 100))
}

// · · · · · · · · · · · · · · · · · · · · · · · · · · · · · · · · · · · · · · ·

func (tt *testThing) Feed(ctx context.Context, ch chan<- any) error {
	defer close(ch)

	for _, v := range tt.input {
		if err := Send[int](ctx, v, ch); err != nil {
			return err
		}
	}

	return nil
}

// · · · · · · · · · · · · · · · · · · · · · · · · · · · · · · · · · · · · · · ·

func (tt *testThing) Collect(ctx context.Context, ch <-chan any) error {
	// var min, max time.Time

	for {
		r, ok, err := Recv[record](ctx, ch)
		switch {
		case err != nil:
			return err
		case !ok:
			// tt.logf("min=%q max=%q diff=%v", min.Format(tf), max.Format(tf), max.Sub(min).Round(time.Microsecond))
			return nil
		}

		// tt.logf("Collected: %v", r)

		tt.output[r.orig] = r

		// if min.IsZero() || r.start.Before(min) {
		// 	min = r.start
		// }

		// if max.IsZero() || r.stop.After(max) {
		// 	max = r.stop
		// }
	}
}

// · · · · · · · · · · · · · · · · · · · · · · · · · · · · · · · · · · · · · · ·

func (tt *testThing) stage1(ctx context.Context, input any) (any, error) {
	r := record{start: time.Now(), orig: input.(int)}
	r.value = r.orig + 1
	time.Sleep(tt.delay)
	r.stop = time.Now()
	return r, nil
}

//╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴

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
