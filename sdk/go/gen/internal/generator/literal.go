package generator

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
)

var (
	ErrUnnamedType   = errors.New("type has no name to render")
	ErrUnsupportedGo = errors.New("no Go literal for kind")
)

// GoLiteral renders a value as the Go expression that rebuilds it, leaving out
// the zero and unexported fields a literal already defaults
func GoLiteral(v any) (string, error) {
	return literalOf(reflect.ValueOf(v))
}

func literalOf(v reflect.Value) (string, error) {
	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			return "nil", nil
		}
		inner, err := literalOf(v.Elem())
		if err != nil {
			return "", err
		}
		return "&" + inner, nil
	case reflect.Struct, reflect.Map, reflect.Slice, reflect.Array:
		return namedComposite(v)
	case reflect.String:
		return strconv.Quote(v.String()), nil
	case reflect.Bool:
		return strconv.FormatBool(v.Bool()), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32,
		reflect.Int64:
		return strconv.FormatInt(v.Int(), 10), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32,
		reflect.Uint64:
		return strconv.FormatUint(v.Uint(), 10), nil
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(v.Float(), 'g', -1, 64), nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedGo, v.Kind())
	}
}

// an empty name renders the braces alone, for an element whose type the
// composite around it already states
func compositeLiteral(name string, v reflect.Value) (string, error) {
	parts, err := partsOf(v)
	if err != nil {
		return "", err
	}
	return composite(name, parts), nil
}

func namedComposite(v reflect.Value) (string, error) {
	name, err := typeName(v.Type())
	if err != nil {
		return "", err
	}
	return compositeLiteral(name, v)
}

func partsOf(v reflect.Value) ([]string, error) {
	switch v.Kind() {
	case reflect.Struct:
		return structParts(v)
	case reflect.Map:
		return mapParts(v)
	default:
		return listParts(v)
	}
}

func structParts(v reflect.Value) ([]string, error) {
	var fields []string
	for i := range v.NumField() {
		f := v.Type().Field(i)
		if !f.IsExported() || v.Field(i).IsZero() {
			continue
		}
		val, err := literalOf(v.Field(i))
		if err != nil {
			return nil, err
		}
		fields = append(fields, f.Name+": "+val+",")
	}
	return fields, nil
}

func mapParts(v reflect.Value) ([]string, error) {
	entries := make([]string, 0, v.Len())
	for i := v.MapRange(); i.Next(); {
		key, err := elementLiteral(i.Key(), v.Type().Key())
		if err != nil {
			return nil, err
		}
		val, err := elementLiteral(i.Value(), v.Type().Elem())
		if err != nil {
			return nil, err
		}
		entries = append(entries, key+": "+val+",")
	}
	slices.Sort(entries)
	return entries, nil
}

func listParts(v reflect.Value) ([]string, error) {
	items := make([]string, 0, v.Len())
	for i := range v.Len() {
		item, err := elementLiteral(v.Index(i), v.Type().Elem())
		if err != nil {
			return nil, err
		}
		items = append(items, item+",")
	}
	return items, nil
}

// elementLiteral renders an element of a composite, leaving out the type Go
// infers from the composite around it
func elementLiteral(v reflect.Value, t reflect.Type) (string, error) {
	if t.Kind() == reflect.Pointer {
		if v.IsNil() {
			return "nil", nil
		}
		if elides(t.Elem()) {
			return compositeLiteral("", v.Elem())
		}
	}
	if elides(t) {
		return compositeLiteral("", v)
	}
	return literalOf(v)
}

// elides reports whether a composite literal of this type states its own type
// only when it stands alone
func elides(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Struct, reflect.Slice, reflect.Array, reflect.Map:
		return true
	default:
		return false
	}
}

func composite(name string, parts []string) string {
	if len(parts) == 0 {
		return name + "{}"
	}
	return name + "{\n" + strings.Join(parts, "\n") + "\n}"
}

// typeName qualifies a named type by the package goimports resolves for it, and
// spells an unnamed composite out of the types it composes
func typeName(t reflect.Type) (string, error) {
	if t.Name() != "" {
		pkg := t.PkgPath()
		if pkg == "" {
			return t.Name(), nil
		}
		return pkg[strings.LastIndex(pkg, "/")+1:] + "." + t.Name(), nil
	}

	switch t.Kind() {
	case reflect.Slice:
		return elemTypeName("[]", t)
	case reflect.Array:
		return elemTypeName("["+strconv.Itoa(t.Len())+"]", t)
	case reflect.Pointer:
		return elemTypeName("*", t)
	case reflect.Map:
		key, err := typeName(t.Key())
		if err != nil {
			return "", err
		}
		return elemTypeName("map["+key+"]", t)
	default:
		return "", fmt.Errorf("%w: %s", ErrUnnamedType, t)
	}
}

func elemTypeName(prefix string, t reflect.Type) (string, error) {
	elem, err := typeName(t.Elem())
	if err != nil {
		return "", err
	}
	return prefix + elem, nil
}
