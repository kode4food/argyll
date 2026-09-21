package generator

import (
	"errors"
	"fmt"
	"go/ast"
	"go/types"
	"maps"
	"regexp"
	"strings"

	"github.com/kode4food/argyll/engine/pkg/api"
)

type (
	stepModel struct {
		spec       *api.Step
		invoke     string
		compensate string
		embedType  string
	}

	compHandlerConfig struct {
		fn     *ast.FuncDecl
		decl   stepDecl
		fields []compField
		wrap   bool
	}

	compField struct {
		attr *api.AttributeSpec
		name string
		typ  types.Type
	}

	compAdapterConfig struct {
		Adapter           string
		CompensateAdapter string
		Type              string
		Call              string
		Fallible          bool
	}

	stepModelConfig struct {
		function    *ast.FuncDecl
		declaration stepDecl
		attributes  api.AttributeSpecs
		invoke      string
		compensate  string
	}

	attributeSets struct {
		inputs  api.AttributeSpecs
		outputs api.AttributeSpecs
	}

	directiveRef struct {
		kind string
		args string
	}

	// wrapNames are the attribute names a wrap directive declares, nil on a
	// side it leaves to the signature
	wrapNames struct {
		inputs  []string
		outputs []string
	}

	// stepDirectives are the directives a step declares alongside its own
	stepDirectives struct {
		tags        api.Tags
		predicate   *api.ScriptConfig
		description string
		http        Options
		work        Options
	}

	// stepDecl is what a step's directives declare
	stepDecl struct {
		tags        api.Tags
		predicate   *api.ScriptConfig
		description string
		id          string
		attrs       string
		handling    api.Handling
		compensate  string
		compTimeout int64
		options     Options
		http        Options
		work        Options
	}
)

const (
	directivePrefix    = "//argyll:"
	stepDirective      = "step"
	wrapDirective      = "wrap"
	propsDirective     = "props"
	httpDirective      = "http"
	workDirective      = "work"
	predicateDirective = "predicate"
	tagsDirective      = "tags"
	descDirective      = "description"
	memoDirective      = "memoize"
	compDirective      = "compensate"
	embedDirective     = "embed"

	// registration prepends the host the step server is reachable on
	healthPath = "/health"
)

var (
	ErrBadSignature = errors.New("unsupported step signature")
	ErrBadDirective = errors.New("invalid argyll directive")
)

var (
	validStepID = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)

	reservedStepTypes = map[api.StepType]bool{
		api.StepTypeService: true,
		api.StepTypeScript:  true,
		api.StepTypeFlow:    true,
	}
)

func (g *pkgGen) addFunc(fn *ast.FuncDecl) error {
	directive, ok := directiveOf(fn)
	if !ok {
		return nil
	}
	sig, err := g.signatureOf(fn)
	if err != nil {
		return err
	}
	wrap := directive.kind == wrapDirective
	decl, err := g.declOf(fn, directive.args, wrap)
	if err != nil {
		return err
	}

	var step stepModel
	if wrap {
		step, err = g.wrapFor(fn, sig, decl)
	} else {
		step, err = g.stepFor(fn, sig, decl)
	}
	if err != nil {
		return err
	}
	if g.dialect == embeddedDialect {
		step.embedType, err = g.claimEmbedType(fn, decl.id)
	}
	if err != nil {
		return err
	}

	g.steps = append(g.steps, step)
	return nil
}

func (g *pkgGen) declOf(
	fn *ast.FuncDecl, args string, wrap bool,
) (stepDecl, error) {
	head, options, err := ParseOptions(args)
	if err != nil {
		return stepDecl{}, g.errorAt(fn, "%w", err)
	}
	decl := SplitHead(head)
	id, err := g.stepIDOf(fn, decl, wrap)
	if err != nil {
		return stepDecl{}, err
	}
	dirs, err := g.directivesOf(fn)
	if err != nil {
		return stepDecl{}, err
	}
	res := stepDecl{
		tags:        dirs.tags,
		predicate:   dirs.predicate,
		description: dirs.description,
		id:          id,
		attrs:       decl.Attrs,
		options:     options,
		http:        dirs.http,
		work:        dirs.work,
	}
	if err := handlingIn(fn, &res); err != nil {
		return stepDecl{}, g.errorAt(fn, "%w", err)
	}
	return res, nil
}

