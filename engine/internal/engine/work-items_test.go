package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/spuds/engine/pkg/api"
)

func TestSingleWorkItem(t *testing.T) {
	items := api.WorkItems{
		"token-1": {
			Status: api.WorkSucceeded,
			Inputs: api.Args{
				"user": "alice",
			},
			Outputs: api.Args{
				"email":      "alice@example.com",
				"message_id": "msg-123",
			},
		},
	}

	step := &api.Step{
		Attributes: api.AttributeSpecs{
			"user": {Role: api.RoleRequired, ForEach: true},
		},
	}

	result := aggregateWorkItemOutputs(items, step)

	assert.Equal(t, "alice@example.com", result["email"])
	assert.Equal(t, "msg-123", result["message_id"])
	assert.Len(t, result, 2)
}

func TestMultipleWorkItems(t *testing.T) {
	items := api.WorkItems{
		"token-1": {
			Status: api.WorkSucceeded,
			Inputs: api.Args{
				"user": "alice",
			},
			Outputs: api.Args{
				"message_id": "msg-123",
			},
		},
		"token-2": {
			Status: api.WorkSucceeded,
			Inputs: api.Args{
				"user": "bob",
			},
			Outputs: api.Args{
				"message_id": "msg-456",
			},
		},
		"token-3": {
			Status: api.WorkSucceeded,
			Inputs: api.Args{
				"user": "charlie",
			},
			Outputs: api.Args{
				"message_id": "msg-789",
			},
		},
	}

	step := &api.Step{
		Attributes: api.AttributeSpecs{
			"user": {Role: api.RoleRequired, ForEach: true},
		},
	}

	result := aggregateWorkItemOutputs(items, step)

	messageIds, ok := result["message_id"].([]map[string]any)
	assert.True(t, ok)
	assert.Len(t, messageIds, 3)

	users := map[string]string{}
	for _, entry := range messageIds {
		user := entry["user"].(string)
		msgId := entry["message_id"].(string)
		users[user] = msgId

		assert.Len(t, entry, 2)
	}

	assert.Equal(t, "msg-123", users["alice"])
	assert.Equal(t, "msg-456", users["bob"])
	assert.Equal(t, "msg-789", users["charlie"])
}

func TestOutputUsesAttributeName(t *testing.T) {
	items := api.WorkItems{
		"token-1": {
			Status: api.WorkSucceeded,
			Inputs: api.Args{
				"user": "alice",
			},
			Outputs: api.Args{
				"result": "success",
			},
		},
		"token-2": {
			Status: api.WorkSucceeded,
			Inputs: api.Args{
				"user": "bob",
			},
			Outputs: api.Args{
				"result": "success",
			},
		},
	}

	step := &api.Step{
		Attributes: api.AttributeSpecs{
			"user": {Role: api.RoleRequired, ForEach: true},
		},
	}

	output := aggregateWorkItemOutputs(items, step)

	results := output["result"].([]map[string]any)
	assert.Len(t, results, 2)

	for _, entry := range results {
		assert.Contains(t, entry, "result")
		assert.NotContains(t, entry, "value")
		assert.Contains(t, entry, "user")
	}
}

func TestMultipleForEachAttributes(t *testing.T) {
	items := api.WorkItems{
		"token-1": {
			Status: api.WorkSucceeded,
			Inputs: api.Args{
				"user":   "alice",
				"action": "notify",
			},
			Outputs: api.Args{
				"result": "notification sent",
			},
		},
		"token-2": {
			Status: api.WorkSucceeded,
			Inputs: api.Args{
				"user":   "alice",
				"action": "log",
			},
			Outputs: api.Args{
				"result": "logged",
			},
		},
		"token-3": {
			Status: api.WorkSucceeded,
			Inputs: api.Args{
				"user":   "bob",
				"action": "notify",
			},
			Outputs: api.Args{
				"result": "notification sent",
			},
		},
	}

	step := &api.Step{
		Attributes: api.AttributeSpecs{
			"user":   {Role: api.RoleRequired, ForEach: true},
			"action": {Role: api.RoleRequired, ForEach: true},
		},
	}

	output := aggregateWorkItemOutputs(items, step)

	results := output["result"].([]map[string]any)
	assert.Len(t, results, 3)

	for _, entry := range results {
		assert.Contains(t, entry, "user")
		assert.Contains(t, entry, "action")
		assert.Contains(t, entry, "result")
		assert.Len(t, entry, 3) // 2 metadata + 1 output
	}

	var aliceNotify map[string]any
	for _, entry := range results {
		if entry["user"] == "alice" && entry["action"] == "notify" {
			aliceNotify = entry
			break
		}
	}
	assert.NotNil(t, aliceNotify)
	assert.Equal(t, "notification sent", aliceNotify["result"])
}

