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
		space := api.Space{
			ID:   "payments",
			Name: "Payments",
			Selector: api.LabelSelector{MatchLabels: api.Labels{
				"domain": "payments",
			}},
		}
		assert.NoError(t, eng.RegisterSpace(space))
		assert.NoError(t, eng.RegisterSpace(space))

		cat, err := eng.GetCatalogState()
		assert.NoError(t, err)
		assert.Equal(t, space, cat.Spaces[space.ID])

		spaces, err := eng.ListSpaces()
		assert.NoError(t, err)
		assert.Equal(t, []api.Space{space}, spaces)

		updated := api.Space{
			ID:          "payments",
			Name:        "Payments",
			Description: "Production payments",
			Selector: api.LabelSelector{MatchLabels: api.Labels{
				"domain":      "payments",
				"environment": "production",
			}},
		}
		_, before, err := eng.GetCatalogStateSeq()
		assert.NoError(t, err)
		assert.NoError(t, eng.UpdateSpace(updated))
		_, after, err := eng.GetCatalogStateSeq()
		assert.NoError(t, err)
		assert.Equal(t, before+1, after)

		cat, err = eng.GetCatalogState()
		assert.NoError(t, err)
		assert.Equal(t, updated, cat.Spaces[space.ID])

		assert.NoError(t, eng.UnregisterSpace(space.ID))
		cat, err = eng.GetCatalogState()
		assert.NoError(t, err)
		assert.NotContains(t, cat.Spaces, space.ID)
	})
}

func TestSpacesRejectInvalid(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		err := eng.RegisterSpace(api.Space{})
		assert.ErrorIs(t, err, engine.ErrInvalidSpace)
		assert.ErrorIs(t, err, api.ErrSpaceIDEmpty)

		err = eng.UpdateSpace(api.Space{ID: "missing", Name: "Missing"})
		assert.ErrorIs(t, err, engine.ErrSpaceNotFound)

		err = eng.UnregisterSpace("missing")
		assert.ErrorIs(t, err, engine.ErrSpaceNotFound)
	})
}

func TestSpaceMembershipEvents(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		st := helpers.NewSimpleStep("existing")
		st.Labels = api.Labels{
			"domain":      "payments",
			"environment": "production",
		}
		assert.NoError(t, eng.RegisterStep(st))
		cat, err := eng.GetCatalogState()
		assert.NoError(t, err)
		st = cat.Steps[st.ID]

		domain := api.Space{
			ID:   "z-domain",
			Name: "Domain",
			Selector: api.LabelSelector{MatchLabels: api.Labels{
				"domain": "payments",
			}},
		}
		assert.NoError(t, eng.RegisterSpace(domain))
		evs, err := eng.GetCatalogEvents()
		assert.NoError(t, err)
		registered, err := evs[len(evs)-1].
			GetValue[api.SpaceRegisteredEvent]()
		assert.NoError(t, err)
		assert.Equal(t, api.Steps{st.ID: st}, registered.Steps)

		environment := api.Space{
			ID:   "a-environment",
			Name: "Environment",
			Selector: api.LabelSelector{MatchLabels: api.Labels{
				"environment": "production",
			}},
		}
		assert.NoError(t, eng.RegisterSpace(environment))

		added := helpers.NewSimpleStep("added")
		added.Labels = st.Labels
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
		updated.Labels = api.Labels{
			"domain":      "payments",
			"environment": "staging",
		}
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
			Selector: api.LabelSelector{MatchLabels: api.Labels{
				"environment": "production",
			}},
		}
		assert.NoError(t, eng.UpdateSpace(domain))
		evs, err = eng.GetCatalogEvents()
		assert.NoError(t, err)
		spaceUpdated, err := evs[len(evs)-1].
			GetValue[api.SpaceUpdatedEvent]()
		assert.NoError(t, err)
		assert.Equal(t, api.Steps{st.ID: st}, spaceUpdated.Steps)
	})
}