func (g *pkgGen) stepIDOf(
	fn *ast.FuncDecl, decl Head, wrap bool,
) (string, error) {
	if decl.Attrs != "" && !wrap {
		return "", g.errorAt(fn,
			"%w: %s takes no attribute spec, so name them in the struct",
			ErrBadDirective, directivePrefix+stepDirective)
	}
	id := decl.Name
	if id == "" {
		id = KebabCase(fn.Name.Name)
	}
	if !validStepID.MatchString(id) {
		return "", g.errorAt(fn, "%w: bad step ID %q", ErrBadDirective, id)
	}
	return id, nil
}

// claimEmbedType is the step type an embedded engine runs a step as, the embed
// directive's or the step's ID, and names one function's handler
func (g *pkgGen) claimEmbedType(fn *ast.FuncDecl, id string) (string, error) {
	typ, err := embedIn(fn)
	if err != nil {
		return "", g.errorAt(fn, "%w", err)
	}
	if typ == "" {
		typ = id
	}
	if !validStepID.MatchString(typ) {
		return "", g.errorAt(fn, "%w: bad step type %q", ErrBadDirective, typ)
	}
	if reservedStepTypes[api.StepType(typ)] {
		return "", g.errorAt(fn, "%w: step type %q is built in",
			ErrBadDirective, typ)
	}
	if other, ok := g.embedTypes[typ]; ok {
		return "", g.errorAt(fn, "%w: step type %q is already %s's",
			ErrBadDirective, typ, other)
	}
	g.embedTypes[typ] = fn.Name.Name
	return typ, nil
}

func (g *pkgGen) directivesOf(fn *ast.FuncDecl) (stepDirectives, error) {
	if err := g.rejectProps(fn); err != nil {
		return stepDirectives{}, err
	}
	http, err := optionsIn(fn, httpDirective)
	if err != nil {
		return stepDirectives{}, g.errorAt(fn, "%w", err)
	}
	work, err := optionsIn(fn, workDirective)
	if err != nil {
		return stepDirectives{}, g.errorAt(fn, "%w", err)
	}
	tags, err := tagsIn(fn)
	if err != nil {
		return stepDirectives{}, g.errorAt(fn, "%w", err)
	}
	predicate, err := parsePredicate(fn)
	if err != nil {
		return stepDirectives{}, g.errorAt(fn, "%w", err)
	}
	description, err := descriptionIn(fn)
	if err != nil {
		return stepDirectives{}, g.errorAt(fn, "%w", err)
	}
	return stepDirectives{
		tags:        tags,
		predicate:   predicate,
		description: description,
		http:        http,
		work:        work,
	}, nil
}

func (g *pkgGen) rejectProps(fn *ast.FuncDecl) error {
	legacy, err := optionsIn(fn, propsDirective)
	if err != nil {
		return g.errorAt(fn, "%w", err)
	}
	if len(legacy) > 0 {
		return g.errorAt(fn, "%w: %s is not supported",
			ErrBadDirective, directivePrefix+propsDirective)
	}
	return nil
}

func (g *pkgGen) model(config *stepModelConfig) (stepModel, error) {
	fn := config.function
	decl := config.declaration
	spec := &api.Step{
		Attributes:  config.attributes,
		Tags:        decl.tags,
		Predicate:   decl.predicate,
		Description: decl.description,
		Type:        api.StepTypeService,
		ID:          api.StepID(decl.id),
		Name:        api.Name(TitleCase(fn.Name.Name)),
		HTTP: &api.HTTPConfig{
			Invoke: api.HTTPAction{Endpoint: "/" + decl.id},
			Health: healthPath,
		},
		Handling: decl.handling,
	}
	if config.compensate != "" {
		spec.HTTP.Compensate = &api.HTTPAction{
			Endpoint: "/" + decl.id + "/compensate",
			Timeout:  decl.compTimeout,
		}
	}
	if err := applyOptions(spec, decl.options, stepSetters); err != nil {
		return stepModel{}, g.errorAt(fn, "%w", err)
	}
	if err := applyOptions(spec.HTTP, decl.http, httpSetters); err != nil {
		return stepModel{}, g.errorAt(fn, "%w", err)
	}
	if len(decl.work) > 0 {
		spec.WorkConfig = &api.WorkConfig{}
	}
	err := applyOptions(spec.WorkConfig, decl.work, workSetters)
	if err != nil {
		return stepModel{}, g.errorAt(fn, "%w", err)
	}
	if err := spec.Validate(); err != nil {
		return stepModel{}, g.errorAt(fn, "%w", err)
	}
	return stepModel{
		spec:       spec,
		invoke:     config.invoke,
		compensate: config.compensate,
	}, nil
}

