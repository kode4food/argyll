package main

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/kode4food/argyll/workers/agent/internal/catalog"
	"github.com/kode4food/argyll/workers/agent/internal/worker"
)

type publishedPrice struct {
	base      worker.Rates
	long      worker.Rates
	inputOver int64
}

const (
	specializedHeader = "| Category | Model | Input | Output | Cached input |" +
		" 30m write | 12h write |"
	tokensPerK         = 1000
	longRateOffset     = 5
	pairedRateColumns  = 2 * longRateOffset
	specializedColumns = 7
	specializedOutput  = 3
)

var (
	ErrIncompletePricing = errors.New("incomplete OpenAI standard pricing table")
)

var (
	priceID      = regexp.MustCompile("^[a-z][a-z0-9.-]*$")
	shortContext = regexp.MustCompile(`^\(<([0-9]+)K context length\)$`)
)

func readPrices(document string) (map[string]publishedPrice, error) {
	_, section, _ := strings.Cut(document, "\n### Standard pricing data\n")
	section, _, _ = strings.Cut(section, "\n### ")
	prices := map[string]publishedPrice{}
	columns := 0
	for line := range strings.SplitSeq(section, "\n") {
		cells := catalog.Cells(line)
		if len(cells) == 0 {
			continue
		}
		if cells[0] == "Model" {
			columns = standardColumns(cells)
			continue
		}
		id, qualifier, _ := strings.Cut(cells[0], " ")
		if !priceID.MatchString(id) {
			continue
		}
		if columns == 0 || len(cells) != columns {
			return nil, fmt.Errorf("%w: %s", ErrIncompletePricing, id)
		}
		if err := readPrice(prices, id, cells[1:]); err != nil {
			return nil, err
		}
		match := shortContext.FindStringSubmatch(qualifier)
		if match == nil {
			continue
		}
		cutoff, err := strconv.ParseInt(match[1], 10, 64)
		if err != nil || cutoff <= 0 {
			return nil, fmt.Errorf("%w: %s", ErrIncompletePricing, id)
		}
		price := prices[id]
		price.inputOver = cutoff * tokensPerK
		prices[id] = price
	}
	if len(prices) == 0 {
		return nil, ErrIncompletePricing
	}
	if err := readSpecializedPrices(prices, document); err != nil {
		return nil, err
	}
	return prices, nil
}

func standardColumns(cells []string) int {
	if slices.Equal(cells, []string{"Model", "Input", "Output", "Cached input"}) {
		return len(cells)
	}
	if slices.Equal(cells, []string{
		"Model", "Short context input", "Short context output", "Short context cached input",
		"Short context 30m write", "Short context 12h write", "Long context input",
		"Long context output", "Long context cached input", "Long context 30m write",
		"Long context 12h write",
	}) {
		return len(cells)
	}
	return 0
}

func readPrice(prices map[string]publishedPrice, id string, cells []string) error {
	base, err := readRates(cells)
	if err != nil {
		return fmt.Errorf("%w: %s", err, id)
	}
	if base.Input == nil || base.Output == nil {
		return fmt.Errorf("%w: %s", ErrIncompletePricing, id)
	}
	price := publishedPrice{base: base}
	if len(cells) != pairedRateColumns {
		prices[id] = price
		return nil
	}
	price.long, err = readRates(cells[longRateOffset:])
	if err != nil {
		return fmt.Errorf("%w: %s", err, id)
	}
	prices[id] = price
	return nil
}

func readSpecializedPrices(prices map[string]publishedPrice, document string) error {
	_, section, ok := strings.Cut(document, "\n"+specializedHeader+"\n")
	if !ok {
		return ErrIncompletePricing
	}
	section, _, _ = strings.Cut(section, "\n\n")
	for line := range strings.SplitSeq(section, "\n") {
		cells := catalog.Cells(line)
		if len(cells) != specializedColumns || cells[specializedOutput] == "-" {
			continue
		}
		id, suffix, _ := strings.Cut(cells[1], " ")
		if !priceID.MatchString(id) {
			continue
		}
		rates, err := readRates(cells[2:])
		if err != nil {
			return fmt.Errorf("%w: %s", err, id)
		}
		price := prices[id]
		if suffix == "(long context)" {
			price.long = rates
		} else {
			price.base = rates
		}
		prices[id] = price
	}
	return nil
}

func readRates(cells []string) (worker.Rates, error) {
	var rates worker.Rates
	fields := []**float64{&rates.Input, &rates.Output, &rates.CacheInput, &rates.CacheOutput}
	for idx, target := range fields[:min(len(cells), len(fields))] {
		rate, err := catalog.Rate(cells[idx])
		if err != nil {
			return worker.Rates{}, err
		}
		*target = rate
	}
	return rates, nil
}
