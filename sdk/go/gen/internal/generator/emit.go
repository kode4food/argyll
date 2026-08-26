package generator

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/types"
	"maps"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"text/template"

	"github.com/kode4food/argyll/engine/pkg/api"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/imports"
)

type (
	sourceModel struct {
		Package      string
		Imports      string
		Declarations []string
		Steps        []sourceStep
	}

	sourceStep struct {
		ID         string
		Spec       string
		Handler    string
		Compensate string
	}

	structDeclaration struct {
		name   string
		owner  string
		fields []codecField
		lazy   bool
	}

	pkgGen struct {
		pkg       *packages.Package
		server    bool
		imports   map[string]string
		codecs    map[string]string
		active    map[string]bool
		recursive map[string]bool
		decls     []string
		steps     []stepModel
	}

	codecField struct {
		attr  string
		field string
		codec string
		owner string
		typ   string
	}

	fieldSpec struct {
		*types.Var
		options Options
		attr    string
	}
)

const (
	// GeneratedFile is the file argyll-gen writes into each package
	GeneratedFile = "zz_argyll_gen.go"

	fieldTag   = "argyll"
	matchTag   = "argyll-match"
	mappingTag = "argyll-mapping"
	skipField  = "-"

	codecPackage   = "github.com/kode4food/argyll/sdk/go/codec"
	runtimePackage = "github.com/kode4food/argyll/sdk/go/gen"
	contextPackage = "context"
	httpPackage    = "net/http"
	slogPackage    = "log/slog"
	osPackage      = "os"

	templatePattern = "templates/*.go.tmpl"
	serverTemplate  = "server.go.tmpl"
	stepsTemplate   = "steps.go.tmpl"
)

var (
	ErrUnsupportedType = errors.New("unsupported attribute type")
	ErrBadTag          = errors.New("invalid argyll field tag")
	ErrAmbiguousField  = errors.New("ambiguous embedded field")
	ErrServerPackage   = errors.New("generated server requires package main")
	ErrMainDeclared    = errors.New("package already declares main")
	ErrSourceTemplate  = errors.New("failed to render source template")
)

//go:embed templates/*.go.tmpl
var templates embed.FS

var sources = template.Must(template.ParseFS(templates, templatePattern))

// Render returns the generated source for a package, or nil when the
// package contains no Argyll directives. Server adds a minimal main function
func Render(pkg *packages.Package, server bool) ([]byte, error) {
	g := &pkgGen{
		pkg:       pkg,
		server:    server,
		imports:   map[string]string{},
		codecs:    map[string]string{},
		active:    map[string]bool{},
		recursive: map[string]bool{},
	}
	mainDeclared := false
	for _, f := range pkg.Syntax {
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok {
				continue
			}
			if !ast.IsGenerated(f) && fn.Name.Name == "main" {
				mainDeclared = true
			}
			if err := g.addFunc(fn); err != nil {
				return nil, err
			}
		}
	}
	if len(g.steps) == 0 {
		return nil, nil
	}
	if server && pkg.Name != "main" {
		return nil, fmt.Errorf("%w: %s", ErrServerPackage, pkg.Name)
	}
	if server && mainDeclared {
		return nil, ErrMainDeclared
	}
	return g.source()
}

func (g *pkgGen) source() ([]byte, error) {
	steps, err := g.sourceSteps()
	if err != nil {
		return nil, err
	}
	model := sourceModel{
		Package:      g.pkg.Name,
		Imports:      g.importBlock(),
		Declarations: g.decls,
		Steps:        steps,
	}
	name := stepsTemplate
	if g.server {
		name = serverTemplate
	}
	var buf bytes.Buffer
	if err := sources.ExecuteTemplate(&buf, name, model); err != nil {
		return nil, errors.Join(ErrSourceTemplate, err)
	}

	src, err := imports.Process("", buf.Bytes(), nil)
	if err != nil {
		return nil, fmt.Errorf("%w:\n%s", err, buf.String())
	}
	return src, nil
}

