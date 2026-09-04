package engine_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestSpaces(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		sp := api.Space{
			ID:   "payments",
			Name: "Payments",
			QBE:  api.SpaceQuery{{"domain:payments"}},
		}
		assert.NoError(t, eng.RegisterSpace(sp))
		assert.NoError(t, eng.RegisterSpace(sp))

		cat, err := eng.GetCatalogState()
		assert.NoError(t, err)
		stored := cat.Spaces[sp.ID]
		assert.Equal(t, sp.QBE, stored.QBE)
		assert.Equal(t, &api.ScriptConfig{
			Language: api.ScriptLangJPath,
			Script:   `$.tags[?@=="domain:payments"]`,
		}, stored.Selector)

		spaces, err := eng.ListSpaces()
		assert.NoError(t, err)
		assert.Equal(t, []api.Space{stored}, spaces)

		updated := api.Space{
			ID:          "payments",
			Name:        "Payments",
			Description: "Production payments",
			QBE: api.SpaceQuery{
				{"domain:payments", "environment:production"},
			},
		}
		_, before, err := eng.GetCatalogStateSeq()
		assert.NoError(t, err)
		assert.NoError(t, eng.UpdateSpace(updated))
		_, after, err := eng.GetCatalogStateSeq()
		assert.NoError(t, err)
		assert.Equal(t, before+1, after)

		cat, err = eng.GetCatalogState()
		assert.NoError(t, err)
		assert.Equal(t, updated.QBE, cat.Spaces[sp.ID].QBE)
		assert.Equal(t, updated.Description,
			cat.Spaces[sp.ID].Description)

		assert.NoError(t, eng.UnregisterSpace(sp.ID))
		cat, err = eng.GetCatalogState()
		assert.NoError(t, err)
		assert.NotContains(t, cat.Spaces, sp.ID)
	})
}

func TestSpaceScriptSelectors(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		risk := helpers.NewSimpleStep("risk")
		risk.Tags = api.Tags{"domain:risk"}
		risk.Attributes["score"] = &api.AttributeSpec{
			Role: api.RoleOutput,
			Type: api.TypeNumber,
		}
		trading := helpers.NewSimpleStep("trading")
		trading.Tags = api.Tags{"domain:trading"}
		assert.NoError(t, eng.RegisterStep(risk))
		assert.NoError(t, eng.RegisterStep(trading))

		luaSpace := api.Space{
			ID: "lua", Name: "Lua",
			Selector: &api.ScriptConfig{
				Language: api.ScriptLangLua,
				Script: `local score = value.attributes.score
return has(value.tags, "domain:risk") and score and
    value.type == "service" and value.handling == "standard" and
    score.role == "output" and score.type == "number" and
    score.compensated == false`,
			},
		}
		jpathSpace := api.Space{
			ID: "jpath", Name: "JPath",
			Selector: &api.ScriptConfig{
				Language: api.ScriptLangJPath,
				Script:   `$.tags[?@=="domain:trading"]`,
			},
		}
		assert.NoError(t, eng.RegisterSpace(luaSpace))
		assert.NoError(t, eng.RegisterSpace(jpathSpace))

		cat, err := eng.GetCatalogState()
		assert.NoError(t, err)
		assert.Equal(t, api.Steps{risk.ID: cat.Steps[risk.ID]},
			cat.SpaceSteps(luaSpace.ID))
		assert.Equal(t, api.Steps{trading.ID: cat.Steps[trading.ID]},
			cat.SpaceSteps(jpathSpace.ID))
	})
}

