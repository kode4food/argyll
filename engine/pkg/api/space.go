package api

import (
	"errors"
	"fmt"
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

	// SpaceQuery matches steps carrying every one of its tags
	SpaceQuery []string
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
// The selector is a tag predicate, so its language is left to the script
// registry to accept or reject
func (s Space) ValidateSelector() error {
	if s.Selector == nil {
		return ErrSpaceSelectorEmpty
	}
	if s.Selector.Script == "" {
		return ErrScriptEmpty
	}
	if len(s.QBE) > MaxTagCount {
		return fmt.Errorf("%w: maximum is %d", ErrTooManyTags, MaxTagCount)
	}
	if slices.Contains(s.QBE, "") {
		return ErrInvalidSpaceQuery
	}
	return nil
}

// Normalize returns the Space with its QBE tags sorted and deduped
func (s Space) Normalize() Space {
	if len(s.QBE) == 0 {
		return s
	}
	s.QBE = slices.Compact(slices.Sorted(slices.Values(s.QBE)))
	return s
}

// Equal returns true if two normalized space definitions are equal
func (s Space) Equal(other Space) bool {
	sameIdentity := s.ID == other.ID && s.Name == other.Name &&
		s.Description == other.Description
	return sameIdentity && s.Selector.Equal(other.Selector) &&
		slices.Equal(s.QBE, other.QBE)
}
