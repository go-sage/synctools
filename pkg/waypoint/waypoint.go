// Copyright © 2024 Timothy E. Peoples
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to
// deal in the Software without restriction, including without limitation the
// rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
// sell copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS
// IN THE SOFTWARE.

//╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴╶╴

// Package waypoint implements an opinionated scheme for coordinating limited
// concurrency over a variably-sized set of cooperating callers. This is
// accomplished primarily through the provided types Worker and Waypoint.
//
// A Worker represents a cooperating unit of work (most often a goroutine)
// intended for concurrent execution along with other Workers. The lifecycle
// for each Worker transitions through three states: Waiting, Active, and
// Finished (albeit, Waiting workers are never visible to the caller).
//
// A Waypoint acts as a coordination point over a group of Workers by ensuring
// only a set number may be in the Active state at any given time. The number
// of Workers that a Waypoint allows in the Active state is referred to as its
// capacity. Workers wishing to become Active while a Waypoint is at (or above)
// its set capacity are blocked until one or more currently Active Workers are
// transitioned to the Finished state.
//
// As an analogy, you could think of a Waypoint as similar to a doorman at a
// busy nightclub whose job it is to prevent the club from becoming too crowded
// (and possibly unsafe) by ensuring there are never too many people in the
// building at the same time.
//
// Here's an (obscenely contrived) example showing how to use a Waypoint:
//
//	func ProcessStuff(ctx context.Context, inch <-chan *Stuff, outch chan<- *Stuff) error {
//	  defer close(outch)
//
//	  // Create a Waypoint with capacity for 5 concurrent Workers.
//	  wp := waypoint.New(5)
//
//	  for stuff := range inch {
//	    // As each "stuff" arrives from "inch" we create a new Worker by
//	    // calling the Waypoint's Wait method. This call will immediately
//	    // return an Active Worker until we reach the Waypoint's capacity
//	    // of 5. Subsequent calls will block until one or more of the
//	    // Active Workers is moved to the Finished state thus exposing
//	    // more capacity.
//	    a, err := wp.Wait(ctx)
//	    if err != nil {
//	      return err
//	    }
//
//	    // Next, we'll create a new goroutine to process our stuff and then
//	    // send it to outch. On return, this function calls a.Done() to move
//	    // this Worker into the Finished state which will unblock one of the
//	    // above blocked calls to Wait (that should be in a separate goroutine,
//	    // right?)
//	    //
//	    // When used in this way, there will never be more than 5 of these
//	    // goroutines at any given time (unless, of course, you alter the
//	    // Waypoint's capacity).
//	    go func(stuff *Stuff) {
//	      defer a.Done()
//	      //
//	      // Do something with the stuff, then send it to 'outch'.
//	      //
//	      outch <- stuff
//	    }(stuff)
//	  }
//
//	  return nil
//	}
//
// For an even better use case, see:
//
//	"github.com/go-sage/synctools/pkg/pipeline"
//
// A Waypoint's capacity is not a set value; it may be adjusted over time
// to allow for dynamic scalling. When capacity is reduced, Active Workers
// are unaffected but as each transitions to the Finished state, Waiting
// Workers will remain blocked until the number of Active Workers drops
// below the new capacity value.  Conversely, if capacity is increased,
// enough Waiting Workers will immediately unblock in order to consume
// the newly available capacity.
//
// Note that a Waypoint is not a queue; the order in which Workers become
// unblocked is unrelated to the order they're created.
package waypoint

import (
	"context"
	"sync"
	"time"
)

type (
	// A Waypoint is a coordination point that ensure only a set number of
	// Workers are allowed to do work concurrently.
	Waypoint struct {
		idSeq       uint64
		capacity    int
		numWaiting  int
		numFinished int
		active      map[uint64]*Worker
		cond        *sync.Cond

		closed bool
		done   chan struct{}
		once   sync.Once

		waitTime   time.Duration
		activeTime time.Duration

		rwMutex
	}

	// A type alias to hide an otherwise exported name
	// for the embedded RWMutex field.
	rwMutex = sync.RWMutex
)

