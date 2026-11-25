package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"maps"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/util"
	"github.com/kode4food/argyll/workers/agent/internal/catalog"
	"github.com/kode4food/argyll/workers/agent/internal/worker"
)

type (
	offerFile struct {
		Products map[string]offerProduct `json:"products"`
		Terms    offerTerms              `json:"terms"`
	}

	offerProduct struct {
		Attributes map[string]string `json:"attributes"`
	}

	offerTerms struct {
		OnDemand map[string]map[string]offerTerm `json:"OnDemand"`
	}

	offerTerm struct {
		PriceDimensions map[string]offerDimension `json:"priceDimensions"`
	}

	offerDimension struct {
		Unit         string            `json:"unit"`
		PricePerUnit map[string]string `json:"pricePerUnit"`
	}

	// rateRow is one standard on-demand rate for a model, by its display name
	rateRow struct {
		scope rateScope
		kind  string
		rate  float64
	}

	rateScope struct {
		model  string
		global bool
		long   bool
	}

	modelInfo struct {
		name         string
		capabilities worker.Capabilities
		inputOver    int64
	}
)

const (
	priceListURL = "https://pricing.us-east-1.amazonaws.com/offers/v1.0/aws/" +
		"%s/current/%s/index.json"
	modelDocsURL          = "https://docs.aws.amazon.com/bedrock/latest/userguide/"
	supportedIcon         = "![supported]"
	defaultDir            = "cmd/argyll-agent-bedrock/internal/bedrock/"
	defaultRegion         = "us-east-1"
	exitFailed            = 1
	perThousandToMillion  = "e3"
	tokensPerThousand     = 1000
	legacySonnetInputOver = 200_000

	modelIDColumn  = 1
	geoIDColumn    = 3
	globalIDColumn = 4
)

var (
	ErrNoPrices         = errors.New("no Bedrock prices matched the published models")
	ErrIncompleteModels = errors.New("incomplete Bedrock model catalog")
	ErrMissingThreshold = errors.New("missing Bedrock long-context threshold")
)

var (
	// priceOffers are the AWS Price List offers that carry Bedrock token rates
	priceOffers = [...]string{
		"AmazonBedrock", "AmazonBedrockFoundationModels", "AmazonBedrockService",
	}

	// excludedRates are usage types other than standard on-demand calls through Converse
	excludedRates = regexp.MustCompile(
		`(?i)mantle|1h|30m|batch|flex|priority|ultrafast|reserved|latencyoptimized`,
	)
	cardLink          = regexp.MustCompile(`\[([^\]]+)\]\((model-card-[a-z0-9-]+\.md)\)`)
	supportedConverse = regexp.MustCompile(`!\[supported\]\([^)]*\) Converse\b`)
	modelID           = regexp.MustCompile(`^[a-z0-9][a-z0-9.:-]*$`)
	inputRate         = regexp.MustCompile(`^(text )?input tokens?$`)
	outputRate        = regexp.MustCompile(`^(text )?output tokens?$`)
	toolUse           = regexp.MustCompile(
		`!\[supported\]\([^)]*\) \[Client-side tool (?:calling|use)\]`,
	)
	legacyTools = regexp.MustCompile(
		`^(?:amazon\.nova-(?:micro|lite|pro)-v1:0|anthropic\.claude-(?:3|opus-4|sonnet-4))`,
	)
	effortLevel = regexp.MustCompile(`\b(?:none|minimal|low|medium|high|xhigh|max)\b`)
	effortList  = regexp.MustCompile(`effort(?: level)?(?: is)? configurable(?: —|:)? ([^;)\n]+)`)
	claudeName  = regexp.MustCompile(
		`Claude (?:Opus|Sonnet|Haiku|Fable|Mythos) [0-9]+(?:\.[0-9]+)*`,
	)
	effortModel = regexp.MustCompile(`^(?:[a-z]+\.)?(?:anthropic\.claude-|amazon\.nova-|openai\.)`)
	longPrompt  = regexp.MustCompile(
		`(?i)(?:prompts (?:that exceed|over)|more than) ([0-9,]+)(K)? (?:input )?tokens`,
	)
)

