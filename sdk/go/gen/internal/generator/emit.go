package generator

import (
	"bytes"
	"embed"
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

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/imports"

	"github.com/kode4food/argyll/engine/pkg/api"
)

type (
	sourceModel struct {
		Package       string
		Imports       string
		Decls         []string
		Steps         []sourceStep
		EmbeddedDecls []string
		Embedded      []sourceStep
		EmbeddedSpecs string
	}

	sourceStep struct {
		ID         string
		Spec       string
		Invoke     string
		Compensate string
	}

	structDeclaration struct {
		dialect dialect
		name    string
		owner   string
		fields  []codecField
		lazy    bool
	}

	pkgGen struct {
		pkg        *packages.Package
		dialect    dialect
		imports    map[string]string
		codecs     map[string]string
		active     map[string]bool
		recursive  map[string]bool
		embedTypes map[string]string
		decls      []string
		steps      []stepModel
	}

	// dialect is the family of value adapters a pass generates: codecs for
	// served HTTP handlers, converters for in-process embedded handlers
	dialect struct {
		pkg        string
		iface      string
		sync       string
		compensate string
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

	// adapterView is the snippet's view of a struct adapter declaration,
	// which reads the dialect and fields the generator carries unexported
	adapterView struct {
		Package   string
		Interface string
		Name      string
		Owner     string
		Lazy      bool
		Fields    []adapterFieldView
	}

	adapterFieldView struct {
		Attribute string
		Adapter   string
		Owner     string
		Type      string
		Field     string
	}

	// wrapBodyView is the snippet's view of a wrapper's call and result
	// assembly
	wrapBodyView struct {
		Function string
		Inputs   []string
		Outputs  []string
		OutType  string
		HasError bool
	}

	loggedView struct {
		ID      api.StepID
		Handler string
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
	convertPackage = "github.com/kode4food/argyll/sdk/go/convert"
	runtimePackage = "github.com/kode4food/argyll/sdk/go/gen"
	apiPackage     = "github.com/kode4food/argyll/engine/pkg/api"
	stepPackage    = "github.com/kode4food/argyll/engine/pkg/step"
	contextPackage = "context"
	httpPackage    = "net/http"
	slogPackage    = "log/slog"
	osPackage      = "os"

	templatePattern = "templates/*.go.tmpl"
	serverTemplate  = "server.go.tmpl"
	stepsTemplate   = "steps.go.tmpl"

	// the fragment templates snippets.go.tmpl defines
	structTypeSnippet        = "structType"
	adapterStructSnippet     = "adapterStruct"
	loggedSnippet            = "loggedHandler"
	syncHandlerSnippet       = "syncHandler"
	syncBodySnippet          = "syncBody"
	wrapBodySnippet          = "wrapBody"
	compensateHandlerSnippet = "compensateHandler"
)

var (
	ErrUnsupportedType = errors.New("unsupported attribute type")
	ErrBadTag          = errors.New("invalid argyll field tag")
	ErrAmbiguousField  = errors.New("ambiguous embedded field")
	ErrServerPackage   = errors.New("generated server requires package main")
	ErrMainDeclared    = errors.New("package already declares main")
	ErrSourceTemplate  = errors.New("failed to render source template")
)

var (
	//go:embed templates/*.go.tmpl
	templates embed.FS

	sources = template.Must(template.ParseFS(templates, templatePattern))

	serviceDialect = dialect{
		pkg:        "codec",
		iface:      "Codec",
		sync:       "gen.Sync(",
		compensate: "gen.Compensate(",
	}

	embeddedDialect = dialect{
		pkg:        "convert",
		iface:      "Converter",
		sync:       "gen.EmbeddedSync(",
		compensate: "gen.EmbeddedCompensate(",
	}

	serverPackages = []string{
		runtimePackage, codecPackage,
		contextPackage, httpPackage, slogPackage, osPackage,
	}

	libraryPackages = []string{
		runtimePackage, codecPackage, convertPackage, apiPackage, stepPackage,
	}
)

// Render returns the generated source for a package, or nil when the package
// contains no Argyll directives. Server adds a minimal main function and leaves
// out the embedded handlers, which only a library offers
func Render(pkg *packages.Package, server bool) ([]byte, error) {
	service := newPkgGen(pkg, serviceDialect)
	mainDeclared, err := service.collect()
	if err != nil {
		return nil, err
	}
	if len(service.steps) == 0 {
		return nil, nil
	}
	if server && pkg.Name != "main" {
		return nil, fmt.Errorf("%w: %s", ErrServerPackage, pkg.Name)
	}
	if server && mainDeclared {
		return nil, ErrMainDeclared
	}
	if server {
		return service.serverSource()
	}
	embedded := newPkgGen(pkg, embeddedDialect)
	if _, err := embedded.collect(); err != nil {
		return nil, err
	}
	return librarySource(service, embedded)
}

// Steps returns the step specifications a package declares, the values Render
// writes as literals into its generated source
func Steps(pkg *packages.Package) ([]*api.Step, error) {
	g := newPkgGen(pkg, serviceDialect)
	if _, err := g.collect(); err != nil {
		return nil, err
	}
	res := make([]*api.Step, 0, len(g.steps))
	for _, s := range g.steps {
		res = append(res, s.spec)
	}
	return res, nil
}

func newPkgGen(pkg *packages.Package, d dialect) *pkgGen {
	return &pkgGen{
		pkg:        pkg,
		dialect:    d,
		imports:    map[string]string{},
		codecs:     map[string]string{},
		active:     map[string]bool{},
		recursive:  map[string]bool{},
		embedTypes: map[string]string{},
	}
}

// collect gathers the package's steps, reporting whether it declares main
func (g *pkgGen) collect() (bool, error) {
	mainDeclared := false
	for _, f := range g.pkg.Syntax {
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok {
				continue
			}
			if !ast.IsGenerated(f) && fn.Name.Name == "main" {
				mainDeclared = true
			}
			if err := g.addFunc(fn); err != nil {
				return false, err
			}
		}
	}
	return mainDeclared, nil
}

func (g *pkgGen) serverSource() ([]byte, error) {
	steps, err := g.serviceSteps(true)
	if err != nil {
		return nil, err
	}
	return render(serverTemplate, sourceModel{
		Package: g.pkg.Name,
		Imports: importBlock(g.imports, serverPackages),
		Decls:   g.decls,
		Steps:   steps,
	})
}

func (g *pkgGen) serviceSteps(logging bool) ([]sourceStep, error) {
	res := make([]sourceStep, 0, len(g.steps))
	for _, s := range g.steps {
		spec, err := GoLiteral(s.spec)
		if err != nil {
			return nil, err
		}
		invoke := s.invoke
		compensate := s.compensate
		if logging {
			invoke = logged(s.spec.ID, invoke)
			compensate = logged(s.spec.ID, compensate)
		}
		res = append(res, sourceStep{
			Spec:       spec,
			Invoke:     invoke,
			Compensate: compensate,
		})
	}
	return res, nil
}

// embeddedSteps are keyed by step type, which is the ID an embedded engine runs
// the step's handler under
func (g *pkgGen) embeddedSteps() []sourceStep {
	res := make([]sourceStep, 0, len(g.steps))
	for _, s := range g.steps {
		res = append(res, sourceStep{
			ID:         strconv.Quote(s.embedType),
			Invoke:     s.invoke,
			Compensate: s.compensate,
		})
	}
	return res
}

// one slice renders each spec as an element, which states only its braces
func (g *pkgGen) embeddedSpecs() (string, error) {
	res := make([]*api.Step, 0, len(g.steps))
	for _, s := range g.steps {
		res = append(res, embeddedSpec(s))
	}
	return GoLiteral(res)
}

func (g *pkgGen) wrapStruct(
	name string, names []string, types []types.Type, output bool,
) (string, api.AttributeSpecs, error) {
	fields := make([]codecField, len(names))
	attrs := api.AttributeSpecs{}
	typeFields := make([]adapterFieldView, len(names))
	for i, n := range names {
		expr, err := g.codecExpr(types[i])
		if err != nil {
			return "", nil, err
		}
		field := ExportedName(n)
		typ := g.typeOf(types[i])
		typeFields[i] = adapterFieldView{Field: field, Type: typ}
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
	g.decls = append(g.decls, snippet(structTypeSnippet, adapterView{
		Name:   name,
		Fields: typeFields,
	}))

	codecVar := g.dialect.pkg + ExportedName(name)
	g.decls = append(g.decls, renderStructDeclaration(structDeclaration{
		dialect: g.dialect,
		name:    codecVar,
		owner:   name,
		fields:  fields,
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
		return g.scalarCodec("Text", t), nil
	case info&types.IsBoolean != 0:
		return g.scalarCodec("Boolean", t), nil
	case info&(types.IsInteger|types.IsFloat) != 0:
		return g.scalarCodec("Number", t), nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedType, g.typeOf(t))
	}
}

func (g *pkgGen) scalarCodec(kind string, t types.Type) string {
	return fmt.Sprintf("%s.%s[%s]()", g.dialect.pkg, kind, g.typeOf(t))
}

func (g *pkgGen) compositeCodec(u types.Type) (string, error) {
	switch u := u.(type) {
	case *types.Slice:
		return g.wrapCodec("Slice", u.Elem())
	case *types.Pointer:
		return g.wrapCodec("Optional", u.Elem())
	case *types.Map:
		if b, ok := u.Key().Underlying().(*types.Basic); !ok ||
			b.Info()&types.IsString == 0 {
			return "", fmt.Errorf("%w: %s keys", ErrUnsupportedType,
				g.typeOf(u.Key()))
		}
		return g.wrapCodec("Map", u.Elem())
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedType, g.typeOf(u))
	}
}

func (g *pkgGen) wrapCodec(kind string, elem types.Type) (string, error) {
	inner, err := g.codecExpr(elem)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s.%s(%s)", g.dialect.pkg, kind, inner), nil
}

func (g *pkgGen) structCodec(t types.Type, u *types.Struct) (string, error) {
	name := g.typeOf(t)
	if v, ok := g.codecs[name]; ok {
		g.recursive[name] = g.recursive[name] || g.active[name]
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
		dialect: g.dialect,
		name:    v,
		owner:   name,
		fields:  fields,
		lazy:    g.recursive[name],
	}))
	return v, nil
}

