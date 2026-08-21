package generator

import (
	"errors"
	"fmt"
	"go/ast"
	"go/types"
	"strings"
)

type (
	stepModel struct {
		labels   map[string]string
		id       string
		name     string
		stepType string
		handler  string
		inputs   []attrModel
		outputs  []attrModel
	}

	attrModel struct {
		name     string
		attrType string
		optional bool
	}
)

const (
	stepDirective  = "//argyll:step"
	wrapDirective  = "//argyll:wrap"
	labelDirective = "//argyll:label"
)

var (
	ErrBadSignature = errors.New("unsupported step signature")
	ErrBadDirective = errors.New("invalid argyll directive")
)

func (g *pkgGen) addFunc(fn *ast.FuncDecl) error {
	directive, args, ok := directiveOf(fn)
	if !ok {
		return nil
	}
	sig, err := g.signatureOf(fn)
	if err != nil {
		return err
	}
	labels, err := labelsOf(fn)
	if err != nil {
		return g.errorAt(fn, "%w", err)
	}

	var step stepModel
	if directive == stepDirective {
		step, err = g.stepFor(fn, sig)
	} else {
		step, err = g.wrapFor(fn, sig, args)
	}
	if err != nil {
		return err
	}

	step.labels = labels
	g.steps = append(g.steps, step)
	return nil
}

func (g *pkgGen) stepFor(
	fn *ast.FuncDecl, sig *types.Signature,
) (stepModel, error) {
	if sig.Params().Len() != 1 {
		return stepModel{}, g.errorAt(fn,
			"%w: %s takes one arguments struct",
			ErrBadSignature, fn.Name.Name)
	}
	in := sig.Params().At(0).Type()
	inCodec, inAttrs, err := g.contract(fn, in)
	if err != nil {
		return stepModel{}, err
	}

	res, hasErr, err := g.results(fn, sig)
	if err != nil {
		return stepModel{}, err
	}
	outCodec, outAttrs, err := g.contract(fn, res)
	if err != nil {
		return stepModel{}, err
	}

	call := fn.Name.Name + "(in)"
	body := syncBody(call, res != nil, hasErr, g.typeOf(res))
	return stepModel{
		id:       KebabCase(fn.Name.Name),
		name:     TitleCase(fn.Name.Name),
		stepType: "api.StepTypeSync",
		inputs:   inAttrs,
		outputs:  outAttrs,
		handler: syncHandler(
			inCodec, outCodec, g.typeOf(in), g.typeOf(res), body,
		),
	}, nil
}

func (g *pkgGen) wrapFor(
	fn *ast.FuncDecl, sig *types.Signature, args string,
) (stepModel, error) {
	inNames, outNames, err := parseWrap(args)
	if err != nil {
		return stepModel{}, g.errorAt(fn, "%w: %s", err, fn.Name.Name)
	}
	if inNames == nil {
		n := sig.Params().Len()
		if inNames, err = g.inferNames(
			fn, sig.Params(), n, "parameter",
		); err != nil {
			return stepModel{}, err
		}
	}
	if outNames == nil {
		res := sig.Results()
		n := res.Len()
		if n > 0 && isError(res.At(n-1).Type()) {
			n--
		}
		if outNames, err = g.inferNames(
			fn, res, n, "result",
		); err != nil {
			return stepModel{}, err
		}
	}
	if sig.Params().Len() != len(inNames) {
		return stepModel{}, g.errorAt(fn,
			"%w: %s declares %d inputs but takes %d",
			ErrBadDirective, fn.Name.Name, len(inNames), sig.Params().Len())
	}
	res, hasErr, err := g.wrapResults(fn, sig, len(outNames))
	if err != nil {
		return stepModel{}, err
	}

	inType := "argyll" + fn.Name.Name + "In"
	outType := "argyll" + fn.Name.Name + "Out"
	inCodec, inAttrs, err := g.wrapStruct(inType, inNames, paramTypes(sig))
	if err != nil {
		return stepModel{}, g.errorAt(fn, "%w", err)
	}
	outCodec, outAttrs, err := g.wrapStruct(outType, outNames, res)
	if err != nil {
		return stepModel{}, g.errorAt(fn, "%w", err)
	}

	return stepModel{
		id:       KebabCase(fn.Name.Name),
		name:     TitleCase(fn.Name.Name),
		stepType: "api.StepTypeSync",
		inputs:   inAttrs,
		outputs:  outAttrs,
		handler: syncHandler(inCodec, outCodec, inType, outType,
			wrapBody(fn.Name.Name, inNames, outNames, outType, hasErr),
		),
	}, nil
}

func (g *pkgGen) contract(
	fn *ast.FuncDecl, t types.Type,
) (string, []attrModel, error) {
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
	specs, err := structFields(st)
	if err != nil {
		return "", nil, g.errorAt(fn, "%w", err)
	}
	var attrs []attrModel
	for _, f := range specs {
		attrs = append(attrs, attrModel{
			name:     f.attr,
			attrType: attributeType(f.Type()),
			optional: isPointer(f.Type()),
		})
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

func directiveOf(fn *ast.FuncDecl) (string, string, bool) {
	if fn.Doc == nil {
		return "", "", false
	}
	for _, c := range fn.Doc.List {
		text := strings.TrimSpace(c.Text)
		for _, d := range []string{stepDirective, wrapDirective} {
			if text == d {
				return d, "", true
			}
			if rest, ok := strings.CutPrefix(text, d+" "); ok {
				return d, strings.TrimSpace(rest), true
			}
		}
	}
	return "", "", false
}

func labelsOf(fn *ast.FuncDecl) (map[string]string, error) {
	if fn.Doc == nil {
		return nil, nil
	}
	labels := map[string]string{}
	for _, c := range fn.Doc.List {
		text := strings.TrimSpace(c.Text)
		rest, ok := strings.CutPrefix(text, labelDirective+" ")
		if !ok {
			continue
		}
		key, value, ok := strings.Cut(rest, "=")
		key = strings.TrimSpace(key)
		if !ok || key == "" {
			return nil, fmt.Errorf("%w: label %q is not key=value",
				ErrBadDirective, strings.TrimSpace(rest))
		}
		labels[key] = strings.TrimSpace(value)
	}
	if len(labels) == 0 {
		return nil, nil
	}
	return labels, nil
}

func parseWrap(args string) ([]string, []string, error) {
	ins, outs, _ := strings.Cut(args, "->")
	inNames, err := attrNames(ins)
	if err != nil {
		return nil, nil, err
	}
	outNames, err := attrNames(outs)
	if err != nil {
		return nil, nil, err
	}
	return inNames, outNames, nil
}

func attrNames(s string) ([]string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	names := strings.Split(s, ",")
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

func syncHandler(inCodec, outCodec, inType, outType, body string) string {
	return fmt.Sprintf(
		"gen.Sync(\n%s, %s,\nfunc(in %s) (%s, error) {\n%s\n})",
		inCodec, outCodec, inType, outType, body,
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
	fn string, inNames, outNames []string, outType string, hasErr bool,
) string {
	call := make([]string, len(inNames))
	for i, n := range inNames {
		call[i] = "in." + ExportedName(n)
	}
	lhs := make([]string, 0, len(outNames)+1)
	assign := make([]string, len(outNames))
	for i, n := range outNames {
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

func isPointer(t types.Type) bool {
	_, ok := t.(*types.Pointer)
	return ok
}
