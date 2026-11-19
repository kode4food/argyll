package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/kode4food/argyll/engine/pkg/util"
	"github.com/kode4food/argyll/workers/agent/internal/catalog"
	"github.com/kode4food/argyll/workers/agent/internal/worker"
)

type (
	modelCapabilities struct {
		Efforts  []string `json:"efforts,omitempty"`
		Endpoint string   `json:"endpoint,omitempty"`
		Hidden   bool     `json:"hidden,omitempty"`
	}

	modelCatalog struct {
		hidden      util.Set[string]
		listed      util.Set[string]
		recommended util.Set[string]
		prices      map[string]publishedPrice
	}
)

const (
	catalogURL  = "https://developers.openai.com/api/docs/models/all.md"
	pricingURL  = "https://developers.openai.com/api/docs/pricing.md"
	modelURL    = "https://developers.openai.com/api/docs/models/"
	defaultFile = "cmd/argyll-agent-openai/internal/openai/model-capabilities.json"
	exitFailed  = 1
)

var (
	ErrIncompleteCatalog = errors.New("missing OpenAI model catalog categories")
)

var (
	hiddenCategories = util.SetOf("Image", "Realtime & audio", "Embedding models", "ChatGPT models")
	modelLink        = regexp.MustCompile(`^- \[[^\]]+\]\(/api/docs/models/([a-z0-9.-]+)\.md\)`)
)

func main() {
	path := flag.String("file", defaultFile, "model capabilities file to update")
	flag.Parse()
	if err := Refresh(*path, catalog.Fetch); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(exitFailed)
	}
}

// Refresh updates the bundled OpenAI prices and model capabilities
func Refresh(path string, fetch func(string) (string, error)) error {
	document, err := fetch(catalogURL)
	if err != nil {
		return err
	}
	listing, err := readCatalog(strings.NewReader(document))
	if err != nil {
		return err
	}
	pricing, err := fetch(pricingURL)
	if err != nil {
		return err
	}
	listing.prices, err = readPrices(pricing)
	if err != nil {
		return err
	}
	models, err := catalog.ReadTable[modelCapabilities](path)
	if err != nil {
		return err
	}
	pricePath := filepath.Join(filepath.Dir(path), "model-prices.json")
	prices, err := catalog.ReadTable[worker.Price](pricePath)
	if err != nil {
		return err
	}
	for key := range listing.hidden {
		caps := models[key]
		caps.Hidden = true
		models[key] = caps
	}
	updateModels(models, prices, listing, fetch)
	if err := catalog.WriteTable(path, models); err != nil {
		return err
	}
	return catalog.WriteTable(pricePath, prices)
}

func readCatalog(r io.Reader) (modelCatalog, error) {
	missing := maps.Clone(hiddenCategories)
	models := modelCatalog{
		hidden:      util.SetOf[string](),
		listed:      util.SetOf[string](),
		recommended: util.SetOf[string](),
	}
	var category string
	s := bufio.NewScanner(r)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if heading, ok := strings.CutPrefix(line, "## "); ok {
			category = heading
			continue
		}
		match := modelLink.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		if hiddenCategories.Contains(category) {
			models.hidden.Add("openai/" + match[1])
			missing.Remove(category)
			continue
		}
		if category == "Open-weight models" || strings.Contains(line, ": Shut down.") {
			continue
		}
		models.listed.Add(match[1])
		if category == "Flagship models" || category == "More models" {
			models.recommended.Add(match[1])
		}
	}
	if err := s.Err(); err != nil {
		return modelCatalog{}, err
	}
	if !missing.IsEmpty() {
		categories := slices.Sorted(maps.Keys(missing))
		return modelCatalog{}, fmt.Errorf("%w: %v", ErrIncompleteCatalog, categories)
	}
	return models, nil
}
