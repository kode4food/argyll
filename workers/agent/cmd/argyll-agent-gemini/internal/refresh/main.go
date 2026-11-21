package main

import (
	"cmp"
	"errors"
	"flag"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/workers/agent/internal/catalog"
	"github.com/kode4food/argyll/workers/agent/internal/worker"
)

type (
	// section is one second-level heading of a Gemini documentation page
	section struct {
		id   string
		body string
	}

	// tieredRate is a published rate and its long-context rates by input cutoff
	tieredRate struct {
		base  *float64
		tiers map[int64]*float64
	}
)

const (
	docsURL           = "https://ai.google.dev/gemini-api/docs/"
	defaultDir        = "cmd/argyll-agent-gemini/internal/gemini/"
	keyPrefix         = "gemini/"
	dateLayout        = "January 2, 2006"
	exitFailed        = 1
	tokensPerThousand = 1000

	standardTab = `{% tab title="Standard" %}`
	tabEnd      = `{% /tab %}`
	tableStart  = `{% table %}`
	tableEnd    = `{% /table %}`
	headerEnd   = `{% hr /%}`
	freeCell    = "Free of charge"
	storageRate = "per hour"
)

var (
	ErrNoPrices         = errors.New("no Gemini prices matched the published models")
	ErrIncompleteModels = errors.New("incomplete Gemini model catalog")
)

