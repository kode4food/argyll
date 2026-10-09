package helpers_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestMockClient(t *testing.T) {
	client := helpers.NewMockClient()
	assert.NotNil(t, client)
}

func TestSetResponse(t *testing.T) {
	cl := helpers.NewMockClient()

	out := api.Args{"result": "success"}
	cl.SetResponse("step-1", out)

	st := &api.Step{ID: "step-1"}
	result, err := cl.Invoke(st, api.Args{}, api.Metadata{})

	assert.NoError(t, err)
	assert.Equal(t, "success", result["result"])
}

func TestSetError(t *testing.T) {
	cl := helpers.NewMockClient()

	expectedErr := assert.AnError
	cl.SetError("step-error", expectedErr)

	st := &api.Step{ID: "step-error"}
	_, err := cl.Invoke(st, api.Args{}, api.Metadata{})

	assert.Equal(t, expectedErr, err)
}

func TestTracksInvocations(t *testing.T) {
	cl := helpers.NewMockClient()

	step1 := &api.Step{ID: "step-1"}
	step2 := &api.Step{ID: "step-2"}

	_, _ = cl.Invoke(step1, api.Args{}, api.Metadata{})
	_, _ = cl.Invoke(step2, api.Args{}, api.Metadata{})

	assert.True(t, cl.WasInvoked("step-1"))
	assert.True(t, cl.WasInvoked("step-2"))
	assert.False(t, cl.WasInvoked("step-3"))

	invocations := cl.GetInvocations()
	assert.Len(t, invocations, 2)
	assert.Equal(t, api.StepID("step-1"), invocations[0])
	assert.Equal(t, api.StepID("step-2"), invocations[1])
}

func TestDefaultResponse(t *testing.T) {
	cl := helpers.NewMockClient()

	st := &api.Step{ID: "unconfigured-step"}
	result, err := cl.Invoke(st, api.Args{}, api.Metadata{})

	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestThreadSafe(t *testing.T) {
	cl := helpers.NewMockClient()
	cl.SetResponse("step-1", api.Args{"result": "value"})

	done := make(chan bool)
	for range 10 {
		go func() {
			st := &api.Step{ID: "step-1"}
			_, _ = cl.Invoke(st, api.Args{}, api.Metadata{})
			done <- true
		}()
	}

	for range 10 {
		<-done
	}

	assert.True(t, cl.WasInvoked("step-1"))
	invocations := cl.GetInvocations()
	assert.Len(t, invocations, 10)
}

func TestLastMetadata(t *testing.T) {
	cl := helpers.NewMockClient()

	st := &api.Step{ID: "step-with-metadata"}
	md1 := api.Metadata{"attempt": "1"}
	md2 := api.Metadata{"attempt": "2"}
	md3 := api.Metadata{"attempt": "3"}

	_, _ = cl.Invoke(st, api.Args{}, md1)
	_, _ = cl.Invoke(st, api.Args{}, md2)
	_, _ = cl.Invoke(st, api.Args{}, md3)

	last := cl.LastMetadata("step-with-metadata")
	assert.NotNil(t, last)
	assert.Equal(t, "3", last["attempt"])
}

func TestMetadataEmpty(t *testing.T) {
	cl := helpers.NewMockClient()

	last := cl.LastMetadata("never-invoked")
	assert.Nil(t, last)
}
