package gen_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	argyll "github.com/kode4food/argyll/sdk/go"
	"github.com/kode4food/argyll/sdk/go/gen"
)

func TestPanicErrorUnwraps(t *testing.T) {
	h := gen.Invoke(sumArgsCodec(), sumResultCodec(),
		func(sumArgs) (sumResult, error) {
			panic(errRefused)
		})

	w := invoke(h, `{"left":1,"right":1}`)
	assert.Equal(t, gen.PanicStatus, w.Code)

	pe := &gen.PanicError{Value: errRefused}
	assert.ErrorIs(t, pe, argyll.ErrHandlerPanic)
	assert.Contains(t, pe.Error(), errRefused.Error())
}
