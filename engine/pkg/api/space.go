package api

import (
	"errors"
	"fmt"
)

type (
	// SpaceID uniquely identifies a planning space
	SpaceID string

	// Spaces contains Spaces by their ID
	Spaces map[SpaceID]Space

	// SpaceSelection contains the Step IDs each Space's selector selected
	SpaceSelection map[SpaceID][]StepID

	// Space defines a dynamic planning scope over registered steps
	Space struct {
		Selector    Labels  `json:"selector"`
		ID          SpaceID `json:"id"`
		Name        Name    `json:"name"`
		Description string  `json:"description,omitempty"`
	}
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
	for key, value := range s.Selector {
		if key == "" || value == "" {
			return ErrInvalidMatchLabels
		}
	}
	return nil
}

// Equal returns true if two space definitions are equal
func (s Space) Equal(other Space) bool {
	return s.ID == other.ID && s.Name == other.Name &&
		s.Description == other.Description &&
		s.Selector.Equal(other.Selector)
}

// Matches returns true when all selector labels match the step's labels
func (s Space) Matches(step *Step) bool {
	for key, value := range s.Selector {
		if step.Labels[key] != value {
			return false
		}
	}
	return true
}
