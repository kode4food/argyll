package openapi

import (
	"slices"

	"github.com/kode4food/argyll/engine/pkg/util"
)

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
