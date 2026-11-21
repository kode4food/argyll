package main

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/util"
	"github.com/kode4food/argyll/workers/agent/internal/worker"
)

type modelDocument string

const (
	responseEndpoint = "responses"
	chatEndpoint     = "chat_completions"
)

var (
	longContext = regexp.MustCompile(`(?i)(?:more than |over |>\s*)([0-9,]+)(k)? input tokens`)
	effortList  = regexp.MustCompile(
		`(?i)(?:reasoning[._ ]effort supports\s*:?|supports reasoning[._ ]effort:)\s+([^.]+)\.`,
	)
	effortLevel = regexp.MustCompile(`\b(?:none|minimal|low|medium|high|xhigh|max)\b`)
	snapshotID  = regexp.MustCompile(`(?m)^- \x60([a-z0-9.-]+)\x60[^\n]*`)
)

func updateModels(
	models map[string]modelCapabilities, prices map[string]worker.Price,
	listing modelCatalog, fetch func(string) (string, error),
) {
	for id := range listing.prices {
		listing.listed.Add(id)
	}
	seen := util.SetOf[string]()
	for _, id := range slices.Sorted(maps.Keys(listing.listed)) {
		key := "openai/" + id
		rates, ok := listing.prices[id]
		recommended := listing.recommended.Contains(id) && !listing.hidden.Contains(key)
		if !ok && models[key].Hidden && !recommended {
			continue
		}
		data, err := fetch(modelURL + id + ".md")
		if err != nil {
			fmt.Printf("Model details unavailable for %s: %v\n", id, err)
		}
		doc := modelDocument(data)
		price, priced := withTiers(id, &rates, doc)
		if priced {
			prices[key] = price
			seen.Add(key)
		}
		updateModel(models, key, doc, recommended)
		for _, alias := range snapshots(doc, id) {
			aliasKey := "openai/" + alias
			if _, ok := listing.prices[alias]; !ok && priced {
				prices[aliasKey] = price
				seen.Add(aliasKey)
			}
			updateModel(models, aliasKey, doc, false)
		}
	}
	for _, key := range slices.Sorted(maps.Keys(prices)) {
		if !seen.Contains(key) {
			fmt.Printf("Retained %s without verified pricing\n", key)
		}
	}
}

func withTiers(id string, rates *publishedPrice, doc modelDocument) (worker.Price, bool) {
	if rates.base.Input == nil || rates.base.Output == nil {
		return worker.Price{}, false
	}
	price := worker.Price{
		Input:       rates.base.Input,
		Output:      rates.base.Output,
		CacheInput:  rates.base.CacheInput,
		CacheOutput: rates.base.CacheOutput,
	}
	if rates.long.Input == nil && rates.long.Output == nil {
		return price, true
	}
	if rates.inputOver > 0 && rates.long.Input != nil && rates.long.Output != nil {
		price.Tiers = []worker.PriceTier{{
			InputOver:   rates.inputOver,
			Input:       rates.long.Input,
			Output:      rates.long.Output,
			CacheInput:  rates.long.CacheInput,
			CacheOutput: rates.long.CacheOutput,
		}}
		return price, true
	}
	match := longContext.FindStringSubmatch(string(doc))
	if match == nil || rates.long.Input == nil || rates.long.Output == nil {
		fmt.Printf("Could not verify the long-context tier for %s\n", id)
		return worker.Price{}, false
	}
	cutoff, err := strconv.ParseInt(strings.ReplaceAll(match[1], ",", ""), 10, 64)
	if err != nil || cutoff <= 0 {
		return worker.Price{}, false
	}
	if match[2] != "" {
		cutoff *= tokensPerK
	}
	price.Tiers = []worker.PriceTier{{
		InputOver:   cutoff,
		Input:       rates.long.Input,
		Output:      rates.long.Output,
		CacheInput:  rates.long.CacheInput,
		CacheOutput: rates.long.CacheOutput,
	}}
	return price, true
}

func updateModel(
	models map[string]modelCapabilities, key string, doc modelDocument, recommended bool,
) {
	caps := models[key]
	if doc == "" {
		return
	}
	hasResponses := strings.Contains(string(doc), "| Responses | `v1/responses` | Supported |")
	hasChat := strings.Contains(
		string(doc), "| Chat Completions | `v1/chat/completions` | Supported |",
	)
	textOutput := strings.Contains(string(doc), "\n- Output modalities: text\n")
	if !textOutput || !hasResponses && !hasChat {
		caps.Hidden = true
		models[key] = caps
		return
	}
	eligible := recommended && textOutput && (hasResponses || hasChat)
	hidden := caps.Hidden
	if hidden && !eligible {
		return
	}
	caps.Hidden = false
	if hasResponses && !hasChat {
		caps.Endpoint = responseEndpoint
	} else if hasChat && !hasResponses {
		caps.Endpoint = chatEndpoint
	}
	plain := strings.ReplaceAll(string(doc), "`", "")
	if levels := readEfforts(plain); len(levels) > 0 {
		caps.Efforts = levels
	} else if len(caps.Efforts) > 0 {
		fmt.Printf("Retained %s effort options without a published list\n", key)
	}
	models[key] = caps
}

func readEfforts(doc string) []api.Effort {
	match := effortList.FindStringSubmatch(doc)
	if match == nil {
		return nil
	}
	var res []api.Effort
	for _, level := range effortLevel.FindAllString(strings.ToLower(match[1]), -1) {
		res = append(res, api.Effort(level))
	}
	return res
}

func snapshots(doc modelDocument, id string) []string {
	_, section, _ := strings.Cut(string(doc), "\n## Snapshots\n")
	section, _, _ = strings.Cut(section, "\n## ")
	var ids []string
	for _, match := range snapshotID.FindAllStringSubmatch(section, -1) {
		if match[1] != id && !strings.Contains(match[0], "Shut down") {
			ids = append(ids, match[1])
		}
	}
	return ids
}
