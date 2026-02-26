package util_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/util"
)

func collectDetach[T any](tree *util.PathTree[T], prefix []string) []T {
	var vals []T
	tree.DetachWith(prefix, func(v T) {
		vals = append(vals, v)
	})
	return vals
}

func TestPathTreeRemovePrunes(t *testing.T) {
	tree := util.NewPathTree[int]()
	tree.Insert([]string{"a", "b", "c"}, 1)
	tree.Insert([]string{"a", "d"}, 2)

	tree.Remove([]string{"a", "b", "c"})

	vals := collectDetach(tree, []string{"a", "b"})
	assert.Nil(t, vals)

	vals = collectDetach(tree, []string{"a"})
	assert.Equal(t, []int{2}, vals)
}

func TestPathTreeDetachPrunesPrefix(t *testing.T) {
	tree := util.NewPathTree[int]()
	tree.Insert([]string{"retry", "f1", "s1"}, 1)
	tree.Insert([]string{"retry", "f1", "s2"}, 2)
	tree.Insert([]string{"retry", "f2", "s1"}, 3)

	vals := collectDetach(tree, []string{"retry", "f1"})
	assert.ElementsMatch(t, []int{1, 2}, vals)

	vals = collectDetach(tree, []string{"retry", "f1"})
	assert.Nil(t, vals)

	vals = collectDetach(tree, []string{"retry"})
	assert.Equal(t, []int{3}, vals)

	vals = collectDetach(tree, []string{"retry"})
	assert.Nil(t, vals)
}

func TestPathTreeExactOverwriteAndRemove(t *testing.T) {
	tree := util.NewPathTree[string]()
	tree.Insert([]string{"x"}, "one")
	tree.Insert([]string{"x"}, "two")

	vals := collectDetach(tree, []string{"x"})
	assert.Equal(t, []string{"two"}, vals)

	tree.Insert([]string{"x", "y"}, "z")
	tree.Remove([]string{"x", "y"})
	vals = collectDetach(tree, []string{"x"})
	assert.Nil(t, vals)
}

func TestPathTreeEmptyPath(t *testing.T) {
	tree := util.NewPathTree[string]()

	tree.Insert([]string{}, "root")
	tree.Insert([]string{"x"}, "child")
	assert.ElementsMatch(t, []string{"root", "child"},
		collectDetach(tree, nil))

	tree.Insert([]string{}, "root")
	tree.Insert([]string{"x"}, "child")
	tree.Remove([]string{})
	assert.Equal(t, []string{"child"}, collectDetach(tree, []string{}))
}

func TestPathTreeRemoveMissingNestedPath(t *testing.T) {
	tree := util.NewPathTree[int]()
	tree.Insert([]string{"a", "b"}, 1)
	tree.Insert([]string{"a", "c"}, 2)

	tree.Remove([]string{"a", "x"})
	assert.ElementsMatch(t, []int{1, 2}, collectDetach(tree, []string{"a"}))
}

func TestPathTreeRemoveNodeWithChildren(t *testing.T) {
	tree := util.NewPathTree[int]()
	tree.Insert([]string{"a"}, 1)
	tree.Insert([]string{"a", "b"}, 2)

	tree.Remove([]string{"a"})
	assert.Equal(t, []int{2}, collectDetach(tree, []string{"a"}))
}

func TestPathTreeDetachMissingNestedPrefix(t *testing.T) {
	tree := util.NewPathTree[int]()
	tree.Insert([]string{"a", "b"}, 1)

	vals := collectDetach(tree, []string{"a", "x"})
	assert.Nil(t, vals)
	assert.Equal(t, []int{1}, collectDetach(tree, []string{"a"}))
}

func TestPathTreeDetachMissingParentPrefix(t *testing.T) {
	tree := util.NewPathTree[int]()
	tree.Insert([]string{"a", "b"}, 1)

	vals := collectDetach(tree, []string{"x", "b"})
	assert.Nil(t, vals)
	assert.Equal(t, []int{1}, collectDetach(tree, []string{"a"}))
}

func TestPathTreeDetachStillDetachesWithoutCallback(t *testing.T) {
	tree := util.NewPathTree[int]()
	tree.Insert([]string{"a", "b"}, 1)
	tree.Insert([]string{"a", "c"}, 2)

	tree.Detach([]string{"a"})

	assert.Nil(t, collectDetach(tree, []string{"a"}))
}