func (g *pkgGen) stepFor(
	fn *ast.FuncDecl, sig *types.Signature, decl stepDecl,
) (stepModel, error) {
	n := sig.Params().Len()
	if n > 1 {
		return stepModel{}, g.errorAt(fn,
			"%w: %s takes zero or one argument struct",
			ErrBadSignature, fn.Name.Name)
	}
	var in types.Type
	call := fn.Name.Name + "()"
	if n == 1 {
		in = sig.Params().At(0).Type()
		call = fn.Name.Name + "(in)"
	}
	inCodec, inAttrs, err := g.contract(fn, in, false)
	if err != nil {
		return stepModel{}, err
	}

	res, hasErr, err := g.results(fn, sig)
	if err != nil {
		return stepModel{}, err
	}
	outCodec, outAttrs, err := g.contract(fn, res, true)
	if err != nil {
		return stepModel{}, err
	}

	body := syncBody(syncBodyArgs{
		Call:       call,
		Output:     res != nil,
		Fallible:   hasErr,
		OutputType: g.typeOf(res),
	})
	compFields, err := taggedCompFields(in, inAttrs)
	if err != nil {
		return stepModel{}, g.errorAt(fn, "%w", err)
	}
	outFields, err := taggedCompFields(res, outAttrs)
	if err != nil {
		return stepModel{}, g.errorAt(fn, "%w", err)
	}
	compFields = append(compFields, outFields...)
	compensate, err := g.compHandler(&compHandlerConfig{
		fn:     fn,
		decl:   decl,
		fields: compFields,
	})
	if err != nil {
		return stepModel{}, err
	}
	attrs := mergeAttributes(attributeSets{inputs: inAttrs, outputs: outAttrs})
	return g.model(&stepModelConfig{
		function:    fn,
		declaration: decl,
		attributes:  attrs,
		invoke: syncHandler(syncHandlerArgs{
			Adapter:    g.dialect.sync,
			InAdapter:  inCodec,
			OutAdapter: outCodec,
			InType:     g.typeOf(in),
			OutType:    g.typeOf(res),
			Body:       body,
		}),
		compensate: compensate,
	})
}

func (g *pkgGen) wrapFor(
	fn *ast.FuncDecl, sig *types.Signature, decl stepDecl,
) (stepModel, error) {
	names, err := parseWrap(decl.attrs)
	if err != nil {
		return stepModel{}, g.errorAt(fn, "%w: %s", err, fn.Name.Name)
	}
	if names.inputs == nil {
		names.inputs, err = g.inferNames(
			fn, sig.Params(), sig.Params().Len(), "parameter",
		)
	}
	if err != nil {
		return stepModel{}, err
	}
	if names.outputs == nil {
		names.outputs, err = g.inferNames(
			fn, sig.Results(), valueCount(sig.Results()), "result",
		)
	}
	if err != nil {
		return stepModel{}, err
	}
	if sig.Params().Len() != len(names.inputs) {
		return stepModel{}, g.errorAt(fn,
			"%w: %s declares %d inputs but takes %d", ErrBadDirective,
			fn.Name.Name, len(names.inputs), sig.Params().Len())
	}
	res, hasErr, err := g.wrapResults(fn, sig, len(names.outputs))
	if err != nil {
		return stepModel{}, err
	}

	inType := fn.Name.Name + "In"
	outType := fn.Name.Name + "Out"
	inCodec, inAttrs, err := g.wrapStruct(
		inType, names.inputs, paramTypes(sig), false,
	)
	if err != nil {
		return stepModel{}, g.errorAt(fn, "%w", err)
	}
	outCodec, outAttrs, err := g.wrapStruct(outType, names.outputs, res, true)
	if err != nil {
		return stepModel{}, g.errorAt(fn, "%w", err)
	}
	compFields := namedCompFields(names.inputs, paramTypes(sig), inAttrs)
	outFields := namedCompFields(names.outputs, res, outAttrs)
	compFields = append(compFields, outFields...)
	compensate, err := g.compHandler(&compHandlerConfig{
		fn:     fn,
		decl:   decl,
		fields: compFields,
		wrap:   true,
	})
	if err != nil {
		return stepModel{}, err
	}

	attrs := mergeAttributes(attributeSets{inputs: inAttrs, outputs: outAttrs})
	return g.model(&stepModelConfig{
		function:    fn,
		declaration: decl,
		attributes:  attrs,
		invoke: syncHandler(syncHandlerArgs{
			Adapter:    g.dialect.sync,
			InAdapter:  inCodec,
			OutAdapter: outCodec,
			InType:     inType,
			OutType:    outType,
			Body:       wrapBody(fn.Name.Name, names, outType, hasErr),
		}),
		compensate: compensate,
	})
}

