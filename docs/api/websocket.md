# WebSocket API

The engine provides a real-time event stream via WebSocket for live monitoring of steps, node health, cluster membership, and flow execution.

## Connection

- Endpoint: `GET /engine/ws`
- Local URL: `ws://localhost:8080/engine/ws`
- Production URL: `wss://.../engine/ws`

## Message Envelopes

### Event Messages

Event messages stream live timebox events:

```json
{
  "type": "flow_started",
  "data": { "flow_id": "wf-123" },
  "id": ["flow", "wf-123"],
  "sub_id": "flow-detail",
  "timestamp": 1704067425000,
  "sequence": 42
}
```

Fields:

- `type`: event type
- `data`: event payload
- `id`: aggregate ID path
- `sub_id`: subscription identifier that produced the event
- `timestamp`: event timestamp in Unix milliseconds
- `sequence`: aggregate sequence number

### Subscribed Messages

When you subscribe, the server sends the current projected state for every requested aggregate before streaming live events:

```json
{
  "type": "subscribed",
  "sub_id": "flow-list",
  "items": [
    {
      "id": ["flow", "wf-123"],
      "data": { "id": "wf-123", "status": "active" },
      "sequence": 42
    },
    {
      "id": ["flow", "wf-456"],
      "data": { "id": "wf-456", "status": "failed" },
      "sequence": 87
    }
  ]
}
```

Each `items` entry contains:

- `id`: aggregate ID path
- `data`: current projected state for that aggregate
- `sequence`: next live sequence boundary for that aggregate

The server suppresses stale live events with a sequence lower than the initial subscribed item sequence for that aggregate.

## Subscriptions

A connection starts idle. Send a `subscribe` message to begin receiving events:

```json
{
  "type": "subscribe",
  "data": {
    "sub_id": "flow-detail",
    "aggregate_ids": [["flow", "wf-123"]],
    "event_types": ["flow_started", "step_completed", "flow_completed"]
  }
}
```

Fields:

- `sub_id`: required client-defined subscription ID
- `aggregate_ids`: optional list of aggregate IDs to follow
- `include_state`: optional flag controlling whether the server sends the initial `subscribed` projection batch. Omit it or set it to `false` for live events only; set it to `true` when you want the current projected state first
- `event_types`: optional event type filter

Single-aggregate subscriptions use a one-element `aggregate_ids` array. There is no separate `aggregate_id` field.

To unsubscribe:

```json
{
  "type": "unsubscribe",
  "data": {
    "sub_id": "flow-detail"
  }
}
```

## Aggregate IDs

- `["catalog"]`: step registry events
- `["nodes"]`: node registry state
- `["node", "node-id"]`: one node's health and last-seen state
- `["node"]`: prefix subscription for per-node health events across the cluster
- `["flow", "flow-id"]`: flow execution events for one flow

## Event Types

### Catalog

- `step_registered`
- `step_unregistered`
- `step_updated`

### Nodes

- `node_seen`
- `step_health_changed`

### Flow

- `flow_started`
- `flow_completed`
- `flow_failed`
- `flow_deactivated`
- `step_started`
- `step_completed`
- `step_failed`
- `step_skipped`
- `attribute_set`
- `work_started`
- `work_succeeded`
- `work_failed`
- `work_not_completed`
- `retry_scheduled`

## Examples

### Subscribe to Catalog Updates

```json
{
  "type": "subscribe",
  "data": {
    "sub_id": "catalog",
    "aggregate_ids": [["catalog"]]
  }
}
```

### Subscribe to Node Health

```json
{
  "type": "subscribe",
  "data": {
    "sub_id": "health",
    "aggregate_ids": [["node"]],
    "event_types": ["step_health_changed"]
  }
}
```

### Subscribe to Node Registry State

```json
{
  "type": "subscribe",
  "data": {
    "sub_id": "nodes",
    "aggregate_ids": [["nodes"]],
    "include_state": true,
    "event_types": ["node_seen"]
  }
}
```

### Subscribe to a Specific Flow

```json
{
  "type": "subscribe",
  "data": {
    "sub_id": "flow-detail",
    "aggregate_ids": [["flow", "wf-123"]],
    "event_types": ["flow_started", "step_completed", "flow_completed", "flow_failed"]
  }
}
```

### Subscribe to Multiple Visible Flows

```json
{
  "type": "subscribe",
  "data": {
    "sub_id": "flow-list",
    "aggregate_ids": [
      ["flow", "wf-123"],
      ["flow", "wf-456"],
      ["flow", "wf-789"]
    ],
    "event_types": ["flow_started", "flow_completed", "flow_failed", "flow_deactivated"]
  }
}
```

## Client Pattern

One WebSocket connection can carry multiple subscriptions. The web UI uses one connection and adds or removes subscriptions as needed:

- catalog events
- node health
- the currently selected flow
- the currently visible flow rows in the selector list

Use distinct `sub_id` values so you can replace or unsubscribe individual streams without reconnecting the socket.

Use `include_state: true` for stateful subscriptions such as catalog, `["nodes"]`, `["node", "node-id"]`, or the currently selected flow. Leave it omitted for high-churn prefix subscriptions where you only care about future events, such as `["node"]` for cluster-wide health events or a short-lived visible-row list in the flow selector.

## Performance Notes

- Filter aggressively with `aggregate_ids` and `event_types`
- Use multi-aggregate subscriptions for short visible lists instead of broad unfiltered streams
- Expect high event volume on busy flows
- Reconnect on `close`, then resubscribe with the same filters

## Example Client

```javascript
const ws = new WebSocket("ws://localhost:8080/engine/ws");

ws.onopen = () => {
  ws.send(JSON.stringify({
    type: "subscribe",
    data: {
      sub_id: "flow-detail",
      aggregate_ids: [["flow", "wf-123"]],
      event_types: ["flow_started", "step_completed", "flow_completed", "flow_failed"],
    },
  }));
};

ws.onmessage = (event) => {
  const msg = JSON.parse(event.data);

  if (msg.type === "subscribed") {
    console.log("initial projection", msg.items[0]);
    return;
  }

  switch (msg.type) {
    case "step_completed":
      console.log("step completed", msg.data.step_id, msg.data.outputs);
      break;
    case "flow_completed":
      console.log("flow completed", msg.data.result);
      break;
    case "flow_failed":
      console.error("flow failed", msg.data.error);
      break;
  }
};
```
