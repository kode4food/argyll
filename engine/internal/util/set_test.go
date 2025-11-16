package util

import (
	"testing"
)

func TestEmptySet(t *testing.T) {
	s := Set[string]{}
	if !s.IsEmpty() {
		t.Errorf("expected empty set, got length %d", s.Len())
	}
	if s.Len() != 0 {
		t.Errorf("expected length 0, got %d", s.Len())
	}
}

func TestSetOf(t *testing.T) {
	s := SetOf("a", "b", "c")
	if s.Len() != 3 {
		t.Errorf("expected length 3, got %d", s.Len())
	}
	if !s.Contains("a") || !s.Contains("b") || !s.Contains("c") {
		t.Error("set should contain all initial elements")
	}
}

func TestSetOfDuplicates(t *testing.T) {
	s := SetOf("a", "b", "a", "c", "b")
	if s.Len() != 3 {
		t.Errorf("expected length 3 (duplicates removed), got %d", s.Len())
	}
}

func TestAdd(t *testing.T) {
	s := Set[int]{}
	s.Add(1)
	s.Add(2)
	s.Add(1) // duplicate

	if s.Len() != 2 {
		t.Errorf("expected length 2, got %d", s.Len())
	}
	if !s.Contains(1) || !s.Contains(2) {
		t.Error("set should contain added elements")
	}
}

func TestRemove(t *testing.T) {
	s := SetOf(1, 2, 3)
	s.Remove(2)

	if s.Len() != 2 {
		t.Errorf("expected length 2, got %d", s.Len())
	}
	if s.Contains(2) {
		t.Error("set should not contain removed element")
	}
	if !s.Contains(1) || !s.Contains(3) {
		t.Error("set should still contain other elements")
	}
}

func TestRemoveNonExistent(t *testing.T) {
	s := SetOf(1, 2)
	s.Remove(99) // doesn't exist

	if s.Len() != 2 {
		t.Errorf("expected length 2, got %d", s.Len())
	}
}

func TestContains(t *testing.T) {
	s := SetOf("foo", "bar")

	if !s.Contains("foo") {
		t.Error("set should contain 'foo'")
	}
	if s.Contains("baz") {
		t.Error("set should not contain 'baz'")
	}
}

func TestIsEmpty(t *testing.T) {
	s := Set[int]{}
	if !s.IsEmpty() {
		t.Error("new set should be empty")
	}

	s.Add(1)
	if s.IsEmpty() {
		t.Error("set with elements should not be empty")
	}

	s.Remove(1)
	if !s.IsEmpty() {
		t.Error("set after removing all elements should be empty")
	}
}
