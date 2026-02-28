package plan_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/engine/plan"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestNoGoals(t *testing.T) {
	cat := makeCatalogState(api.Steps{})

	_, err := plan.Create(cat, []api.StepID{}, api.Args{})
	assert.Error(t, err)
}

func TestGoalStepNotFound(t *testing.T) {
	cat := makeCatalogState(api.Steps{})

	_, err := plan.Create(cat, []api.StepID{"nonexistent"}, api.Args{})
	assert.Error(t, err)
}

func TestSimpleResolver(t *testing.T) {

	resolverStep := &api.Step{
		ID:   "resolver",
		Name: "Resolver",
		Type: api.StepTypeSync,
		Attributes: api.AttributeSpecs{
			"data": {Role: api.RoleOutput, Type: api.TypeString},
		},
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test",
			Timeout:  30 * api.Second,
		},
	}

	cat := makeCatalogState(api.Steps{
		"resolver": resolverStep,
	})
	pl, err := plan.Create(cat, []api.StepID{"resolver"}, api.Args{})
	assert.NoError(t, err)

	assert.Len(t, pl.Goals, 1)
	assert.Equal(t, api.StepID("resolver"), pl.Goals[0])

	assert.Len(t, pl.Steps, 1)
	assert.Contains(t, pl.Steps, api.StepID("resolver"))

	assert.Empty(t, pl.Required)
}

func TestProcessorWithInit(t *testing.T) {

	processorStep := &api.Step{
		ID:   "processor",
		Name: "Processor",
		Type: api.StepTypeSync,
		Attributes: api.AttributeSpecs{
			"input":  {Role: api.RoleRequired, Type: api.TypeString},
			"output": {Role: api.RoleOutput, Type: api.TypeString},
		},
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test",
			Timeout:  30 * api.Second,
		},
	}

	cat := makeCatalogState(api.Steps{
		"processor": processorStep,
	})
	init := api.Args{"input": "test-value"}
	pl, err := plan.Create(cat, []api.StepID{"processor"}, init)
	assert.NoError(t, err)

	assert.Len(t, pl.Steps, 1)
	assert.Empty(t, pl.Required)
}

func TestInitSatisfiedExcluded(t *testing.T) {

	providerStep := &api.Step{
		ID:   "provider",
		Name: "Provider",
		Type: api.StepTypeSync,
		Attributes: api.AttributeSpecs{
			"seed": {Role: api.RoleRequired, Type: api.TypeString},
			"data": {Role: api.RoleOutput, Type: api.TypeString},
		},
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test",
			Timeout:  30 * api.Second,
		},
	}

	consumerStep := &api.Step{
		ID:   "consumer",
		Name: "Consumer",
		Type: api.StepTypeSync,
		Attributes: api.AttributeSpecs{
			"data":   {Role: api.RoleRequired, Type: api.TypeString},
			"result": {Role: api.RoleOutput, Type: api.TypeString},
		},
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test",
			Timeout:  30 * api.Second,
		},
	}

	cat := makeCatalogState(api.Steps{
		"provider": providerStep,
		"consumer": consumerStep,
	})

	pl, err := plan.Create(
		cat, []api.StepID{"consumer"}, api.Args{"data": "ready"},
	)
	assert.NoError(t, err)

	assert.Len(t, pl.Steps, 1)
	assert.NotContains(t, pl.Steps, api.StepID("provider"))
	assert.Contains(t, pl.Steps, api.StepID("consumer"))

	excluded := pl.Excluded
	if assert.NotNil(t, excluded) {
		if assert.Contains(t, excluded.Satisfied, api.StepID("provider")) {
			assert.Equal(t, []api.Name{"data"}, excluded.Satisfied["provider"])
		}
	}
}

func TestProcessorNoInit(t *testing.T) {

	processorStep := &api.Step{
		ID:   "processor",
		Name: "Processor",
		Type: api.StepTypeSync,
		Attributes: api.AttributeSpecs{
			"input":  {Role: api.RoleRequired, Type: api.TypeString},
			"output": {Role: api.RoleOutput, Type: api.TypeString},
		},
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test",
			Timeout:  30 * api.Second,
		},
	}

	cat := makeCatalogState(api.Steps{
		"processor": processorStep,
	})
	pl, err := plan.Create(cat, []api.StepID{"processor"}, api.Args{})
	assert.NoError(t, err)

	assert.Len(t, pl.Required, 1)
	assert.Equal(t, api.Name("input"), pl.Required[0])
}

