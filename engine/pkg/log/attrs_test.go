package log_test

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/log"
)

type errStub string

func TestFlowID(t *testing.T) {
	attr := log.FlowID(api.FlowID("flow-123"))
	assertAttrEqual(t, attr, "flow_id", "flow-123")
}

func TestStepID(t *testing.T) {
	attr := log.StepID(api.StepID("step-abc"))
	assertAttrEqual(t, attr, "step_id", "step-abc")
}

func TestStatus(t *testing.T) {
	attr := log.Status("completed")
	assertAttrEqual(t, attr, "status", "completed")
}

func TestToken(t *testing.T) {
	attr := log.Token("token-xyz")
	assertAttrEqual(t, attr, "token", "token-xyz")
}

func TestError(t *testing.T) {
	attr := log.Error(nil)
	assertAttrEqual(t, attr, "error", "")

	attr = log.Error(errStub("boom"))
	assertAttrEqual(t, attr, "error", "boom")
}

func TestErrorString(t *testing.T) {
	attr := log.ErrorString("badness")
	assertAttrEqual(t, attr, "error", "badness")
}

func (e errStub) Error() string { return string(e) }

func assertAttrEqual(t *testing.T, attr slog.Attr, key, value string) {
	t.Helper()
	assert.Equal(t, key, attr.Key)
	assert.Equal(t, value, attr.Value.String())
}