func TestMultipleOutputAttributes(t *testing.T) {
	items := api.WorkItems{
		"token-1": {
			Status: api.WorkSucceeded,
			Inputs: api.Args{
				"user": "alice",
			},
			Outputs: api.Args{
				"email":      "alice@example.com",
				"message_id": "msg-123",
				"status":     "sent",
			},
		},
		"token-2": {
			Status: api.WorkSucceeded,
			Inputs: api.Args{
				"user": "bob",
			},
			Outputs: api.Args{
				"email":      "bob@example.com",
				"message_id": "msg-456",
				"status":     "sent",
			},
		},
	}

	step := &api.Step{
		Attributes: api.AttributeSpecs{
			"user": {Role: api.RoleRequired, ForEach: true},
		},
	}

	output := aggregateWorkItemOutputs(items, step)

	emails, ok := output["email"].([]map[string]any)
	assert.True(t, ok)
	assert.Len(t, emails, 2)

	messageIds, ok := output["message_id"].([]map[string]any)
	assert.True(t, ok)
	assert.Len(t, messageIds, 2)

	statuses, ok := output["status"].([]map[string]any)
	assert.True(t, ok)
	assert.Len(t, statuses, 2)

	for _, entry := range emails {
		assert.Contains(t, entry, "user")
		assert.Contains(t, entry, "email")
		assert.Len(t, entry, 2)
	}
}

func TestNoCompletedItems(t *testing.T) {
	items := api.WorkItems{
		"token-1": {
			Status: api.WorkFailed,
			Inputs: api.Args{"user": "alice"},
		},
		"token-2": {
			Status: api.WorkPending,
			Inputs: api.Args{"user": "bob"},
		},
	}

	step := &api.Step{
		Attributes: api.AttributeSpecs{
			"user": {Role: api.RoleRequired, ForEach: true},
		},
	}

	result := aggregateWorkItemOutputs(items, step)
	assert.Nil(t, result)
}

func TestMixedSuccessAndFailure(t *testing.T) {
	items := api.WorkItems{
		"token-1": {
			Status:  api.WorkSucceeded,
			Inputs:  api.Args{"user": "alice"},
			Outputs: api.Args{"result": "success"},
		},
		"token-2": {
			Status: api.WorkFailed,
			Inputs: api.Args{"user": "bob"},
		},
		"token-3": {
			Status:  api.WorkSucceeded,
			Inputs:  api.Args{"user": "charlie"},
			Outputs: api.Args{"result": "success"},
		},
	}

	step := &api.Step{
		Attributes: api.AttributeSpecs{
			"user": {Role: api.RoleRequired, ForEach: true},
		},
	}

	output := aggregateWorkItemOutputs(items, step)

	results := output["result"].([]map[string]any)
	assert.Len(t, results, 2) // Only alice and charlie

	var users []string
	for _, entry := range results {
		users = append(users, entry["user"].(string))
	}
	assert.Contains(t, users, "alice")
	assert.Contains(t, users, "charlie")
	assert.NotContains(t, users, "bob")
}

func TestNoForEachAttributes(t *testing.T) {
	items := api.WorkItems{
		"token-1": {
			Status: api.WorkSucceeded,
			Inputs: api.Args{
				"input": "value",
			},
			Outputs: api.Args{
				"output": "result",
			},
		},
	}

	step := &api.Step{
		Attributes: api.AttributeSpecs{
			"input": {Role: api.RoleRequired, ForEach: false},
		},
	}

	result := aggregateWorkItemOutputs(items, step)

	assert.Equal(t, "result", result["output"])
}

func TestNilStep(t *testing.T) {
	items := api.WorkItems{
		"token-1": {
			Status:  api.WorkSucceeded,
			Inputs:  api.Args{"input": "value"},
			Outputs: api.Args{"output": "result1"},
		},
		"token-2": {
			Status:  api.WorkSucceeded,
			Inputs:  api.Args{"input": "value"},
			Outputs: api.Args{"output": "result2"},
		},
	}

	result := aggregateWorkItemOutputs(items, nil)

	outputs := result["output"].([]map[string]any)
	assert.Len(t, outputs, 2)

	for _, entry := range outputs {
		assert.Contains(t, entry, "output")
		assert.Len(t, entry, 1)
	}
}

func TestPreventValueCollision(t *testing.T) {
	items := api.WorkItems{
		"token-1": {
			Status: api.WorkSucceeded,
			Inputs: api.Args{
				"value": "input-value-1", // for_each attribute named "value"
			},
			Outputs: api.Args{
				"result": "output-value-1",
			},
		},
		"token-2": {
			Status: api.WorkSucceeded,
			Inputs: api.Args{
				"value": "input-value-2",
			},
			Outputs: api.Args{
				"result": "output-value-2",
			},
		},
	}

	step := &api.Step{
		Attributes: api.AttributeSpecs{
			"value": {Role: api.RoleRequired, ForEach: true},
		},
	}

	output := aggregateWorkItemOutputs(items, step)

	results := output["result"].([]map[string]any)
	assert.Len(t, results, 2)

	for _, entry := range results {
		assert.Contains(t, entry, "value")
		assert.Contains(t, entry, "result")

		inputValue := entry["value"].(string)
		outputValue := entry["result"].(string)
		assert.NotEqual(t, inputValue, outputValue)
		assert.Contains(t, inputValue, "input-value")
		assert.Contains(t, outputValue, "output-value")
	}
}
