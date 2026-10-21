package policy

import (
	"time"

	"github.com/kode4food/argyll/engine/pkg/api"
)

type (
	// RequiredInputState describes available values and providers for an input
	RequiredInputState struct {
		Attr        *api.AttributeSpec
		HasInit     bool
		HasProvider bool
		Values      []*api.AttributeValue
		Match       api.Matcher
	}

	ProviderSummary struct {
		// Terminal is true when every planned provider has reached a terminal
		// step state, whether successful, failed, or skipped
		Terminal bool

		// AllSucceeded is true when every planned provider completed
		// successfully and produced a value for the collected attribute
		AllSucceeded bool
	}

	// InputOutlook describes what a pending step's required input has and
	// what its providers can still deliver
	InputOutlook struct {
		HasInit        bool
		HasValue       bool
		AllCollectable bool
		Providable     func() bool
	}
)

// InitBlocksInput reports whether an initial flow value makes an input
// impossible to satisfy. collect:none means the step is waiting for absence,
// so a present init value blocks that step during planning
func InitBlocksInput(collect api.InputCollect, hasInit bool) bool {
	return hasInit && collect == api.InputCollectNone
}

// InitProviderComplete reports whether init data lets match policy treat
// provider state as complete. Init data is complete only when it exists and no
// planned provider can produce the same attribute later
func InitProviderComplete(st *RequiredInputState) bool {
	return st.HasInit && !st.HasProvider
}

// InitSatisfiesRequired reports whether init data can satisfy an input during
// planning. An init value always prunes upstream providers for that input,
// regardless of collect mode, unless a match predicate rejects it
func InitSatisfiesRequired(st *RequiredInputState) bool {
	if !st.HasInit {
		return false
	}
	if !InputHasMatch(st.Attr) {
		return true
	}
	complete := InitProviderComplete(st)
	status, err := RequiredMatchStatus(RequiredMatchSpec{
		Attr:   st.Attr,
		Values: st.Values,
		Provider: ProviderSummary{
			Terminal:     complete,
			AllSucceeded: complete,
		},
		Match: st.Match,
	})
	return err == nil && status == MatchMatched
}

// InitBlocksRuntime reports whether init data blocks a required input.
// Optional inputs can fall back instead
func InitBlocksRuntime(attr *api.AttributeSpec, hasInit bool) bool {
	return attr.IsRequired() && InitBlocksInput(attr.Collect(), hasInit)
}

// RequiredInputMissing reports whether a required input is missing. Providers
// satisfy it speculatively and collect:none never needs a supplied value, so
// otherwise an init value is required
func RequiredInputMissing(st *RequiredInputState) bool {
	if st.HasProvider {
		return false
	}
	if st.Attr.Collect() == api.InputCollectNone {
		return false
	}
	return !st.HasInit
}

// RequiresAllProviders reports whether planning must keep every available
// provider for the attribute. collect:all cannot be satisfied by a subset of
// providers, unlike first, last, or some
func RequiresAllProviders(collect api.InputCollect) bool {
	return collect == api.InputCollectAll
}

// RequiredInputsAvailable reports whether required values are speculatively
// available. collect:none waits for absence instead
func RequiredInputsAvailable(
	st *api.Step, available func(api.Name) bool,
) bool {
	for name, attr := range st.Attributes {
		if attr.IsRequired() && attr.Collect() != api.InputCollectNone &&
			!available(name) {
			return false
		}
	}
	return true
}

// StepOutputsSatisfied reports whether a step's declared outputs are already
// satisfied. A step with no declared outputs is not satisfied by output state
func StepOutputsSatisfied(st *api.Step, satisfied func(api.Name) bool) bool {
	hasOutputs := false
	for name, attr := range st.Attributes {
		if !attr.IsOutput() {
			continue
		}
		hasOutputs = true
		if !satisfied(name) {
			return false
		}
	}
	return hasOutputs
}

// StepInputGuaranteed reports whether an input is guaranteed to have a value
// when planning a child flow. Required inputs, const inputs, meta inputs, and
// optional inputs with defaults are guaranteed
func StepInputGuaranteed(attr *api.AttributeSpec) bool {
	if attr.IsRequired() || attr.IsConst() || attr.IsMeta() {
		return true
	}
	return attr.IsOptional() && attr.OptionalDefault() != ""
}

