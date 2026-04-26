package util_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/util"
)

type copySubject struct {
	Name   string
	Values []string
}

func TestMutableCopy(t *testing.T) {
	t.Run("nil returns zero value", func(t *testing.T) {
		res := util.MutableCopy[copySubject](nil)

		assert.NotNil(t, res)
		assert.Equal(t, copySubject{}, *res)
	})

	t.Run("returns distinct value", func(t *testing.T) {
		src := &copySubject{Name: "original"}

		res := util.MutableCopy(src)
		res.Name = "updated"

		assert.NotSame(t, src, res)
		assert.Equal(t, "original", src.Name)
		assert.Equal(t, "updated", res.Name)
	})

	t.Run("copy is shallow", func(t *testing.T) {
		src := &copySubject{
			Name:   "original",
			Values: []string{"a"},
		}

		res := util.MutableCopy(src)
		res.Values[0] = "b"

		assert.Equal(t, "b", src.Values[0])
		assert.Equal(t, "b", res.Values[0])
	})
}