// emptyCodec is the codec of a side a step's function leaves empty
func (g *pkgGen) emptyCodec() string {
	return g.dialect.pkg + ".Struct[struct{}]()"
}

func (g *pkgGen) codecName(t types.Type) string {
	if named, ok := t.(*types.Named); ok {
		return g.dialect.pkg + ExportedName(named.Obj().Name())
	}
	return fmt.Sprintf("%sAnon%d", g.dialect.pkg, len(g.codecs))
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

func librarySource(service, embedded *pkgGen) ([]byte, error) {
	steps, err := service.serviceSteps(false)
	if err != nil {
		return nil, err
	}
	specs, err := embedded.embeddedSpecs()
	if err != nil {
		return nil, err
	}
	paths := maps.Clone(service.imports)
	maps.Copy(paths, embedded.imports)
	return render(stepsTemplate, sourceModel{
		Package:       service.pkg.Name,
		Imports:       importBlock(paths, libraryPackages),
		Decls:         service.decls,
		Steps:         steps,
		EmbeddedDecls: embedded.decls,
		Embedded:      embedded.embeddedSteps(),
		EmbeddedSpecs: specs,
	})
}

func render(name string, model sourceModel) ([]byte, error) {
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

func importBlock(paths map[string]string, extra []string) string {
	all := maps.Clone(paths)
	for _, p := range extra {
		all[p] = ""
	}

	var sb strings.Builder
	sb.WriteString("import (\n")
	for _, p := range slices.Sorted(maps.Keys(all)) {
		_, _ = fmt.Fprintf(&sb, "%q\n", p)
	}
	sb.WriteString(")\n\n")
	return sb.String()
}

// embeddedSpec is the step an embedded engine runs itself, typed to name its
// in-process handler
func embeddedSpec(s stepModel) *api.Step {
	embedded := s.spec.Copy()
	embedded.Type = api.StepType(s.embedType)
	embedded.HTTP = nil
	return embedded
}

// logged wraps a server's handler expression with invocation logging, leaving
// the absent compensation handler of a step absent
func logged(id api.StepID, handler string) string {
	if handler == "" {
		return ""
	}
	return snippet(loggedSnippet, loggedView{ID: id, Handler: handler})
}

func renderStructDeclaration(decl structDeclaration) string {
	fields := make([]adapterFieldView, len(decl.fields))
	for i, f := range decl.fields {
		fields[i] = adapterFieldView{
			Attribute: f.attr,
			Adapter:   f.codec,
			Owner:     f.owner,
			Type:      f.typ,
			Field:     f.field,
		}
	}
	return snippet(adapterStructSnippet, adapterView{
		Package:   decl.dialect.pkg,
		Interface: decl.dialect.iface,
		Name:      decl.name,
		Owner:     decl.owner,
		Lazy:      decl.lazy,
		Fields:    fields,
	})
}

// syncHandlerArgs are the pieces of a generated synchronous handler
type syncHandlerArgs struct {
	Adapter    string
	InAdapter  string
	OutAdapter string
	InType     string
	OutType    string
	Body       string
}

func syncHandler(args syncHandlerArgs) string {
	return snippet(syncHandlerSnippet, args)
}

type syncBodyArgs struct {
	Call       string
	Output     bool
	Fallible   bool
	OutputType string
}

func syncBody(args syncBodyArgs) string {
	return snippet(syncBodySnippet, args)
}

func wrapBody(
	fn string, names wrapNames, outType string, hasErr bool,
) string {
	inputs := make([]string, len(names.inputs))
	for i, n := range names.inputs {
		inputs[i] = ExportedName(n)
	}
	outputs := make([]string, len(names.outputs))
	for i, n := range names.outputs {
		outputs[i] = ExportedName(n)
	}
	return snippet(wrapBodySnippet, wrapBodyView{
		Function: fn,
		Inputs:   inputs,
		Outputs:  outputs,
		OutType:  outType,
		HasError: hasErr,
	})
}

func compAdapter(cfg compAdapterConfig) string {
	return snippet(compensateHandlerSnippet, cfg)
}

// a failed fragment renders short, which the gofmt pass in render reports
// against the whole file
func snippet(name string, data any) string {
	var sb strings.Builder
	_ = sources.ExecuteTemplate(&sb, name, data)
	return sb.String()
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
		return basicType(u.Info())
	case *types.Slice:
		return api.TypeArray
	case *types.Struct, *types.Map:
		return api.TypeObject
	default:
		return api.TypeAny
	}
}

func basicType(info types.BasicInfo) api.AttributeType {
	switch {
	case info&types.IsString != 0:
		return api.TypeString
	case info&types.IsBoolean != 0:
		return api.TypeBoolean
	case info&(types.IsInteger|types.IsFloat) != 0:
		return api.TypeNumber
	default:
		return api.TypeAny
	}
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
		fields, err := fieldSpecs(f, s.Tag(i))
		if err != nil {
			return nil, err
		}
		out = append(out, fields...)
	}
	if err := checkAmbiguous(out); err != nil {
		return nil, err
	}
	return out, nil
}

// fieldSpecs returns the attributes one struct field contributes: the fields of
// an embedded struct it flattens into, or the field itself
func fieldSpecs(f *types.Var, tag string) ([]fieldSpec, error) {
	if inner, ok := embeddedStruct(f, tag); ok {
		return structFields(inner)
	}
	spec, ok, err := attrOf(f, tag)
	if err != nil || !ok {
		return nil, err
	}
	return []fieldSpec{spec}, nil
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
