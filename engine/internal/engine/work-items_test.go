package engine_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/spuds/engine/internal/assert/helpers"
	"github.com/kode4food/spuds/engine/pkg/api"
)

const workItemTimeout = 5 * time.Second

func TestForEachAggregatesOutputs(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()
	env.Engine.Start()

	ctx := context.Background()

	step := &api.Step{
		ID:      "foreach-step",
		Name:    "For Each Step",
		Type:    api.StepTypeSync,
		Version: "1.0.0",
		HTTP: &api.HTTPConfig{
			Endpoint: "http://example.com",
			Timeout:  30 * api.Second,
		},
		Attributes: api.AttributeSpecs{
			"item": {
				Role:    api.RoleRequired,
				Type:    api.TypeArray,
				ForEach: true,
			},
			"result": {
				Role: api.RoleOutput,
				Type: api.TypeString,
			},
		},
	}

	assert.NoError(t, env.Engine.RegisterStep(ctx, step))
	env.MockClient.SetResponse(step.ID, api.Args{"result": "ok"})

	plan := &api.ExecutionPlan{
		Goals: []api.StepID{step.ID},
		Steps: api.Steps{step.ID: step},
	}

	err := env.Engine.StartFlow(
		ctx, "wf-foreach", plan, api.Args{
			"item": []any{"a", "b"},
		}, api.Metadata{},
	)
	assert.NoError(t, err)

	flow := env.WaitForFlowStatus(t, ctx, "wf-foreach", workItemTimeout)
	assert.Equal(t, api.FlowCompleted, flow.Status)

	attrs := flow.GetAttributes()
	rawResults, ok := attrs["result"].([]any)
	assert.True(t, ok)
	assert.Len(t, rawResults, 2)

	results := make([]map[string]any, 0, len(rawResults))
	for _, r := range rawResults {
		entry, ok := r.(map[string]any)
		assert.True(t, ok)
		results = append(results, entry)
	}

	assertContainsEntry(t, results, map[string]any{
		"item":   "a",
		"result": "ok",
	})
	assertContainsEntry(t, results, map[string]any{
		"item":   "b",
		"result": "ok",
	})
}

func assertContainsEntry(
	t *testing.T, entries []map[string]any, expected map[string]any,
) {
	t.Helper()
	for _, entry := range entries {
		if entry["item"] == expected["item"] &&
			entry["result"] == expected["result"] {
			return
		}
	}
	t.Fatalf("expected entry %v not found in %v", expected, entries)
}