func TestSpaceMetadataSelectors(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		risk := helpers.NewSimpleStep("risk")
		risk.Attributes["risk_score"] = &api.AttributeSpec{
			Role: api.RoleOutput,
			Type: api.TypeNumber,
		}
		reversible := helpers.NewSimpleStep("reversible")
		reversible.Handling = api.HandlingCompensated
		reversible.HTTP.Compensate = &api.HTTPAction{
			Endpoint: "http://test:8080/compensate",
		}
		assert.NoError(t, eng.RegisterStep(risk))
		assert.NoError(t, eng.RegisterStep(reversible))

		cases := []struct {
			name     string
			script   string
			expected []api.StepID
		}{
			{
				name:     "handling",
				script:   `$.type == "service" && $.handling == "compensated"`,
				expected: []api.StepID{reversible.ID},
			},
			{
				name: "attribute contract",
				script: `$.attributes.risk_score.role == "output" &&
    $.attributes.risk_score.type == "number"`,
				expected: []api.StepID{risk.ID},
			},
		}
		for _, tt := range cases {
			t.Run(tt.name, func(t *testing.T) {
				preview, err := eng.PreviewSpace(api.Space{
					Selector: &api.ScriptConfig{
						Language: api.ScriptLangJPath,
						Script:   tt.script,
					},
				})
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, preview.StepIDs)
			})
		}
	})
}

func TestSpaceMarketSelector(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		steps := []*api.Step{
			helpers.NewSimpleStep("europe"),
			helpers.NewSimpleStep("all"),
			helpers.NewSimpleStep("unclassified"),
			helpers.NewSimpleStep("us"),
		}
		steps[0].Tags = api.Tags{"market:europe"}
		steps[1].Tags = api.Tags{"market:all"}
		steps[2].Tags = api.Tags{"domain:fulfillment"}
		steps[3].Tags = api.Tags{"market:us"}
		for _, st := range steps {
			assert.NoError(t, eng.RegisterStep(st))
		}

		preview, err := eng.PreviewSpace(api.Space{
			Selector: &api.ScriptConfig{
				Language: api.ScriptLangJPath,
				Script: `$.tags[?@=="market:europe" || @=="market:all"] ||
    !$.tags[?search(@, "^market:")]`,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, []api.StepID{"all", "europe", "unclassified"},
			preview.StepIDs)
	})
}

func TestPreviewSpace(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		risk := helpers.NewSimpleStep("risk")
		risk.Tags = api.Tags{"domain:risk"}
		trading := helpers.NewSimpleStep("trading")
		trading.Tags = api.Tags{"domain:trading"}
		assert.NoError(t, eng.RegisterStep(risk))
		assert.NoError(t, eng.RegisterStep(trading))

		// A Space being drafted has no ID or Name yet
		preview, err := eng.PreviewSpace(api.Space{
			QBE: api.SpaceQuery{{"domain:risk"}, {"domain:risk"}},
		})
		assert.NoError(t, err)
		assert.Equal(t, []api.StepID{risk.ID}, preview.StepIDs)
		assert.Equal(t, api.SpaceQuery{{"domain:risk"}}, preview.Space.QBE)
		assert.Equal(t, &api.ScriptConfig{
			Language: api.ScriptLangJPath,
			Script:   `$.tags[?@=="domain:risk"]`,
		}, preview.Space.Selector)

		preview, err = eng.PreviewSpace(api.Space{
			Selector: &api.ScriptConfig{
				Language: api.ScriptLangLua,
				Script:   `return has(value.tags, "unknown")`,
			},
		})
		assert.NoError(t, err)
		assert.Empty(t, preview.StepIDs)

		_, err = eng.PreviewSpace(api.Space{})
		assert.ErrorIs(t, err, engine.ErrInvalidSpace)
	})
}

