# WebSocket API

The engine provides a real-time event stream via WebSocket for live monitoring of flows and step execution.

## Connection

**Endpoint:** `GET /engine/ws`

**URL:** `ws://localhost:8080/engine/ws`

**Connection types:**
- `ws://` for local development
- `wss://` for secure production

## Example Connection

```javascript
// Connect to WebSocket
const ws = new WebSocket('ws://localhost:8080/engine/ws');

ws.onopen = () => {
  console.log('Connected to engine');
};

ws.onmessage = (event) => {
  const engineEvent = JSON.parse(event.data);
  console.log('Event:', engineEvent);
};

ws.onerror = (error) => {
  console.error('WebSocket error:', error);
};

ws.onclose = () => {
  console.log('Disconnected from engine');
};
```

## Event Format

Event messages use this envelope:

```json
{
  "type": "event_type",
  "data": { /* event-specific fields */ },
  "id": ["aggregate", "id"],
  "timestamp": 1704067425000,
  "sequence": 42
}
```

**Envelope Fields (event messages):**
- `type`: Event type constant (e.g., "flow_started", "step_completed")
- `data`: Event-specific data (structure varies by event type)
- `id`: Aggregate ID path (e.g., ["engine"], ["flow", "flow-id"])
- `timestamp`: Milliseconds since epoch (Unix time × 1000)
- `sequence`: Global event sequence number for ordering

`subscribed` messages are different: they include `type`, `id`, `data`, and
`sequence`, but no `timestamp`.

## Event Types

### Flow Events

**flow_started** — Emitted when flow execution begins. Includes the execution plan and initial arguments.

```json
{
  "type": "flow_started",
  "data": {
    "flow_id": "wf-123",
    "init": { "amount": 100.00, "order_id": "ord-456" },
    "plan": {
      "goals": ["process_payment", "send_notification"],
      "required": ["amount", "order_id"],
      "steps": { /* step definitions */ },
      "attributes": { /* attribute dependency graph */ }
    },
    "metadata": { "user_id": "user-789" }
  },
  "id": ["flow", "wf-123"],
  "timestamp": 1704067425000,
  "sequence": 1
}
```

**flow_completed** — Emitted when all goal steps are satisfied and flow completes successfully.

```json
{
  "type": "flow_completed",
  "data": {
    "flow_id": "wf-123",
    "result": { "transaction_id": "txn-789", "notification_sent": true }
  },
  "id": ["flow", "wf-123"],
  "timestamp": 1704067430000,
  "sequence": 47
}
```

**flow_failed** — Emitted when a goal step fails or becomes unreachable, making success impossible.

```json
{
  "type": "flow_failed",
  "data": {
    "flow_id": "wf-123",
    "error": "step process_payment failed: insufficient funds"
  },
  "id": ["flow", "wf-123"],
  "timestamp": 1704067428000,
  "sequence": 45
}
```

