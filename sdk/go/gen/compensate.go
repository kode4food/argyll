package gen

import (
	"net/http"

	"github.com/kode4food/argyll/sdk/go/codec"
)

type compensateArgs[I, O any] struct {
	Input  I
	Output O
}

// Compensate adapts a typed compensation function to an HTTP handler
func Compensate[I, O any](
	in codec.Codec[I], out codec.Codec[O], fn func(I, O) error,
) http.HandlerFunc {
	args := codec.Struct(
		codec.Field("input", in,
			func(v *compensateArgs[I, O]) *I {
				return &v.Input
			}),
		codec.Field("output", out,
			func(v *compensateArgs[I, O]) *O {
				return &v.Output
			}),
	)
	return func(w http.ResponseWriter, r *http.Request) {
		body, ok := decodeBody(w, r, args)
		if !ok {
			return
		}
		_, err := invoke(
			func(body compensateArgs[I, O]) (struct{}, error) {
				return struct{}{}, fn(body.Input, body.Output)
			},
			body,
		)
		if err != nil {
			writeFailure(
				w, err, http.StatusInternalServerError, "Compensation",
			)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
