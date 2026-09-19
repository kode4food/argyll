package step_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/step"
)

func TestHandlersWith(t *testing.T) {
	base := &step.Handler{}
	first := &step.Handler{}
	second := &step.Handler{}
	h := step.Handlers{"base": base, "shared": base}

	res := h.With(
		step.Handlers{"shared": first, "first": first},
		step.Handlers{"shared": second},
	)

	assert.Equal(t, step.Handlers{
		"base":   base,
		"shared": second,
		"first":  first,
	}, res)
	assert.Equal(t, step.Handlers{"base": base, "shared": base}, h)
	assert.Equal(t,
		step.Handlers{"first": first},
		step.Handlers(nil).With(step.Handlers{"first": first}))
}
