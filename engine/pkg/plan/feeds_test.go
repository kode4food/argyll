package plan_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/plan"
)

func feedPlan(cat api.CatalogState, goals ...api.StepID) (*api.ExecutionPlan, error) {
	return plan.Create(&plan.Request{
		Match:   testEval,
		Catalog: cat,
		Steps:   cat.Steps,
		Goals:   api.Goals{Steps: goals},
	})
}

func TestPlanFeeds(t *testing.T) {
	charge := planProvider("charge", "amount")
	refund := planProvider("refund", "amount")
	reader := planProvider("reader", "ran")
	reader.Attributes["total"] = &api.AttributeSpec{
		Role: api.RoleFeed,
		Feed: &api.FeedConfig{ID: "order-total"},
	}
	unrelated := planProvider("unrelated", "x")
	total := &api.Feed{
		ID:         "order-total",
		Attributes: api.Slice[api.Name]{"amount"},
	}
	idle := &api.Feed{
		ID:         "idle",
		Attributes: api.Slice[api.Name]{"missing"},
	}

	t.Run("includes consumed and sourced Feeds", func(t *testing.T) {
		cat := makeCatalogState(api.Steps{
			"charge":    charge,
			"reader":    reader,
			"unrelated": unrelated,
		})
		cat.Feeds = api.Feeds{
			total.ID: total,
			idle.ID:  idle,
		}

		pl, err := feedPlan(cat, "charge", "reader")
		assert.NoError(t, err)
		sourced := api.PlanFeed{
			Feed:    total,
			Sources: []api.StepID{"charge"},
		}
		assert.Equal(t, api.PlanFeeds{total.ID: sourced}, pl.Feeds)
	})

	t.Run("includes a read Feed without sources", func(t *testing.T) {
		cat := makeCatalogState(api.Steps{"reader": reader})
		cat.Feeds = api.Feeds{total.ID: total}

		pl, err := feedPlan(cat, "reader")
		assert.NoError(t, err)
		read := api.PlanFeed{Feed: total}
		assert.Equal(t, api.PlanFeeds{total.ID: read}, pl.Feeds)
	})

	t.Run("sources every provider of the Attribute", func(t *testing.T) {
		cat := makeCatalogState(api.Steps{
			"charge": charge,
			"refund": refund,
		})
		cat.Feeds = api.Feeds{total.ID: total}

		pl, err := feedPlan(cat, "charge", "refund")
		assert.NoError(t, err)
		sources := api.Slice[api.StepID]{"charge", "refund"}
		assert.Equal(t, sources, pl.Feeds[total.ID].Sources)
	})

	t.Run("restricts sources to the Feed's Space", func(t *testing.T) {
		cat := makeCatalogState(api.Steps{
			"charge": charge,
			"refund": refund,
		})
		spaced := &api.Feed{
			ID:         "order-total",
			Attributes: api.Slice[api.Name]{"amount"},
			SpaceID:    "billing",
		}
		cat.Feeds = api.Feeds{spaced.ID: spaced}
		cat.Spaces.Selected = api.Map[api.SpaceID, api.Slice[api.StepID]]{
			"billing": {"refund"},
		}

		pl, err := feedPlan(cat, "charge", "refund")
		assert.NoError(t, err)
		assert.Equal(t, api.Slice[api.StepID]{"refund"}, pl.Feeds[spaced.ID].Sources)

		pl, err = feedPlan(cat, "charge")
		assert.NoError(t, err)
		assert.Nil(t, pl.Feeds)
	})

	t.Run("omits Feeds when none apply", func(t *testing.T) {
		cat := makeCatalogState(api.Steps{"unrelated": unrelated})
		cat.Feeds = api.Feeds{total.ID: total}

		pl, err := feedPlan(cat, "unrelated")
		assert.NoError(t, err)
		assert.Nil(t, pl.Feeds)
	})
}
