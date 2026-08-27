package api_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestSpaceMatches(t *testing.T) {
	space := api.Space{Selector: api.Labels{
		"domain":      "payments",
		"environment": "production",
	}}

	tests := []struct {
		name   string
		labels api.Labels
		match  bool
	}{
		{
			name: "matches all labels",
			labels: api.Labels{
				"domain":      "payments",
				"environment": "production",
				"team":        "core",
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
		Selector: api.Labels{"domain": "payments"},
	})
	assert.NoError(t, err)
	assert.JSONEq(t, `{
		"id": "payments",
		"name": "Payments",
		"selector": {"domain": "payments"}
	}`, string(data))
}
