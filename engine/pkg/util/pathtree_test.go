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

func TestPathTreeEmptyPath(t *testing.T) {
	tree := util.NewPathTree[string]()

	tree.Insert([]string{}, "root")
	tree.Insert([]string{"x"}, "child")
	assert.ElementsMatch(t, []string{"root", "child"}, tree.Detach(nil))

	tree.Insert([]string{}, "root")
	tree.Insert([]string{"x"}, "child")
	tree.Remove([]string{})
	assert.Equal(t, []string{"child"}, tree.Detach([]string{}))
}

func TestPathTreeRemoveMissingNestedPath(t *testing.T) {
	tree := util.NewPathTree[int]()
	tree.Insert([]string{"a", "b"}, 1)
	tree.Insert([]string{"a", "c"}, 2)

	tree.Remove([]string{"a", "x"})
	assert.ElementsMatch(t, []int{1, 2}, tree.Detach([]string{"a"}))
}

func TestPathTreeRemoveNodeWithChildren(t *testing.T) {
	tree := util.NewPathTree[int]()
	tree.Insert([]string{"a"}, 1)
	tree.Insert([]string{"a", "b"}, 2)

	tree.Remove([]string{"a"})
	assert.Equal(t, []int{2}, tree.Detach([]string{"a"}))
}

func TestPathTreeDetachMissingNestedPrefix(t *testing.T) {
	tree := util.NewPathTree[int]()
	tree.Insert([]string{"a", "b"}, 1)

	vals := tree.Detach([]string{"a", "x"})
	assert.Nil(t, vals)
	assert.Equal(t, []int{1}, tree.Detach([]string{"a"}))
}

func TestPathTreeDetachMissingParentPrefix(t *testing.T) {
	tree := util.NewPathTree[int]()
	tree.Insert([]string{"a", "b"}, 1)

	vals := tree.Detach([]string{"x", "b"})
	assert.Nil(t, vals)
	assert.Equal(t, []int{1}, tree.Detach([]string{"a"}))
}
