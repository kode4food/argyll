package generator

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"maps"
	"regexp"
	"strings"

	"github.com/kode4food/argyll/engine/pkg/api"
)

type (
	stepModel struct {
		spec       *api.Step
		handler    string
		compensate string
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
		codec    string
		typ      string
		call     string
		fallible bool
	}

	stepModelConfig struct {
		function    *ast.FuncDecl
		declaration stepDecl
		attributes  api.AttributeSpecs
		handler     string
		compensate  string
	}

	attributeSets struct {
		inputs  api.AttributeSpecs
		outputs api.AttributeSpecs
	}

	directiveText struct {
		comment   string
		directive string
	}

	// directiveRef is a step or wrap directive and the text following it
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

	// stepDecl is what a directive declares, gathered from its own line and
	// from any props and labels directives
	stepDecl struct {
		labels      api.Labels
		id          string
		attrs       string
		handling    api.Handling
		compensate  string
		compTimeout int64
		props       Options
	}
)

const (
	stepDirective   = "//argyll:step"
	wrapDirective   = "//argyll:wrap"
	propsDirective  = "//argyll:props"
	labelsDirective = "//argyll:labels"
	memoDirective   = "//argyll:memoize"
	compDirective   = "//argyll:compensate"

	// registration prepends the host the step server is reachable on
	healthPath = "/health"
)

var (
	ErrBadSignature = errors.New("unsupported step signature")
	ErrBadDirective = errors.New("invalid argyll directive")
)

var validStepID = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)

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
	if decl.Attrs != "" && !wrap {
		return stepDecl{}, g.errorAt(fn,
			"%w: %s takes no attribute spec, so name them in the struct",
			ErrBadDirective, stepDirective)
	}
	id := decl.Name
	if id == "" {
		id = KebabCase(fn.Name.Name)
	}
	if !validStepID.MatchString(id) {
		return stepDecl{}, g.errorAt(fn, "%w: bad step ID %q",
			ErrBadDirective, id)
	}

	props, err := optionsIn(fn, propsDirective)
	if err != nil {
		return stepDecl{}, g.errorAt(fn, "%w", err)
	}
	declared, err := optionsIn(fn, labelsDirective)
	if err != nil {
		return stepDecl{}, g.errorAt(fn, "%w", err)
	}
	var labels api.Labels
	for _, o := range declared {
		if labels == nil {
			labels = api.Labels{}
		}
		labels[o.Key] = o.Value
	}
	res := stepDecl{
		labels: labels,
		id:     id,
		attrs:  decl.Attrs,
		props:  append(options, props...),
	}
	if err := handlingIn(fn, &res); err != nil {
		return stepDecl{}, g.errorAt(fn, "%w", err)
	}
	return res, nil
}