func (g *pkgGen) compHandler(cfg *compHandlerConfig) (string, error) {
	name := cfg.decl.compensate
	if name == "" {
		return "", nil
	}
	sig, err := g.compSignature(cfg.fn, name)
	if err != nil {
		return "", err
	}
	if sig.Params().Len() == 0 {
		return compAdapter(compAdapterConfig{
			Adapter:           g.dialect.compensate,
			CompensateAdapter: g.emptyCodec(),
			Type:              "struct{}",
			Call:              name + "()",
			Fallible:          sig.Results().Len() == 1,
		}), nil
	}
	if cfg.wrap {
		return g.wrapCompHandler(cfg, sig)
	}
	return g.stepCompHandler(cfg, sig)
}

func (g *pkgGen) compSignature(
	fn *ast.FuncDecl, name string,
) (*types.Signature, error) {
	obj := g.pkg.Types.Scope().Lookup(name)
	if obj == nil {
		return nil, g.errorAt(fn, "%w: compensator %s not found",
			ErrBadSignature, name)
	}
	declared, ok := obj.(*types.Func)
	if !ok {
		return nil, g.errorAt(fn,
			"%w: compensator %s must be a function", ErrBadSignature, name)
	}
	sig := declared.Type().(*types.Signature)
	if sig.Recv() != nil || sig.TypeParams() != nil || sig.Variadic() {
		return nil, g.errorAt(fn,
			"%w: compensator %s must be a plain non-variadic function",
			ErrBadSignature, name)
	}
	res := sig.Results()
	if res.Len() > 1 || res.Len() == 1 && !isError(res.At(0).Type()) {
		return nil, g.errorAt(fn,
			"%w: compensator %s must return nothing or error",
			ErrBadSignature, name)
	}
	return sig, nil
}

func (g *pkgGen) stepCompHandler(
	cfg *compHandlerConfig, sig *types.Signature,
) (string, error) {
	name := cfg.decl.compensate
	params := sig.Params()
	if params.Len() > 1 {
		return "", g.errorAt(cfg.fn,
			"%w: compensator %s takes zero or one argument struct",
			ErrBadSignature, name)
	}

	typ := params.At(0).Type()
	st, ok := typ.Underlying().(*types.Struct)
	if !ok {
		return "", g.errorAt(cfg.fn,
			"%w: compensator %s argument %s is not a struct",
			ErrBadSignature, name, g.typeOf(typ))
	}
	fields, err := structFields(st)
	if err != nil {
		return "", g.errorAt(cfg.fn, "%w", err)
	}
	available := make(map[string]types.Type, len(cfg.fields))
	for _, f := range cfg.fields {
		available[f.name] = f.typ
	}
	seen := make(map[string]bool, len(fields))
	for _, f := range fields {
		fieldName := resolveInnerName(f)
		want, ok := available[fieldName]
		if !ok {
			return "", g.errorAt(cfg.fn,
				"%w: compensator %s field %s is not compensated",
				ErrBadSignature, name, fieldName)
		}
		if seen[fieldName] {
			return "", g.errorAt(cfg.fn,
				"%w: compensator %s repeats field %s",
				ErrBadSignature, name, fieldName)
		}
		seen[fieldName] = true
		if !types.Identical(f.Type(), want) {
			return "", g.errorAt(cfg.fn,
				"%w: compensator %s field %s has type %s; want %s",
				ErrBadSignature, name, fieldName, g.typeOf(f.Type()),
				g.typeOf(want))
		}
	}
	codec, err := g.codecExpr(typ)
	if err != nil {
		return "", g.errorAt(cfg.fn, "%w", err)
	}
	return compAdapter(compAdapterConfig{
		Adapter:           g.dialect.compensate,
		CompensateAdapter: codec,
		Type:              g.typeOf(typ),
		Call:              name + "(in)",
		Fallible:          sig.Results().Len() == 1,
	}), nil
}

