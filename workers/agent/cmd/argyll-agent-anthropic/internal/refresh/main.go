package main

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/kode4food/argyll/workers/agent/internal/catalog"
	"github.com/kode4food/argyll/workers/agent/internal/worker"
)

const (
	pricingURL = "https://platform.claude.com/docs/en/about-claude/pricing.md"
	cachingURL = "https://platform.claude.com/docs/en/build-with-claude/prompt-caching.md"
	defaultDir = "cmd/argyll-agent-anthropic/internal/anthropic/"
	exitFailed = 1

	inputColumn      = 1
	cacheWriteColumn = 2
	cacheReadColumn  = 4
	outputColumn     = 5
)

var (
	ErrIncompletePricing    = errors.New("incomplete Anthropic pricing table")
	ErrIncompleteThresholds = errors.New("incomplete Anthropic cache thresholds")
)

var (
	modelName = regexp.MustCompile(
		"Claude (?:Fable|Mythos|Opus|Sonnet|Haiku) (?:[0-9.]+|Preview)",
	)
	publishedID  = regexp.MustCompile(`(?m)^\s+- (claude-[a-z0-9-]+)\s*$`)
	snapshot     = regexp.MustCompile("-20[0-9]{6}$")
	longPrompt   = regexp.MustCompile("for prompts over ([0-9,]+) tokens")
	cacheMinimum = regexp.MustCompile(`^\* ([0-9,]+) tokens for `)
	priceHeader  = [...]string{
		"Model", "Base input tokens", "5m cache writes", "1h cache writes",
		"Cache hits and refreshes", "Output tokens",
	}
)

func main() {
	if err := Refresh(defaultDir, catalog.Fetch); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(exitFailed)
	}
}

// Refresh updates the bundled Anthropic prices and cache thresholds
func Refresh(dir string, fetch func(string) (string, error)) error {
	pricing, err := fetch(pricingURL)
	if err != nil {
		return err
	}
	caching, err := fetch(cachingURL)
	if err != nil {
		return err
	}
	prices, err := readPrices(pricing)
	if err != nil {
		return err
	}
	minimums, err := readMinimums(caching)
	if err != nil {
		return err
	}
	pricePath := filepath.Join(dir, "model-prices.json")
	thresholdPath := filepath.Join(dir, "prefix-thresholds.json")
	oldPrices, err := catalog.ReadTable[worker.Price](pricePath)
	if err != nil {
		return err
	}
	oldMinimums, err := catalog.ReadTable[int](thresholdPath)
	if err != nil {
		return err
	}
	for _, match := range publishedID.FindAllStringSubmatch(caching, -1) {
		copySnapshot(prices, "anthropic/"+match[1])
		copySnapshot(minimums, match[1])
	}
	updateDefaults(updateDefaultsArgs[worker.Price]{
		existing:  oldPrices,
		published: prices,
	})
	updateDefaults(updateDefaultsArgs[int]{
		existing:  oldMinimums,
		published: minimums,
	})
	if err := catalog.WriteTable(pricePath, oldPrices); err != nil {
		return err
	}
	return catalog.WriteTable(thresholdPath, oldMinimums)
}

func readPrices(document string) (map[string]worker.Price, error) {
	_, section, _ := strings.Cut(document, "\n## Model pricing\n")
	section, _, _ = strings.Cut(section, "\n## ")
	prices := map[string]worker.Price{}
	headerFound := false
	for line := range strings.SplitSeq(section, "\n") {
		cells := catalog.Cells(line)
		if len(cells) == 0 {
			continue
		}
		if cells[0] == "Model" {
			headerFound = slices.Equal(cells, priceHeader[:])
			continue
		}
		id := modelID(modelName.FindString(cells[0]))
		if id == "" {
			continue
		}
		if !headerFound || len(cells) != len(priceHeader) {
			return nil, ErrIncompletePricing
		}
		rates, err := readRates(cells)
		if err != nil {
			return nil, fmt.Errorf("%w: %s: %w", ErrIncompletePricing, id, err)
		}
		key := "anthropic/" + id
		match := longPrompt.FindStringSubmatch(cells[0])
		if match == nil {
			prices[key] = worker.Price{Rates: rates}
			continue
		}
		cutoff, err := strconv.ParseInt(strings.ReplaceAll(match[1], ",", ""), 10, 64)
		price := prices[key]
		if err != nil || price.Input == nil || cutoff <= 0 {
			return nil, fmt.Errorf("%w: %s", ErrIncompletePricing, id)
		}
		price.Tiers = append(price.Tiers, worker.PriceTier{
			InputOver: cutoff,
			Rates:     rates,
		})
		prices[key] = price
	}
	if len(prices) == 0 {
		return nil, ErrIncompletePricing
	}
	return prices, nil
}

func readRates(cells []string) (worker.Rates, error) {
	var rates worker.Rates
	fields := []**float64{&rates.Input, &rates.CacheOutput, &rates.CacheInput, &rates.Output}
	for idx, column := range []int{inputColumn, cacheWriteColumn, cacheReadColumn, outputColumn} {
		amount, _, ok := strings.Cut(cells[column], " / MTok")
		if !ok {
			return worker.Rates{}, fmt.Errorf("%w: %s", ErrIncompletePricing, cells[column])
		}
		rate, err := catalog.Rate(amount)
		if err != nil || rate == nil {
			return worker.Rates{}, fmt.Errorf("%w: %s", ErrIncompletePricing, cells[column])
		}
		*fields[idx] = rate
	}
	return rates, nil
}

func readMinimums(document string) (map[string]int, error) {
	_, section, _ := strings.Cut(document, "the minimum cacheable prompt length is:")
	section, _, _ = strings.Cut(strings.TrimSpace(section), "\n\n")
	minimums := map[string]int{}
	for line := range strings.SplitSeq(section, "\n") {
		match := cacheMinimum.FindStringSubmatch(line)
		if match == nil {
			return nil, ErrIncompleteThresholds
		}
		count, err := strconv.Atoi(strings.ReplaceAll(match[1], ",", ""))
		names := modelName.FindAllString(line[len(match[0]):], -1)
		if err != nil || count <= 0 || len(names) == 0 {
			return nil, ErrIncompleteThresholds
		}
		for _, name := range names {
			minimums[modelID(name)] = count
		}
	}
	return minimums, nil
}

func modelID(name string) string {
	return strings.NewReplacer(" ", "-", ".", "-").Replace(strings.ToLower(name))
}

func copySnapshot[T any](table map[string]T, key string) {
	model := snapshot.ReplaceAllString(key, "")
	if value, ok := table[model]; ok {
		table[key] = value
	}
}

type updateDefaultsArgs[T any] struct {
	existing  map[string]T
	published map[string]T
}

func updateDefaults[T any](args updateDefaultsArgs[T]) {
	for _, key := range slices.Sorted(maps.Keys(args.existing)) {
		copySnapshot(args.published, key)
		if _, ok := args.published[key]; !ok {
			fmt.Printf("Retained %s without a published value\n", key)
		}
	}
	maps.Copy(args.existing, args.published)
}
