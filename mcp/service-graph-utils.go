package mcp

import (
	"slices"
	"strings"

	"github.com/kode4food/argyll/engine/pkg/util"
)

func sharedKeys(a, b []string) []string {
	var res []string
	for _, item := range sharedStrings(a, b) {
		if strings.HasSuffix(item, "_id") {
			res = append(res, item)
		}
	}
	return res
}

func sharedStrings(a, b []string) []string {
	set := util.Set[string]{}
	for _, item := range b {
		set.Add(item)
	}
	var res []string
	for _, item := range a {
		if set.Contains(item) {
			res = append(res, item)
		}
	}
	return uniqueStrings(res)
}

func tokenOverlap(a, b string) int {
	src := tokenSet(a)
	dst := tokenSet(b)
	shared := 0
	for token := range src {
		if dst.Contains(token) {
			shared++
		}
	}
	return shared
}

func tokenSet(name string) util.Set[string] {
	res := util.Set[string]{}
	for token := range strings.SplitSeq(name, "_") {
		if token == "" {
			continue
		}
		res.Add(token)
	}
	return res
}

func uniqueStrings(items []string) []string {
	seen := util.Set[string]{}
	var res []string
	for _, item := range items {
		if item == "" {
			continue
		}
		if seen.Contains(item) {
			continue
		}
		seen.Add(item)
		res = append(res, item)
	}
	slices.Sort(res)
	return res
}

func sameType(a, b string) bool {
	if a == "" || b == "" {
		return true
	}
	return a == b
}
