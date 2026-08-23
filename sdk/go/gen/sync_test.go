package gen_test

import (
	"context"
	"encoding/json/jsontext"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kode4food/argyll/engine/pkg/api"
	argyll "github.com/kode4food/argyll/sdk/go"
	"github.com/kode4food/argyll/sdk/go/codec"
	"github.com/kode4food/argyll/sdk/go/gen"
	"github.com/stretchr/testify/assert"
)

type (
	sumArgs struct {
		Left  int
		Right int
	}

	sumResult struct {
		Total   int
		Doubled int
	}

	compArgs struct {
		Left  int
		Total int
	}

	failCodec struct{}
)

var errRefused = errors.New("refused")

func TestSyncOutputs(t *testing.T) {
	h := gen.Sync(sumArgsCodec(), sumResultCodec(),
		func(in sumArgs) (sumResult, error) {
			total := in.Left + in.Right
			return sumResult{Total: total, Doubled: total * 2}, nil
		})

	w := invoke(h, `{"left":2,"right":3}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, api.JSONContentType, w.Header().Get("Content-Type"))
	assert.JSONEq(t, `{"total":5,"doubled":10}`, w.Body.String())
}

func TestSyncNoOutputs(t *testing.T) {
	called := false
	h := gen.Sync(sumArgsCodec(), codec.Struct[struct{}](),
		func(sumArgs) (struct{}, error) {
			called = true
			return struct{}{}, nil
		})

	w := invoke(h, `{"left":1,"right":1}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
	assert.JSONEq(t, `{}`, w.Body.String())
}

func TestSyncError(t *testing.T) {
	h := gen.Sync(sumArgsCodec(), sumResultCodec(),
		func(sumArgs) (sumResult, error) {
			return sumResult{}, errRefused
		})

	w := invoke(h, `{"left":1,"right":1}`)
	assert.Equal(t, gen.FailureStatus, w.Code)
	assert.Equal(t, api.ProblemJSONContentType,
		w.Header().Get("Content-Type"))
	assert.Contains(t, w.Body.String(), errRefused.Error())
}

func TestSyncPanic(t *testing.T) {
	h := gen.Sync(sumArgsCodec(), sumResultCodec(),
		func(sumArgs) (sumResult, error) {
			panic("kaboom")
		})

	w := invoke(h, `{"left":1,"right":1}`)
	assert.Equal(t, gen.PanicStatus, w.Code)
	assert.Contains(t, w.Body.String(), "kaboom")
	assert.Contains(t, w.Body.String(), argyll.ErrHandlerPanic.Error())
}

func TestSyncBadInputs(t *testing.T) {
	h := gen.Sync(sumArgsCodec(), sumResultCodec(),
		func(sumArgs) (sumResult, error) {
			return sumResult{}, nil
		})

	w := invoke(h, `{"left":"two"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), gen.ErrInvalidInputs.Error())
}

func TestSyncBadOutput(t *testing.T) {
	h := gen.Sync(sumArgsCodec(), failCodec{},
		func(sumArgs) (sumResult, error) {
			return sumResult{}, nil
		})

	w := invoke(h, `{"left":1,"right":1}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), errRefused.Error())
}

func TestSyncMethodNotAllowed(t *testing.T) {
	h := gen.Sync(sumArgsCodec(), sumResultCodec(),
		func(sumArgs) (sumResult, error) {
			return sumResult{}, nil
		})

	r := httptest.NewRequest(http.MethodGet, "/sum", nil)
	w := httptest.NewRecorder()
	h(w, r)
	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestCompensate(t *testing.T) {
	var got compArgs
	h := gen.Compensate(
		compArgsCodec(),
		func(args compArgs) error {
			got = args
			return nil
		},
	)

	w := invoke(h, `{
		"left":2,"right":3,"total":5,"doubled":10
	}`)
	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, compArgs{Left: 2, Total: 5}, got)
}

