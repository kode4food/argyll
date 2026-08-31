package api_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestSpaceJSON(t *testing.T) {
	space := api.Space{
		ID:   "payments",
		Name: "Payments",
		QBE:  api.SpaceQuery{"domain": {"payments", "risk"}},
		Selector: &api.ScriptConfig{
			Language: api.ScriptLangJPath,
			Script:   `$.domain == "payments"`,
		},
	}.Normalize()
	data, err := json.Marshal(space)
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
	space := api.Space{
		ID: "payments", Name: "Payments",
		QBE: api.SpaceQuery{"domain": {}},
		Selector: &api.ScriptConfig{
			Language: api.ScriptLangJPath,
			Script:   `$.domain == "payments"`,
		},
	}.Normalize()
	assert.ErrorIs(t, space.Validate(), api.ErrInvalidSpaceQuery)

	space.QBE = api.SpaceQuery{"domain": {""}}
	assert.ErrorIs(t, space.Validate(), api.ErrInvalidSpaceQuery)

	space.QBE = nil
	assert.NoError(t, space.Validate())

	space.Selector = nil
	assert.ErrorIs(t, space.Validate(), api.ErrSpaceSelectorEmpty)
}

func TestSpaceEqual(t *testing.T) {
	space := api.Space{
		ID: "payments", Name: "Payments",
		QBE: api.SpaceQuery{"domain": {"payments", "risk"}},
	}.Normalize()
	other := space
	other.QBE = api.SpaceQuery{"domain": {"risk", "payments"}}
	assert.False(t, space.Equal(other))
	assert.True(t, space.Equal(other.Normalize()))

	other = api.Space{
		ID: "payments", Name: "Payments",
		QBE: api.SpaceQuery{"domain": {"inventory"}},
	}.Normalize()
	assert.False(t, space.Equal(other))
}

func TestSpaceNormalize(t *testing.T) {
	space := api.Space{
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
	normalized := space.Normalize()
	assert.Equal(t, api.SpaceQuery{
		"domain": {"payments", "risk"},
		"tier":   {"gold"},
	}, normalized.QBE)
	assert.Equal(t, space.Selector, normalized.Selector)
	assert.Equal(t,
		[]string{"risk", "payments", "risk"}, space.QBE["domain"])
}