func (g *pkgGen) sourceSteps() ([]sourceStep, error) {
	steps := make([]sourceStep, 0, len(g.steps))
	for _, s := range g.steps {
		spec, err := json.Marshal(s.spec)
		if err != nil {
			return nil, err
		}
		handler := s.handler
		compensate := s.compensate
		if g.server {
			logged := func(h string) string {
				return fmt.Sprintf("logged(%q, %s)", s.spec.ID, h)
			}
			handler = logged(handler)
			if compensate != "" {
				compensate = logged(compensate)
			}
		}
		steps = append(steps, sourceStep{
			ID:         strconv.Quote(string(s.spec.ID)),
			Spec:       strconv.Quote(string(spec)),
			Handler:    handler,
			Compensate: compensate,
		})
	}
	return steps, nil
}

func (g *pkgGen) importBlock() string {
	paths := maps.Clone(g.imports)
	for _, p := range []string{runtimePackage, codecPackage} {
		paths[p] = ""
	}
	if g.server {
		for _, p := range []string{
			contextPackage, httpPackage, slogPackage, osPackage,
		} {
			paths[p] = ""
		}
	}

	var sb strings.Builder
	sb.WriteString("import (\n")
	for _, p := range slices.Sorted(maps.Keys(paths)) {
		fmt.Fprintf(&sb, "%q\n", p)
	}
	sb.WriteString(")\n\n")
	return sb.String()
}

func (g *pkgGen) wrapStruct(
	name string, names []string, types []types.Type, output bool,
) (string, api.AttributeSpecs, error) {
	fields := make([]codecField, len(names))
	attrs := api.AttributeSpecs{}
	var decl strings.Builder
	fmt.Fprintf(&decl, "type %s struct {\n", name)
	for i, n := range names {
		expr, err := g.codecExpr(types[i])
		if err != nil {
			return "", nil, err
		}
		field, typ := ExportedName(n), g.typeOf(types[i])
		fmt.Fprintf(&decl, "%s %s\n", field, typ)
		fields[i] = codecField{
			attr:  n,
			field: field,
			codec: expr,
			owner: name,
			typ:   typ,
		}
		spec, err := newAttr(types[i], nil, output)
		if err != nil {
			return "", nil, err
		}
		attrs[api.Name(n)] = spec
	}
	decl.WriteString("}")
	g.decls = append(g.decls, decl.String())

	codecVar := "codec" + ExportedName(name)
	g.decls = append(g.decls, renderStructDeclaration(structDeclaration{
		name:   codecVar,
		owner:  name,
		fields: fields,
	}))
	return codecVar, attrs, nil
}

func (g *pkgGen) codecExpr(t types.Type) (string, error) {
	switch u := t.Underlying().(type) {
	case *types.Basic:
		return g.basicCodec(t, u)
	case *types.Struct:
		return g.structCodec(t, u)
	case *types.Slice, *types.Pointer, *types.Map:
		if _, named := t.(*types.Named); named {
			return "", fmt.Errorf("%w: %s", ErrUnsupportedType, g.typeOf(t))
		}
		return g.compositeCodec(u)
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedType, g.typeOf(t))
	}
}

func (g *pkgGen) basicCodec(t types.Type, u *types.Basic) (string, error) {
	info := u.Info()
	switch {
	case info&types.IsString != 0:
		return fmt.Sprintf("codec.Text[%s]()", g.typeOf(t)), nil
	case info&types.IsBoolean != 0:
		return fmt.Sprintf("codec.Boolean[%s]()", g.typeOf(t)), nil
	case info&(types.IsInteger|types.IsFloat) != 0:
		return fmt.Sprintf("codec.Number[%s]()", g.typeOf(t)), nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedType, g.typeOf(t))
	}
}

func (g *pkgGen) compositeCodec(u types.Type) (string, error) {
	switch u := u.(type) {
	case *types.Slice:
		return g.wrapCodec("codec.Slice", u.Elem())
	case *types.Pointer:
		return g.wrapCodec("codec.Optional", u.Elem())
	case *types.Map:
		if b, ok := u.Key().Underlying().(*types.Basic); !ok ||
			b.Info()&types.IsString == 0 {
			return "", fmt.Errorf("%w: %s keys", ErrUnsupportedType,
				g.typeOf(u.Key()))
		}
		return g.wrapCodec("codec.Map", u.Elem())
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedType, g.typeOf(u))
	}
}

