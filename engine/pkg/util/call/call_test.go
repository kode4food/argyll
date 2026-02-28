package call_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/util/call"
)

func TestPerform(t *testing.T) {
	t.Run("runs calls in order", func(t *testing.T) {
		var order []int

		err := call.Perform(
			func() error {
				order = append(order, 1)
				return nil
			},
			func() error {
				order = append(order, 2)
				return nil
			},
		)

		assert.NoError(t, err)
		assert.Equal(t, []int{1, 2}, order)
	})

	t.Run("stops on first error", func(t *testing.T) {
		want := errors.New("boom")
		var order []int

		err := call.Perform(
			func() error {
				order = append(order, 1)
				return nil
			},
			func() error {
				order = append(order, 2)
				return want
			},
			func() error {
				order = append(order, 3)
				return nil
			},
		)

		assert.Equal(t, want, err)
		assert.Equal(t, []int{1, 2}, order)
	})
}

func TestWithArg(t *testing.T) {
	var got int

	err := call.WithArg(func(v int) error {
		got = v
		return nil
	}, 42)()

	assert.NoError(t, err)
	assert.Equal(t, 42, got)
}

func TestWithArgs(t *testing.T) {
	var got string

	err := call.WithArgs(func(a, b string) error {
		got = a + b
		return nil
	}, "foo", "bar")()

	assert.NoError(t, err)
	assert.Equal(t, "foobar", got)
}