func main() {
	dir := flag.String("dir", defaultDir, "directory of the model defaults to update")
	region := flag.String("region", defaultRegion, "AWS region whose prices to bundle")
	flag.Parse()
	err := Refresh(RefreshArgs{
		Dir:    *dir,
		Region: *region,
		Fetch:  catalog.Fetch,
	})
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(exitFailed)
	}
}

// RefreshArgs names the bundled defaults directory, the AWS region whose prices to bundle, and how
// to fetch published documents
type RefreshArgs struct {
	Dir    string
	Region string
	Fetch  func(string) (string, error)
}

// Refresh updates the bundled Bedrock model capabilities from the Bedrock model cards, and the
// bundled prices from the AWS Price List for the region
func Refresh(args RefreshArgs) error {
	models, err := modelNames(args.Fetch)
	if err != nil {
		return err
	}
	capsPath := filepath.Join(args.Dir, "model-capabilities.json")
	caps, err := catalog.ReadTable[worker.Capabilities](capsPath)
	if err != nil {
		return err
	}
	publishedCaps := map[string]worker.Capabilities{}
	for id, model := range models {
		publishedCaps["bedrock/"+id] = model.capabilities
	}
	var rows []rateRow
	for _, offer := range priceOffers {
		doc, err := args.Fetch(fmt.Sprintf(priceListURL, offer, args.Region))
		if res, ok := errors.AsType[*catalog.HTTPError](err); ok &&
			res.Status == http.StatusNotFound {
			fmt.Printf("No %s prices for %s\n", offer, args.Region)
			continue
		}
		if err != nil {
			return err
		}
		var f offerFile
		if err := json.Unmarshal([]byte(doc), &f); err != nil {
			return err
		}
		rows = append(rows, offerRates(&f)...)
	}
	rates := map[rateScope]worker.Rates{}
	for _, r := range rows {
		rates[r.scope] = withRate(rates[r.scope], r.kind, r.rate)
	}
	published := map[string]worker.Price{}
	for _, id := range slices.Sorted(maps.Keys(models)) {
		model := models[id]
		if model.capabilities.Hidden {
			continue
		}
		scope := rateScope{
			model:  priceName(model.name),
			global: strings.HasPrefix(id, "global."),
		}
		base, ok := rates[scope]
		if !ok || base.Input == nil || base.Output == nil {
			fmt.Printf("No published price for bedrock/%s (%s)\n", id, model.name)
			continue
		}
		price := worker.Price{
			Input:       base.Input,
			Output:      base.Output,
			CacheInput:  base.CacheInput,
			CacheOutput: base.CacheOutput,
		}
		scope.long = true
		if tier, ok := rates[scope]; ok {
			// Sonnet 4's beta long-context threshold is in the Messages request documentation
			if model.inputOver == 0 && strings.Contains(id, "anthropic.claude-sonnet-4-20250514") {
				model.inputOver = legacySonnetInputOver
			}
			if model.inputOver <= 0 {
				return fmt.Errorf("%w: %s", ErrMissingThreshold, id)
			}
			price.Tiers = []worker.PriceTier{{
				InputOver:   model.inputOver,
				Input:       tier.Input,
				Output:      tier.Output,
				CacheInput:  tier.CacheInput,
				CacheOutput: tier.CacheOutput,
			}}
		}
		published["bedrock/"+id] = price
	}
	if len(published) == 0 {
		return ErrNoPrices
	}
	path := filepath.Join(args.Dir, "model-prices.json")
	existing, err := catalog.ReadTable[worker.Price](path)
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
			Path: path,
			Values: catalog.Merge(catalog.MergeArgs[worker.Price]{
				Existing:  existing,
				Published: published,
			}),
		},
	)
}

