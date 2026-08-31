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
		Selector    *ScriptConfig `json:"selector"`
		QBE         SpaceQuery    `json:"qbe,omitempty"`
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

	// SpaceQuery matches label keys with AND and their values with OR
	SpaceQuery map[string][]string
)

var (
	ErrSpaceIDEmpty       = errors.New("space ID empty")
	ErrSpaceIDInvalid     = errors.New("space ID contains invalid characters")
	ErrSpaceNameEmpty     = errors.New("space name empty")
	ErrSpaceSelectorEmpty = errors.New("space selector empty")
	ErrInvalidSpaceQuery  = errors.New("invalid space query")
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
	return s.ValidateSelector()
}

// ValidateSelector checks if the space selector and its QBE source are valid.
// The selector is a label predicate, so its language is left to the script
// registry to accept or reject
func (s Space) ValidateSelector() error {
	if s.Selector == nil {
		return ErrSpaceSelectorEmpty
	}
	if s.Selector.Script == "" {
		return ErrScriptEmpty
	}
	if len(s.QBE) > MaxLabelCount {
		return fmt.Errorf("%w: maximum is %d", ErrTooManyLabels, MaxLabelCount)
	}
	for key, values := range s.QBE {
		if key == "" || len(values) == 0 || slices.Contains(values, "") {
			return ErrInvalidSpaceQuery
		}
	}
	return nil
}

// Normalize returns the Space with its QBE values sorted and deduped
func (s Space) Normalize() Space {
	if len(s.QBE) == 0 {
		return s
	}
	qbe := make(SpaceQuery, len(s.QBE))
	for key, values := range s.QBE {
		qbe[key] = slices.Compact(slices.Sorted(slices.Values(values)))
	}
	s.QBE = qbe
	return s
}

// Equal returns true if two normalized space definitions are equal
func (s Space) Equal(other Space) bool {
	return s.ID == other.ID && s.Name == other.Name &&
		s.Description == other.Description &&
		s.Selector.Equal(other.Selector) &&
		maps.EqualFunc(s.QBE, other.QBE, slices.Equal)
}
