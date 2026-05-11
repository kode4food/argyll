package openapi

import (
	"slices"

	"github.com/kode4food/argyll/engine/pkg/util"
)

func intersect(a, b []string) []string {
	set := util.Set[string]{}
	for _, v := range b {
		set.Add(v)
	}
	var res []string
	for _, v := range a {
		if set.Contains(v) {
			res = append(res, v)
		}
	}
	return uniqueStrings(res)
}

func diff(a, b []string) []string {
	set := util.Set[string]{}
	for _, v := range b {
		set.Add(v)
	}
	var res []string
	for _, v := range a {
		if !set.Contains(v) {
			res = append(res, v)
		}
	}
	return uniqueStrings(res)
}

func unionStrings(parts ...[]string) []string {
	var all []string
	for _, part := range parts {
		all = append(all, part...)
	}
	return uniqueStrings(all)
}

func uniqueStrings(in []string) []string {
	seen := util.Set[string]{}
	var res []string
	for _, v := range in {
		if v == "" {
			continue
		}
		if seen.Contains(v) {
			continue
		}
		seen.Add(v)
		res = append(res, v)
	}
	slices.Sort(res)
	return res
}
