package step

import (
	"github.com/kode4food/argyll/engine/internal/client"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func httpHandler(c client.Client) *Handler {
	return &Handler{
		Execute:    httpExecutor(c),
		Compensate: c.InvokeCompensate,
	}
}

func httpExecutor(c client.Client) ExecuteFunc {
	return func(
		rt Runtime, st *api.Step, inputs api.Args, tkn api.Token,
	) error {
		async := st.HTTP != nil && st.HTTP.Invoke.Async()
		meta := httpMetaForToken(rt, tkn)
		if async {
			meta[api.MetaWebhookURL] = rt.WebhookURL(tkn)
		}
		inputs = applyMetaInputs(st, inputs, meta)
		outputs, err := c.Invoke(st, inputs, meta)
		if err != nil {
			return err
		}
		if async {
			return nil
		}
		return rt.CompleteWork(tkn, outputs)
	}
}

func httpMetaForToken(rt Runtime, tkn api.Token) api.Metadata {
	return rt.Metadata().Apply(api.Metadata{
		api.MetaFlowID:       rt.FlowID(),
		api.MetaStepID:       rt.StepID(),
		api.MetaReceiptToken: tkn,
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
		if val, ok := meta[attr.MetaKey()]; ok {
			mapped, _ := st.MappedName(name)
			metaArgs[mapped] = val
		}
	}
	return inputs.Apply(metaArgs)
}
