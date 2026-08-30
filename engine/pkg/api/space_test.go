package api_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestSpaceMatches(t *testing.T) {
	space := api.Space{Selector: api.SpaceSelector{
		"domain":      {"payments", "risk"},
		"environment": {"production"},
	}}

	tests := []struct {
		name   string
		labels api.Labels
		match  bool
	}{
		{
			name: "matches first value and all keys",
			labels: api.Labels{
				"domain":      "payments",
				"environment": "production",
				"team":        "core",
			},
			match: true,
		},
		{
			name: "matches alternative value and all keys",
			labels: api.Labels{
				"domain":      "risk",
				"environment": "production",
			},
			match: true,
		},
		{
			name: "rejects different value",
			labels: api.Labels{
				"domain":      "payments",
				"environment": "staging",
			},
		},
		{
			name:   "rejects missing label",
			labels: api.Labels{"domain": "payments"},
		},
		{name: "rejects nil labels"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			step := &api.Step{Labels: test.labels}
			assert.Equal(t, test.match, space.Matches(step))
		})
	}
	assert.True(t, api.Space{}.Matches(&api.Step{}))
}

func TestSpaceJSON(t *testing.T) {
	data, err := json.Marshal(api.Space{
		ID:       "payments",
		Name:     "Payments",
		Selector: api.SpaceSelector{"domain": {"payments", "risk"}},
	})
	assert.NoError(t, err)
	assert.JSONEq(t, `{
		"id": "payments",
		"name": "Payments",
		"selector": {"domain": ["payments", "risk"]}
	}`, string(data))
}

func TestSpaceValidate(t *testing.T) {
	space := api.Space{
		ID:       "payments",
		Name:     "Payments",
		Selector: api.SpaceSelector{"domain": {}},
	}
	assert.ErrorIs(t, space.Validate(), api.ErrInvalidMatchLabels)

	space.Selector = api.SpaceSelector{"domain": {""}}
	assert.ErrorIs(t, space.Validate(), api.ErrInvalidMatchLabels)
}

func TestSpaceEqual(t *testing.T) {
	space := api.Space{
		ID:       "payments",
		Name:     "Payments",
		Selector: api.SpaceSelector{"domain": {"payments", "risk"}},
	}
	other := space
	other.Selector = api.SpaceSelector{"domain": {"risk", "payments"}}
	assert.True(t, space.Equal(other))

	other.Selector = api.SpaceSelector{"domain": {"inventory"}}
	assert.False(t, space.Equal(other))
}

func TestSpaceNormalize(t *testing.T) {
	space := api.Space{
		ID:   "payments",
		Name: "Payments",
		Selector: api.SpaceSelector{
			"domain": {"risk", "payments", "risk"},
		},
	}
	normalized := space.Normalize()
	assert.Equal(t, api.SpaceSelector{
		"domain": {"payments", "risk"},
	}, normalized.Selector)
	assert.Equal(t, []string{"risk", "payments", "risk"},
		space.Selector["domain"],
	)
}
