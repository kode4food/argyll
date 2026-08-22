package gen

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/kode4food/argyll/engine/pkg/api"
	argyll "github.com/kode4food/argyll/sdk/go"
	"github.com/kode4food/argyll/sdk/go/codec"
)

// PanicError carries a value recovered from a panicking step function
type PanicError struct {
	Value any
	Stack []byte
}

const (
	// FailureStatus reports a step function that returned an error
	FailureStatus = http.StatusUnprocessableEntity

	// PanicStatus reports a step function that panicked
	PanicStatus = http.StatusInternalServerError
)

var (
	ErrInvalidInputs = errors.New("invalid step inputs")
	ErrMethod        = errors.New("method not allowed")
)

// Sync adapts a plain Go function to a synchronous Argyll step handler
func Sync[I, O any](
	in codec.Codec[I], out codec.Codec[O], fn func(I) (O, error),
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		args, ok := decodeBody(w, r, in)
		if !ok {
			return
		}
		res, err := invoke(fn, args)
		if err != nil {
			writeFailure(w, err)
			return
		}
		writeValue(w, http.StatusOK, out, res)
	}
}

func (e *PanicError) Error() string {
	return fmt.Sprintf("%s: %v", argyll.ErrHandlerPanic, e.Value)
}

func (e *PanicError) Unwrap() error {
	return argyll.ErrHandlerPanic
}

func invoke[I, O any](fn func(I) (O, error), in I) (out O, err error) {
	defer func() {
		if r := recover(); r != nil {
			var zero O
			out = zero
			err = &PanicError{Value: r, Stack: debug.Stack()}
		}
	}()
	return fn(in)
}

func decodeBody[T any](
	w http.ResponseWriter, r *http.Request, c codec.Codec[T],
) (T, bool) {
	var zero T
	if r.Method != http.MethodPost {
		argyll.WriteProblem(w, http.StatusMethodNotAllowed, ErrMethod.Error())
		return zero, false
	}
	v, err := codec.DecodeFrom(c, r.Body)
	if err != nil {
		argyll.WriteProblem(w, http.StatusBadRequest,
			fmt.Sprintf("%s: %s", ErrInvalidInputs, err))
		return zero, false
	}
	return v, true
}

func writeFailure(w http.ResponseWriter, err error) {
	if pe, ok := errors.AsType[*PanicError](err); ok {
		slog.Error("Step function panicked",
			slog.Any("panic", pe.Value),
			slog.String("stack", string(pe.Stack)))
		argyll.WriteProblem(w, PanicStatus, pe.Error())
		return
	}
	if he, ok := errors.AsType[*argyll.HTTPError](err); ok {
		argyll.WriteProblem(w, he.StatusCode, he.Message)
		return
	}
	argyll.WriteProblem(w, FailureStatus, err.Error())
}

func writeValue[T any](
	w http.ResponseWriter, status int, c codec.Codec[T], v T,
) {
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
