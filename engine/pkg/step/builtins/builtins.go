// Package builtins provides Argyll's standard step implementations
package builtins

import (
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/step"
)

type (
	// Client invokes HTTP-backed steps
	Client interface {
		Invoke(*api.Step, api.Args, api.Metadata) (api.Args, error)
		Compensate(step.CompensateRequest) error
	}

	// CallbackURL returns the callback endpoint for asynchronous HTTP work
	CallbackURL func(api.FlowStep, api.Token, api.CallbackAction) string
)

// BaseCallbackURL constructs callback URLs below a fixed base URL
func BaseCallbackURL(base string) CallbackURL {
	return func(
		fs api.FlowStep, token api.Token, action api.CallbackAction,
	) string {
		return base + api.CallbackPath(fs.FlowID, fs.StepID, token, action)
	}
}

// All returns all standard handlers using the supplied HTTP client
func All(client Client, callback CallbackURL) step.Handlers {
	return step.Handlers{
		api.StepTypeScript:  Script(),
		api.StepTypeService: HTTP(client, callback),
		api.StepTypeFlow:    Flow(),
	}
}
