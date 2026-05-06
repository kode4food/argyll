package policy

import "github.com/kode4food/argyll/engine/pkg/api"

// ProviderSummary is the executor's concrete view of all providers for an
// attribute. It lets collect-mode policy reason about provider completion
// without depending on FlowState directly
type ProviderSummary struct {
	// Terminal is true when every planned provider has reached a terminal step
	// state, whether successful, failed, or skipped
	Terminal bool

	// AllSucceeded is true when every planned provider completed successfully
	// and produced a value for the collected attribute
	AllSucceeded bool
}

// InitSatisfiesInput reports whether an initial flow value is enough to
// satisfy an input during planning. Only collect:first treats an init value as
// complete and can therefore prune upstream providers from the plan
func InitSatisfiesInput(collect api.InputCollect, hasInit bool) bool {
	return hasInit && collect == api.InputCollectFirst
}

// InitBlocksInput reports whether an initial flow value makes an input
// impossible to satisfy. collect:none means the step is waiting for absence,
// so a present init value blocks that step during planning
func InitBlocksInput(collect api.InputCollect, hasInit bool) bool {
	return hasInit && collect == api.InputCollectNone
}

// RequiredInputMissing reports whether a required input should be surfaced as
// missing to the caller building a plan. Providers satisfy the requirement
// speculatively, collect:none never requires a supplied value, and otherwise
// an init value is required
func RequiredInputMissing(
	collect api.InputCollect, hasProvider, hasInit bool,
) bool {
	if hasProvider {
		return false
	}
	if collect == api.InputCollectNone {
		return false
	}
	return !hasInit
}

// RequiresAllProviders reports whether planning must keep every available
// provider for the attribute. collect:all cannot be satisfied by a subset of
// providers, unlike first, last, or some
func RequiresAllProviders(collect api.InputCollect) bool {
	return collect == api.InputCollectAll
}

// ProviderOutputNeeded reports whether a provider should still run for a
// pending consumer of this attribute. Existing values satisfy collect:first,
// but other collect modes may still need later or additional provider output
func ProviderOutputNeeded(
	collect api.InputCollect, hasValue, canCollectAll bool,
) bool {
	if collect == api.InputCollectAll && !canCollectAll {
		return false
	}
	return !hasValue || collect != api.InputCollectFirst
}

// InputFulfilled reports whether the executor has enough real flow state to
// start a step input. It applies collect-mode semantics over concrete value
// count and provider terminal/success state
func InputFulfilled(
	collect api.InputCollect, valueCount int, providers ProviderSummary,
) bool {
	switch collect {
	case api.InputCollectNone:
		return providers.Terminal && valueCount == 0
	case api.InputCollectFirst:
		return valueCount > 0
	case api.InputCollectLast, api.InputCollectSome:
		return valueCount > 0 && providers.Terminal
	case api.InputCollectAll:
		return valueCount > 0 && providers.Terminal && providers.AllSucceeded
	default:
		return valueCount > 0
	}
}

// TimeoutCanUseValues reports whether an optional input timeout may use values
// collected up to the timeout cutoff rather than falling back to its default
func TimeoutCanUseValues(collect api.InputCollect) bool {
	switch collect {
	case api.InputCollectLast, api.InputCollectSome, api.InputCollectNone:
		return true
	default:
		return false
	}
}

// ResolveInputValue returns the runtime argument value implied by a collect
// mode from already-filtered attribute values. Callers are responsible for
// applying any deadline cutoff before calling this helper
func ResolveInputValue(
	collect api.InputCollect, values []*api.AttributeValue,
) (any, bool) {
	if len(values) == 0 {
		return nil, false
	}
	switch collect {
	case api.InputCollectLast:
		return values[len(values)-1].Value, true
	case api.InputCollectAll, api.InputCollectSome:
		res := make([]any, 0, len(values))
		for _, v := range values {
			res = append(res, v.Value)
		}
		return res, true
	case api.InputCollectNone:
		return nil, false
	default:
		return values[0].Value, true
	}
}