// ProviderOutputNeeded reports whether a provider should still run for a
// pending consumer of this attribute. An init declaration always answers for
// it. Without one, collect:first stops at the first value and others continue
func ProviderOutputNeeded(collect api.InputCollect, o InputOutlook) bool {
	if o.HasInit {
		return false
	}
	if collect == api.InputCollectAll && !o.AllCollectable {
		return false
	}
	return !o.HasValue || collect != api.InputCollectFirst
}

// InputFulfilled applies collect-mode semantics over value count and provider
// state. An init declaration lets collect:some and collect:all resolve with
// zero values
func InputFulfilled(
	collect api.InputCollect, valueCount int, providers ProviderSummary,
	hasInit bool,
) bool {
	collected := valueCount > 0 || hasInit
	switch collect {
	case api.InputCollectNone:
		return providers.Terminal && valueCount == 0
	case api.InputCollectLast:
		return valueCount > 0 && providers.Terminal
	case api.InputCollectSome:
		return collected && providers.Terminal
	case api.InputCollectAll:
		return collected && providers.Terminal && providers.AllSucceeded
	default: // api.InputCollectFirst
		return valueCount > 0
	}
}

// InputSatisfiable reports whether a required input can still be satisfied.
// collect:none needs absence, collect:all needs every provider to deliver, and
// an init declaration satisfies collect:some and collect:all with zero values
func InputSatisfiable(collect api.InputCollect, o InputOutlook) bool {
	switch collect {
	case api.InputCollectNone:
		return !o.HasValue
	case api.InputCollectAll:
		return o.AllCollectable && (o.HasInit || o.HasValue || o.Providable())
	case api.InputCollectSome:
		return o.HasInit || o.HasValue || o.Providable()
	default:
		return o.HasValue || o.Providable()
	}
}

// FulfilledAt returns when a fulfilled input became ready, which anchors its
// optional deadlines. collect:none is ready when its providers finished
func FulfilledAt(
	collect api.InputCollect, values []*api.AttributeValue,
	completedAt time.Time,
) time.Time {
	var at time.Time
	switch collect {
	case api.InputCollectNone:
		return completedAt
	case api.InputCollectLast:
		at = values[len(values)-1].SetAt
	case api.InputCollectAll, api.InputCollectSome:
		at = lastSetAt(values)
	default: // api.InputCollectFirst
		return values[0].SetAt
	}
	if completedAt.After(at) {
		return completedAt
	}
	return at
}

// CollectWindow returns values accepted before expiry and the resulting
// inactivity deadline. An expired window cannot be extended by later values
func CollectWindow(
	anchor time.Time, timeoutMS int64, values []*api.AttributeValue,
) ([]*api.AttributeValue, time.Time) {
	gap := time.Duration(timeoutMS) * time.Millisecond
	at := anchor.Add(gap)
	for i, value := range values {
		if timeoutMS == 0 {
			if value.SetAt.After(anchor) {
				return values[:i], at
			}
			continue
		}
		if !value.SetAt.Before(at) {
			return values[:i], at
		}
		if next := value.SetAt.Add(gap); next.After(at) {
			at = next
		}
	}
	return values, at
}

// ResolveInputValue selects a collect mode's argument from already-filtered
// values, where collect:some and collect:all resolve zero values to an empty
// list. Callers apply any deadline cutoff or match filtering first
func ResolveInputValue(
	collect api.InputCollect, values []*api.AttributeValue,
) (any, bool) {
	switch collect {
	case api.InputCollectAll, api.InputCollectSome:
		res := make([]any, 0, len(values))
		for _, v := range values {
			res = append(res, v.Value)
		}
		return res, true
	case api.InputCollectNone:
		return nil, false
	}
	if len(values) == 0 {
		return nil, false
	}
	if collect == api.InputCollectLast {
		return values[len(values)-1].Value, true
	}
	return values[0].Value, true
}

func lastSetAt(values []*api.AttributeValue) time.Time {
	var at time.Time
	for _, v := range values {
		if !v.SetAt.IsZero() && (at.IsZero() || v.SetAt.After(at)) {
			at = v.SetAt
		}
	}
	return at
}