func TestChained(t *testing.T) {

	resolverStep := &api.Step{
		ID:   "resolver",
		Name: "Resolver",
		Type: api.StepTypeSync,
		Attributes: api.AttributeSpecs{
			"data": {Role: api.RoleOutput, Type: api.TypeString},
		},
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test",
			Timeout:  30 * api.Second,
		},
	}

	processorStep := &api.Step{
		ID:   "processor",
		Name: "Processor",
		Type: api.StepTypeSync,
		Attributes: api.AttributeSpecs{
			"data":   {Role: api.RoleRequired, Type: api.TypeString},
			"result": {Role: api.RoleOutput, Type: api.TypeString},
		},
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test",
			Timeout:  30 * api.Second,
		},
	}

	collectorStep := &api.Step{
		ID:   "collector",
		Name: "Collector",
		Type: api.StepTypeSync,
		Attributes: api.AttributeSpecs{
			"result": {Role: api.RoleRequired, Type: api.TypeString},
		},
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test",
			Timeout:  30 * api.Second,
		},
	}

	cat := makeCatalogState(api.Steps{
		"resolver":  resolverStep,
		"processor": processorStep,
		"collector": collectorStep,
	})
	pl, err := plan.Create(cat, []api.StepID{"collector"}, api.Args{})
	assert.NoError(t, err)

	assert.Len(t, pl.Steps, 3)

	assert.Contains(t, pl.Steps, api.StepID("resolver"))
	assert.Contains(t, pl.Steps, api.StepID("processor"))
	assert.Contains(t, pl.Steps, api.StepID("collector"))

	assert.Empty(t, pl.Required)
}

func TestMultipleGoals(t *testing.T) {

	step1 := &api.Step{
		ID:   "step1",
		Name: "Step 1",
		Type: api.StepTypeSync,
		Attributes: api.AttributeSpecs{
			"output1": {Role: api.RoleOutput, Type: api.TypeString},
		},
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test",
			Timeout:  30 * api.Second,
		},
	}

	step2 := &api.Step{
		ID:   "step2",
		Name: "Step 2",
		Type: api.StepTypeSync,
		Attributes: api.AttributeSpecs{
			"output2": {Role: api.RoleOutput, Type: api.TypeString},
		},
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test",
			Timeout:  30 * api.Second,
		},
	}

	cat := makeCatalogState(api.Steps{
		"step1": step1,
		"step2": step2,
	})
	pl, err := plan.Create(
		cat, []api.StepID{"step1", "step2"}, api.Args{},
	)
	assert.NoError(t, err)

	assert.Len(t, pl.Goals, 2)
	assert.Len(t, pl.Steps, 2)
}

func TestExistingOutputs(t *testing.T) {

	step := &api.Step{
		ID:   "step",
		Name: "Step",
		Type: api.StepTypeSync,
		Attributes: api.AttributeSpecs{
			"data": {Role: api.RoleOutput, Type: api.TypeString},
		},
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test",
			Timeout:  30 * api.Second,
		},
	}

	cat := makeCatalogState(api.Steps{
		"step": step,
	})
	init := api.Args{"data": "already-available"}
	pl, err := plan.Create(cat, []api.StepID{"step"}, init)
	assert.NoError(t, err)

	assert.Empty(t, pl.Steps)
}

func TestComplexGraph(t *testing.T) {

	resolver1 := &api.Step{
		ID:   "resolver1",
		Name: "Resolver 1",
		Type: api.StepTypeSync,
		Attributes: api.AttributeSpecs{
			"a": {Role: api.RoleOutput, Type: api.TypeString},
		},
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test",
			Timeout:  30 * api.Second,
		},
	}

	resolver2 := &api.Step{
		ID:   "resolver2",
		Name: "Resolver 2",
		Type: api.StepTypeSync,
		Attributes: api.AttributeSpecs{
			"b": {Role: api.RoleOutput, Type: api.TypeString},
		},
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test",
			Timeout:  30 * api.Second,
		},
	}

	processor1 := &api.Step{
		ID:   "processor1",
		Name: "Processor 1",
		Type: api.StepTypeSync,
		Attributes: api.AttributeSpecs{
			"a": {Role: api.RoleRequired, Type: api.TypeString},
			"b": {Role: api.RoleRequired, Type: api.TypeString},
			"c": {Role: api.RoleOutput, Type: api.TypeString},
		},
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test",
			Timeout:  30 * api.Second,
		},
	}

	processor2 := &api.Step{
		ID:   "processor2",
		Name: "Processor 2",
		Type: api.StepTypeSync,
		Attributes: api.AttributeSpecs{
			"c": {Role: api.RoleRequired, Type: api.TypeString},
			"d": {Role: api.RoleOutput, Type: api.TypeString},
		},
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test",
			Timeout:  30 * api.Second,
		},
	}

	cat := makeCatalogState(api.Steps{
		"resolver1":  resolver1,
		"resolver2":  resolver2,
		"processor1": processor1,
		"processor2": processor2,
	})
	pl, err := plan.Create(cat, []api.StepID{"processor2"}, api.Args{})
	assert.NoError(t, err)

	assert.Len(t, pl.Steps, 4)

	requiredSteps := []api.StepID{
		"resolver1", "resolver2", "processor1", "processor2",
	}
	for _, stepID := range requiredSteps {
		assert.Contains(t, pl.Steps, stepID)
	}

	assert.Empty(t, pl.Required)
}