**flow_activated** — Emitted when a flow becomes active (part of the engine's active flow list).

```json
{
  "type": "flow_activated",
  "data": {
    "flow_id": "wf-123",
    "parent_flow_id": "wf-parent-456"
  },
  "id": ["flow", "wf-123"],
  "timestamp": 1704067425100,
  "sequence": 2
}
```

**flow_deactivated** — Emitted when a flow is terminal (completed or failed) AND no active work items remain.

```json
{
  "type": "flow_deactivated",
  "data": {
    "flow_id": "wf-123"
  },
  "id": ["flow", "wf-123"],
  "timestamp": 1704067435000,
  "sequence": 50
}
```

**flow_archiving** — Emitted when a deactivated flow is selected for archiving to external storage.

```json
{
  "type": "flow_archiving",
  "data": {
    "flow_id": "wf-123"
  },
  "id": ["flow", "wf-123"],
  "timestamp": 1704067440000,
  "sequence": 51
}
```

**flow_archived** — Emitted when a flow is successfully archived.

```json
{
  "type": "flow_archived",
  "data": {
    "flow_id": "wf-123"
  },
  "id": ["flow", "wf-123"],
  "timestamp": 1704067445000,
  "sequence": 52
}
```

**flow_digest_updated** — Emitted when the flow's summary status changes (internal event).

```json
{
  "type": "flow_digest_updated",
  "data": {
    "flow_id": "wf-123",
    "status": "completed",
    "completed_at": "2025-01-30T15:24:02Z",
    "error": ""
  },
  "id": ["flow", "wf-123"],
  "timestamp": 1704067430500,
  "sequence": 48
}
```

### Step Registry Events

**step_registered** — Emitted when a step is added to the engine registry.

```json
{
  "type": "step_registered",
  "data": {
    "step": {
      "id": "lookup_customer",
      "name": "Lookup Customer",
      "type": "sync",
      "http": { "endpoint": "http://api.local/customer", "timeout": 5000 },
      "attributes": {
        "customer_id": { "input": true },
        "customer_name": { "output": true }
      },
      "memoizable": false
    }
  },
  "id": ["engine"],
  "timestamp": 1704067200000,
  "sequence": 1
}
```

**step_unregistered** — Emitted when a step is removed from the engine registry.

```json
{
  "type": "step_unregistered",
  "data": {
    "step_id": "lookup_customer"
  },
  "id": ["engine"],
  "timestamp": 1704067500000,
  "sequence": 10
}
```

**step_updated** — Emitted when a step definition is modified.

```json
{
  "type": "step_updated",
  "data": {
    "step": {
      "id": "lookup_customer",
      "name": "Lookup Customer",
      "type": "sync",
      "http": { "endpoint": "http://api.local/customer/v2", "timeout": 10000 },
      "attributes": {
        "customer_id": { "input": true },
        "customer_name": { "output": true },
        "customer_tier": { "output": true }
      },
      "memoizable": true
    }
  },
  "id": ["engine"],
  "timestamp": 1704067600000,
  "sequence": 12
}
```

**step_health_changed** — Emitted when a step's health status changes (e.g., endpoint unreachable).

```json
{
  "type": "step_health_changed",
  "data": {
    "step_id": "lookup_customer",
    "status": "unhealthy",
    "error": "health check failed: connection refused"
  },
  "id": ["engine"],
  "timestamp": 1704067700000,
  "sequence": 15
}
```

### Step Events

**step_started** — Emitted when a step begins execution.

```json
{
  "type": "step_started",
  "data": {
    "flow_id": "wf-123",
    "step_id": "process_payment",
    "inputs": { "amount": 100.00, "currency": "USD" },
    "work_items": {
      "tok-1": { "amount": 50.00 },
      "tok-2": { "amount": 50.00 }
    }
  },
  "id": ["flow", "wf-123"],
  "timestamp": 1704067427000,
  "sequence": 25
}
```

**step_completed** — Emitted when a step finishes successfully.

```json
{
  "type": "step_completed",
  "data": {
    "flow_id": "wf-123",
    "step_id": "process_payment",
    "outputs": { "transaction_id": "txn-789", "status": "approved" },
    "duration": 523
  },
  "id": ["flow", "wf-123"],
  "timestamp": 1704067428000,
  "sequence": 26
}
```

**step_failed** — Emitted when a step encounters an error.

```json
{
  "type": "step_failed",
  "data": {
    "flow_id": "wf-123",
    "step_id": "process_payment",
    "error": "connection timeout"
  },
  "id": ["flow", "wf-123"],
  "timestamp": 1704067429000,
  "sequence": 27
}
```

**step_skipped** — Emitted when a step is skipped due to its predicate evaluating to false.

```json
{
  "type": "step_skipped",
  "data": {
    "flow_id": "wf-123",
    "step_id": "send_premium_notification",
    "reason": "predicate returned false"
  },
  "id": ["flow", "wf-123"],
  "timestamp": 1704067428500,
  "sequence": 30
}
```

### Work Item Events

**work_started** — Emitted when a work item begins execution (for async steps or for_each expansion).

```json
{
  "type": "work_started",
  "data": {
    "flow_id": "wf-123",
    "step_id": "process_items",
    "token": "tok-abc-123",
    "inputs": { "item_id": "item-789", "quantity": 5 }
  },
  "id": ["flow", "wf-123"],
  "timestamp": 1704067430000,
  "sequence": 33
}
```

**work_succeeded** — Emitted when a work item completes successfully.

```json
{
  "type": "work_succeeded",
  "data": {
    "flow_id": "wf-123",
    "step_id": "process_items",
    "token": "tok-abc-123",
    "outputs": { "processed_count": 100, "status": "ok" }
  },
  "id": ["flow", "wf-123"],
  "timestamp": 1704067431000,
  "sequence": 34
}
```

**work_failed** — Emitted when a work item fails permanently (error is unrecoverable).

```json
{
  "type": "work_failed",
  "data": {
    "flow_id": "wf-123",
    "step_id": "process_items",
    "token": "tok-abc-123",
    "error": "invalid input format"
  },
  "id": ["flow", "wf-123"],
  "timestamp": 1704067432000,
  "sequence": 35
}
```

**work_not_completed** — Emitted when work reports transient failure (retry will be attempted).

```json
{
  "type": "work_not_completed",
  "data": {
    "flow_id": "wf-123",
    "step_id": "process_items",
    "token": "tok-abc-123",
    "error": "service temporarily unavailable"
  },
  "id": ["flow", "wf-123"],
  "timestamp": 1704067434000,
  "sequence": 36
}
```

**retry_scheduled** — Emitted when a failed work item is scheduled for retry.

```json
{
  "type": "retry_scheduled",
  "data": {
    "flow_id": "wf-123",
    "step_id": "process_items",
    "token": "tok-abc-123",
    "retry_count": 1,
    "next_retry_at": "2025-01-30T15:24:00Z",
    "error": "service temporarily unavailable"
  },
  "id": ["flow", "wf-123"],
  "timestamp": 1704067433000,
  "sequence": 37
}
```

### Attribute and Status Events

**attribute_set** — Emitted when a flow attribute is set from step outputs (with provenance tracking).

```json
{
  "type": "attribute_set",
  "data": {
    "flow_id": "wf-123",
    "step_id": "lookup_customer",
    "key": "customer_name",
    "value": "Alice Smith"
  },
  "id": ["flow", "wf-123"],
  "timestamp": 1704067428000,
  "sequence": 31
}
```

## Subscriptions and Filtering

A new WebSocket connection starts with no event stream. Send a `subscribe`
message to begin receiving events:

```json
{
  "type": "subscribe",
  "data": {
    "aggregate_id": ["flow", "wf-123"],
    "event_types": ["flow_started", "step_completed", "flow_completed"]
  }
}
```

**Subscription Fields:**
- `aggregate_id` (optional): Array identifying the aggregate to filter on. Use `["engine"]` for all engine-level events, or `["flow", "flow-id"]` for a specific flow's events.
- `event_types` (optional): Array of event types to receive. If omitted, all event types for the aggregate are sent.

**Examples:**

Subscribe to all engine events (step registry, health changes):
```json
{
  "type": "subscribe",
  "data": {
    "aggregate_id": ["engine"]
  }
}
```

Subscribe to a specific flow's events:
```json
{
  "type": "subscribe",
  "data": {
    "aggregate_id": ["flow", "wf-123"]
  }
}
```

Subscribe to only completion events for a flow:
```json
{
  "type": "subscribe",
  "data": {
    "aggregate_id": ["flow", "wf-123"],
    "event_types": ["flow_completed", "flow_failed", "step_completed"]
  }
}
```

### Client Implementation Pattern

The web UI maintains **two separate WebSocket connections**:
1. One subscribed to `["engine"]` for all engine-level events (step registry, health changes)
2. One subscribed to `["flow", flowId]` for a specific flow's events

This allows efficient filtering at the server level rather than discarding events on the client:

```javascript
// Connection for engine-level events
const engineWs = new WebSocket('ws://localhost:8080/engine/ws');
engineWs.onopen = () => {
  engineWs.send(JSON.stringify({
    type: 'subscribe',
    data: { aggregate_id: ['engine'] }
  }));
};

// Connection for specific flow
const flowWs = new WebSocket('ws://localhost:8080/engine/ws');
flowWs.onopen = () => {
  flowWs.send(JSON.stringify({
    type: 'subscribe',
    data: { aggregate_id: ['flow', 'wf-123'] }
  }));
};
```

## Connection Management

### Reconnection

The WebSocket connection may drop. Your client should:
1. Detect the disconnection (`onclose` event)
2. Implement exponential backoff for reconnect attempts
3. Resubscribe after reconnection with your desired filters

**Example backoff with resubscription:**
```javascript
let reconnectDelay = 1000; // 1 second
const maxDelay = 30000;    // 30 seconds
const subscriptionFilter = {
  type: 'subscribe',
  data: { aggregate_id: ['flow', 'wf-123'] }
};

ws.onclose = () => {
  setTimeout(() => {
    ws = new WebSocket('ws://localhost:8080/engine/ws');
    ws.onopen = () => {
      ws.send(JSON.stringify(subscriptionFilter));
    };
    reconnectDelay = Math.min(reconnectDelay * 2, maxDelay);
  }, reconnectDelay);
};
```

## Performance Considerations

- **Event volume**: High-concurrency scenarios may generate many events per second
- **Reduce event traffic**: Use subscription filters to receive only relevant events, reducing network bandwidth and client processing overhead
- **Multiple connections**: Consider using separate WebSocket connections for different aggregate IDs (e.g., one for engine-level events, one for specific flow monitoring)
- **Message queueing**: Large backlogs can cause memory growth in the server
- **Parsing overhead**: Decode JSON and extract relevant fields on the client
- **Best practice**: Subscribe with filters to minimize unnecessary data transfer; process and discard events as needed

## Error Handling

- **Network errors**: Handle `onerror` and `onclose` with reconnection logic
- **Malformed events**: Add try-catch around JSON parsing
- **Stale events**: WebSocket events are real-time but may arrive out-of-order across network boundaries

## Example: Monitor a Specific Flow

Subscribe to only the events you care about on the server, eliminating the need for client-side filtering:

```javascript
const flowIDToMonitor = 'wf-123';

const ws = new WebSocket('ws://localhost:8080/engine/ws');

ws.onopen = () => {
  // Subscribe to specific flow's completion-related events
  ws.send(JSON.stringify({
    type: 'subscribe',
    data: {
      aggregate_id: ['flow', flowIDToMonitor],
      event_types: ['flow_started', 'step_completed', 'flow_completed', 'flow_failed']
    }
  }));
};

ws.onmessage = (event) => {
  const engineEvent = JSON.parse(event.data);

  switch (engineEvent.type) {
    case 'flow_started':
      console.log('Flow started:', engineEvent.data);
      break;
    case 'step_completed':
      console.log(`Step ${engineEvent.data.step_id} completed`, engineEvent.data.outputs);
      break;
    case 'flow_completed':
      console.log('Flow completed with result:', engineEvent.data.result);
      break;
    case 'flow_failed':
      console.error('Flow failed:', engineEvent.data.error);
      break;
  }
};
```
