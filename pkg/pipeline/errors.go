// Copyright © 2024 Timothy E. Peoples

package pipeline

type errstr string

func (s errstr) Error() string {
	return string(s)
}

const (
	ErrCorrupted    = errstr("pipeline state is corrupted")
	ErrIsStarted    = errstr("pipeline is already started")
	ErrNameConflict = errstr("stage name conflict")
	ErrNameUnknown  = errstr("stage name not found")
	ErrNilReceiver  = errstr("nil receiver")
	ErrNoStages     = errstr("no pipeline stages registered")
)
