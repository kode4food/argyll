package gen_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
	argyll "github.com/kode4food/argyll/sdk/go"
	"github.com/kode4food/argyll/sdk/go/codec"
	"github.com/kode4food/argyll/sdk/go/gen"
)

func TestRegisterFailure(t *testing.T) {
	engine := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
	defer engine.Close()

	client := argyll.NewClient(engine.URL, time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := gen.Register(ctx, client, "http://host:1", sumStep())
	assert.Error(t, err)
}

func TestRegisterConflict(t *testing.T) {
	var posts int
	var puts int
	engine := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodPost:
				posts++
				w.WriteHeader(http.StatusConflict)
			case http.MethodPut:
				puts++
				w.WriteHeader(http.StatusOK)
			}
		}))
	defer engine.Close()

	client := argyll.NewClient(engine.URL, time.Second)
	err := gen.Register(context.Background(), client, "http://host:1", sumStep())
	assert.NoError(t, err)
	assert.Equal(t, 1, posts)
	assert.Equal(t, 1, puts)
}

func TestServeFailure(t *testing.T) {
	t.Setenv("ARGYLL_URL", "http://localhost:1")
	t.Setenv("STEP_PORT", "0")

	assert.Error(t, gen.Serve(context.Background(), sumStep()))
}

func TestMux(t *testing.T) {
	st := sumStep()
	st.Compensate = gen.Comp(
		codec.Struct[struct{}](), func(struct{}) error { return nil },
	)
	srv := httptest.NewServer(gen.Mux(st))
	defer srv.Close()

	res, err := http.Get(srv.URL + "/health")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.NoError(t, res.Body.Close())

	res, err = http.Post(srv.URL+"/sum", api.JSONContentType,
		strings.NewReader(`{"left":1,"right":2}`))
	assert.NoError(t, err)
	defer func() { _ = res.Body.Close() }()
	assert.Equal(t, http.StatusOK, res.StatusCode)

	res, err = http.Post(
		srv.URL+"/sum/compensate", api.JSONContentType, strings.NewReader(`{}`),
	)
	assert.NoError(t, err)
	defer func() { _ = res.Body.Close() }()
	assert.Equal(t, http.StatusNoContent, res.StatusCode)
}

// sumStep stands in for what argyll-gen writes
func sumStep() gen.StepDef {
	return gen.StepDef{
		Step: &api.Step{
			ID:   "sum",
			Name: "Sum",
			Type: api.StepTypeService,
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{Endpoint: "/sum"},
				Health: "/health",
			},
			Attributes: api.AttributeSpecs{
				"left": {
					Role: api.RoleRequired,
				},
				"total": {
					Role: api.RoleOutput,
				},
			},
		},
		Invoke: gen.Invoke(sumArgsCodec(), sumResultCodec(),
			func(in sumArgs) (sumResult, error) {
				return sumResult{Total: in.Left + in.Right}, nil
			}),
	}
}