func TestSpacesRejectInvalid(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		err := eng.RegisterSpace(api.Space{})
		assert.ErrorIs(t, err, engine.ErrInvalidSpace)
		assert.ErrorIs(t, err, api.ErrSpaceSelectorEmpty)

		err = eng.RegisterSpace(api.Space{
			Name: "No ID",
			QBE:  api.SpaceQuery{{"domain:payments"}},
		})
		assert.ErrorIs(t, err, engine.ErrInvalidSpace)
		assert.ErrorIs(t, err, api.ErrSpaceIDEmpty)

		err = eng.UpdateSpace(api.Space{
			ID:   "missing",
			Name: "Missing",
			QBE:  api.SpaceQuery{{"domain:payments"}},
		})
		assert.ErrorIs(t, err, engine.ErrSpaceNotFound)

		err = eng.RegisterSpace(api.Space{
			ID: "invalid-script", Name: "Invalid Script",
			Selector: &api.ScriptConfig{
				Language: api.ScriptLangLua,
				Script:   "return value[",
			},
		})
		assert.ErrorIs(t, err, engine.ErrInvalidSpace)

		err = eng.RegisterSpace(api.Space{
			ID: "unknown-language", Name: "Unknown Language",
			Selector: &api.ScriptConfig{
				Language: "prolog",
				Script:   "labels(risk).",
			},
		})
		assert.ErrorIs(t, err, engine.ErrInvalidSpace)
		assert.ErrorIs(t, err, api.ErrInvalidScriptLanguage)

		err = eng.UnregisterSpace("missing")
		assert.ErrorIs(t, err, engine.ErrSpaceNotFound)
	})
}

func TestSpaceSelectionEvents(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		st := helpers.NewSimpleStep("existing")
		st.Tags = api.Tags{"domain:payments", "environment:production"}
		assert.NoError(t, eng.RegisterStep(st))
		cat, err := eng.GetCatalogState()
		assert.NoError(t, err)
		st = cat.Steps[st.ID]

		domain := api.Space{
			ID:   "z-domain",
			Name: "Domain",
			QBE:  api.SpaceQuery{{"domain:payments"}},
		}
		assert.NoError(t, eng.RegisterSpace(domain))
		evs, err := eng.GetCatalogEvents()
		assert.NoError(t, err)
		registered, err := evs[len(evs)-1].
			GetValue[api.SpaceRegisteredEvent]()
		assert.NoError(t, err)
		assert.Equal(t, []api.StepID{st.ID}, registered.StepIDs)

		environment := api.Space{
			ID:   "a-environment",
			Name: "Environment",
			QBE:  api.SpaceQuery{{"environment:production"}},
		}
		assert.NoError(t, eng.RegisterSpace(environment))

		added := helpers.NewSimpleStep("added")
		added.Tags = st.Tags
		assert.NoError(t, eng.RegisterStep(added))
		evs, err = eng.GetCatalogEvents()
		assert.NoError(t, err)
		stepRegistered, err := evs[len(evs)-1].
			GetValue[api.StepRegisteredEvent]()
		assert.NoError(t, err)
		assert.Equal(t,
			[]api.SpaceID{"a-environment", "z-domain"},
			stepRegistered.Spaces)

		updated := added.Copy()
		updated.Tags = api.Tags{"domain:payments", "environment:staging"}
		assert.NoError(t, eng.UpdateStep(updated))
		evs, err = eng.GetCatalogEvents()
		assert.NoError(t, err)
		stepUpdated, err := evs[len(evs)-1].
			GetValue[api.StepUpdatedEvent]()
		assert.NoError(t, err)
		assert.Equal(t, []api.SpaceID{"z-domain"}, stepUpdated.Spaces)

		assert.NoError(t, eng.UnregisterStep(updated.ID))
		evs, err = eng.GetCatalogEvents()
		assert.NoError(t, err)
		unregistered, err := evs[len(evs)-1].
			GetValue[api.StepUnregisteredEvent]()
		assert.NoError(t, err)
		assert.Equal(t, []api.SpaceID{"z-domain"}, unregistered.Spaces)

		domain = api.Space{
			ID:   domain.ID,
			Name: domain.Name,
			QBE:  api.SpaceQuery{{"environment:production"}},
		}
		assert.NoError(t, eng.UpdateSpace(domain))
		evs, err = eng.GetCatalogEvents()
		assert.NoError(t, err)
		spaceUpdated, err := evs[len(evs)-1].
			GetValue[api.SpaceUpdatedEvent]()
		assert.NoError(t, err)
		assert.Equal(t, []api.StepID{st.ID}, spaceUpdated.StepIDs)
	})
}