func (g *pkgGen) wrapCompHandler(
	cfg *compHandlerConfig, sig *types.Signature,
) (string, error) {
	name := cfg.decl.compensate
	params := sig.Params()
	names := make([]string, params.Len())
	selectedTypes := make([]types.Type, params.Len())
	args := make([]string, params.Len())
	used := map[string]bool{}
	for i := range params.Len() {
		param := params.At(i)
		paramName := SnakeCase(param.Name())
		if param.Name() == "" || param.Name() == "_" {
			return "", g.errorAt(cfg.fn,
				"%w: compensator %s argument %d is unnamed",
				ErrBadSignature, name, i+1)
		}
		var found *compField
		for j := range cfg.fields {
			field := &cfg.fields[j]
			if SnakeCase(ExportedName(field.name)) != paramName {
				continue
			}
			if found != nil {
				return "", g.errorAt(cfg.fn,
					"%w: compensator %s argument %s is ambiguous",
					ErrBadSignature, name, param.Name())
			}
			found = field
		}
		if found == nil {
			return "", g.errorAt(cfg.fn,
				"%w: compensator %s argument %s is not a step attribute",
				ErrBadSignature, name, param.Name())
		}
		if used[found.name] {
			return "", g.errorAt(cfg.fn,
				"%w: compensator %s repeats attribute %s",
				ErrBadSignature, name, found.name)
		}
		used[found.name] = true
		if !types.Identical(param.Type(), found.typ) {
			return "", g.errorAt(cfg.fn,
				"%w: compensator %s argument %s has type %s; want %s",
				ErrBadSignature, name, param.Name(), g.typeOf(param.Type()),
				g.typeOf(found.typ))
		}
		found.attr.Compensated = true
		names[i] = found.name
		selectedTypes[i] = found.typ
		args[i] = "in." + ExportedName(found.name)
	}

	inType := cfg.fn.Name.Name + "CompIn"
	codec, _, err := g.wrapStruct(inType, names, selectedTypes, false)
	if err != nil {
		return "", g.errorAt(cfg.fn, "%w", err)
	}
	call := fmt.Sprintf("%s(%s)", name, strings.Join(args, ", "))
	return compAdapter(compAdapterConfig{
		Adapter:           g.dialect.compensate,
		CompensateAdapter: codec,
		Type:              inType,
		Call:              call,
		Fallible:          sig.Results().Len() == 1,
	}), nil
}

func (g *pkgGen) contract(
	fn *ast.FuncDecl, t types.Type, output bool,
) (string, api.AttributeSpecs, error) {
	if t == nil {
		return g.emptyCodec(), nil, nil
	}
	st, ok := t.Underlying().(*types.Struct)
	if !ok {
		return "", nil, g.errorAt(fn, "%w: %s is not a struct",
			ErrBadSignature, g.typeOf(t))
	}
	expr, err := g.codecExpr(t)
	if err != nil {
		return "", nil, g.errorAt(fn, "%w", err)
	}
	fields, err := structFields(st)
	if err != nil {
		return "", nil, g.errorAt(fn, "%w", err)
	}
	attrs := api.AttributeSpecs{}
	for _, f := range fields {
		spec, err := newAttr(f.Type(), f.options, output)
		if err != nil {
			return "", nil, g.errorAt(fn, "%w on %s", err, f.Name())
		}
		attrs[api.Name(f.attr)] = spec
	}
	return expr, attrs, nil
}

