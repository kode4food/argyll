package api_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
)

func newFeed() *api.Feed {
	return &api.Feed{
		ID:    "order-total",
		Name:  "Order Total",
		Value: "0",
		Fold: &api.ScriptConfig{
			Language: api.ScriptLangLua,
			Source:   "return feed + amount",
		},
		Attributes: api.Slice[api.Name]{"amount"},
	}
}

func TestFeedValidate(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*api.Feed)
		want   error
	}{
		{"valid", func(*api.Feed) {}, nil},
		{"empty ID", func(f *api.Feed) { f.ID = "" }, api.ErrFeedIDEmpty},
		{"bad ID", func(f *api.Feed) { f.ID = "Bad ID" }, api.ErrFeedIDInvalid},
		{"empty name", func(f *api.Feed) { f.Name = "" }, api.ErrFeedNameEmpty},
		{
			"empty value",
			func(f *api.Feed) { f.Value = "" },
			api.ErrFeedValueRequired,
		},
		{
			"bad value",
			func(f *api.Feed) { f.Value = "{" },
			api.ErrFeedValueRequired,
		},
		{"nil fold", func(f *api.Feed) { f.Fold = nil }, api.ErrFeedFoldEmpty},
		{
			"empty fold",
			func(f *api.Feed) { f.Fold.Source = "" },
			api.ErrFeedFoldEmpty,
		},
		{
			"no attributes",
			func(f *api.Feed) { f.Attributes = nil },
			api.ErrFeedAttributesEmpty,
		},
		{
			"empty attribute",
			func(f *api.Feed) { f.Attributes = api.Slice[api.Name]{""} },
			api.ErrFeedAttributeEmpty,
		},
		{
			"reserved attribute",
			func(f *api.Feed) {
				f.Attributes = api.Slice[api.Name]{api.FeedCurrent}
			},
			api.ErrFeedAttributeReserved,
		},
		{
			"repeated attribute",
			func(f *api.Feed) {
				f.Attributes = api.Slice[api.Name]{"amount", "amount"}
			},
			api.ErrFeedAttributeRepeated,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFeed()
			tt.mutate(f)
			err := f.Validate()
			if tt.want == nil {
				assert.NoError(t, err)
				return
			}
			assert.True(t, errors.Is(err, api.ErrInvalidFeed))
			assert.True(t, errors.Is(err, tt.want))
		})
	}
}

func TestFeedVersion(t *testing.T) {
	b := newFeed()
	vb := b.Version()
	assert.True(t, b.Equal(newFeed()))

	t.Run("name and description keep the version", func(t *testing.T) {
		c := newFeed()
		c.Name = "Renamed"
		c.Description = "described"
		assert.Equal(t, vb, c.Version())
		assert.False(t, c.Equal(b))
	})

	t.Run("attribute order keeps the version", func(t *testing.T) {
		ab := newFeed()
		ab.Attributes = api.Slice[api.Name]{"a", "b"}
		ba := newFeed()
		ba.Attributes = api.Slice[api.Name]{"b", "a"}
		assert.Equal(t, ab.Version(), ba.Version())
	})

	t.Run("seed, fold, attributes, and space change it", func(t *testing.T) {
		seed := newFeed()
		seed.Value = "1"
		fold := newFeed()
		fold.Fold = &api.ScriptConfig{Language: "lua", Source: "return 1"}
		attr := newFeed()
		attr.Attributes = api.Slice[api.Name]{"amount", "other"}
		space := newFeed()
		space.SpaceID = "risk"
		for _, f := range []*api.Feed{seed, fold, attr, space} {
			assert.NotEqual(t, vb, f.Version())
			assert.False(t, f.Equal(b))
		}
	})
}

func TestFeedSeed(t *testing.T) {
	f := newFeed()
	seed, err := f.Seed()
	assert.NoError(t, err)
	assert.EqualValues(t, 0, seed)

	f.Value = "{"
	_, err = f.Seed()
	assert.True(t, errors.Is(err, api.ErrFeedValueRequired))
}

func TestFeedFolds(t *testing.T) {
	charge := &api.Step{
		ID: "charge",
		Attributes: api.AttributeSpecs{
			"amount": {Role: api.RoleOutput},
		},
	}
	reader := &api.Step{
		ID: "reader",
		Attributes: api.AttributeSpecs{
			"amount": {Role: api.RoleRequired},
		},
	}
	f := newFeed()
	assert.True(t, f.Folds(charge, nil))
	assert.False(t, f.Folds(reader, nil))

	f.Attributes = api.Slice[api.Name]{"refunded", "amount"}
	assert.True(t, f.Folds(charge, nil))

	f.SpaceID = "risk"
	assert.False(t, f.Folds(charge, nil))
	assert.True(t, f.Folds(charge, []api.StepID{"charge"}))
}

func TestFeedAttributeRole(t *testing.T) {
	ok := &api.AttributeSpec{
		Role: api.RoleFeed,
		Feed: &api.FeedConfig{ID: "order-total"},
	}
	assert.NoError(t, ok.Validate("total"))
	assert.True(t, ok.IsFeed())
	assert.True(t, ok.IsRuntimeInput())
	assert.False(t, ok.IsInput())
	assert.Equal(t, api.FeedID("order-total"), ok.FeedID())

	missing := &api.AttributeSpec{Role: api.RoleFeed}
	assert.True(t,
		errors.Is(missing.Validate("total"), api.ErrFeedIDRequired))
	assert.Equal(t, api.FeedID(""), missing.FeedID())

	wrong := &api.AttributeSpec{
		Role:  api.RoleConst,
		Const: &api.ConstConfig{Value: "1"},
		Feed:  &api.FeedConfig{ID: "order-total"},
	}
	assert.True(t,
		errors.Is(wrong.Validate("total"), api.ErrWrongRoleConfig))

	other := &api.AttributeSpec{
		Role: api.RoleFeed,
		Feed: &api.FeedConfig{ID: "other"},
	}
	assert.True(t, ok.Equal(ok))
	assert.False(t, ok.Equal(other))
	assert.False(t, ok.Equal(missing))
}

func TestCatalogFeeds(t *testing.T) {
	f := newFeed()
	cat := api.CatalogState{
		Steps: api.Steps{
			"reader": {
				ID: "reader",
				Attributes: api.AttributeSpecs{
					"total": {
						Role: api.RoleFeed,
						Feed: &api.FeedConfig{ID: f.ID},
					},
				},
			},
			"plain": {ID: "plain", Attributes: api.AttributeSpecs{}},
		},
		Feeds: api.Feeds{},
	}
	cat = cat.SetFeed(f.ID, f)
	assert.Equal(t, f, cat.Feeds[f.ID])
	assert.Equal(t, []api.StepID{"reader"}, cat.FeedConsumers(f.ID))
	assert.Empty(t, cat.FeedConsumers("other"))

	cat = cat.DeleteFeed(f.ID)
	assert.Empty(t, cat.Feeds)
}
