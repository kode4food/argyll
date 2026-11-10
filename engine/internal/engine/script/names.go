package script

import (
	"maps"
	"regexp"

	"github.com/kode4food/argyll/engine/pkg/api"
)

type bareName struct {
	name  string
	input string
}

var (
	identPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	identTail    = regexp.MustCompile(`[A-Za-z0-9_]+$`)
)

// bareNames pairs each short name source can use with the input it reads. A
// usable name wins over a derived tail, and tails two inputs share bind none
func bareNames(names []string, usable func(string) bool) []bareName {
	taken := map[string]bool{}
	for _, name := range names {
		taken[name] = usable(name)
	}

	tails := map[string]int{}
	for _, name := range names {
		if taken[name] {
			continue
		}
		if tail, ok := derivedName(name, usable); ok && !taken[tail] {
			tails[tail]++
		}
	}

	var res []bareName
	for _, name := range names {
		if taken[name] {
			res = append(res, bareName{name: name, input: name})
			continue
		}
		if tail, ok := derivedName(name, usable); ok && tails[tail] == 1 {
			res = append(res, bareName{name: tail, input: name})
		}
	}
	return res
}

func aliasesOf(names []bareName) []bareName {
	var res []bareName
	for _, b := range names {
		if b.name == b.input {
			continue
		}
		res = append(res, b)
	}
	return res
}

// withAliases builds what source reads through this: every input by its full
// name and by each derived short name
func withAliases(inputs api.Args, aliases []bareName) api.Args {
	if len(aliases) == 0 {
		return inputs
	}
	res := maps.Clone(inputs)
	for _, b := range aliases {
		if v, ok := inputs[api.Name(b.input)]; ok {
			res[api.Name(b.name)] = v
		}
	}
	return res
}

func derivedName(name string, usable func(string) bool) (string, bool) {
	tail := identTail.FindString(name)
	return tail, usable(tail)
}