func (g *pkgGen) model(config *stepModelConfig) (stepModel, error) {
	fn := config.function
	decl := config.declaration
	spec := &api.Step{
		Attributes: config.attributes,
		Labels:     decl.labels,
		Type:       api.StepTypeSync,
		ID:         api.StepID(decl.id),
		Name:       api.Name(TitleCase(fn.Name.Name)),
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
	for _, o := range decl.props {
		set, ok := stepSetters[o.Key]
		if !ok {
			return stepModel{}, g.errorAt(fn, "%w: unknown property %q",
				ErrBadProp, o.Key)
		}
		if err := set(spec, o.Value); err != nil {
			return stepModel{}, g.errorAt(fn, "%w", err)
		}
	}
	if err := spec.Validate(); err != nil {
		return stepModel{}, g.errorAt(fn, "%w", err)
	}
	return stepModel{
		spec:       spec,
		handler:    config.handler,
		compensate: config.compensate,
	}, nil
}

// props and labels are their own directives, repeatable and options only,
// so a long set can spread across lines
func optionsIn(fn *ast.FuncDecl, directive string) (Options, error) {
	var res Options
	for _, c := range fn.Doc.List {
		text := strings.TrimSpace(c.Text)
		rest, ok := directiveArgs(directiveText{
			comment:   text,
			directive: directive,
		})
		if !ok {
			continue
		}
		head, options, err := ParseOptions(rest)
		if err != nil {
			return nil, err
		}
		if head != "" || len(options) == 0 {
			return nil, fmt.Errorf("%w: %s takes key%svalue options, got %q",
				ErrBadDirective, directive, optionAssign, rest)
		}
		res = append(res, options...)
	}
	return res, nil
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
		call:       call,
		output:     res != nil,
		fallible:   hasErr,
		outputType: g.typeOf(res),
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
		handler: syncHandler(syncHandlerArgs{
			inCodec:  inCodec,
			outCodec: outCodec,
			inType:   g.typeOf(in),
			outType:  g.typeOf(res),
			body:     body,
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
		n := sig.Params().Len()
		if names.inputs, err = g.inferNames(
			fn, sig.Params(), n, "parameter",
		); err != nil {
			return stepModel{}, err
		}
	}
	if names.outputs == nil {
		res := sig.Results()
		n := res.Len()
		if n > 0 && isError(res.At(n-1).Type()) {
			n--
		}
		if names.outputs, err = g.inferNames(
			fn, res, n, "result",
		); err != nil {
			return stepModel{}, err
		}
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
	compFields := namedCompFields(
		names.inputs, paramTypes(sig), inAttrs,
	)
	compFields = append(compFields, namedCompFields(
		names.outputs, res, outAttrs,
	)...)
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
		handler: syncHandler(syncHandlerArgs{
			inCodec:  inCodec,
			outCodec: outCodec,
			inType:   inType,
			outType:  outType,
			body:     wrapBody(fn.Name.Name, names, outType, hasErr),
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
			codec:    "codec.Struct[struct{}]()",
			typ:      "struct{}",
			call:     name + "()",
			fallible: sig.Results().Len() == 1,
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
		codec:    codec,
		typ:      g.typeOf(typ),
		call:     name + "(in)",
		fallible: sig.Results().Len() == 1,
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
		codec:    codec,
		typ:      inType,
		call:     call,
		fallible: sig.Results().Len() == 1,
	}), nil
}

func (g *pkgGen) contract(
	fn *ast.FuncDecl, t types.Type, output bool,
) (string, api.AttributeSpecs, error) {
	if t == nil {
		return "codec.Struct[struct{}]()", nil, nil
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
	n := res.Len()
	hasErr := n > 0 && isError(res.At(n-1).Type())
	if hasErr {
		n--
	}
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

func directiveOf(fn *ast.FuncDecl) (directiveRef, bool) {
	if fn.Doc == nil {
		return directiveRef{}, false
	}
	for _, c := range fn.Doc.List {
		text := strings.TrimSpace(c.Text)
		for _, d := range []string{stepDirective, wrapDirective} {
			if args, ok := directiveArgs(directiveText{
				comment:   text,
				directive: d,
			}); ok {
				return directiveRef{kind: d, args: args}, true
			}
		}
	}
	return directiveRef{}, false
}

// the head may follow the directive on a space, or its options directly on a
// separator
func directiveArgs(text directiveText) (string, bool) {
	rest, ok := strings.CutPrefix(text.comment, text.directive)
	if !ok {
		return "", false
	}
	if rest != "" && !strings.ContainsAny(rest[:1], " \t"+optionSeparator) {
		return "", false
	}
	return strings.TrimSpace(rest), true
}

func parseWrap(attrs string) (wrapNames, error) {
	ins, outs, _ := strings.Cut(attrs, arrow)
	inputs, err := attrNames(ins)
	if err != nil {
		return wrapNames{}, err
	}
	outputs, err := attrNames(outs)
	if err != nil {
		return wrapNames{}, err
	}
	return wrapNames{inputs: inputs, outputs: outputs}, nil
}

// an omitted list is inferred from the signature, an empty one is empty
func attrNames(s string) ([]string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	body, ok := strings.CutPrefix(s, "(")
	if !ok {
		return nil, fmt.Errorf("%w: attributes %q are not parenthesized",
			ErrBadDirective, s)
	}
	body, ok = strings.CutSuffix(strings.TrimSpace(body), ")")
	if !ok {
		return nil, fmt.Errorf("%w: attributes %q are not parenthesized",
			ErrBadDirective, s)
	}
	if strings.TrimSpace(body) == "" {
		return []string{}, nil
	}
	names := strings.Split(body, ",")
	for i, n := range names {
		n = strings.TrimSpace(n)
		if n == "" || strings.ContainsAny(n, " \t") {
			return nil, fmt.Errorf("%w: bad attribute name %q",
				ErrBadDirective, n)
		}
		names[i] = n
	}
	return names, nil
}

// syncHandlerArgs are the pieces of a generated synchronous handler
type syncHandlerArgs struct {
	inCodec  string
	outCodec string
	inType   string
	outType  string
	body     string
}

func syncHandler(args syncHandlerArgs) string {
	return fmt.Sprintf(
		"gen.Sync(\n%s, %s,\nfunc(in %s) (%s, error) {\n%s\n})",
		args.inCodec, args.outCodec, args.inType, args.outType, args.body,
	)
}

type syncBodyArgs struct {
	call       string
	output     bool
	fallible   bool
	outputType string
}

func syncBody(args syncBodyArgs) string {
	switch {
	case args.output && args.fallible:
		return "return " + args.call
	case args.output:
		return "return " + args.call + ", nil"
	case args.fallible:
		return fmt.Sprintf("return %s{}, %s", args.outputType, args.call)
	default:
		return fmt.Sprintf("%s\nreturn %s{}, nil", args.call,
			args.outputType)
	}
}

func wrapBody(
	fn string, names wrapNames, outType string, hasErr bool,
) string {
	call := make([]string, len(names.inputs))
	for i, n := range names.inputs {
		call[i] = "in." + ExportedName(n)
	}
	lhs := make([]string, 0, len(names.outputs)+1)
	assign := make([]string, len(names.outputs))
	for i, n := range names.outputs {
		lhs = append(lhs, fmt.Sprintf("r%d", i))
		assign[i] = fmt.Sprintf("%s: r%d", ExportedName(n), i)
	}
	if hasErr {
		lhs = append(lhs, "err")
	}
	res := fmt.Sprintf("return %s{%s}, nil", outType,
		strings.Join(assign, ", "))

	var sb strings.Builder
	if len(lhs) > 0 {
		fmt.Fprintf(&sb, "%s := ", strings.Join(lhs, ", "))
	}
	fmt.Fprintf(&sb, "%s(%s)\n", fn, strings.Join(call, ", "))
	if hasErr {
		fmt.Fprintf(&sb, "if err != nil {\nreturn %s{}, err\n}\n", outType)
	}
	sb.WriteString(res)
	return sb.String()
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

func compAdapter(cfg compAdapterConfig) string {
	body := cfg.call + "\nreturn nil"
	if cfg.fallible {
		body = "return " + cfg.call
	}
	return fmt.Sprintf(
		"gen.Compensate(\n%s,\nfunc(in %s) error {\n%s\n})",
		cfg.codec, cfg.typ, body,
	)
}

func handlingIn(fn *ast.FuncDecl, decl *stepDecl) error {
	var memoized bool
	for _, c := range fn.Doc.List {
		text := strings.TrimSpace(c.Text)
		if value, ok := directiveArgs(directiveText{
			comment:   text,
			directive: memoDirective,
		}); ok {
			if value != "" {
				return fmt.Errorf("%w: %s takes no value",
					ErrBadDirective, memoDirective)
			}
			if memoized {
				return fmt.Errorf("%w: %s repeats",
					ErrBadDirective, memoDirective)
			}
			memoized = true
			continue
		}

		value, ok := directiveArgs(directiveText{
			comment:   text,
			directive: compDirective,
		})
		if !ok {
			continue
		}
		if decl.compensate != "" {
			return fmt.Errorf("%w: %s repeats", ErrBadDirective,
				compDirective)
		}
		name, opts, err := ParseOptions(value)
		if err != nil {
			return err
		}
		if !token.IsIdentifier(name) {
			return fmt.Errorf("%w: %s needs a function name",
				ErrBadDirective, compDirective)
		}
		for _, o := range opts {
			if o.Key != timeoutProp {
				return fmt.Errorf("%w: unknown property %q", ErrBadProp, o.Key)
			}
			ms, err := parseMillis(o)
			if err != nil {
				return err
			}
			decl.compTimeout = ms
		}
		decl.compensate = name
	}
	if memoized && decl.compensate != "" {
		return fmt.Errorf("%w: %s and %s are mutually exclusive",
			ErrBadDirective, memoDirective, compDirective,
		)
	}
	if memoized {
		decl.handling = api.HandlingMemoized
	}
	if decl.compensate != "" {
		decl.handling = api.HandlingCompensated
	}
	return nil
}
