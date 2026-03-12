package event

import (
	"slices"
	"sync"

	"github.com/kode4food/caravan"
	"github.com/kode4food/caravan/topic"
	"github.com/kode4food/timebox"
)

type (
	// Hub filters events based on active subscriptions
	Hub struct {
		inner    topic.Topic[*timebox.Event]
		producer topic.Producer[*timebox.Event]
		registry *registry
	}

	// Consumer filters events based on interests
	Consumer struct {
		inner     topic.Consumer[*timebox.Event]
		interests *interests
		registry  *registry
		filtered  <-chan *timebox.Event
		once      sync.Once
		closeOnce sync.Once
	}

	// registry tracks active subscriptions and counts references
	registry struct {
		mu             sync.RWMutex
		anyType        *prefixNode
		byType         map[timebox.EventType]*prefixNode
		allEventsCount int64
	}

	// interests describes what events a consumer is interested in
	interests struct {
		eventTypes map[timebox.EventType]bool // empty = all event types
		prefixes   []timebox.AggregateID      // empty = all aggregates
	}

	prefixNode struct {
		count    int64
		children map[timebox.ID]*prefixNode
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
			anyType: &prefixNode{},
			byType:  make(map[timebox.EventType]*prefixNode),
		},
	}
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

// NewAggregateConsumer creates a consumer interested in events from aggregates
// matching the provided prefix. If no event types are specified, the consumer
// receives all events for aggregates matching the prefix
func (h *Hub) NewAggregateConsumer(
	prefix timebox.AggregateID, eventTypes ...timebox.EventType,
) *Consumer {
	return h.NewAggregatesConsumer([]timebox.AggregateID{prefix}, eventTypes...)
}

// NewAggregatesConsumer creates a consumer interested in events from
// aggregates matching any provided prefix. If no event types are specified,
// the consumer receives all events for aggregates matching those prefixes
func (h *Hub) NewAggregatesConsumer(
	prefixes []timebox.AggregateID, eventTypes ...timebox.EventType,
) *Consumer {
	i := &interests{
		prefixes: prefixes,
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
	}
}

// Receive returns a channel of events filtered by the consumer's interests
func (c *Consumer) Receive() <-chan *timebox.Event {
	c.once.Do(func() {
		filtered := make(chan *timebox.Event, 1)

		go func() {
			defer close(filtered)
			for ev := range c.inner.Receive() {
				if c.matches(ev) {
					filtered <- ev
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
	if len(c.interests.prefixes) > 0 {
		if !slices.ContainsFunc(
			c.interests.prefixes, ev.AggregateID.HasPrefix,
		) {
			return false
		}
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

	if len(i.prefixes) == 0 && len(i.eventTypes) == 0 {
		r.allEventsCount++
		return
	}

	if len(i.eventTypes) == 0 {
		for _, pfx := range i.prefixes {
			r.anyType.add(pfx)
		}
		return
	}

	if len(i.prefixes) == 0 {
		for et := range i.eventTypes {
			r.getOrCreateNode(et).add(nil)
		}
		return
	}

	for _, pfx := range i.prefixes {
		for et := range i.eventTypes {
			r.getOrCreateNode(et).add(pfx)
		}
	}
}

// unregister removes a subscription from the registry
func (r *registry) unregister(i *interests) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(i.prefixes) == 0 && len(i.eventTypes) == 0 {
		r.allEventsCount--
		return
	}

	if len(i.eventTypes) == 0 {
		for _, pfx := range i.prefixes {
			r.anyType.remove(pfx)
		}
		return
	}

	if len(i.prefixes) == 0 {
		for et := range i.eventTypes {
			if node, ok := r.byType[et]; ok {
				node.remove(nil)
				if node.isEmpty() {
					delete(r.byType, et)
				}
			}
		}
		return
	}

	for _, pfx := range i.prefixes {
		for et := range i.eventTypes {
			if node, ok := r.byType[et]; ok {
				node.remove(pfx)
				if node.isEmpty() {
					delete(r.byType, et)
				}
			}
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

	if r.anyType.hasPrefixMatch(id) {
		return true
	}

	if node, ok := r.byType[typ]; ok {
		if node.hasPrefixMatch(id) {
			return true
		}
	}

	return false
}

func (r *registry) getOrCreateNode(et timebox.EventType) *prefixNode {
	if node, ok := r.byType[et]; ok {
		return node
	}
	node := &prefixNode{}
	r.byType[et] = node
	return node
}

func (p *prefixNode) add(prefix timebox.AggregateID) {
	node := p
	for _, part := range prefix {
		if node.children == nil {
			node.children = make(map[timebox.ID]*prefixNode)
		}
		child := node.children[part]
		if child == nil {
			child = &prefixNode{}
			node.children[part] = child
		}
		node = child
	}
	node.count++
}

func (p *prefixNode) remove(prefix timebox.AggregateID) {
	node := p
	type pathEntry struct {
		node *prefixNode
		key  timebox.ID
	}
	var path []pathEntry

	for _, part := range prefix {
		if node.children == nil {
			return
		}
		child := node.children[part]
		if child == nil {
			return
		}
		path = append(path, pathEntry{node: node, key: part})
		node = child
	}

	node.count--
	if node.count > 0 || len(node.children) > 0 {
		return
	}

	for i := len(path) - 1; i >= 0; i-- {
		parent := path[i].node
		delete(parent.children, path[i].key)
		if parent.count > 0 || len(parent.children) > 0 {
			return
		}
	}
}

func (p *prefixNode) hasPrefixMatch(id timebox.AggregateID) bool {
	node := p
	if node.count > 0 {
		return true
	}
	for _, part := range id {
		if node.children == nil {
			return false
		}
		child := node.children[part]
		if child == nil {
			return false
		}
		node = child
		if node.count > 0 {
			return true
		}
	}
	return false
}

func (p *prefixNode) isEmpty() bool {
	return p.count == 0 && len(p.children) == 0
}
