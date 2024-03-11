// Copyright © 2024 Timothy E. Peoples

package waypoint

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestFoo(t *testing.T) {
	var (
		inch  = make(chan rune)
		outch = make(chan rune)
		ctx   = context.Background()
		wg    sync.WaitGroup
	)

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(inch)
		for c := 'A'; c <= 'Z'; c++ {
			t.Logf("In %q", c)
			select {
			case <-ctx.Done():
				t.Logf("#1 context: %v", ctx.Err())
				return
			case inch <- c:
				// ok
			}
		}
		t.Logf("#1 done")
	}()

	go func() {
		wp := New(5)

		var (
			c  rune
			ok bool
		)

		for {
			select {
			case c, ok = <-inch:
				if !ok {
					t.Logf("#2 done")
					return
				}

			case <-ctx.Done():
				t.Logf("#2 context: %v", ctx.Err())
				return
			}

			t.Logf("Received %q", c)

			wg.Add(1)
			go func(c rune) {
				defer wg.Done()

				a, err := wp.Wait(ctx)
				if err != nil {
					t.Logf("waiting: %v", err)
					return
				}

				t.Logf("Ready %q", c)

				defer a.Done()

				m := wp.Metrics()
				t.Logf("%s: %d/%d/%d", m.Timestamp.Format("15:04:05.000000"), m.Waiting, m.Active, m.Finished)

				time.Sleep(100 * time.Millisecond)

				select {
				case outch <- c:
					// ok
				case <-ctx.Done():
					t.Logf("#3 context: %v", ctx.Err())
					return
				}
			}(c)
		}
	}()

	go func() {
		wg.Wait()
		close(outch)
	}()

	go func() {
		for {
			c, ok := <-outch
			if !ok {
				return
			}

			t.Logf("Out %q", c)
		}
	}()

	wg.Wait()
}
