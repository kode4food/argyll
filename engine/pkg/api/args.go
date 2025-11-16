package api

import "maps"

type (
	// Args represents a map of named arguments passed to or from steps
	Args map[Name]any

	// Name is a string identifier for arguments and attributes
	Name string
)

// Set creates a new Args with the specified name-value pair added
func (a Args) Set(name Name, value any) Args {
	res := maps.Clone(a)
	res[name] = value
	return res
}

// GetString retrieves a string value from args, returning defaultValue if not
// found or wrong type
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

// GetBool retrieves a boolean value from args, returning defaultValue if not
// found or wrong type
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

// GetInt retrieves an integer value from args, returning defaultValue if not
// found or wrong type. Supports both int and float64 (converting from JSON
// numbers)
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
