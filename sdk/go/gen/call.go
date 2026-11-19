package gen

import (
	"errors"
	"fmt"
	"runtime/debug"

	argyll "github.com/kode4food/argyll/sdk/go"
)

// PanicError carries a value recovered from a panicking step function
type PanicError struct {
	Value any
	Stack []byte
}

var (
	ErrInvalidInputs = errors.New("invalid step inputs")
)

func (e *PanicError) Error() string {
	return fmt.Sprintf("%s: %v", argyll.ErrHandlerPanic, e.Value)
}

func (e *PanicError) Unwrap() error {
	return argyll.ErrHandlerPanic
}

func call[I, O any](fn func(I) (O, error), in I) (out O, err error) {
	defer func() {
		if r := recover(); r != nil {
			var zero O
			out = zero
			err = &PanicError{
				Value: r,
				Stack: debug.Stack(),
			}
		}
	}()
	return fn(in)
}
