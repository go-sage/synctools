// Copyright © 2024 Timothy E. Peoples

// A NOTE TO DEVELOPERS
//
// Some unexported method names may begin with an underscore. This is
// a shorthand indicator which should be interpreted to mean:
//
// 		"I assume my receiver is currently locked."
//
// Please behave accordingly.

package waypoint

import "time"

// State is the type used to populate a Worker's State field.
type State string

const (
	// Waiting Workers are those that are ready to do work but are not yet
	// allowed to do so because their associated Waypoint is at or above its
	// set capacity.
	Waiting = State("Waiting")

	// Active Workers are those that are currently consuming capacity from
	// their associated Waypoint.
	Active = State("Active")

	// Finished Workers are those who have completed their work and are no
	// longer consuming capacity from their associated Waypoint.
	Finished = State("Finished")
)

type (
	// A Worker represents an individual unit of work (most often a goroutine)
	// issued by an associated Waypoint. Each Worker has a unique ID and exists
	// in one of three states: Waiting, Active, or Finished.
	//
	// Note that, since each Worker maintains a reference to the Waypoint
	// that created it, no Worker should be copied once it has been created.
	Worker struct {
		ID    uint64
		State State

		created  time.Time
		started  time.Time
		finished time.Time

		// An embedded reference to the creating Waypoint
		// (and its embedded RWMutex)
		*waypoint
	}

	// A type alias to hide an otherwise exported name
	// for the embedded Waypoint field.
	waypoint = Waypoint
)

// _next is called at the beginning of Waypoint's Wait method to create
// a Waiting worker with an ID unique to its receiver. Note that _next
// assumes that its receiver has already been locked.
func (w *Waypoint) _next() *Worker {
	w.idSeq++

	return &Worker{
		ID:       w.idSeq,
		State:    Waiting,
		created:  time.Now(),
		waypoint: w,
	}
}

// _start is called at the end of Waypoint's Wait method to transition its
// receiver into the Active state. Note that _start assumes that its receiver
// has already been locked.
func (w *Worker) _start() *Worker {
	now := time.Now()
	w.started = now
	w.waitTime += now.Sub(w.created)
	w.State = Active
	w.active[w.ID] = w
	return w
}

// Done is called to transition the receiver to the Finished state. If this
// drops the associated Waypoint below its set, non-zero, capacity -- and the
// Waypoint has not yet been closed -- a Worker from the associated Waypoint's
// pool of Waiting Workers will be moved to the Active state to begin work.
func (w *Worker) Done() {
	w.Lock()
	defer w.Unlock()

	w.State = Finished
	w.finished = time.Now()

	w.numFinished++
	w.activeTime += w.finished.Sub(w.started)

	// Note that calling cond.Signal() will likely trigger a call to the
	// above _start method (if there are Workers "Waiting" in the wings).
	w.cond.Signal()

	// n.b. This must be called *after* cond.Signal() to allow a closed
	//      Waypoint, with non-zero capacity, to continue activating
	//      Waiting Workers -- otherwise, removing the only Active Worker
	//      from a closed Waypoint would shut this whole thing down.
	w._removeWorker(w.ID)
}
