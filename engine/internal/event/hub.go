package event

import (
	"slices"
	"sync"

	"github.com/kode4food/caravan"
	"github.com/kode4food/caravan/closer"
	"github.com/kode4food/caravan/topic"
	"github.com/kode4food/timebox"
)

type (
	// Hub filters events based on active subscriptions
	Hub struct {
		inner     topic.Topic[*timebox.Event]
		producer  topic.Producer[*timebox.Event]
		registry  *registry
		closed    chan struct{}
		closeOnce sync.Once
	}

	// Consumer filters events based on interests
	Consumer struct {
		inner     topic.Consumer[*timebox.Event]
		interests *interests
		registry  *registry
		filtered  <-chan *timebox.Event
		closed    chan struct{}
		once      sync.Once
		closeOnce sync.Once
	}

	// registry tracks active subscriptions and counts references
	registry struct {
		anyType        *aggregateNode
		byType         map[timebox.EventType]*aggregateNode
		allEventsCount int64
		mu             sync.RWMutex
	}

	// interests describes what events a consumer is interested in
	interests struct {
		eventTypes map[timebox.EventType]bool // empty = all event types
		ids        []timebox.AggregateID      // empty = all aggregates
	}

	aggregateNode struct {
		byID map[timebox.AggregateID]int64
		all  int64
	}
)

// NewHub creates a new Hub with an internal in-memory topic
func NewHub() *Hub {
	return NewHubWithTopic(caravan.NewTopic[*timebox.Event]())
}

// NewHubWithTopic creates a new Hub that filters events based on active
// subscriptions
func NewHubWithTopic(inner topic.Topic[*timebox.Event]) *Hub {
	return &Hub{
		inner:    inner,
		producer: inner.NewProducer(),
		registry: &registry{
			anyType: &aggregateNode{},
			byType:  make(map[timebox.EventType]*aggregateNode),
		},
		closed: make(chan struct{}),
	}
}

// Close releases the Hub's underlying topic resources when supported
func (h *Hub) Close() {
	h.closeOnce.Do(func() {
		if c, ok := h.inner.(closer.Closer); ok {
			c.Close()
		}
		close(h.closed)
	})
}

// IsClosed reports whether the underlying topic has been closed
func (h *Hub) IsClosed() <-chan struct{} {
	return h.closed
}

// Publish sends committed events to matching subscribers
func (h *Hub) Publish(evs ...*timebox.Event) {
	for _, ev := range evs {
		if ev == nil || !h.hasSubscribers(ev.Type, ev.AggregateID) {
			continue
		}
		h.producer.Send() <- ev
	}
}

// NewConsumer creates a consumer that receives all events
func (h *Hub) NewConsumer() *Consumer {
	return h.NewAggregatesConsumer(nil)
}

// NewTypeConsumer creates a consumer interested in specific event types
func (h *Hub) NewTypeConsumer(eventTypes ...timebox.EventType) *Consumer {
	return h.NewAggregatesConsumer(nil, eventTypes...)
}

// NewAggregateConsumer creates a consumer interested in events from one
// aggregate. If no event types are specified, it receives all of its events
func (h *Hub) NewAggregateConsumer(
	id timebox.AggregateID, eventTypes ...timebox.EventType,
) *Consumer {
	return h.NewAggregatesConsumer([]timebox.AggregateID{id}, eventTypes...)
}

// NewAggregatesConsumer creates a consumer for the specified aggregates and
// event types. An empty filter places no restriction on that dimension
func (h *Hub) NewAggregatesConsumer(
	ids []timebox.AggregateID, eventTypes ...timebox.EventType,
) *Consumer {
	i := &interests{
		ids: ids,
	}

	if len(eventTypes) > 0 {
		i.eventTypes = make(map[timebox.EventType]bool)
		for _, et := range eventTypes {
			i.eventTypes[et] = true
		}
	}

	h.registry.register(i)

	return &Consumer{
		inner:     h.inner.NewConsumer(),
		interests: i,
		registry:  h.registry,
		closed:    make(chan struct{}),
	}
}

