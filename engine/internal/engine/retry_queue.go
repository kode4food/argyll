package engine

import (
	"sync"
	"time"

	"github.com/kode4food/argyll/engine/pkg/api"
)

type (

	// RetryQueue is a thread-safe queue for scheduled retries
	RetryQueue struct {
		mu      sync.Mutex
		items   map[retryKey]*RetryItem
		next    *RetryItem
		notify  chan struct{}
		stopped bool
	}

	// RetryItem represents a scheduled retry
	RetryItem struct {
		FlowID      api.FlowID
		StepID      api.StepID
		Token       api.Token
		NextRetryAt time.Time
	}

	retryTimer struct {
		timer *time.Timer
	}

	retryKey struct {
		FlowID api.FlowID
		StepID api.StepID
		Token  api.Token
	}
)

// NewRetryQueue creates a new retry queue
func NewRetryQueue() *RetryQueue {
	return &RetryQueue{
		items:  make(map[retryKey]*RetryItem),
		notify: make(chan struct{}, 1),
	}
}

// Push adds or updates a retry item and reports if the next deadline changed
func (q *RetryQueue) Push(item *RetryItem) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.stopped {
		return false
	}

	key := retryKey{
		FlowID: item.FlowID,
		StepID: item.StepID,
		Token:  item.Token,
	}
	prevNext := q.next
	prevTime := time.Time{}
	if prevNext != nil {
		prevTime = prevNext.NextRetryAt
	}
	q.items[key] = item
	q.recalcNext()
	if q.next == nil {
		return false
	}
	if prevNext == q.next && q.next.NextRetryAt.Equal(prevTime) {
		return false
	}
	q.signal()
	return true
}

// Remove removes a retry item from the queue
func (q *RetryQueue) Remove(
	flowID api.FlowID, stepID api.StepID, token api.Token,
) {
	q.mu.Lock()
	defer q.mu.Unlock()

	key := retryKey{
		FlowID: flowID,
		StepID: stepID,
		Token:  token,
	}
	item := q.items[key]

	delete(q.items, key)
	if q.next == item {
		q.recalcNext()
	}
}

// RemoveFlow removes all retry items for a flow
func (q *RetryQueue) RemoveFlow(flowID api.FlowID) {
	q.mu.Lock()
	defer q.mu.Unlock()

	needsRecalc := false
	for key, item := range q.items {
		if key.FlowID == flowID {
			delete(q.items, key)
			if q.next == item {
				needsRecalc = true
			}
		}
	}

	if needsRecalc {
		q.recalcNext()
	}
}

// Peek returns the earliest retry time
func (q *RetryQueue) Peek() (time.Time, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.next == nil {
		return time.Time{}, false
	}
	return q.next.NextRetryAt, true
}

// PopReady removes and returns all items whose retry time has passed
func (q *RetryQueue) PopReady(now time.Time) []*RetryItem {
	q.mu.Lock()
	defer q.mu.Unlock()

	var ready []*RetryItem
	for key, item := range q.items {
		if !item.NextRetryAt.After(now) {
			ready = append(ready, item)
			delete(q.items, key)
		}
	}

	if len(ready) > 0 {
		q.recalcNext()
	}
	return ready
}

// Notify returns the channel that signals queue changes
func (q *RetryQueue) Notify() <-chan struct{} {
	return q.notify
}

// Stop stops the queue and prevents further pushes
func (q *RetryQueue) Stop() {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.stopped {
		return
	}
	q.stopped = true
	close(q.notify)
}

// Len returns the number of items in the queue
func (q *RetryQueue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.items)
}

func (q *RetryQueue) recalcNext() {
	q.next = nil
	for _, item := range q.items {
		if q.next == nil || item.NextRetryAt.Before(q.next.NextRetryAt) {
			q.next = item
		}
	}
}

func (q *RetryQueue) signal() {
	select {
	case q.notify <- struct{}{}:
	default:
	}
}

func (t *retryTimer) Reset(nextTime time.Time) <-chan time.Time {
	delay := max(time.Until(nextTime), 0)
	if t.timer == nil {
		t.timer = time.NewTimer(delay)
		return t.timer.C
	}
	if !t.timer.Stop() {
		select {
		case <-t.timer.C:
		default:
		}
	}
	t.timer.Reset(delay)
	return t.timer.C
}

func (t *retryTimer) Stop() {
	if t.timer != nil {
		t.timer.Stop()
	}
}
