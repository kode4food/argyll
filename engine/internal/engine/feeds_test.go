package engine_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
)

func newTestFeed() *api.Feed {
	return &api.Feed{
		ID:    "order-total",
		Name:  "Order Total",
		Value: "0",
		Fold: &api.ScriptConfig{
			Language: api.ScriptLangLua,
			Script:   "return feed + amount",
		},
		Attributes: api.Slice[api.Name]{"amount"},
	}
}

// withProvider registers a Step that outputs amount, which a Feed must have
// before it can be registered
func withProvider(t *testing.T, env *helpers.TestEngineEnv) {
	t.Helper()
	charge := helpers.NewStepWithOutputs("charge", "amount")
	assert.NoError(t, env.Engine.RegisterStep(charge))
}

func billingSpace() api.Space {
	return api.Space{
		ID:   "billing",
		Name: "Billing",
		QBE:  api.SpaceQuery{{"domain:billing"}},
	}
}

func TestFeedLifecycle(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		withProvider(t, env)
		f := newTestFeed()
		assert.NoError(t, env.Engine.RegisterFeed(f))
		assert.NoError(t, env.Engine.RegisterFeed(f))

		res, err := env.Engine.GetFeed(f.ID)
		assert.NoError(t, err)
		assert.Equal(t, f.ID, res.Feed.ID)
		assert.EqualValues(t, 0, res.State.Value)
		assert.NotEmpty(t, res.State.Version)

		clash := newTestFeed()
		clash.Value = "1"
		err = env.Engine.RegisterFeed(clash)
		assert.True(t, errors.Is(err, api.ErrFeedExists))

		renamed := newTestFeed()
		renamed.Name = "Renamed"
		assert.NoError(t, env.Engine.UpdateFeed(renamed))
		assert.NoError(t, env.Engine.UpdateFeed(renamed))
		res, err = env.Engine.GetFeed(f.ID)
		assert.NoError(t, err)
		assert.Equal(t, api.Name("Renamed"), res.Feed.Name)

		missing := newTestFeed()
		missing.ID = "missing"
		err = env.Engine.UpdateFeed(missing)
		assert.True(t, errors.Is(err, api.ErrFeedNotFound))

		feeds, err := env.Engine.ListFeeds()
		assert.NoError(t, err)
		assert.Len(t, feeds, 1)
		assert.Equal(t, f.ID, feeds[0].ID)

		assert.NoError(t, env.Engine.UnregisterFeed(f.ID))
		_, err = env.Engine.GetFeed(f.ID)
		assert.True(t, errors.Is(err, api.ErrFeedNotFound))
		err = env.Engine.UnregisterFeed(f.ID)
		assert.True(t, errors.Is(err, api.ErrFeedNotFound))
	})
}

func TestFeedRegisterInvalid(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		withProvider(t, env)
		err := env.Engine.RegisterFeed(nil)
		assert.True(t, errors.Is(err, api.ErrInvalidFeed))

		noAttributes := newTestFeed()
		noAttributes.Attributes = nil
		err = env.Engine.RegisterFeed(noAttributes)
		assert.True(t, errors.Is(err, api.ErrInvalidFeed))

		unknown := newTestFeed()
		unknown.Attributes = api.Slice[api.Name]{"amount", "nobody"}
		err = env.Engine.RegisterFeed(unknown)
		assert.True(t, errors.Is(err, api.ErrInvalidFeed))
		assert.True(t, errors.Is(err, api.ErrFeedAttributeUnknown))

		consumed := newTestFeed()
		reader := helpers.NewSimpleStep("reader")
		reader.Attributes["only_read"] = &api.AttributeSpec{
			Role: api.RoleRequired,
		}
		assert.NoError(t, env.Engine.RegisterStep(reader))
		consumed.Attributes = api.Slice[api.Name]{"only_read"}
		err = env.Engine.RegisterFeed(consumed)
		assert.True(t, errors.Is(err, api.ErrFeedAttributeUnknown))

		badLang := newTestFeed()
		badLang.Fold.Language = "cobol"
		err = env.Engine.RegisterFeed(badLang)
		assert.True(t, errors.Is(err, api.ErrInvalidFeed))

		badSyntax := newTestFeed()
		badSyntax.Fold.Script = "return +"
		err = env.Engine.RegisterFeed(badSyntax)
		assert.True(t, errors.Is(err, api.ErrInvalidFeed))
	})
}

