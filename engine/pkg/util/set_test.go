package util_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/spuds/engine/pkg/util"
)

func TestEmptySet(t *testing.T) {
	s := util.Set[string]{}
	assert.True(t, s.IsEmpty())
	assert.Equal(t, 0, s.Len())
}

func TestSetOf(t *testing.T) {
	s := util.SetOf("a", "b", "c")
	assert.Equal(t, 3, s.Len())
	assert.True(t, s.Contains("a"))
	assert.True(t, s.Contains("b"))
	assert.True(t, s.Contains("c"))
}

func TestSetOfDuplicates(t *testing.T) {
	s := util.SetOf("a", "b", "a", "c", "b")
	assert.Equal(t, 3, s.Len())
}

func TestAdd(t *testing.T) {
	s := util.Set[int]{}
	s.Add(1)
	s.Add(2)
	s.Add(1)

	assert.Equal(t, 2, s.Len())
	assert.True(t, s.Contains(1))
	assert.True(t, s.Contains(2))
}

func TestRemove(t *testing.T) {
	s := util.SetOf(1, 2, 3)
	s.Remove(2)

	assert.Equal(t, 2, s.Len())
	assert.False(t, s.Contains(2))
	assert.True(t, s.Contains(1))
	assert.True(t, s.Contains(3))
}

func TestRemoveNonExistent(t *testing.T) {
	s := util.SetOf(1, 2)
	s.Remove(99)

	assert.Equal(t, 2, s.Len())
}

func TestContains(t *testing.T) {
	s := util.SetOf("foo", "bar")

	assert.True(t, s.Contains("foo"))
	assert.False(t, s.Contains("baz"))
}

func TestIsEmpty(t *testing.T) {
	s := util.Set[int]{}
	assert.True(t, s.IsEmpty())

	s.Add(1)
	assert.False(t, s.IsEmpty())

	s.Remove(1)
	assert.True(t, s.IsEmpty())
}