func (g *pkgGen) wrapCodec(fn string, elem types.Type) (string, error) {
	inner, err := g.codecExpr(elem)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s(%s)", fn, inner), nil
}

func (g *pkgGen) structCodec(t types.Type, u *types.Struct) (string, error) {
	name := g.typeOf(t)
	if v, ok := g.codecs[name]; ok {
		if g.active[name] {
			g.recursive[name] = true
		}
		return v, nil
	}
	v := g.codecName(t)
	g.codecs[name] = v
	g.active[name] = true
	defer delete(g.active, name)

	specs, err := structFields(u)
	if err != nil {
		return "", err
	}
	var fields []codecField
	for _, f := range specs {
		expr, err := g.codecExpr(f.Type())
		if err != nil {
			return "", err
		}
		fields = append(fields, codecField{
			attr:  resolveInnerName(f),
			field: f.Name(),
			codec: expr,
			owner: name,
			typ:   g.typeOf(f.Type()),
		})
	}
	g.decls = append(g.decls, renderStructDeclaration(structDeclaration{
		name:   v,
		owner:  name,
		fields: fields,
		lazy:   g.recursive[name],
	}))
	return v, nil
}

func (g *pkgGen) codecName(t types.Type) string {
	if named, ok := t.(*types.Named); ok {
		return "codec" + ExportedName(named.Obj().Name())
	}
	return fmt.Sprintf("codecAnon%d", len(g.codecs))
}

func (g *pkgGen) typeOf(t types.Type) string {
	if t == nil {
		return "struct{}"
	}
	return types.TypeString(t, g.qualifier)
}

func (g *pkgGen) qualifier(p *types.Package) string {
	if p == g.pkg.Types {
		return ""
	}
	g.imports[p.Path()] = p.Name()
	return p.Name()
}

func renderStructDeclaration(decl structDeclaration) string {
	if len(decl.fields) == 0 {
		return fmt.Sprintf("%s := codec.Struct[%s]()", decl.name, decl.owner)
	}
	var sb strings.Builder
	sb.WriteString("codec.Struct(\n")
	for _, f := range decl.fields {
		fmt.Fprintf(&sb, "codec.Field(%q, %s,\nfunc(v *%s) *%s {\n"+
			"return &v.%s\n},\n),\n", f.attr, f.codec, f.owner, f.typ,
			f.field)
	}
	sb.WriteString(")")
	if !decl.lazy {
		return fmt.Sprintf("%s := %s", decl.name, sb.String())
	}
	// a self-referential initializer is an initialization cycle
	return fmt.Sprintf("var %sImpl codec.Codec[%s]\n\n"+
		"%s := codec.Ref(&%sImpl)\n\n"+
		"%sImpl = %s", decl.name, decl.owner, decl.name, decl.name,
		decl.name,
		sb.String())
}

func newAttr(
	t types.Type, options Options, output bool,
) (*api.AttributeSpec, error) {
	role := declaredRole(t, options, output)
	if output && role != api.RoleOutput {
		return nil, fmt.Errorf("%w: an output takes no role %q",
			ErrBadProp, role)
	}
	spec := &api.AttributeSpec{Type: attributeType(t)}
	if err := setRole(spec, role); err != nil {
		return nil, err
	}
	for _, o := range options {
		if o.Key == roleProp {
			continue
		}
		set, ok := attrSetters[o.Key]
		if !ok {
			return nil, fmt.Errorf("%w: unknown property %q",
				ErrBadProp, o.Key)
		}
		if err := set(spec, o.Value); err != nil {
			return nil, err
		}
	}
	return spec, nil
}

func declaredRole(
	t types.Type, options Options, output bool,
) api.AttributeRole {
	for _, o := range options {
		if o.Key == roleProp {
			return api.AttributeRole(o.Value)
		}
	}
	switch {
	case output:
		return api.RoleOutput
	case isPointer(t):
		return api.RoleOptional
	default:
		return api.RoleRequired
	}
}

