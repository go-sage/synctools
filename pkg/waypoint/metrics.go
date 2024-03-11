// Copyright © 2024 Timothy E. Peoples

package waypoint

import (
	"time"
)

// Metrics represents point-in-time metrics for a Waypoint.
type Metrics struct {
	Timestamp  time.Time     // Time these metrics were gathered
	Capacity   int           // Waypoint's current capacity
	Waiting    int           // Current number of waiting Workers
	Active     int           // Current number of active Workers
	Finished   int           // Current number of finished Workers
	WaitTime   time.Duration // Total accumulated Wait time
	ActiveTime time.Duration // Total accumulated Active time
}

// Metrics returns a point-in-time Metrics value for the receiver.
func (w *Waypoint) Metrics() Metrics {
	if w == nil {
		return Metrics{}
	}

	w.RLock()
	defer w.RUnlock()

	return Metrics{
		Timestamp:  time.Now(),
		Capacity:   w.capacity,
		Waiting:    w.numWaiting,
		Active:     len(w.active),
		Finished:   w.numFinished,
		WaitTime:   w.waitTime,
		ActiveTime: w.activeTime,
	}
}
