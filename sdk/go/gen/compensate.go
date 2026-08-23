package gen

import (
	"net/http"

	"github.com/kode4food/argyll/sdk/go/codec"
)

// Compensate adapts a typed compensation function to an HTTP handler
func Compensate[I any](in codec.Codec[I], fn func(I) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		args, ok := decodeBody(w, r, in)
		if !ok {
			return
		}
		_, err := invoke(
			func(in I) (struct{}, error) {
				return struct{}{}, fn(in)
			},
			args,
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
