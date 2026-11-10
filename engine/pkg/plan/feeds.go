package plan

import (
	"maps"
	"slices"

	"github.com/kode4food/argyll/engine/pkg/api"
)

// planFeeds selects the Feeds a plan reads or folds into, and resolves each Feed's Space into the
// planned Steps whose outputs it folds
func planFeeds(cat *api.CatalogState, steps api.Steps) api.PlanFeeds {
	var res api.PlanFeeds
	add := func(pf api.PlanFeed) {
		if res == nil {
			res = api.PlanFeeds{}
		}
		res[pf.Feed.ID] = pf
	}
	for _, st := range steps {
		for _, attr := range st.Attributes {
			if attr == nil || !attr.IsFeed() {
				continue
			}
			if f := cat.Feeds[attr.FeedID()]; f != nil {
				add(api.PlanFeed{Feed: f})
			}
		}
	}
	sids := slices.Sorted(maps.Keys(steps))
	for _, f := range cat.Feeds {
		selected := cat.Spaces.Selected[f.SpaceID]
		var sources api.Slice[api.StepID]
		for _, sid := range sids {
			if f.Folds(steps[sid], selected) {
				sources = append(sources, sid)
			}
		}
		if len(sources) == 0 {
			continue
		}
		add(api.PlanFeed{
			Feed:    f,
			Sources: sources,
		})
	}
	return res
}
