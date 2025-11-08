package api

import "maps"

type (
	Args map[Name]any

	Name string
)

func (a Args) Set(name Name, value any) Args {
	res := maps.Clone(a)
	res[name] = value
	return res
}

func (a Args) GetString(name Name, defaultValue string) string {
	val, ok := a[name]
	if !ok {
		return defaultValue
	}
	str, ok := val.(string)
	if !ok {
		return defaultValue
	}
	return str
}

func (a Args) GetBool(name Name, defaultValue bool) bool {
	val, ok := a[name]
	if !ok {
		return defaultValue
	}
	b, ok := val.(bool)
	if !ok {
		return defaultValue
	}
	return b
}

func (a Args) GetInt(name Name, defaultValue int) int {
	val, ok := a[name]
	if !ok {
		return defaultValue
	}
	if i, ok := val.(int); ok {
		return i
	}
	if f, ok := val.(float64); ok {
		return int(f)
	}
	return defaultValue
}