func (g *pkgGen) results(
	fn *ast.FuncDecl, sig *types.Signature,
) (types.Type, bool, error) {
	res := sig.Results()
	switch {
	case res.Len() == 0:
		return nil, false, nil
	case res.Len() == 1 && isError(res.At(0).Type()):
		return nil, true, nil
	case res.Len() == 1:
		return res.At(0).Type(), false, nil
	case res.Len() == 2 && isError(res.At(1).Type()):
		return res.At(0).Type(), true, nil
	default:
		return nil, false, g.errorAt(fn,
			"%w: %s returns more than an outputs struct and an error",
			ErrBadSignature, fn.Name.Name)
	}
}

func (g *pkgGen) inferNames(
	fn *ast.FuncDecl, vars *types.Tuple, n int, kind string,
) ([]string, error) {
	names := make([]string, n)
	for i := range n {
		switch name := vars.At(i).Name(); name {
		case "", "_":
			return nil, g.errorAt(fn,
				"%w: %s %s %d is unnamed, so name the %ss in the directive",
				ErrBadDirective, fn.Name.Name, kind, i+1, kind)
		default:
			names[i] = SnakeCase(name)
		}
	}
	return names, nil
}

func (g *pkgGen) wrapResults(
	fn *ast.FuncDecl, sig *types.Signature, want int,
) ([]types.Type, bool, error) {
	res := sig.Results()
	n := valueCount(res)
	hasErr := n < res.Len()
	if n != want {
		return nil, false, g.errorAt(fn,
			"%w: %s declares %d outputs but returns %d",
			ErrBadDirective, fn.Name.Name, want, n)
	}
	out := make([]types.Type, n)
	for i := range n {
		out[i] = res.At(i).Type()
	}
	return out, hasErr, nil
}

func (g *pkgGen) signatureOf(fn *ast.FuncDecl) (*types.Signature, error) {
	obj := g.pkg.TypesInfo.Defs[fn.Name]
	if obj == nil {
		return nil, g.errorAt(fn, "%w: %s could not be type checked",
			ErrBadSignature, fn.Name.Name)
	}
	sig, ok := obj.Type().(*types.Signature)
	if !ok || sig.Recv() != nil || sig.TypeParams() != nil {
		return nil, g.errorAt(fn,
			"%w: %s must be a plain generic-free function",
			ErrBadSignature, fn.Name.Name)
	}
	return sig, nil
}

func (g *pkgGen) errorAt(fn *ast.FuncDecl, format string, a ...any) error {
	pos := g.pkg.Fset.Position(fn.Pos())
	return fmt.Errorf("%w: %s", fmt.Errorf(format, a...), pos)
}

func mergeAttributes(attrs attributeSets) api.AttributeSpecs {
	res := make(api.AttributeSpecs, len(attrs.inputs)+len(attrs.outputs))
	maps.Copy(res, attrs.inputs)
	maps.Copy(res, attrs.outputs)
	return res
}

func paramTypes(sig *types.Signature) []types.Type {
	params := sig.Params()
	res := make([]types.Type, params.Len())
	for i := range params.Len() {
		res[i] = params.At(i).Type()
	}
	return res
}

func isError(t types.Type) bool {
	named, ok := t.(*types.Named)
	return ok && named.Obj().Pkg() == nil && named.Obj().Name() == "error"
}

// valueCount is how many results are values, leaving out a trailing error
func valueCount(res *types.Tuple) int {
	n := res.Len()
	if n > 0 && isError(res.At(n-1).Type()) {
		return n - 1
	}
	return n
}

func taggedCompFields(
	t types.Type, attrs api.AttributeSpecs,
) ([]compField, error) {
	if t == nil {
		return nil, nil
	}
	st, ok := t.Underlying().(*types.Struct)
	if !ok {
		return nil, fmt.Errorf("%w: compensation input is not a struct",
			ErrBadSignature)
	}
	fields, err := structFields(st)
	if err != nil {
		return nil, err
	}
	var res []compField
	for _, f := range fields {
		attr := attrs[api.Name(f.attr)]
		if attr == nil || !attr.Compensated {
			continue
		}
		res = append(res, compField{
			attr: attr,
			name: resolveInnerName(f),
			typ:  f.Type(),
		})
	}
	return res, nil
}

func namedCompFields(
	names []string, types []types.Type, attrs api.AttributeSpecs,
) []compField {
	res := make([]compField, len(names))
	for i, name := range names {
		res[i] = compField{
			attr: attrs[api.Name(name)],
			name: name,
			typ:  types[i],
		}
	}
	return res
}
