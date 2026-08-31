package api_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestSpaceJSON(t *testing.T) {
	sp := api.Space{
		ID:   "payments",
		Name: "Payments",
		QBE:  api.SpaceQuery{"domain": {"payments", "risk"}},
		Selector: &api.ScriptConfig{
			Language: api.ScriptLangJPath,
			Script:   `$.domain == "payments"`,
		},
	}.Normalize()
	data, err := json.Marshal(sp)
	assert.NoError(t, err)
	assert.JSONEq(t, `{
		"id": "payments",
		"name": "Payments",
		"qbe": {"domain": ["payments", "risk"]},
		"selector": {
			"language": "jpath",
			"script": "$.domain == \"payments\""
		}
	}`, string(data))
}

func TestSpaceValidate(t *testing.T) {
	sp := api.Space{
		ID: "payments", Name: "Payments",
		QBE: api.SpaceQuery{"domain": {}},
		Selector: &api.ScriptConfig{
			Language: api.ScriptLangJPath,
			Script:   `$.domain == "payments"`,
		},
	}.Normalize()
	assert.ErrorIs(t, sp.Validate(), api.ErrInvalidSpaceQuery)

	sp.QBE = api.SpaceQuery{"domain": {""}}
	assert.ErrorIs(t, sp.Validate(), api.ErrInvalidSpaceQuery)

	sp.QBE = nil
	assert.NoError(t, sp.Validate())

	sp.Selector = nil
	assert.ErrorIs(t, sp.Validate(), api.ErrSpaceSelectorEmpty)
}

func TestSpaceEqual(t *testing.T) {
	sp := api.Space{
		ID: "payments", Name: "Payments",
		QBE: api.SpaceQuery{"domain": {"payments", "risk"}},
	}.Normalize()
	other := sp
	other.QBE = api.SpaceQuery{"domain": {"risk", "payments"}}
	assert.False(t, sp.Equal(other))
	assert.True(t, sp.Equal(other.Normalize()))

	other = api.Space{
		ID: "payments", Name: "Payments",
		QBE: api.SpaceQuery{"domain": {"inventory"}},
	}.Normalize()
	assert.False(t, sp.Equal(other))
}

func TestSpaceNormalize(t *testing.T) {
	sp := api.Space{
		ID: "payments", Name: "Payments",
		Selector: &api.ScriptConfig{
			Language: api.ScriptLangJPath,
			Script:   `$.domain == "payments"`,
		},
		QBE: api.SpaceQuery{
			"domain": {"risk", "payments", "risk"},
			"tier":   {"gold"},
		},
	}
	normalized := sp.Normalize()
	assert.Equal(t, api.SpaceQuery{
		"domain": {"payments", "risk"},
		"tier":   {"gold"},
	}, normalized.QBE)
	assert.Equal(t, sp.Selector, normalized.Selector)
	assert.Equal(t,
		[]string{"risk", "payments", "risk"}, sp.QBE["domain"])
}