func TestFeedSpaceMustExist(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		withProvider(t, env)
		f := newTestFeed()
		f.SpaceID = "billing"
		err := env.Engine.RegisterFeed(f)
		assert.True(t, errors.Is(err, api.ErrInvalidFeed))
		assert.True(t, errors.Is(err, api.ErrSpaceNotFound))

		assert.NoError(t, env.Engine.RegisterFeed(newTestFeed()))
		err = env.Engine.UpdateFeed(f)
		assert.True(t, errors.Is(err, api.ErrSpaceNotFound))

		assert.NoError(t, env.Engine.RegisterSpace(billingSpace()))
		assert.NoError(t, env.Engine.UpdateFeed(f))
	})
}

func TestStepRequiresDefinedFeed(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		reader := helpers.NewStepWithOutputs("reader", "ran")
		reader.Attributes["total"] = &api.AttributeSpec{
			Role: api.RoleFeed,
			Feed: &api.FeedConfig{ID: "order-total"},
		}
		err := env.Engine.RegisterStep(reader)
		assert.True(t, errors.Is(err, api.ErrInvalidStep))
		assert.True(t, errors.Is(err, api.ErrFeedNotFound))

		withProvider(t, env)
		assert.NoError(t, env.Engine.RegisterFeed(newTestFeed()))
		assert.NoError(t, env.Engine.RegisterStep(reader))

		dangling := helpers.NewStepWithOutputs("reader", "ran")
		dangling.Attributes["total"] = &api.AttributeSpec{
			Role: api.RoleFeed,
			Feed: &api.FeedConfig{ID: "missing"},
		}
		err = env.Engine.UpdateStep(dangling)
		assert.True(t, errors.Is(err, api.ErrFeedNotFound))
	})
}

func TestFeedStateSeq(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		charge := newChargeStep("charge")
		assert.NoError(t, env.Engine.RegisterStep(charge))
		f := newTestFeed()
		assert.NoError(t, env.Engine.RegisterFeed(f))
		key := events.FeedKey(f.ID, f.Version())

		state, seq, err := env.Engine.GetFeedStateSeq(key)
		assert.NoError(t, err)
		assert.Empty(t, state.Version)
		assert.Zero(t, seq)

		env.MockClient.SetResponse(charge.ID, api.Args{"amount": testCharge})
		runChargeFlow(env, "flow-seq", charge.ID)
		state, seq, err = env.Engine.GetFeedStateSeq(key)
		assert.NoError(t, err)
		assert.EqualValues(t, testCharge, state.Value)
		assert.Positive(t, seq)
	})
}

func TestFeedVersioning(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		withProvider(t, env)
		f := newTestFeed()
		assert.NoError(t, env.Engine.RegisterFeed(f))
		before, err := env.Engine.GetFeed(f.ID)
		assert.NoError(t, err)

		changed := newTestFeed()
		changed.Value = "100"
		assert.NoError(t, env.Engine.UpdateFeed(changed))
		after, err := env.Engine.GetFeed(f.ID)
		assert.NoError(t, err)
		assert.NotEqual(t, before.State.Version, after.State.Version)
		assert.EqualValues(t, 100, after.State.Value)
	})
}

func TestFeedUnregisterInUse(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		withProvider(t, env)
		f := newTestFeed()
		assert.NoError(t, env.Engine.RegisterFeed(f))

		reader := helpers.NewSimpleStep("reader")
		reader.Attributes["total"] = &api.AttributeSpec{
			Role: api.RoleFeed,
			Feed: &api.FeedConfig{ID: f.ID},
		}
		assert.NoError(t, env.Engine.RegisterStep(reader))

		err := env.Engine.UnregisterFeed(f.ID)
		assert.True(t, errors.Is(err, engine.ErrFeedInUse))

		assert.NoError(t, env.Engine.UnregisterStep(reader.ID))
		assert.NoError(t, env.Engine.UnregisterFeed(f.ID))
	})
}