// New returns a new Waypoint initialized to the provided capacity.
func New(capacity int) *Waypoint {
	w := &Waypoint{
		capacity: capacity,
		active:   make(map[uint64]*Worker),
		done:     make(chan struct{}),
	}

	w.cond = sync.NewCond(w)

	return w
}

// Wait returns an Active *Worker ready to do some work.  If the receiver
// has available capacity, Wait returns immediately, otherwise it blocks
// until capacity is made available. If the provided context is canceled
// or times out while waiting, a nil *Worker is returned along with the
// error value returned by ctx.Err().
func (w *Waypoint) Wait(ctx context.Context) (*Worker, error) {
	done := make(chan struct{})
	defer close(done)

	// n.b. Since sync.Cond.Wait does not accept a Context, we'll need
	// this extra goroutine to watch for context cancelation. If/when
	// the provided context is canceled, we'll send a broadcast to the
	// receiver's 'cond' thus waking up all goroutines blocked on this
	// method. If this method returns before the context gets canceled,
	// then the closing of the local 'done' channel will allow this
	// func to return as well.
	go func() {
		select {
		case <-ctx.Done():
			w.cond.Broadcast()
		case <-done:
			return
		}
	}()

	w.Lock()
	defer w.Unlock()

	w.numWaiting++
	defer func() { w.numWaiting-- }()

	a := w._next()

	for len(w.active) >= w.capacity {
		w.cond.Wait()

		// Before we turn around and recheck the above condition (since
		// that's what the docs for cond.Wait() tell us to do), we'll need
		// to check whether we were awoken by the broadcast in the above
		// anonymous goroutine.
		//
		// If so, we'll return ctx.Err() -- otherwise, we can check our
		// condition and act accordingly.
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			continue
		}
	}

	return a._start(), nil
}

// Resize sets the receiver's capacity to newcap returning the previous
// capacity value. A value of -1 is returned if a) the receiver is nil,
// b) newcap is less than zero, or c) the receiver has been closed.
//
// This method may be called at any time on a non-closed Wayopint having
// any number of Workers in any State.
//
// If the receiver's capacity is increased by this call, the reciever will
// activate Waiting workers allowing this new capacity to be consumed. If
// capacity is reduced, Worker completion will not start new workers until
// the number of Active Workers drops below the new capacity level.
//
// Note that setting capacity to zero will allow currently Active Workers
// to complete but will not allow Waiting workers to become Active until
// such time that capacity is increased. Also, since this method cannot
// be called on a closed Waypoint, setting capacity to zero then closing
// the Waypoint will abandon all Waiting Workers.
func (w *Waypoint) Resize(newcap int) int {
	if w == nil || newcap < 0 || w.closed {
		return -1
	}

	oldcap := w.capacity
	w.capacity = newcap

	if newcap > oldcap {
		// We have more capacity!!
		// Let's tell everyone!
		w.cond.Broadcast()
	}

	return oldcap
}

// Done marks the receiver as closed thus denying any new Workers to be
// added through its Wait method. Currently Active Workers are allowed
// to continue and currently Waiting Workers will become Active when/if
// capacity becomes available.  However, since capacity cannot be altered
// on a closed Waypoint, if Done is called with zero capacity, all Waiting
// Workers will be abandoned and will never become Active.
//
// The returned channel will be closed once all actionable Workers have
// reached the Finished state.
func (w *Waypoint) Done() <-chan struct{} {
	w.Lock()
	defer w.Unlock()

	w.closed = true
	w._stop()

	return w.done
}

func (w *Waypoint) _removeWorker(id uint64) {
	if _, ok := w.active[id]; ok {
		delete(w.active, id)
	}

	if len(w.active) == 0 {
		w._stop()
	}
}

func (w *Waypoint) _stop() {
	if w.closed {
		w.once.Do(func() {
			close(w.done)
		})
	}
}
