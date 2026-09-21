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

// goLiteral renders a value as the Go expression that rebuilds it, leaving out
// the zero and unexported fields a literal already defaults
func goLiteral(v any) (string, error) {
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
	case reflect.Struct:
		return structLiteral(v)
	case reflect.Map:
		return mapLiteral(v)
	case reflect.Slice:
		return sliceLiteral(v)
	case reflect.String:
		return scalarLiteral(v, strconv.Quote(v.String()))
	case reflect.Bool:
		return scalarLiteral(v, strconv.FormatBool(v.Bool()))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32,
		reflect.Int64:
		return scalarLiteral(v, strconv.FormatInt(v.Int(), 10))
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedGo, v.Kind())
	}
}

// scalarLiteral wraps a basic value in a conversion when its type is named,
// so a StepID stays a StepID rather than becoming an untyped string
func scalarLiteral(v reflect.Value, basic string) (string, error) {
	if isBasicType(v.Type()) {
		return basic, nil
	}
	name, err := typeName(v.Type())
	if err != nil {
		return "", err
	}
	return name + "(" + basic + ")", nil
}

func structLiteral(v reflect.Value) (string, error) {
	name, err := typeName(v.Type())
	if err != nil {
		return "", err
	}

	var fields []string
	for i := range v.NumField() {
		f := v.Type().Field(i)
		if !f.IsExported() || v.Field(i).IsZero() {
			continue
		}
		val, err := literalOf(v.Field(i))
		if err != nil {
			return "", err
		}
		fields = append(fields, f.Name+": "+val+",")
	}
	return composite(name, fields), nil
}

func mapLiteral(v reflect.Value) (string, error) {
	name, err := typeName(v.Type())
	if err != nil {
		return "", err
	}

	entries := make([]string, 0, v.Len())
	for i := v.MapRange(); i.Next(); {
		key, err := literalOf(i.Key())
		if err != nil {
			return "", err
		}
		val, err := literalOf(i.Value())
		if err != nil {
			return "", err
		}
		entries = append(entries, key+": "+val+",")
	}
	slices.Sort(entries)
	return composite(name, entries), nil
}

func sliceLiteral(v reflect.Value) (string, error) {
	name, err := typeName(v.Type())
	if err != nil {
		return "", err
	}

	items := make([]string, 0, v.Len())
	for i := range v.Len() {
		item, err := literalOf(v.Index(i))
		if err != nil {
			return "", err
		}
		items = append(items, item+",")
	}
	return composite(name, items), nil
}

func composite(name string, parts []string) string {
	if len(parts) == 0 {
		return name + "{}"
	}
	return name + "{\n" + strings.Join(parts, "\n") + "\n}"
}

// typeName qualifies a named type by its package, matching the import names
// goimports resolves for the generated file
func typeName(t reflect.Type) (string, error) {
	if t.Name() == "" {
		return "", fmt.Errorf("%w: %s", ErrUnnamedType, t)
	}
	pkg := t.PkgPath()
	if pkg == "" {
		return t.Name(), nil
	}
	return pkg[strings.LastIndex(pkg, "/")+1:] + "." + t.Name(), nil
}

// isBasicType reports whether a type is a predeclared one, which a literal
// states without a conversion
func isBasicType(t reflect.Type) bool {
	return t.PkgPath() == "" && t.Name() == t.Kind().String()
}
