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
		spec    *api.Step
		handler string
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
		labels api.Labels
		id     string
		attrs  string
		props  Options
	}
)

const (
	stepDirective   = "//argyll:step"
	wrapDirective   = "//argyll:wrap"
	propsDirective  = "//argyll:props"
	labelsDirective = "//argyll:labels"

	// registration prepends the host the step server is reachable on
	healthPath = "/health"
)

var (
	ErrBadSignature = errors.New("unsupported step signature")
	ErrBadDirective = errors.New("invalid argyll directive")
)

// a step ID also names the generated var, so it stays kebab-case
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
	return stepDecl{
		labels: labels,
		id:     id,
		attrs:  decl.Attrs,
		props:  append(options, props...),
	}, nil
}

func (g *pkgGen) model(
	fn *ast.FuncDecl, decl stepDecl, attrs api.AttributeSpecs, handler string,
) (stepModel, error) {
	spec := &api.Step{
		Attributes: attrs,
		Labels:     decl.labels,
		Type:       api.StepTypeSync,
		ID:         api.StepID(decl.id),
		Name:       api.Name(TitleCase(fn.Name.Name)),
		HTTP: &api.HTTPConfig{
			Invoke: api.HTTPAction{Endpoint: "/" + decl.id},
			Health: healthPath,
		},
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
	return stepModel{spec: spec, handler: handler}, nil
}

// props and labels are their own directives, repeatable and options only,
// so a long set can spread across lines
func optionsIn(fn *ast.FuncDecl, directive string) (Options, error) {
	var res Options
	for _, c := range fn.Doc.List {
		text := strings.TrimSpace(c.Text)
		rest, ok := directiveArgs(text, directive)
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
	if sig.Params().Len() != 1 {
		return stepModel{}, g.errorAt(fn,
			"%w: %s takes one arguments struct",
			ErrBadSignature, fn.Name.Name)
	}
	in := sig.Params().At(0).Type()
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

	call := fn.Name.Name + "(in)"
	body := syncBody(call, res != nil, hasErr, g.typeOf(res))
	return g.model(fn, decl, merge(inAttrs, outAttrs), syncHandler(
		syncHandlerArgs{
			inCodec:  inCodec,
			outCodec: outCodec,
			inType:   g.typeOf(in),
			outType:  g.typeOf(res),
			body:     body,
		},
	))
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

	inType := "argyll" + fn.Name.Name + "In"
	outType := "argyll" + fn.Name.Name + "Out"
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

	return g.model(fn, decl, merge(inAttrs, outAttrs), syncHandler(
		syncHandlerArgs{
			inCodec:  inCodec,
			outCodec: outCodec,
			inType:   inType,
			outType:  outType,
			body:     wrapBody(fn.Name.Name, names, outType, hasErr),
		},
	))
}

func merge(in, out api.AttributeSpecs) api.AttributeSpecs {
	res := make(api.AttributeSpecs, len(in)+len(out))
	maps.Copy(res, in)
	maps.Copy(res, out)
	return res
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
	return fmt.Errorf("%s: %w", pos, fmt.Errorf(format, a...))
}

func directiveOf(fn *ast.FuncDecl) (directiveRef, bool) {
	if fn.Doc == nil {
		return directiveRef{}, false
	}
	for _, c := range fn.Doc.List {
		text := strings.TrimSpace(c.Text)
		for _, d := range []string{stepDirective, wrapDirective} {
			if args, ok := directiveArgs(text, d); ok {
				return directiveRef{kind: d, args: args}, true
			}
		}
	}
	return directiveRef{}, false
}

// the head may follow the directive on a space, or its options directly on a
// separator
func directiveArgs(comment, directive string) (string, bool) {
	rest, ok := strings.CutPrefix(comment, directive)
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

func syncBody(call string, hasOut, hasErr bool, outType string) string {
	switch {
	case hasOut && hasErr:
		return "return " + call
	case hasOut:
		return "return " + call + ", nil"
	case hasErr:
		return fmt.Sprintf("return %s{}, %s", outType, call)
	default:
		return fmt.Sprintf("%s\nreturn %s{}, nil", call, outType)
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
