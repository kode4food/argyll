package util_test

import (
	"bytes"
	"encoding/json"
	"runtime"
	"strconv"
	"testing"

	"github.com/kode4food/argyll/engine/pkg/util"
)

func BenchmarkEqualJSON(b *testing.B) {
	object := func() map[string]any {
		res := map[string]any{}
		for i := range 16 {
			res[strconv.Itoa(i)] = []any{float64(i), "pending", true}
		}
		return res
	}
	changed := object()
	changed["0"] = "changed"
	for _, tc := range []struct {
		name  string
		left  any
		right any
	}{
		{name: "scalar", left: 7.0, right: 7.0},
		{name: "equal object", left: object(), right: object()},
		{name: "changed object", left: object(), right: changed},
	} {
		b.Run(tc.name+"/structural", func(b *testing.B) {
			var equal bool
			for b.Loop() {
				equal = util.EqualJSON(tc.left, tc.right)
			}
			runtime.KeepAlive(equal)
		})
		b.Run(tc.name+"/marshal", func(b *testing.B) {
			var equal bool
			for b.Loop() {
				left, err := json.Marshal(tc.left)
				if err != nil {
					b.Fatal(err)
				}
				right, err := json.Marshal(tc.right)
				if err != nil {
					b.Fatal(err)
				}
				equal = bytes.Equal(left, right)
			}
			runtime.KeepAlive(equal)
		})
	}
}
