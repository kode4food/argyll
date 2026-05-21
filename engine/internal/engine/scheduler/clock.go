package scheduler

import (
	"sync"
	"time"
)

type (
	// Clock provides the current time for scheduling and retries
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

var Now Clock = func() Clock {
	var mu sync.Mutex
	var last time.Time

	return func() time.Time {
		now := time.Now()

		mu.Lock()
		if !now.After(last) {
			now = last.Add(time.Nanosecond)
		}
		last = now
		mu.Unlock()

		return now
	}
}()

// NewTimer builds the default system-backed scheduler timer
func NewTimer(delay time.Duration) Timer {
	return &systemTimer{
		Timer: time.NewTimer(delay),
	}
}

func (t *systemTimer) Channel() <-chan time.Time {
	return t.C
}
