package builtins

import (
	"errors"
	"fmt"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/step"
)

var (
	ErrNoCallbackURL = errors.New("callback URL required")
)

// HTTP returns the standard HTTP-backed step handler
func HTTP(client Client, callback CallbackURL) *step.Handler {
	return &step.Handler{
		Validate:   validateHTTP(callback),
		Execute:    executeHTTP(client, callback),
		Compensate: compensateHTTP(client, callback),
	}
}

// validateHTTP refuses async work whose outcome the host cannot hear about,
// which would otherwise register a step that never settles
func validateHTTP(callback CallbackURL) step.ValidateFunc {
	return func(st *api.Step) error {
		if callback != nil {
			return nil
		}
		if st.HTTP.Invoke.Async() || st.HTTP.Compensate.Async() {
			return fmt.Errorf("%w: %s", ErrNoCallbackURL, st.ID)
		}
		return nil
	}
}

func compensateHTTP(client Client, callback CallbackURL) step.CompensateFunc {
	return func(req step.CompensateRequest) (bool, error) {
		st := req.Step
		if callback != nil && st.HTTP.Compensate.Async() {
			req.Metadata = req.Metadata.Apply(api.Metadata{
				api.MetaWebhookURL: callback(api.FlowStep{
					FlowID: req.FlowID,
					StepID: st.ID,
				}, req.Token, api.ActionCompensate),
			})
		}
		if err := client.InvokeCompensate(req); err != nil {
			return false, err
		}
		return !st.HTTP.Compensate.Async(), nil
	}
}

func executeHTTP(client Client, callback CallbackURL) step.ExecuteFunc {
	return func(
		rt step.Runtime, st *api.Step, inputs api.Args, token api.Token,
	) error {
		async := st.HTTP.Invoke.Async()
		meta := metadata(rt, token)
		if async && callback != nil {
			meta[api.MetaWebhookURL] = callback(api.FlowStep{
				FlowID: rt.FlowID(),
				StepID: rt.StepID(),
			}, token, api.ActionInvoke)
		}
		inputs = applyMetaInputs(st, inputs, meta)
		outputs, err := client.Invoke(st, inputs, meta)
		if err != nil {
			return err
		}
		if async {
			return nil
		}
		return rt.CompleteWork(token, outputs)
	}
}

func metadata(rt step.Runtime, token api.Token) api.Metadata {
	return rt.Metadata().Apply(api.Metadata{
		api.MetaFlowID:       rt.FlowID(),
		api.MetaStepID:       rt.StepID(),
		api.MetaReceiptToken: token,
	})
}

func applyMetaInputs(
	st *api.Step, inputs api.Args, meta api.Metadata,
) api.Args {
	metaArgs := api.Args{}
	for name, attr := range st.Attributes {
		if !attr.IsMeta() {
			continue
		}
		if value, ok := meta[attr.MetaKey()]; ok {
			mapped, _ := st.MappedName(name)
			metaArgs[mapped] = value
		}
	}
	return inputs.Apply(metaArgs)
}