func TestCompensateError(t *testing.T) {
	h := gen.Compensate(
		codec.Struct[struct{}](),
		func(struct{}) error { return errRefused },
	)

	w := invoke(h, `{}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), errRefused.Error())
}

func TestRegisterFailure(t *testing.T) {
	engine := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
	defer engine.Close()

	client := argyll.NewClient(engine.URL, time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := gen.Register(
		ctx, client, "http://host:1", sumStep(),
	)
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
	err := gen.Register(
		context.Background(), client, "http://host:1", sumStep(),
	)
	assert.NoError(t, err)
	assert.Equal(t, 1, posts)
	assert.Equal(t, 1, puts)
}

func TestRegisterBadSpec(t *testing.T) {
	client := argyll.NewClient("http://127.0.0.1:1", time.Second)
	err := gen.Register(context.Background(), client, "http://host:1",
		gen.StepDef{ID: "sum", Spec: "{"})
	assert.Error(t, err)
}

func TestServeFailure(t *testing.T) {
	t.Setenv("ARGYLL_ENGINE_URL", "http://127.0.0.1:1")
	t.Setenv("STEP_PORT", "0")

	assert.Error(t, gen.Serve(context.Background(), sumStep()))
}

func TestMux(t *testing.T) {
	step := sumStep()
	step.Compensate = gen.Compensate(
		codec.Struct[struct{}](), func(struct{}) error { return nil },
	)
	srv := httptest.NewServer(gen.Mux(step))
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

func TestPanicErrorUnwraps(t *testing.T) {
	h := gen.Sync(sumArgsCodec(), sumResultCodec(),
		func(sumArgs) (sumResult, error) {
			panic(errRefused)
		})

	w := invoke(h, `{"left":1,"right":1}`)
	assert.Equal(t, gen.PanicStatus, w.Code)

	pe := &gen.PanicError{Value: errRefused}
	assert.ErrorIs(t, pe, argyll.ErrHandlerPanic)
	assert.Contains(t, pe.Error(), errRefused.Error())
}

// sumStep stands in for what argyll-gen writes, the specification in the wire
// form the engine accepts
func sumStep() gen.StepDef {
	return gen.StepDef{
		ID: "sum",
		Spec: `{"id":"sum","name":"Sum","type":"sync",` +
			`"http":{"invoke":{"endpoint":"/sum"},"health":"/health"},` +
			`"attributes":{` +
			`"left":{"role":"required","type":"number"},` +
			`"total":{"role":"output","type":"number"}}}`,
		Handler: gen.Sync(sumArgsCodec(), sumResultCodec(),
			func(in sumArgs) (sumResult, error) {
				return sumResult{Total: in.Left + in.Right}, nil
			}),
	}
}

func sumArgsCodec() codec.Codec[sumArgs] {
	return codec.Struct(
		codec.Field("left", codec.Int, func(v *sumArgs) *int {
			return &v.Left
		}),
		codec.Field("right", codec.Int, func(v *sumArgs) *int {
			return &v.Right
		}),
	)
}

func sumResultCodec() codec.Codec[sumResult] {
	return codec.Struct(
		codec.Field("total", codec.Int, func(v *sumResult) *int {
			return &v.Total
		}),
		codec.Field("doubled", codec.Int, func(v *sumResult) *int {
			return &v.Doubled
		}),
	)
}

func compArgsCodec() codec.Codec[compArgs] {
	return codec.Struct(
		codec.Field("left", codec.Int, func(v *compArgs) *int {
			return &v.Left
		}),
		codec.Field("total", codec.Int, func(v *compArgs) *int {
			return &v.Total
		}),
	)
}

func invoke(h http.HandlerFunc, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodPost, "/sum",
		strings.NewReader(body))
	w := httptest.NewRecorder()
	h(w, r)
	return w
}

func (failCodec) Decode(*jsontext.Decoder) (sumResult, error) {
	return sumResult{}, errRefused
}

func (failCodec) Encode(*jsontext.Encoder, sumResult) error {
	return errRefused
}
