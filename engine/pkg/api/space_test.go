package api_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestLabelSelector(t *testing.T) {
	selector := api.LabelSelector{MatchLabels: api.Labels{
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
			assert.Equal(t, test.match, selector.Matches(test.labels))
		})
	}
	assert.True(t, (api.LabelSelector{}).Matches(nil))
}

func TestLabelSelectorJSON(t *testing.T) {
	data, err := json.Marshal(api.LabelSelector{MatchLabels: api.Labels{
		"domain": "payments",
	}})
	assert.NoError(t, err)
	assert.JSONEq(t, `{"match_labels":{"domain":"payments"}}`, string(data))
}
