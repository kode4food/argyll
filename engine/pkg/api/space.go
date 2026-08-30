package api

import (
	"errors"
	"fmt"
	"maps"
	"slices"
)

type (
	// Space defines a dynamic planning scope over registered steps
	Space struct {
		Selector    SpaceSelector `json:"selector"`
		ID          SpaceID       `json:"id"`
		Name        Name          `json:"name"`
		Description string        `json:"description,omitempty"`
	}

	// SpaceID uniquely identifies a planning space
	SpaceID string

	// Spaces contains Spaces by their ID
	Spaces map[SpaceID]Space

	// SpaceSelection contains the Step IDs each Space's selector selected
	SpaceSelection map[SpaceID][]StepID

	// SpaceSelector matches label keys with AND and their values with OR
	SpaceSelector map[string][]string
)

var (
	ErrSpaceIDEmpty       = errors.New("space ID empty")
	ErrSpaceIDInvalid     = errors.New("space ID contains invalid characters")
	ErrSpaceNameEmpty     = errors.New("space name empty")
	ErrSpaceSelectorEmpty = errors.New("space selector empty")
	ErrInvalidMatchLabels = errors.New("invalid match labels")
)

// Validate checks if the space definition is valid
func (s Space) Validate() error {
	if s.ID == "" {
		return ErrSpaceIDEmpty
	}
	if SanitizeID(s.ID) != s.ID {
		return ErrSpaceIDInvalid
	}
	if s.Name == "" {
		return ErrSpaceNameEmpty
	}
	if len(s.Selector) == 0 {
		return ErrSpaceSelectorEmpty
	}
	if len(s.Selector) > MaxLabelCount {
		return fmt.Errorf("%w: maximum is %d", ErrTooManyLabels, MaxLabelCount)
	}
	for key, values := range s.Selector {
		if key == "" || len(values) == 0 || slices.Contains(values, "") {
			return ErrInvalidMatchLabels
		}
	}
	return nil
}

// Normalize returns the Space with its selector values sorted and deduped
func (s Space) Normalize() Space {
	selector := make(SpaceSelector, len(s.Selector))
	for key, values := range s.Selector {
		selector[key] = slices.Compact(sorted(values))
	}
	s.Selector = selector
	return s
}

// Equal returns true if two space definitions are equal
func (s Space) Equal(other Space) bool {
	return s.ID == other.ID && s.Name == other.Name &&
		s.Description == other.Description &&
		selectorsEqual(s.Selector, other.Selector)
}

// Matches returns true when every selector key matches one of its values
func (s Space) Matches(step *Step) bool {
	for key, values := range s.Selector {
		if !slices.Contains(values, step.Labels[key]) {
			return false
		}
	}
	return true
}

func selectorsEqual(left, right SpaceSelector) bool {
	return maps.EqualFunc(left, right, func(l, r []string) bool {
		return slices.Equal(sorted(l), sorted(r))
	})
}

func sorted(values []string) []string {
	return slices.Sorted(slices.Values(values))
}
