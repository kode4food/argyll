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

	// SpaceQuery matches steps against alternative tag sets, selecting a step
	// that carries every tag of any one of them
	SpaceQuery []SpaceQueryTerm

	// SpaceQueryTerm matches steps carrying every one of its tags
	SpaceQueryTerm []string
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
	return s.QBE.Validate()
}

// Normalize returns the Space with its QBE terms sorted and deduped
func (s Space) Normalize() Space {
	if len(s.QBE) == 0 {
		return s
	}
	s.QBE = s.QBE.Normalize()
	return s
}

// Equal returns true if two normalized space definitions are equal
func (s Space) Equal(other Space) bool {
	sameIdentity := s.ID == other.ID && s.Name == other.Name &&
		s.Description == other.Description
	return sameIdentity && s.Selector.Equal(other.Selector) &&
		slices.EqualFunc(s.QBE, other.QBE, slices.Equal)
}

// Validate checks that every term carries at least one non-empty tag and that
// the query stays within the tag limit
func (q SpaceQuery) Validate() error {
	tags := 0
	for _, term := range q {
		if len(term) == 0 || slices.Contains(term, "") {
			return ErrInvalidSpaceQuery
		}
		tags += len(term)
	}
	if tags > MaxTagCount {
		return fmt.Errorf("%w: maximum is %d", ErrTooManyTags, MaxTagCount)
	}
	return nil
}

// Normalize returns the query with each term's tags sorted and deduped, and the
// terms themselves sorted and deduped
func (q SpaceQuery) Normalize() SpaceQuery {
	res := make(SpaceQuery, 0, len(q))
	for _, term := range q {
		res = append(res, slices.Compact(slices.Sorted(slices.Values(term))))
	}
	slices.SortFunc(res, slices.Compare)
	return slices.CompactFunc(res, slices.Equal)
}
