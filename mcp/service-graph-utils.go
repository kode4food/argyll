package mcp

import (
	"slices"

	"github.com/kode4food/argyll/engine/pkg/util"
)

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