// modelNames reads each model card's runtime IDs, tool support, effort levels, and price threshold
func modelNames(fetch func(string) (string, error)) (map[string]modelInfo, error) {
	overview, err := fetch(modelDocsURL + "model-cards.md")
	if err != nil {
		return nil, err
	}
	links := cardLink.FindAllStringSubmatch(overview, -1)
	if len(links) == 0 {
		return nil, ErrIncompleteModels
	}
	models := map[string]modelInfo{}
	var thinking string
	for _, link := range links {
		card, err := fetch(modelDocsURL + link[2])
		if res, ok := errors.AsType[*catalog.HTTPError](err); ok &&
			res.Status == http.StatusNotFound {
			fmt.Printf("No model card for %s\n", link[1])
			continue
		}
		if err != nil {
			return nil, err
		}
		ids := runtimeIDs(card)
		eligible := conversable(card) && supportsTools(card, ids) && textOnlyOutput(card)
		caps := worker.Capabilities{Hidden: !eligible}
		effort := slices.ContainsFunc(ids, effortModel.MatchString)
		if !caps.Hidden && effort {
			caps.Efforts = readEfforts(card)
		}
		adaptive := !caps.Hidden && effort && len(caps.Efforts) == 0 &&
			strings.Contains(card, "claude-messages-adaptive-thinking.html")
		if adaptive && thinking == "" {
			thinking, err = fetch(modelDocsURL + "claude-messages-adaptive-thinking.md")
			if err != nil {
				return nil, err
			}
		}
		if adaptive {
			caps.Efforts = readAdaptiveEfforts(thinking, api.Name(link[1]))
		}
		var cutoff int64
		if match := longPrompt.FindStringSubmatch(card); match != nil {
			cutoff, err = strconv.ParseInt(strings.ReplaceAll(match[1], ",", ""), 10, 64)
			if err != nil {
				return nil, err
			}
			if match[2] != "" {
				cutoff *= tokensPerThousand
			}
		}
		for _, id := range ids {
			models[id] = modelInfo{
				name:         strings.ReplaceAll(link[1], "\\", ""),
				capabilities: caps,
				inputOver:    cutoff,
			}
		}
	}
	if len(models) == 0 {
		return nil, ErrIncompleteModels
	}
	return models, nil
}

// conversable reports whether a model card lists Converse, the API the worker calls, as supported.
// Cards show it either as a cell in one combined table or as a column of an API table
func conversable(card string) bool {
	if supportedConverse.MatchString(card) {
		return true
	}
	lines := strings.Split(card, "\n")
	for i, line := range lines {
		col := slices.Index(catalog.Cells(line), "**Converse**")
		if col < 0 || i+2 >= len(lines) {
			continue
		}
		row := catalog.Cells(lines[i+2])
		return col < len(row) && strings.Contains(row[col], supportedIcon)
	}
	return false
}

func textOnlyOutput(card string) bool {
	column := -1
	text := false
	for line := range strings.SplitSeq(card, "\n") {
		cells := catalog.Cells(line)
		if col := slices.IndexFunc(cells, func(cell string) bool {
			return strings.Trim(cell, "*") == "Output Modalities"
		}); col >= 0 {
			column = col
			continue
		}
		if column < 0 {
			continue
		}
		if len(cells) <= column {
			return text
		}
		cell := cells[column]
		if !strings.Contains(cell, supportedIcon) {
			continue
		}
		if !strings.HasSuffix(cell, " Text") {
			return false
		}
		text = true
	}
	return text
}

// supportsTools also recognizes older cards covered by the Nova and Claude tool-use schemas
func supportsTools(card string, ids []string) bool {
	if toolUse.MatchString(card) ||
		strings.Contains(card, "Tool use and response streaming are supported separately") ||
		strings.Contains(card, "Client tools and toolsets using `bedrock-runtime`") {
		return true
	}
	return slices.ContainsFunc(ids, legacyTools.MatchString)
}

func readEfforts(card string) []api.Effort {
	for line := range strings.SplitSeq(card, "\n") {
		if !strings.Contains(line, "**Reasoning:**") {
			continue
		}
		match := effortList.FindStringSubmatch(line)
		if match == nil {
			return nil
		}
		var res []api.Effort
		for _, level := range effortLevel.FindAllString(match[1], -1) {
			res = append(res, api.Effort(level))
		}
		return slices.Compact(res)
	}
	return nil
}

