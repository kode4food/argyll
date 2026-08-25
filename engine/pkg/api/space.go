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

	// Space defines a dynamic planning scope over registered steps
	Space struct {
		Selector    LabelSelector `json:"selector"`
		ID          SpaceID       `json:"id"`
		Name        Name          `json:"name"`
		Description string        `json:"description,omitempty"`
	}

	// LabelSelector selects steps by labels
	LabelSelector struct {
		MatchLabels Labels `json:"match_labels,omitempty"`
	}
)

var (
	ErrSpaceIDEmpty       = errors.New("space ID empty")
	ErrSpaceIDInvalid     = errors.New("space ID contains invalid characters")
	ErrSpaceNameEmpty     = errors.New("space name empty")
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
	if len(s.Selector.MatchLabels) > MaxLabelCount {
		return fmt.Errorf("%w: maximum is %d", ErrTooManyLabels, MaxLabelCount)
	}
	for key, value := range s.Selector.MatchLabels {
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
		s.Selector.MatchLabels.Equal(other.Selector.MatchLabels)
}

// Matches returns true when the step matches the space selector
func (s Space) Matches(step *Step) bool {
	return s.Selector.Matches(step.Labels)
}

// Matches returns true when all selector labels match the supplied labels
func (s LabelSelector) Matches(labels Labels) bool {
	for key, value := range s.MatchLabels {
		if labels[key] != value {
			return false
		}
	}
	return true
}
