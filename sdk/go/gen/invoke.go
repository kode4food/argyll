package gen

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/kode4food/argyll/engine/pkg/api"
	argyll "github.com/kode4food/argyll/sdk/go"
	"github.com/kode4food/argyll/sdk/go/codec"
)

const (
	// FailureStatus reports a step function that returned an error
	FailureStatus = http.StatusUnprocessableEntity

	// PanicStatus reports a step function that panicked
	PanicStatus = http.StatusInternalServerError
)

var (
	ErrMethod = errors.New("method not allowed")
)

// Invoke adapts a plain Go function to a synchronous Argyll step handler
func Invoke[I, O any](
	in codec.Codec[I], out codec.Codec[O], fn func(I) (O, error),
) http.HandlerFunc {
	return invokeHTTP(in, out, func(_ *http.Request, args I) (O, error) {
		return fn(args)
	})
}

// InvokeCtx adapts a typed function that accepts the Step's HTTP request context
func InvokeCtx[I, O any](
	in codec.Codec[I], out codec.Codec[O], fn func(argyll.StepContext, I) (O, error),
) http.HandlerFunc {
	return invokeHTTP(in, out, func(r *http.Request, args I) (O, error) {
		return fn(requestCtx(r), args)
	})
}

// Comp adapts a typed compensation function to an HTTP handler
func Comp[I any](in codec.Codec[I], fn func(I) error) http.HandlerFunc {
	return compHTTP(in, func(_ *http.Request, args I) error {
		return fn(args)
	})
}

// CompCtx adapts a compensation function that accepts the Step's HTTP request context
func CompCtx[I any](in codec.Codec[I], fn func(argyll.StepContext, I) error) http.HandlerFunc {
	return compHTTP(in, func(r *http.Request, args I) error {
		return fn(requestCtx(r), args)
	})
}

func invokeHTTP[I, O any](
	in codec.Codec[I], out codec.Codec[O], fn func(*http.Request, I) (O, error),
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		args, ok := decodeBody(w, r, in)
		if !ok {
			return
		}
		res, err := call(func(in I) (O, error) { return fn(r, in) }, args)
		if err != nil {
			writeFailure(w, err, FailureStatus, "Step")
			return
		}
		writeValue(w, http.StatusOK, out, res)
	}
}

func compHTTP[I any](in codec.Codec[I], fn func(*http.Request, I) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		args, ok := decodeBody(w, r, in)
		if !ok {
			return
		}
		_, err := call(
			func(in I) (struct{}, error) {
				return struct{}{}, fn(r, in)
			},
			args,
		)
		if err != nil {
			writeFailure(w, err, http.StatusInternalServerError, "Compensation")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func requestCtx(r *http.Request) argyll.StepContext {
	meta := api.MetadataFromHeaders(r.Header)
	fid, _ := meta.GetString[api.FlowID](api.MetaFlowID)
	sid, _ := meta.GetString[api.StepID](api.MetaStepID)
	return argyll.NewStepContext(r.Context(), fid, sid, meta)
}

func decodeBody[T any](w http.ResponseWriter, r *http.Request, c codec.Codec[T]) (T, bool) {
	var zero T
	if r.Method != http.MethodPost {
		argyll.WriteProblem(w, http.StatusMethodNotAllowed, ErrMethod.Error())
		return zero, false
	}
	v, err := codec.DecodeFrom(c, r.Body)
	if err != nil {
		argyll.WriteProblem(w, http.StatusBadRequest, fmt.Sprintf("%s: %s", ErrInvalidInputs, err))
		return zero, false
	}
	return v, true
}

func writeFailure(w http.ResponseWriter, err error, fallback int, action string) {
	if pe, ok := errors.AsType[*PanicError](err); ok {
		slog.Error(action+" function panicked",
			slog.Any("panic", pe.Value),
			slog.String("stack", string(pe.Stack)))
		argyll.WriteProblem(w, PanicStatus, pe.Error())
		return
	}
	if he, ok := errors.AsType[*argyll.HTTPError](err); ok {
		argyll.WriteProblem(w, he.StatusCode, he.Message)
		return
	}
	argyll.WriteProblem(w, fallback, err.Error())
}

func writeValue[T any](w http.ResponseWriter, status int, c codec.Codec[T], v T) {
	body, err := encodeValue(c, v)
	if err != nil {
		argyll.WriteProblem(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", api.JSONContentType)
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func encodeValue[T any](c codec.Codec[T], v T) ([]byte, error) {
	var buf bytes.Buffer
	if err := codec.EncodeTo(c, &buf, v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
