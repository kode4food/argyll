package gen_test

import (
	"encoding/json/jsontext"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
	argyll "github.com/kode4food/argyll/sdk/go"
	"github.com/kode4food/argyll/sdk/go/codec"
	"github.com/kode4food/argyll/sdk/go/gen"
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

func TestInvokeOutputs(t *testing.T) {
	h := gen.Invoke(sumArgsCodec(), sumResultCodec(),
		func(in sumArgs) (sumResult, error) {
			total := in.Left + in.Right
			return sumResult{
				Total:   total,
				Doubled: total * 2,
			}, nil
		})

	w := invoke(h, `{"left":2,"right":3}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, api.JSONContentType, w.Header().Get("Content-Type"))
	assert.JSONEq(t, `{"total":5,"doubled":10}`, w.Body.String())
}

func TestInvokeNoOutputs(t *testing.T) {
	called := false
	h := gen.Invoke(sumArgsCodec(), codec.Struct[struct{}](),
		func(sumArgs) (struct{}, error) {
			called = true
			return struct{}{}, nil
		})

	w := invoke(h, `{"left":1,"right":1}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
	assert.JSONEq(t, `{}`, w.Body.String())
}

func TestInvokeError(t *testing.T) {
	h := gen.Invoke(sumArgsCodec(), sumResultCodec(),
		func(sumArgs) (sumResult, error) {
			return sumResult{}, errRefused
		})

	w := invoke(h, `{"left":1,"right":1}`)
	assert.Equal(t, gen.FailureStatus, w.Code)
	assert.Equal(t, api.ProblemJSONContentType, w.Header().Get("Content-Type"))
	assert.Contains(t, w.Body.String(), errRefused.Error())
}

func TestInvokeSemanticErrors(t *testing.T) {
	for _, tt := range []struct {
		err     error
		want    int
		message string
	}{
		{
			argyll.NotFound("product %s", "missing"),
			http.StatusNotFound,
			"product missing",
		},
		{argyll.Conflict("order %d already exists", 42),
			http.StatusConflict, "order 42 already exists"},
	} {
		t.Run(tt.err.Error(), func(t *testing.T) {
			h := gen.Invoke(sumArgsCodec(), sumResultCodec(),
				func(sumArgs) (sumResult, error) {
					return sumResult{}, tt.err
				})

			w := invoke(h, `{"left":1,"right":1}`)
			assert.Equal(t, tt.want, w.Code)
			assert.Contains(t, w.Body.String(), tt.message)
		})
	}
}

func TestInvokePanic(t *testing.T) {
	h := gen.Invoke(sumArgsCodec(), sumResultCodec(),
		func(sumArgs) (sumResult, error) {
			panic("kaboom")
		})

	w := invoke(h, `{"left":1,"right":1}`)
	assert.Equal(t, gen.PanicStatus, w.Code)
	assert.Contains(t, w.Body.String(), "kaboom")
	assert.Contains(t, w.Body.String(), argyll.ErrHandlerPanic.Error())
}

func TestInvokeBadInputs(t *testing.T) {
	h := gen.Invoke(sumArgsCodec(), sumResultCodec(),
		func(sumArgs) (sumResult, error) {
			return sumResult{}, nil
		})

	w := invoke(h, `{"left":"two"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), gen.ErrInvalidInputs.Error())
}

func TestInvokeBadOutput(t *testing.T) {
	h := gen.Invoke(sumArgsCodec(), failCodec{},
		func(sumArgs) (sumResult, error) {
			return sumResult{}, nil
		})

	w := invoke(h, `{"left":1,"right":1}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), errRefused.Error())
}

func TestInvokeMethodNotAllowed(t *testing.T) {
	h := gen.Invoke(sumArgsCodec(), sumResultCodec(),
		func(sumArgs) (sumResult, error) {
			return sumResult{}, nil
		})

	r := httptest.NewRequest(http.MethodGet, "/sum", nil)
	w := httptest.NewRecorder()
	h(w, r)
	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestHTTPContext(t *testing.T) {
	newRequest := func(path string) *http.Request {
		r := httptest.NewRequestWithContext(
			t.Context(), http.MethodPost, path, strings.NewReader(`{"left":2}`),
		)
		r.Header.Set(api.HeaderFlowID, "flow-1")
		r.Header.Set(api.HeaderStepID, "sum")
		r.Header.Set(api.HeaderReceiptToken, "tkn")
		return r
	}

	r := newRequest("/sum")
	check := func(ctx argyll.StepContext, in sumArgs) error {
		assert.Equal(t, r.Context().Done(), ctx.Done())
		assert.Equal(t, api.StepID("sum"), ctx.StepID())
		assert.Equal(t, api.FlowID("flow-1"), ctx.FlowID())
		assert.Equal(t, "tkn", ctx.Metadata()[api.MetaReceiptToken])
		assert.Equal(t, 2, in.Left)
		return nil
	}
	h := gen.InvokeCtx(
		sumArgsCodec(), codec.Struct[struct{}](),
		func(ctx argyll.StepContext, in sumArgs) (struct{}, error) {
			return struct{}{}, check(ctx, in)
		},
	)
	w := httptest.NewRecorder()
	h(w, r)
	assert.Equal(t, http.StatusOK, w.Code)

	w = httptest.NewRecorder()
	r = newRequest("/sum/compensate")
	gen.CompCtx(sumArgsCodec(), check)(w, r)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestCompensate(t *testing.T) {
	var got compArgs
	h := gen.Comp(
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
	assert.Equal(t, compArgs{
		Left:  2,
		Total: 5,
	}, got)
}

func TestCompensateError(t *testing.T) {
	h := gen.Comp(
		codec.Struct[struct{}](), func(struct{}) error { return errRefused },
	)

	w := invoke(h, `{}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), errRefused.Error())
}

func (failCodec) Decode(*jsontext.Decoder) (sumResult, error) {
	return sumResult{}, errRefused
}

func (failCodec) Encode(*jsontext.Encoder, sumResult) error {
	return errRefused
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
	r := httptest.NewRequest(http.MethodPost, "/sum", strings.NewReader(body))
	w := httptest.NewRecorder()
	h(w, r)
	return w
}