func attributeType(t types.Type) api.AttributeType {
	if p, ok := t.(*types.Pointer); ok {
		return attributeType(p.Elem())
	}
	switch u := t.Underlying().(type) {
	case *types.Basic:
		info := u.Info()
		switch {
		case info&types.IsString != 0:
			return api.TypeString
		case info&types.IsBoolean != 0:
			return api.TypeBoolean
		case info&(types.IsInteger|types.IsFloat) != 0:
			return api.TypeNumber
		}
	case *types.Slice:
		return api.TypeArray
	case *types.Struct, *types.Map:
		return api.TypeObject
	}
	return api.TypeAny
}

func isPointer(t types.Type) bool {
	_, ok := t.(*types.Pointer)
	return ok
}

func structFields(s *types.Struct) ([]fieldSpec, error) {
	var out []fieldSpec
	for i := range s.NumFields() {
		f := s.Field(i)
		if !f.Exported() {
			continue
		}
		tag := s.Tag(i)
		if inner, ok := embeddedStruct(f, tag); ok {
			fields, err := structFields(inner)
			if err != nil {
				return nil, err
			}
			out = append(out, fields...)
			continue
		}
		spec, ok, err := attrOf(f, tag)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, spec)
		}
	}
	if err := checkAmbiguous(out); err != nil {
		return nil, err
	}
	return out, nil
}

// embeddedStruct reports the struct an untagged embedded field flattens into,
// leaving a field its tag names to be an attribute of its own
func embeddedStruct(f *types.Var, tag string) (*types.Struct, bool) {
	if !f.Embedded() {
		return nil, false
	}
	if strings.TrimSpace(reflect.StructTag(tag).Get(fieldTag)) != "" {
		return nil, false
	}
	st, ok := f.Type().Underlying().(*types.Struct)
	return st, ok
}

// checkAmbiguous rejects what flattening can collide: two attributes of the
// same name, or two Go fields the generated accessor cannot tell apart
func checkAmbiguous(fields []fieldSpec) error {
	attrs := make(map[string]bool, len(fields))
	names := make(map[string]bool, len(fields))
	for _, f := range fields {
		if attrs[f.attr] {
			return fmt.Errorf("%w: attribute %q", ErrAmbiguousField, f.attr)
		}
		if names[f.Name()] {
			return fmt.Errorf("%w: field %s", ErrAmbiguousField, f.Name())
		}
		attrs[f.attr] = true
		names[f.Name()] = true
	}
	return nil
}

func attrOf(f *types.Var, tag string) (fieldSpec, bool, error) {
	tags := reflect.StructTag(tag)
	text := strings.TrimSpace(tags.Get(fieldTag))
	if text == skipField {
		return fieldSpec{}, false, nil
	}
	head, options, err := ParseOptions(text)
	if err != nil {
		return fieldSpec{}, false, fmt.Errorf("%w: %s", err, f.Name())
	}
	decl := SplitHead(head)
	if decl.Attrs != "" {
		return fieldSpec{}, false, fmt.Errorf(
			"%w: %s is a field, so it names one attribute", ErrBadTag,
			f.Name())
	}
	name := decl.Name
	if strings.ContainsAny(name, " \t") {
		return fieldSpec{}, false, fmt.Errorf(
			"%w: bad attribute name %q on %s", ErrBadTag, name, f.Name())
	}
	if name == "" {
		name = SnakeCase(f.Name())
	}
	if script, ok := tags.Lookup(matchTag); ok {
		options = append(options, Option{Key: matchTag, Value: script})
	}
	if script, ok := tags.Lookup(mappingTag); ok {
		options = append(options, Option{Key: mappingTag, Value: script})
	}
	return fieldSpec{Var: f, attr: name, options: options}, true, nil
}

func resolveInnerName(f fieldSpec) string {
	for _, o := range f.options {
		if o.Key == mappingProp {
			return o.Value
		}
	}
	return f.attr
}
