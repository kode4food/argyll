package config

import (
	"errors"
	"fmt"

	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
	"github.com/kode4food/argyll/engine/pkg/util"
)

// Config holds the settings an Engine runs with, whether embedded or served
type Config struct {
	Timebox       timebox.Config
	NodeID        api.NodeID
	Nodes         []api.NodeID
	Work          api.WorkConfig
	StepTimeout   int64
	MemoCacheSize int
}

const (
	DefaultStepTimeout = 30 * api.Second

	DefaultFlowSnapshotRatio = 2.0
	DefaultTimeboxCacheSize  = 32768
	DefaultMemoCacheSize     = 65536
	DefaultNodeID            = "argyll-1"

	DefaultRetryMaxRetries  = 10
	DefaultRetryInitBackoff = 1000
	DefaultMaxRetryBackoff  = 60000
	DefaultRetryBackoffType = api.BackoffTypeExponential

	EngineStoreCacheSize = 512
)

var (
	ErrInvalidStepTimeout     = errors.New("step timeout must be positive")
	ErrInvalidRetryMaxRetries = errors.New(
		"retry max retries cannot be zero",
	)
	ErrInvalidRetryInitBackoff = errors.New(
		"retry initial backoff must be positive",
	)
	ErrInvalidRetryMaxBackoff = errors.New(
		"retry max backoff must be positive",
	)
	ErrRetryMaxBackoffTooSmall = errors.New(
		"retry max backoff must be >= retry initial backoff",
	)
	ErrInvalidRetryBackoffType = errors.New("invalid retry backoff type")
)

// NewDefaultConfig creates a configuration with sensible defaults for all
// engine settings, stores, and retry behavior
func NewDefaultConfig() *Config {
	return &Config{
		NodeID:  DefaultNodeID,
		Timebox: DefaultTimebox(),
		Work: api.WorkConfig{
			MaxRetries:  DefaultRetryMaxRetries,
			InitBackoff: DefaultRetryInitBackoff,
			MaxBackoff:  DefaultMaxRetryBackoff,
			BackoffType: DefaultRetryBackoffType,
		},
		StepTimeout:   DefaultStepTimeout,
		MemoCacheSize: DefaultMemoCacheSize,
	}
}

// DefaultTimebox returns the top-level Timebox defaults Argyll expects
func DefaultTimebox() timebox.Config {
	return timebox.DefaultConfig().With(timebox.Config{
		CacheSize: DefaultTimeboxCacheSize,
	})
}

// EngineStoreConfig returns the engine metadata store configuration derived
// from the shared Timebox defaults
func (c *Config) EngineStoreConfig() timebox.Config {
	return c.Timebox.With(timebox.Config{
		CacheSize:     EngineStoreCacheSize,
		TrimEvents:    true,
		SnapshotRatio: timebox.DefaultSnapshotRatio,
	})
}

// FlowStoreConfig returns the flow store configuration derived from the shared
// Timebox defaults
func (c *Config) FlowStoreConfig() timebox.Config {
	return c.Timebox.With(timebox.Config{
		SnapshotRatio: DefaultFlowSnapshotRatio,
		Indexer:       events.FlowIndexer,
	})
}

// WithWorkDefaults returns a copy of the config with zero-valued work fields
// filled in from defaults
func (c *Config) WithWorkDefaults() *Config {
	res := util.MutableCopy(c)
	if res.Work.MaxRetries == 0 {
		res.Work.MaxRetries = DefaultRetryMaxRetries
	}
	if res.Work.InitBackoff <= 0 {
		res.Work.InitBackoff = DefaultRetryInitBackoff
	}
	if res.Work.MaxBackoff <= 0 {
		res.Work.MaxBackoff = DefaultMaxRetryBackoff
	}
	if res.Work.BackoffType == "" {
		res.Work.BackoffType = DefaultRetryBackoffType
	}
	return res
}

// Validate checks that all configuration values are valid
func (c *Config) Validate() error {
	if c.StepTimeout <= 0 {
		return ErrInvalidStepTimeout
	}

	if c.Work.MaxRetries == 0 {
		return ErrInvalidRetryMaxRetries
	}

	if c.Work.InitBackoff <= 0 {
		return ErrInvalidRetryInitBackoff
	}

	if c.Work.MaxBackoff <= 0 {
		return ErrInvalidRetryMaxBackoff
	}

	if c.Work.MaxBackoff < c.Work.InitBackoff {
		return ErrRetryMaxBackoffTooSmall
	}

	if c.Work.BackoffType != api.BackoffTypeFixed &&
		c.Work.BackoffType != api.BackoffTypeLinear &&
		c.Work.BackoffType != api.BackoffTypeExponential {
		return fmt.Errorf("%w: %s", ErrInvalidRetryBackoffType,
			c.Work.BackoffType)
	}
	return nil
}