// readAdaptiveEfforts applies the model restrictions in the adaptive-thinking effort table
func readAdaptiveEfforts(doc string, name api.Name) []api.Effort {
	levels := util.Set[api.Effort]{}
	for line := range strings.SplitSeq(doc, "\n") {
		cells := catalog.Cells(line)
		if len(cells) != 2 {
			continue
		}
		level := effortLevel.FindString(cells[0])
		if level == "" || !strings.HasPrefix(cells[0], level) {
			continue
		}
		models := claudeName.FindAllString(cells[1], -1)
		if len(models) == 0 || slices.Contains(models, string(name)) {
			levels.Add(api.Effort(level))
		}
	}
	var res []api.Effort
	for _, level := range []api.Effort{"low", "medium", "high", "xhigh", "max"} {
		if levels.Contains(level) {
			res = append(res, level)
		}
	}
	return res
}

func priceName(name string) string {
	if name == "Nova 2 Lite" {
		name = "Nova 2.0 Lite"
	}
	name = strings.TrimPrefix(name, "OpenAI ")
	name = strings.TrimSuffix(name, " Instruct")
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, name)
}

// runtimeIDs reads the model and inference IDs from a model card's bedrock-runtime row
func runtimeIDs(card string) []string {
	for line := range strings.SplitSeq(card, "\n") {
		cells := catalog.Cells(line)
		if len(cells) <= globalIDColumn || cells[0] != "bedrock-runtime" {
			continue
		}
		var res []string
		for _, column := range []int{modelIDColumn, geoIDColumn, globalIDColumn} {
			for id := range strings.SplitSeq(cells[column], "<br />") {
				id = strings.Trim(strings.TrimSpace(id), "`")
				if modelID.MatchString(id) {
					res = append(res, id)
				}
			}
		}
		return res
	}
	return nil
}

// offerRates reads an offer's standard on-demand token rates per million tokens
func offerRates(f *offerFile) []rateRow {
	var res []rateRow
	for sku, product := range f.Products {
		a := product.Attributes
		if a["model"] == "" || excludedRates.MatchString(a["usagetype"]) {
			continue
		}
		if feature := a["feature"]; feature != "" && feature != "On-demand Inference" {
			continue
		}
		rate, ok := onDemandRate(f.Terms.OnDemand[sku])
		if !ok {
			continue
		}
		kind := strings.ToLower(a["featureType"] + " " + a["featuretype"])
		if !strings.Contains(kind, "cache") {
			kind = strings.ToLower(a["inferenceType"])
		}
		if kind == "" {
			kind = strings.ToLower(a["tokenType"])
		}
		kind, global := strings.CutSuffix(kind, " global")
		usage := strings.ToLower(a["usagetype"])
		long := strings.Contains(usage, "long_ctx") || strings.Contains(usage, "long-context")
		kind = strings.TrimSuffix(kind, " long context")
		scope := rateScope{
			model:  priceName(a["model"]),
			global: global || strings.Contains(strings.ToLower(a["usagetype"]), "global"),
			long:   long,
		}
		res = append(res, rateRow{
			scope: scope,
			kind:  kind,
			rate:  rate,
		})
	}
	return res
}

func onDemandRate(terms map[string]offerTerm) (float64, bool) {
	for _, term := range terms {
		for _, d := range term.PriceDimensions {
			price := d.PricePerUnit["USD"]
			switch strings.ToLower(d.Unit) {
			case "1m tokens":
			case "1k tokens":
				// Shifting the decimal while parsing avoids the noise of multiplying floats
				price += perThousandToMillion
			default:
				return 0, false
			}
			rate, err := strconv.ParseFloat(price, 64)
			return rate, err == nil && rate >= 0 && !math.IsNaN(rate) && !math.IsInf(rate, 0)
		}
	}
	return 0, false
}

func withRate(r worker.Rates, kind string, rate float64) worker.Rates {
	switch {
	case inputRate.MatchString(kind):
		r.Input = &rate
	case outputRate.MatchString(kind):
		r.Output = &rate
	case strings.Contains(kind, "cache read"):
		r.CacheInput = &rate
	case strings.Contains(kind, "cache write"):
		r.CacheOutput = &rate
	}
	return r
}
