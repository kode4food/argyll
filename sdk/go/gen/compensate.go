package gen

import (
	"net/http"

	argyll "github.com/kode4food/argyll/sdk/go"
	"github.com/kode4food/argyll/sdk/go/codec"
)

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

func compHTTP[I any](in codec.Codec[I], fn func(*http.Request, I) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		args, ok := decodeBody(w, r, in)
		if !ok {
			return
		}
		_, err := invoke(
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