func TestReceipts(t *testing.T) {

	step := &api.Step{
		ID:   "step",
		Name: "Step",
		Type: api.StepTypeSync,
		Attributes: api.AttributeSpecs{
			"data": {Role: api.RoleOutput, Type: api.TypeString},
		},
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test",
			Timeout:  30 * api.Second,
		},
	}

	cat := makeCatalogState(api.Steps{
		"step": step,
	})
	pl, err := plan.Create(cat, []api.StepID{"step"}, api.Args{})
	assert.NoError(t, err)

	// Verify plan was created successfully
	assert.NotNil(t, pl)
	assert.Len(t, pl.Steps, 1)
}

func TestMissingDependency(t *testing.T) {

	step := &api.Step{
		ID:   "step",
		Name: "Step",
		Type: api.StepTypeSync,
		Attributes: api.AttributeSpecs{
			"nonexistent": {Role: api.RoleRequired, Type: api.TypeString},
			"output":      {Role: api.RoleOutput, Type: api.TypeString},
		},
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test",
			Timeout:  30 * api.Second,
		},
	}

	cat := makeCatalogState(api.Steps{
		"step": step,
	})
	pl, err := plan.Create(cat, []api.StepID{"step"}, api.Args{})
	assert.NoError(t, err)

	assert.Len(t, pl.Required, 1)
	assert.Equal(t, api.Name("nonexistent"), pl.Required[0])
}

func TestOptionalInput(t *testing.T) {

	resolverStep := &api.Step{
		ID:   "resolver",
		Name: "Resolver",
		Type: api.StepTypeSync,
		Attributes: api.AttributeSpecs{
			"optional_data": {Role: api.RoleOutput, Type: api.TypeString},
		},
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test",
			Timeout:  30 * api.Second,
		},
	}

	processorStep := &api.Step{
		ID:   "processor",
		Name: "Processor",
		Type: api.StepTypeSync,
		Attributes: api.AttributeSpecs{
			"optional_data": {Role: api.RoleOptional, Type: api.TypeString},
			"result":        {Role: api.RoleOutput, Type: api.TypeString},
		},
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test",
			Timeout:  30 * api.Second,
		},
	}

	cat := makeCatalogState(api.Steps{
		"resolver":  resolverStep,
		"processor": processorStep,
	})
	pl, err := plan.Create(cat, []api.StepID{"processor"}, api.Args{})
	assert.NoError(t, err)

	assert.Len(t, pl.Steps, 2)

	assert.Contains(t, pl.Steps, api.StepID("resolver"))
	assert.Contains(t, pl.Steps, api.StepID("processor"))

	assert.Empty(t, pl.Required)
}

func TestOptionalMissing(t *testing.T) {

	processorStep := &api.Step{
		ID:   "processor",
		Name: "Processor",
		Type: api.StepTypeSync,
		Attributes: api.AttributeSpecs{
			"optional_data": {Role: api.RoleOptional, Type: api.TypeString},
			"result":        {Role: api.RoleOutput, Type: api.TypeString},
		},
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test",
			Timeout:  30 * api.Second,
		},
	}

	cat := makeCatalogState(api.Steps{
		"processor": processorStep,
	})
	pl, err := plan.Create(cat, []api.StepID{"processor"}, api.Args{})
	assert.NoError(t, err)

	assert.Len(t, pl.Steps, 1)
	assert.Contains(t, pl.Steps, api.StepID("processor"))

	assert.Empty(t, pl.Required)
}

