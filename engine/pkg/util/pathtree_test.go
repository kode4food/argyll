package util_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/util"
)

func TestPathTreeRemovePrunes(t *testing.T) {
	tree := util.NewPathTree[int]()
	tree.Insert([]string{"a", "b", "c"}, 1)
	tree.Insert([]string{"a", "d"}, 2)

	tree.Remove([]string{"a", "b", "c"})

	vals := tree.Detach([]string{"a", "b"})
	assert.Nil(t, vals)

	vals = tree.Detach([]string{"a"})
	assert.Equal(t, []int{2}, vals)
}

func TestPathTreeDetachPrunesPrefix(t *testing.T) {
	tree := util.NewPathTree[int]()
	tree.Insert([]string{"retry", "f1", "s1"}, 1)
	tree.Insert([]string{"retry", "f1", "s2"}, 2)
	tree.Insert([]string{"retry", "f2", "s1"}, 3)

	vals := tree.Detach([]string{"retry", "f1"})
	assert.ElementsMatch(t, []int{1, 2}, vals)

	vals = tree.Detach([]string{"retry", "f1"})
	assert.Nil(t, vals)

	vals = tree.Detach([]string{"retry"})
	assert.Equal(t, []int{3}, vals)

	vals = tree.Detach([]string{"retry"})
	assert.Nil(t, vals)
}

func TestPathTreeExactOverwriteAndRemove(t *testing.T) {
	tree := util.NewPathTree[string]()
	tree.Insert([]string{"x"}, "one")
	tree.Insert([]string{"x"}, "two")

	vals := tree.Detach([]string{"x"})
	assert.Equal(t, []string{"two"}, vals)

	tree.Insert([]string{"x", "y"}, "z")
	tree.Remove([]string{"x", "y"})
	vals = tree.Detach([]string{"x"})
	assert.Nil(t, vals)
}
