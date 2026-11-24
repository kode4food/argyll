package builtins

import (
	"errors"
	"fmt"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/step"
)

var (
	ErrNoCallbackURL    = errors.New("callback URL required")
	ErrNoCallbackSecret = errors.New("callback secret required")
)

// HTTP returns the standard HTTP-backed step handler
func HTTP(client Client, callback CallbackURL) *step.Handler {
	return &step.Handler{
		Validate:   validateHTTP(callback),
		Invoke:     invokeHTTP(client, callback),
		Compensate: compensateHTTP(client, callback),
	}
}

// validateHTTP refuses async work whose outcome the host cannot hear about, which would otherwise
// register a step that never settles
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

func compensateHTTP(client Client, callback CallbackURL) step.WorkFunc {
	return func(
		rt step.Runtime, st *api.Step, args api.Args, token api.Token,
	) error {
		async := st.HTTP.Compensate.Async()
		meta := metadata(rt, token)
		if async && callback != nil {
			url, err := callbackURL(rt, callback, token, api.ActionCompensate)
			if err != nil {
				return err
			}
			meta[api.MetaWebhookURL] = url
		}
		if err := client.Compensate(st, args, meta); err != nil {
			return err
		}
		if async {
			return nil
		}
		return rt.CompleteWork(token, nil)
	}
}

func invokeHTTP(client Client, callback CallbackURL) step.WorkFunc {
	return func(
		rt step.Runtime, st *api.Step, inputs api.Args, token api.Token,
	) error {
		async := st.HTTP.Invoke.Async()
		meta := metadata(rt, token)
		if async && callback != nil {
			url, err := callbackURL(rt, callback, token, api.ActionInvoke)
			if err != nil {
				return err
			}
			meta[api.MetaWebhookURL] = url
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

func callbackURL(
	rt step.Runtime, callback CallbackURL, tkn api.Token, action api.CallbackAction,
) (string, error) {
	c, ok := rt.(interface{ CallbackSecret(api.Token) string })
	if !ok {
		return "", ErrNoCallbackSecret
	}
	secret := c.CallbackSecret(tkn)
	if secret == "" {
		return "", ErrNoCallbackSecret
	}
	fs := api.FlowStep{
		FlowID: rt.FlowID(),
		StepID: rt.StepID(),
	}
	return callback(fs, tkn, action, secret), nil
}

func metadata(rt step.Runtime, token api.Token) api.Metadata {
	return rt.Metadata().Apply(api.Metadata{
		api.MetaFlowID:    rt.FlowID(),
		api.MetaStepID:    rt.StepID(),
		api.MetaWorkToken: token,
	})
}

func applyMetaInputs(st *api.Step, inputs api.Args, meta api.Metadata) api.Args {
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
