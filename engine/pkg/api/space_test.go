package api_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestSpaceJSON(t *testing.T) {
	sp := api.Space{
		ID:   "payments",
		Name: "Payments",
		QBE:  api.SpaceQuery{"domain:payments", "domain:risk"},
		Selector: &api.ScriptConfig{
			Language: api.ScriptLangJPath,
			Script:   `$["domain:payments"]`,
		},
	}.Normalize()
	data, err := json.Marshal(sp)
	assert.NoError(t, err)
	assert.JSONEq(t, `{
		"id": "payments",
		"name": "Payments",
		"qbe": ["domain:payments", "domain:risk"],
		"selector": {
			"language": "jpath",
			"script": "$[\"domain:payments\"]"
		}
	}`, string(data))
}

func TestSpaceValidate(t *testing.T) {
	sp := api.Space{
		ID: "payments", Name: "Payments",
		QBE: api.SpaceQuery{"domain:payments"},
		Selector: &api.ScriptConfig{
			Language: api.ScriptLangJPath,
			Script:   `$["domain:payments"]`,
		},
	}.Normalize()
	assert.NoError(t, sp.Validate())

	sp.QBE = api.SpaceQuery{""}
	assert.ErrorIs(t, sp.Validate(), api.ErrInvalidSpaceQuery)

	sp.QBE = api.SpaceQuery{"domain:payments", ""}
	assert.ErrorIs(t, sp.Validate(), api.ErrInvalidSpaceQuery)

	sp.QBE = nil
	assert.NoError(t, sp.Validate())

	sp.Selector = nil
	assert.ErrorIs(t, sp.Validate(), api.ErrSpaceSelectorEmpty)
}

func TestSpaceEqual(t *testing.T) {
	sp := api.Space{
		ID: "payments", Name: "Payments",
		QBE: api.SpaceQuery{"domain:payments", "domain:risk"},
	}.Normalize()
	other := sp
	other.QBE = api.SpaceQuery{"domain:risk", "domain:payments"}
	assert.False(t, sp.Equal(other))
	assert.True(t, sp.Equal(other.Normalize()))

	other = api.Space{
		ID: "payments", Name: "Payments",
		QBE: api.SpaceQuery{"domain:inventory"},
	}.Normalize()
	assert.False(t, sp.Equal(other))
}

func TestSpaceNormalize(t *testing.T) {
	sp := api.Space{
		ID: "payments", Name: "Payments",
		Selector: &api.ScriptConfig{
			Language: api.ScriptLangJPath,
			Script:   `$["domain:payments"]`,
		},
		QBE: api.SpaceQuery{
			"domain:risk", "domain:payments", "domain:risk", "tier:gold",
		},
	}
	normalized := sp.Normalize()
	assert.Equal(t, api.SpaceQuery{
		"domain:payments", "domain:risk", "tier:gold",
	}, normalized.QBE)
	assert.Equal(t, sp.Selector, normalized.Selector)
}

func TestSpaceValidateIdentity(t *testing.T) {
	selector := &api.ScriptConfig{
		Language: api.ScriptLangJPath,
		Script:   `$["domain:payments"]`,
	}

	invalidID := api.Space{
		ID: "not a valid id", Name: "Payments", Selector: selector,
	}
	assert.ErrorIs(t, invalidID.Validate(), api.ErrSpaceIDInvalid)

	noName := api.Space{ID: "payments", Selector: selector}
	assert.ErrorIs(t, noName.Validate(), api.ErrSpaceNameEmpty)
}

func TestSpaceValidateSelectorScript(t *testing.T) {
	noScript := api.Space{
		ID: "payments", Name: "Payments",
		Selector: &api.ScriptConfig{Language: api.ScriptLangJPath},
	}
	assert.ErrorIs(t, noScript.ValidateSelector(), api.ErrScriptEmpty)

	qbe := make(api.SpaceQuery, api.MaxTagCount+1)
	for i := range qbe {
		qbe[i] = fmt.Sprintf("tag:%d", i)
	}
	tooManyTags := api.Space{
		ID: "payments", Name: "Payments", QBE: qbe,
		Selector: &api.ScriptConfig{
			Language: api.ScriptLangJPath,
			Script:   `$["domain:payments"]`,
		},
	}
	assert.ErrorIs(t, tooManyTags.ValidateSelector(), api.ErrTooManyTags)
}
