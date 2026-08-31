package example_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kode4food/argyll/engine/pkg/api"
	argyll "github.com/kode4food/argyll/sdk/go"
	"github.com/kode4food/argyll/sdk/go/example"
	"github.com/kode4food/argyll/sdk/go/gen"
	"github.com/stretchr/testify/assert"
)

func TestSyncStep(t *testing.T) {
	srv := stepServer(t)

	res := invoke(t, srv, "calculate-risk",
		`{"customer_id":"c-1","amount":5000,"tags":["vip"],"note":null}`)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.JSONEq(t, `{"score":50,"approved":true}`, bodyOf(t, res))
}

func TestNestedAndMapStep(t *testing.T) {
	srv := stepServer(t)

	res := invoke(t, srv, "enroll",
		`{"address":{"city":"Dublin","zip":"D02"},"limits":{"daily":10},`+
			`"iso_currency":"EUR","scratch":"ignored"}`)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.JSONEq(t,
		`{"city":"Dublin","limits":{"daily":10},"iso_currency":"EUR"}`,
		bodyOf(t, res))
}

func TestRecursiveStep(t *testing.T) {
	srv := stepServer(t)

	tree := `{"root":{"name":"a","children":[` +
		`{"name":"b","children":[{"name":"c","children":[],"next":null}],` +
		`"next":null}],"next":{"name":"z","children":[],"next":null}}}`

	res := invoke(t, srv, "walk", tree)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.JSONEq(t, tree, bodyOf(t, res))
}

func TestSyncStepError(t *testing.T) {
	srv := stepServer(t)

	res := invoke(t, srv, "calculate-risk",
		`{"customer_id":"c-1","amount":-1}`)
	assert.Equal(t, gen.FailureStatus, res.StatusCode)
	assert.Contains(t, bodyOf(t, res), example.ErrAmountNegative.Error())
}

func TestSyncStepPanic(t *testing.T) {
	srv := stepServer(t)

	res := invoke(t, srv, "explode", `{"customer_id":"c-1"}`)
	assert.Equal(t, gen.PanicStatus, res.StatusCode)
	assert.Contains(t, bodyOf(t, res), "boom: c-1")
}

func TestAnonymousStructStep(t *testing.T) {
	srv := stepServer(t)

	res := invoke(t, srv, "greet", `{"name":"ada"}`)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.JSONEq(t, `{"greeting":"hello ada"}`, bodyOf(t, res))
}

func TestZeroOutputStep(t *testing.T) {
	srv := stepServer(t)

	res := invoke(t, srv, "audit", `{"event":"login"}`)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.JSONEq(t, `{}`, bodyOf(t, res))

	res = invoke(t, srv, "audit", `{"event":""}`)
	assert.Equal(t, gen.FailureStatus, res.StatusCode)
}

func TestWrappedStep(t *testing.T) {
	srv := stepServer(t)

	res := invoke(t, srv, "score-customer",
		`{"customer-id":"c-1","amount":20000}`)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.JSONEq(t, `{"score":200,"approved":false}`, bodyOf(t, res))

	res = invoke(t, srv, "score-customer",
		`{"customer-id":"","amount":1}`)
	assert.Equal(t, gen.FailureStatus, res.StatusCode)
}

func TestInferredWrappedStep(t *testing.T) {
	srv := stepServer(t)

	body := `{"customer_id":"c-1","amount":5000}`
	want := `{"score":50,"approved":true}`

	for _, id := range []string{"rate-customer-v2", "grade-customer"} {
		t.Run(id, func(t *testing.T) {
			res := invoke(t, srv, id, body)
			assert.Equal(t, http.StatusOK, res.StatusCode)
			assert.JSONEq(t, want, bodyOf(t, res))
		})
	}
}

func TestRegistration(t *testing.T) {
	byID := registerSteps(t, example.ArgyllSteps()...)

	risk := byID["calculate-risk"]
	assert.Equal(t, api.StepTypeService, risk.Type)
	assert.Equal(t, "http://step-host:9000/calculate-risk",
		risk.HTTP.Invoke.Endpoint)
	assert.True(t, risk.Attributes["customer_id"].IsRequired())
	assert.True(t, risk.Attributes["note"].IsOptional())
	assert.True(t, risk.Attributes["score"].IsOutput())
	assert.Equal(t, api.TypeNumber, risk.Attributes["amount"].Type)
}

func TestSemanticErrorStatus(t *testing.T) {
	srv := stepServer(t)
	res := invoke(t, srv, "reject", `{"reason":"no such customer"}`)
	assert.Equal(t, http.StatusNotFound, res.StatusCode)
	assert.Contains(t, bodyOf(t, res), "no such customer")
}

func TestRegisteredTags(t *testing.T) {
	byID := registerSteps(t, example.ArgyllSteps()...)

	assert.Equal(t, api.Tags{"domain:risk", "scoring"},
		byID["calculate-risk"].Tags)
}

func TestRegisteredAttributeOptions(t *testing.T) {
	byID := registerSteps(t, example.ArgyllSteps()...)

	st := byID["charge-card-v2"]
	assert.Equal(t, api.Name("Charge Card (v2)"), st.Name)

	attrs := st.Attributes
	assert.Equal(t, api.TypeArray, attrs["order_id"].Type)
	assert.True(t, attrs["order_id"].Required.ForEach)
	assert.True(t, attrs["note"].IsOptional())
	assert.Equal(t, `"USD"`, attrs["currency"].Optional.Default)
	assert.Equal(t, `"stripe"`, attrs["gateway"].Const.Value)
	assert.Equal(t, "flow_id", attrs["flow"].Meta.Key)
}

func TestForEachStep(t *testing.T) {
	srv := stepServer(t)

	res := invoke(t, srv, "charge-card-v2",
		`{"order_id":"ord-1","note":"","currency":"EUR",`+
			`"gateway":"stripe","flow":"wf-1"}`)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.JSONEq(t, `{"charge_id":"stripe:ord-1:EUR"}`, bodyOf(t, res))
}

func stepServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(gen.Mux(example.ArgyllSteps()...))
	t.Cleanup(srv.Close)
	return srv
}

func invoke(
	t *testing.T, srv *httptest.Server, id, body string,
) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/"+id,
		strings.NewReader(body))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", api.JSONContentType)
	res, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	t.Cleanup(func() { _ = res.Body.Close() })
	return res
}

func registerSteps(
	t *testing.T, steps ...gen.StepDef,
) map[api.StepID]*api.Step {
	t.Helper()
	registered := make(chan *api.Step, len(steps))
	engine := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			var st api.Step
			assert.NoError(t, json.NewDecoder(r.Body).Decode(&st))
			registered <- &st
			w.WriteHeader(http.StatusCreated)
		}))
	defer engine.Close()

	client := argyll.NewClient(engine.URL, time.Second)
	err := gen.Register(context.Background(), client,
		"http://step-host:9000", steps...)
	assert.NoError(t, err)
	close(registered)

	byID := map[api.StepID]*api.Step{}
	for st := range registered {
		assert.NoError(t, st.Validate())
		byID[st.ID] = st
	}
	return byID
}

func bodyOf(t *testing.T, res *http.Response) string {
	t.Helper()
	body, err := io.ReadAll(res.Body)
	assert.NoError(t, err)
	return string(body)
}