var (
	// mediaSections list models that generate audio, images, video, or music
	mediaSections = [...]string{"audio-models", "generative-media-models", "music-models"}

	// deprecated marks a section whose listed models no longer serve new users
	deprecated = regexp.MustCompile(`(?:is|are) \[deprecated\]\(/docs/deprecations`)
	heading    = regexp.MustCompile(`(?m)^## .*\{% id="([^"]+)" %\}[ \t]*$`)
	tableModel = regexp.MustCompile("(?m)^\\s*\\* `([a-z0-9][a-z0-9.-]*)`\\s*$")
	linkModel  = regexp.MustCompile("\\[`([a-z0-9][a-z0-9.-]*)`\\]")
	modelID    = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]*$`)
	dollars    = regexp.MustCompile(`\$([0-9][0-9,]*(?:\.[0-9]+)?)`)
	through    = regexp.MustCompile(`through ([A-Z][a-z]+ [0-9]{1,2}, [0-9]{4})`)
	starting   = regexp.MustCompile(`starting ([A-Z][a-z]+ [0-9]{1,2}, [0-9]{4})`)
	longPrompt = regexp.MustCompile(`prompts > ([0-9]+)k`)
	modality   = regexp.MustCompile(`\(([^)]*)\)`)
)

func main() {
	dir := flag.String("dir", defaultDir, "directory of the model defaults to update")
	flag.Parse()
	err := Refresh(RefreshArgs{
		Dir:   *dir,
		Fetch: catalog.Fetch,
		Today: time.Now(),
	})
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(exitFailed)
	}
}

// RefreshArgs names the bundled defaults directory, how to fetch published documents, and the date
// whose prices apply
type RefreshArgs struct {
	Dir   string
	Fetch func(string) (string, error)
	Today time.Time
}

// Refresh updates the bundled Gemini model capabilities from the models and thinking guides, and
// the bundled prices from the pricing guide
func Refresh(args RefreshArgs) error {
	models, err := args.Fetch(docsURL + "models")
	if err != nil {
		return err
	}
	hidden := hiddenModels(models)
	thinking, err := args.Fetch(docsURL + "thinking")
	if err != nil {
		return err
	}
	efforts := thinkingLevels(thinking)
	if len(hidden) == 0 || len(efforts) == 0 {
		return ErrIncompleteModels
	}
	publishedCaps := map[string]worker.Capabilities{}
	for id, levels := range efforts {
		publishedCaps[keyPrefix+id] = worker.Capabilities{Efforts: levels}
	}
	for id := range hidden {
		publishedCaps[keyPrefix+id] = worker.Capabilities{Hidden: true}
	}
	pricing, err := args.Fetch(docsURL + "pricing")
	if err != nil {
		return err
	}
	published, err := modelPrices(pricing, args.Today, hidden)
	if err != nil {
		return err
	}
	capsPath := filepath.Join(args.Dir, "model-capabilities.json")
	caps, err := catalog.ReadTable[worker.Capabilities](capsPath)
	if err != nil {
		return err
	}
	pricesPath := filepath.Join(args.Dir, "model-prices.json")
	prices, err := catalog.ReadTable[worker.Price](pricesPath)
	if err != nil {
		return err
	}
	return catalog.WriteTables(
		catalog.Table{
			Path: capsPath,
			Values: catalog.Merge(catalog.MergeArgs[worker.Capabilities]{
				Existing:  caps,
				Published: publishedCaps,
			}),
		},
		catalog.Table{
			Path: pricesPath,
			Values: catalog.Merge(catalog.MergeArgs[worker.Price]{
				Existing:  prices,
				Published: published,
			}),
		},
	)
}

// hiddenModels reads the models guide's audio, media, music, and deprecated sections
func hiddenModels(doc string) map[string]bool {
	res := map[string]bool{}
	for _, s := range sections(doc) {
		if slices.Contains(mediaSections[:], s.id) || deprecated.MatchString(s.body) {
			for _, m := range tableModel.FindAllStringSubmatch(s.body, -1) {
				res[m[1]] = true
			}
		}
	}
	return res
}

// thinkingLevels reads the thinking guide's table of levels each model supports
func thinkingLevels(doc string) map[string][]api.Effort {
	res := map[string][]api.Effort{}
	for line := range strings.Lines(doc) {
		if id, levels := levelRow(line); len(levels) > 0 {
			res[id] = levels
		}
	}
	return res
}

func levelRow(line string) (string, []api.Effort) {
	cells := catalog.Cells(line)
	if len(cells) != 3 || !modelID.MatchString(cells[0]) {
		return "", nil
	}
	var levels []api.Effort
	for level := range strings.SplitSeq(cells[2], ",") {
		if level = strings.TrimSpace(level); level != "" {
			levels = append(levels, api.Effort(level))
		}
	}
	return cells[0], levels
}

// modelPrices reads each pricing section's standard paid rates that apply on the given day
func modelPrices(
	doc string, today time.Time, hidden map[string]bool,
) (map[string]worker.Price, error) {
	res := map[string]worker.Price{}
	for _, s := range sections(doc) {
		header, _, _ := strings.Cut(s.body, headerEnd)
		var ids []string
		for _, m := range linkModel.FindAllStringSubmatch(header, -1) {
			if hidden[m[1]] {
				continue
			}
			ids = append(ids, m[1])
		}
		if len(ids) == 0 {
			continue
		}
		rows := standardRows(s.body)
		input, err := cellRates(rows["Input price"], today)
		if err != nil {
			return nil, err
		}
		output, err := cellRates(rows["Output price"], today)
		if err != nil {
			return nil, err
		}
		cache, err := cellRates(rows["Context caching price"], today)
		if err != nil {
			return nil, err
		}
		if input.base == nil || output.base == nil {
			fmt.Printf("No published price for %s%s\n", keyPrefix, strings.Join(ids, ", "))
			continue
		}
		price := worker.Price{
			Input:      input.base,
			Output:     output.base,
			CacheInput: cache.base,
		}
		for _, over := range slices.Sorted(maps.Keys(input.tiers)) {
			price.Tiers = append(price.Tiers, worker.PriceTier{
				InputOver:  over,
				Input:      input.tiers[over],
				Output:     cmp.Or(output.tiers[over], output.base),
				CacheInput: cmp.Or(cache.tiers[over], cache.base),
			})
		}
		for _, id := range ids {
			res[keyPrefix+id] = price
		}
	}
	if len(res) == 0 {
		return nil, ErrNoPrices
	}
	return res, nil
}

// sections splits a page at its second-level headings
func sections(doc string) []section {
	idx := heading.FindAllStringSubmatchIndex(doc, -1)
	res := make([]section, 0, len(idx))
	for i, m := range idx {
		end := len(doc)
		if i+1 < len(idx) {
			end = idx[i+1][0]
		}
		res = append(res, section{
			id:   doc[m[2]:m[3]],
			body: doc[m[1]:end],
		})
	}
	return res
}

// standardRows maps each row label of a section's standard-tier table to its paid-tier cell
func standardRows(body string) map[string]string {
	if _, tab, ok := strings.Cut(body, standardTab); ok {
		body, _, _ = strings.Cut(tab, tabEnd)
	}
	_, table, ok := strings.Cut(body, tableStart)
	if !ok {
		return nil
	}
	table, _, _ = strings.Cut(table, tableEnd)
	res := map[string]string{}
	var cells []string
	flush := func() {
		if len(cells) == 3 {
			label, _, _ := strings.Cut(cells[0], "{%")
			label, _, _ = strings.Cut(label, " (")
			res[strings.TrimSpace(label)] = cells[2]
		}
		cells = nil
	}
	for line := range strings.Lines(table) {
		line = strings.TrimSpace(line)
		switch {
		case line == "---":
			flush()
		case strings.HasPrefix(line, "* "):
			cells = append(cells, strings.TrimPrefix(line, "* "))
		case len(cells) > 0 && line != "":
			cells[len(cells)-1] += "\n" + line
		}
	}
	flush()
	return res
}

// cellRates reads a paid-tier cell's base rate and long-context rates
func cellRates(cell string, today time.Time) (tieredRate, error) {
	res := tieredRate{tiers: map[int64]*float64{}}
	if strings.HasPrefix(cell, freeCell) {
		res.base = new(float64)
		return res, nil
	}
	for line := range strings.Lines(cell) {
		rate, over, err := lineRate(line, today)
		switch {
		case err != nil:
			return res, err
		case rate == nil:
		case over > 0:
			res.tiers[over] = rate
		case res.base == nil:
			res.base = rate
		}
	}
	return res, nil
}

// lineRate reads one rate and its long-context cutoff. Storage, audio-only, and out-of-date rates
// read as no rate
func lineRate(line string, today time.Time) (*float64, int64, error) {
	amount := dollars.FindStringSubmatch(line)
	if amount == nil || strings.Contains(line, storageRate) || audioOnly(line) {
		return nil, 0, nil
	}
	current, err := inEffect(line, today)
	if err != nil || !current {
		return nil, 0, err
	}
	rate, err := catalog.Rate("$" + amount[1])
	if err != nil {
		return nil, 0, err
	}
	over, err := cutoff(line)
	return rate, over, err
}

func audioOnly(line string) bool {
	m := modality.FindStringSubmatch(line)
	return m != nil && strings.Contains(m[1], "audio") && !strings.Contains(m[1], "text")
}

// inEffect reports whether a dated rate applies on the given day
func inEffect(line string, today time.Time) (bool, error) {
	day := today.Format(time.DateOnly)
	until, err := lineDate(through, line)
	if err != nil {
		return false, err
	}
	from, err := lineDate(starting, line)
	if err != nil {
		return false, err
	}
	return (until == "" || day <= until) && (from == "" || day >= from), nil
}

// lineDate reads a rate's dated bound, or nothing when the line has none
func lineDate(bound *regexp.Regexp, line string) (string, error) {
	m := bound.FindStringSubmatch(line)
	if m == nil {
		return "", nil
	}
	date, err := time.Parse(dateLayout, m[1])
	if err != nil {
		return "", err
	}
	return date.Format(time.DateOnly), nil
}

// cutoff reads a long-context rate's input cutoff, or zero for a base rate
func cutoff(line string) (int64, error) {
	m := longPrompt.FindStringSubmatch(line)
	if m == nil {
		return 0, nil
	}
	over, err := strconv.ParseInt(m[1], 10, 64)
	return over * tokensPerThousand, err
}
