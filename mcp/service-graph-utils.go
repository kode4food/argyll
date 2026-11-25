package mcp

import (
	"slices"

	"github.com/kode4food/argyll/engine/pkg/util"
)

func sharedStrings[S ~string](a, b []S) []S {
	set := util.Set[S]{}
	for _, item := range b {
		set.Add(item)
	}
	var res []S
	for _, item := range a {
		if set.Contains(item) {
			res = append(res, item)
		}
	}
	return uniqueStrings(res)
}

func uniqueStrings[S ~string](items []S) []S {
	seen := util.Set[S]{}
	var res []S
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
