package gen

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/kode4food/argyll/engine/pkg/api"
)

// StepDef is a generated step: its specification in the wire form the engine
// accepts, and the handlers serving it
type StepDef struct {
	Handler    http.HandlerFunc
	Compensate http.HandlerFunc
	ID         api.StepID
	Spec       string
}

// Step decodes the step specification
func (s StepDef) Step() (*api.Step, error) {
	var step api.Step
	if err := json.Unmarshal([]byte(s.Spec), &step); err != nil {
		return nil, fmt.Errorf("%w: %s", err, s.ID)
	}
	return &step, nil
}
