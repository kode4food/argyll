package api_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestCreateFlowRequestValidate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		req := &api.CreateFlowRequest{
			ID:    "my-flow",
			Goals: api.Goals{Steps: []api.StepID{"step-1"}},
		}
		assert.NoError(t, req.Validate())
	})

	t.Run("empty id", func(t *testing.T) {
		req := &api.CreateFlowRequest{
			Goals: api.Goals{Steps: []api.StepID{"step-1"}},
		}
		assert.ErrorIs(t, req.Validate(), api.ErrFlowIDEmpty)
	})

	t.Run("invalid id", func(t *testing.T) {
		req := &api.CreateFlowRequest{
			ID:    "my:flow",
			Goals: api.Goals{Steps: []api.StepID{"step-1"}},
		}
		assert.ErrorIs(t, req.Validate(), api.ErrFlowIDInvalid)
	})

	t.Run("no goals", func(t *testing.T) {
		req := &api.CreateFlowRequest{
			ID: "my-flow",
		}
		assert.ErrorIs(t, req.Validate(), api.ErrGoalsRequired)
	})

	t.Run("empty goal set", func(t *testing.T) {
		req := &api.CreateFlowRequest{
			ID: "my-flow",
			Goals: api.Goals{
				Steps: []api.StepID{"step-1"},
				Else:  &api.Goals{},
			},
		}
		assert.ErrorIs(t, req.Validate(), api.ErrGoalsRequired)
	})

	t.Run("array goals JSON rejected", func(t *testing.T) {
		for _, goals := range []string{`["step-1"]`, `[["step-1"]]`} {
			var req api.CreateFlowRequest
			err := json.Unmarshal(
				[]byte(`{"id":"my-flow","goals":`+goals+`}`), &req,
			)
			assert.Error(t, err)
		}
	})

	t.Run("chained goals JSON", func(t *testing.T) {
		var req api.CreateFlowRequest
		err := json.Unmarshal([]byte(`{"id":"my-flow","goals":`+
			`{"steps":["a","b"],"else":{"steps":["c"]}}}`), &req,
		)
		assert.NoError(t, err)
		assert.Equal(t,
			[][]api.StepID{{"a", "b"}, {"c"}}, req.Goals.Sets())
	})

	t.Run("ID too long", func(t *testing.T) {
		req := &api.CreateFlowRequest{
			ID:    api.FlowID(strings.Repeat("a", api.MaxFlowIDLen+1)),
			Goals: api.Goals{Steps: []api.StepID{"step-1"}},
		}
		assert.ErrorIs(t, req.Validate(), api.ErrFlowIDTooLong)
	})

	t.Run("too many goals", func(t *testing.T) {
		goals := make([]api.StepID, api.MaxGoalCount+1)
		for i := range goals {
			goals[i] = api.StepID(fmt.Sprintf("step-%d", i))
		}
		req := &api.CreateFlowRequest{
			ID:    "my-flow",
			Goals: api.Goals{Steps: goals},
		}
		assert.ErrorIs(t, req.Validate(), api.ErrTooManyGoals)
	})

	t.Run("goals repeated across sets count once", func(t *testing.T) {
		goals := make([]api.StepID, api.MaxGoalCount)
		for i := range goals {
			goals[i] = api.StepID(fmt.Sprintf("step-%d", i))
		}
		req := &api.CreateFlowRequest{
			ID:    "my-flow",
			Goals: api.Goals{Steps: goals, Else: &api.Goals{Steps: goals}},
		}
		assert.NoError(t, req.Validate())
	})

	t.Run("too many init keys", func(t *testing.T) {
		init := api.InitArgs{}
		for i := range api.MaxInitKeys + 1 {
			init[api.Name(fmt.Sprintf("key-%d", i))] = []any{"value"}
		}
		req := &api.CreateFlowRequest{
			ID:    "my-flow",
			Goals: api.Goals{Steps: []api.StepID{"step-1"}},
			Init:  init,
		}
		assert.ErrorIs(t, req.Validate(), api.ErrTooManyInit)
	})

	t.Run("too many tags", func(t *testing.T) {
		tags := make(api.Tags, api.MaxTagCount+1)
		for i := range tags {
			tags[i] = fmt.Sprintf("tag-%d", i)
		}
		req := &api.CreateFlowRequest{
			ID:    "my-flow",
			Goals: api.Goals{Steps: []api.StepID{"step-1"}},
			Tags:  tags,
		}
		assert.ErrorIs(t, req.Validate(), api.ErrTooManyTags)
	})

	t.Run("empty tag", func(t *testing.T) {
		req := &api.CreateFlowRequest{
			ID:    "my-flow",
			Goals: api.Goals{Steps: []api.StepID{"step-1"}},
			Tags:  api.Tags{""},
		}
		assert.ErrorIs(t, req.Validate(), api.ErrTagEmpty)
	})
}