func TestProvidersWithInit(t *testing.T) {

	providerWithInput := &api.Step{
		ID:   "provider-with-input",
		Name: "Provider With Input",
		Type: api.StepTypeSync,
		Attributes: api.AttributeSpecs{
			"product_id":   {Role: api.RoleRequired, Type: api.TypeString},
			"product_info": {Role: api.RoleOutput, Type: api.TypeString},
		},
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test",
			Timeout:  30 * api.Second,
		},
	}

	providerWithoutInput := &api.Step{
		ID:   "provider-without-input",
		Name: "Provider Without Input",
		Type: api.StepTypeSync,
		Attributes: api.AttributeSpecs{
			"product_info": {Role: api.RoleOutput, Type: api.TypeString},
		},
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test",
			Timeout:  30 * api.Second,
		},
	}

	consumer := &api.Step{
		ID:   "consumer",
		Name: "Consumer",
		Type: api.StepTypeSync,
		Attributes: api.AttributeSpecs{
			"product_info": {Role: api.RoleRequired, Type: api.TypeString},
			"result":       {Role: api.RoleOutput, Type: api.TypeString},
		},
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test",
			Timeout:  30 * api.Second,
		},
	}

	cat := makeCatalogState(api.Steps{
		"provider-with-input":    providerWithInput,
		"provider-without-input": providerWithoutInput,
		"consumer":               consumer,
	})

	withInit, err := plan.Create(
		cat, []api.StepID{"consumer"}, api.Args{"product_id": "123"},
	)
	assert.NoError(t, err)
	assert.Len(t, withInit.Steps, 3)
	assert.Contains(t, withInit.Steps, api.StepID("provider-with-input"))
	assert.Contains(t, withInit.Steps, api.StepID("provider-without-input"))
	assert.Contains(t, withInit.Steps, api.StepID("consumer"))
	assert.Empty(t, withInit.Required)

	withInitExcluded := withInit.Excluded
	if assert.NotNil(t, withInitExcluded) {
		assert.Empty(t, withInitExcluded.Missing)
		assert.Empty(t, withInitExcluded.Satisfied)
	}

	withoutInit, err := plan.Create(
		cat, []api.StepID{"consumer"}, api.Args{},
	)
	assert.NoError(t, err)
	assert.Len(t, withoutInit.Steps, 2)
	assert.NotContains(t, withoutInit.Steps, api.StepID("provider-with-input"))
	assert.Contains(t, withoutInit.Steps, api.StepID("provider-without-input"))
	assert.Contains(t, withoutInit.Steps, api.StepID("consumer"))
	assert.Empty(t, withoutInit.Required)

	withoutInitExcluded := withoutInit.Excluded
	if assert.NotNil(t, withoutInitExcluded) {
		assert.Empty(t, withoutInitExcluded.Missing)
		assert.Empty(t, withoutInitExcluded.Satisfied)
	}
}

func TestMixedInputs(t *testing.T) {

	resolver1 := &api.Step{
		ID:   "resolver1",
		Name: "Resolver 1",
		Type: api.StepTypeSync,
		Attributes: api.AttributeSpecs{
			"required_data": {Role: api.RoleOutput, Type: api.TypeString},
		},
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test",
			Timeout:  30 * api.Second,
		},
	}

	resolver2 := &api.Step{
		ID:   "resolver2",
		Name: "Resolver 2",
		Type: api.StepTypeSync,
		Attributes: api.AttributeSpecs{
			"optional_data": {Role: api.RoleOutput, Type: api.TypeString},
		},
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test",
			Timeout:  30 * api.Second,
		},
	}

	processorStep := &api.Step{
		ID:   "processor",
		Name: "Processor",
		Type: api.StepTypeSync,
		Attributes: api.AttributeSpecs{
			"required_data": {Role: api.RoleRequired, Type: api.TypeString},
			"optional_data": {Role: api.RoleOptional, Type: api.TypeString},
			"result":        {Role: api.RoleOutput, Type: api.TypeString},
		},
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test",
			Timeout:  30 * api.Second,
		},
	}

	cat := makeCatalogState(api.Steps{
		"resolver1": resolver1,
		"resolver2": resolver2,
		"processor": processorStep,
	})
	pl, err := plan.Create(cat, []api.StepID{"processor"}, api.Args{})
	assert.NoError(t, err)

	assert.Len(t, pl.Steps, 3)

	assert.Contains(t, pl.Steps, api.StepID("resolver1"))
	assert.Contains(t, pl.Steps, api.StepID("resolver2"))
	assert.Contains(t, pl.Steps, api.StepID("processor"))

	assert.Empty(t, pl.Required)
}

func makeCatalogState(steps api.Steps) *api.CatalogState {
	graph := api.AttributeGraph{}
	for _, step := range steps {
		graph = graph.AddStep(step)
	}
	return &api.CatalogState{
		Steps:      steps,
		Attributes: graph,
	}
}
