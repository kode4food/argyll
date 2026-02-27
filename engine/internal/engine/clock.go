package engine

import "time"

type (
	// Clock provides the current time for engine scheduling and retries
	Clock func() time.Time

	// Timer represents a resettable scheduler timer
	Timer interface {
		Channel() <-chan time.Time
		Reset(delay time.Duration) bool
		Stop() bool
	}

	// TimerConstructor builds a scheduler timer with the given delay
	TimerConstructor func(delay time.Duration) Timer

	systemTimer struct {
		*time.Timer
	}
)

// Now returns the current wall time from Engine's configured clock
func (e *Engine) Now() time.Time {
	return e.clock()
}

// NewTimer builds the default system-backed scheduler timer
func NewTimer(delay time.Duration) Timer {
	return &systemTimer{
		Timer: time.NewTimer(delay),
	}
}

func (t *systemTimer) Channel() <-chan time.Time {
	return t.C
}
