package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"
)

type (
	// Feed folds settled values of its output Attributes from every Flow into
	// one Catalog-wide value that Steps read as an input
	Feed struct {
		Fold        *ScriptConfig `json:"fold"`
		ID          FeedID        `json:"id"`
		Name        Name          `json:"name"`
		Description string        `json:"description,omitempty"`
		Value       string        `json:"value"`
		Attributes  Slice[Name]   `json:"attributes"`
		SpaceID     SpaceID       `json:"space_id,omitempty"`
	}

	// FeedID uniquely identifies a Feed
	FeedID string

	// FeedState is the folded value of one Feed definition version
	FeedState struct {
		LastUpdated time.Time `json:"last_updated"`
		Value       any       `json:"value"`
		ID          FeedID    `json:"id"`
		Version     string    `json:"version"`
	}

	// FeedValues holds Feed values captured when a Flow starts
	FeedValues = Map[FeedID, any]

	// Feeds contains a map of Feeds by their ID
	Feeds = Map[FeedID, *Feed]

	// PlanFeed is a Feed a plan reads or folds into, with the planned Steps
	// whose outputs it folds
	PlanFeed struct {
		Feed    *Feed         `json:"feed"`
		Sources Slice[StepID] `json:"sources,omitempty"`
	}

	// PlanFeeds contains the Feeds of a plan by their ID
	PlanFeeds = Map[FeedID, PlanFeed]

	feedHash struct {
		Fold       *ScriptConfig `json:"fold"`
		Value      string        `json:"value"`
		Attributes []Name        `json:"attributes"`
		SpaceID    SpaceID       `json:"space_id"`
	}
)

// FeedCurrent names the current value inside a fold script
const FeedCurrent = Name("feed")

var (
	ErrFeedIDEmpty           = errors.New("feed ID empty")
	ErrFeedIDInvalid         = errors.New("feed ID contains invalid characters")
	ErrFeedNameEmpty         = errors.New("feed name empty")
	ErrFeedValueRequired     = errors.New("feed value required")
	ErrFeedFoldEmpty         = errors.New("feed fold empty")
	ErrFeedAttributesEmpty   = errors.New("feed attributes empty")
	ErrFeedAttributeEmpty    = errors.New("feed attribute empty")
	ErrFeedAttributeReserved = errors.New("feed attribute uses reserved name")
	ErrFeedAttributeRepeated = errors.New("feed attribute repeated")
	ErrFeedAttributeUnknown  = errors.New("feed attribute is no Step output")
	ErrFeedNotFound          = errors.New("feed not found")
	ErrFeedExists            = errors.New("feed exists")
	ErrInvalidFeed           = errors.New("invalid feed")
)

// Validate checks the definition's shape. The fold's language and syntax are
// left to the script registry, and the Space to the Catalog
func (f *Feed) Validate() error {
	if err := f.validateShape(); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidFeed, err)
	}
	return nil
}

// Equal returns true if two definitions are equal
func (f *Feed) Equal(other *Feed) bool {
	return f.ID == other.ID && f.Name == other.Name &&
		f.Description == other.Description && f.Value == other.Value &&
		slices.Equal(f.Attributes, other.Attributes) &&
		f.SpaceID == other.SpaceID &&
		f.Fold.Equal(other.Fold)
}

// Version hashes the seed, fold, Attributes, and Space of a definition. The
// Attributes are sorted, so their order never starts a new value
func (f *Feed) Version() string {
	// The hash holds only strings, which always marshal
	data, _ := json.Marshal(feedHash{
		Value:      f.Value,
		Fold:       f.Fold,
		Attributes: slices.Sorted(slices.Values(f.Attributes)),
		SpaceID:    f.SpaceID,
	})
	return sha256Hex(string(data))
}

// Seed returns the parsed initial value
func (f *Feed) Seed() (any, error) {
	var res any
	if err := json.Unmarshal([]byte(f.Value), &res); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFeedValueRequired, err)
	}
	return res, nil
}

// Folds reports whether a Step's outputs feed this Feed, given the Steps its
// Space selects
func (f *Feed) Folds(st *Step, selected []StepID) bool {
	if f.SpaceID != NoSpace && !slices.Contains(selected, st.ID) {
		return false
	}
	return slices.ContainsFunc(f.Attributes, func(name Name) bool {
		attr, ok := st.Attributes[name]
		return ok && attr.IsOutput()
	})
}

func (f *Feed) validateShape() error {
	if f.ID == "" {
		return ErrFeedIDEmpty
	}
	if SanitizeID(f.ID) != f.ID {
		return ErrFeedIDInvalid
	}
	if f.Name == "" {
		return ErrFeedNameEmpty
	}
	if f.Value == "" {
		return ErrFeedValueRequired
	}
	if err := validateJSON(f.Value); err != nil {
		return fmt.Errorf("%w: %w", ErrFeedValueRequired, err)
	}
	if f.Fold == nil || f.Fold.Script == "" {
		return ErrFeedFoldEmpty
	}
	return f.validateAttributes()
}

func (f *Feed) validateAttributes() error {
	if len(f.Attributes) == 0 {
		return ErrFeedAttributesEmpty
	}
	for i, name := range f.Attributes {
		if name == "" {
			return ErrFeedAttributeEmpty
		}
		if name == FeedCurrent {
			return fmt.Errorf("%w: %s", ErrFeedAttributeReserved, name)
		}
		if slices.Contains(f.Attributes[:i], name) {
			return fmt.Errorf("%w: %s", ErrFeedAttributeRepeated, name)
		}
	}
	return nil
}