// Receive returns a channel of events filtered by the consumer's interests
func (c *Consumer) Receive() <-chan *timebox.Event {
	c.once.Do(func() {
		filtered := make(chan *timebox.Event, 1)

		go func() {
			defer close(filtered)
			for {
				select {
				case <-c.closed:
					return
				case ev, ok := <-c.inner.Receive():
					if !ok {
						return
					}
					if !c.matches(ev) {
						continue
					}
					select {
					case <-c.closed:
						return
					case filtered <- ev:
					}
				}
			}
		}()

		c.filtered = filtered
	})

	return c.filtered
}

// Close unregisters the consumer
func (c *Consumer) Close() {
	c.closeOnce.Do(func() {
		c.registry.unregister(c.interests)
		close(c.closed)
		c.inner.Close()
	})
}

func (h *Hub) hasSubscribers(
	typ timebox.EventType, id timebox.AggregateID,
) bool {
	return h.registry.hasSubscribers(typ, id)
}

// matches checks if an event matches the consumer's interests
func (c *Consumer) matches(ev *timebox.Event) bool {
	if !c.interests.wantsAggregate(ev.AggregateID) {
		return false
	}

	if len(c.interests.eventTypes) > 0 && !c.interests.eventTypes[ev.Type] {
		return false
	}

	return true
}

// register adds a subscription to the registry
func (r *registry) register(i *interests) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if i.wantsEverything() {
		r.allEventsCount++
		return
	}

	if len(i.eventTypes) == 0 {
		i.addTo(r.anyType)
		return
	}

	for et := range i.eventTypes {
		i.addTo(r.getOrCreateNode(et))
	}
}

// unregister removes a subscription from the registry
func (r *registry) unregister(i *interests) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if i.wantsEverything() {
		r.allEventsCount--
		return
	}

	if len(i.eventTypes) == 0 {
		i.removeFrom(r.anyType)
		return
	}

	for et := range i.eventTypes {
		node, ok := r.byType[et]
		if !ok {
			continue
		}
		i.removeFrom(node)
		if node.isEmpty() {
			delete(r.byType, et)
		}
	}
}

func (r *registry) hasSubscribers(
	typ timebox.EventType, id timebox.AggregateID,
) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.allEventsCount > 0 {
		return true
	}

	if r.anyType.hasSubscriber(id) {
		return true
	}

	if node, ok := r.byType[typ]; ok {
		if node.hasSubscriber(id) {
			return true
		}
	}

	return false
}

func (r *registry) getOrCreateNode(et timebox.EventType) *aggregateNode {
	if node, ok := r.byType[et]; ok {
		return node
	}
	node := &aggregateNode{}
	r.byType[et] = node
	return node
}

func (i *interests) wantsEverything() bool {
	return len(i.ids) == 0 && len(i.eventTypes) == 0
}

// wantsAggregate is satisfied by interests naming no aggregate at all
func (i *interests) wantsAggregate(id timebox.AggregateID) bool {
	return len(i.ids) == 0 || slices.Contains(i.ids, id)
}

func (i *interests) addTo(node *aggregateNode) {
	if len(i.ids) == 0 {
		node.all++
		return
	}
	for _, id := range i.ids {
		node.addID(id)
	}
}

func (i *interests) removeFrom(node *aggregateNode) {
	if len(i.ids) == 0 {
		node.all--
		return
	}
	for _, id := range i.ids {
		node.removeID(id)
	}
}

func (a *aggregateNode) addID(id timebox.AggregateID) {
	if a.byID == nil {
		a.byID = map[timebox.AggregateID]int64{}
	}
	a.byID[id]++
}

func (a *aggregateNode) removeID(id timebox.AggregateID) {
	if a.byID[id]--; a.byID[id] <= 0 {
		delete(a.byID, id)
	}
}

func (a *aggregateNode) hasSubscriber(id timebox.AggregateID) bool {
	return a.all > 0 || a.byID[id] > 0
}

func (a *aggregateNode) isEmpty() bool {
	return a.all == 0 && len(a.byID) == 0
}
