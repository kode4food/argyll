package generator

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/kode4food/argyll/engine/pkg/api"
)

// http, work and tags are repeatable option-only directives, so a long set can
// spread across lines
func optionsIn(fn *ast.FuncDecl, directive string) (Options, error) {
	var res Options
	for _, c := range fn.Doc.List {
		ref, ok := parseDirective(c.Text)
		if !ok || ref.kind != directive {
			continue
		}
		head, options, err := ParseOptions(ref.args)
		if err != nil {
			return nil, err
		}
		if head != "" || len(options) == 0 {
			return nil, fmt.Errorf("%w: %s takes key%svalue options, got %q",
				ErrBadDirective, directivePrefix+directive, optionAssign,
				ref.args)
		}
		res = append(res, options...)
	}
	return res, nil
}

func directiveOf(fn *ast.FuncDecl) (directiveRef, bool) {
	if fn.Doc == nil {
		return directiveRef{}, false
	}
	for _, c := range fn.Doc.List {
		ref, ok := parseDirective(c.Text)
		if !ok {
			continue
		}
		switch ref.kind {
		case stepDirective, wrapDirective:
			return ref, true
		}
	}
	return directiveRef{}, false
}

func parseDirective(text string) (directiveRef, bool) {
	rest, ok := strings.CutPrefix(strings.TrimSpace(text), directivePrefix)
	if !ok || rest == "" {
		return directiveRef{}, false
	}
	i := strings.IndexAny(rest, " \t"+optionSeparator)
	if i < 0 {
		return directiveRef{kind: rest}, true
	}
	return directiveRef{
		kind: rest[:i],
		args: strings.TrimSpace(rest[i:]),
	}, true
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

// a tag is an opaque string, so the directive takes plain semicolon separated
// values rather than the key:value options the others parse
func tagsIn(fn *ast.FuncDecl) (api.Tags, error) {
	var res api.Tags
	for _, c := range fn.Doc.List {
		ref, ok := parseDirective(c.Text)
		if !ok || ref.kind != tagsDirective {
			continue
		}
		if strings.TrimSpace(ref.args) == "" {
			return nil, fmt.Errorf("%w: %s takes tags, got %q",
				ErrBadDirective, directivePrefix+tagsDirective, ref.args)
		}
		for seg := range strings.SplitSeq(ref.args, optionSeparator) {
			tag := strings.TrimSpace(seg)
			if tag == "" {
				return nil, fmt.Errorf("%w: %s has an empty tag",
					ErrBadOption, directivePrefix+tagsDirective)
			}
			res = append(res, tag)
		}
	}
	return res.Normalize(), nil
}

// a description is free prose, so the directive takes its whole argument
func descriptionIn(fn *ast.FuncDecl) (string, error) {
	var res string
	for _, c := range fn.Doc.List {
		ref, ok := parseDirective(c.Text)
		if !ok || ref.kind != descDirective {
			continue
		}
		if res != "" {
			return "", fmt.Errorf("%w: %s repeats",
				ErrBadDirective, directivePrefix+descDirective)
		}
		res = strings.TrimSpace(ref.args)
		if res == "" {
			return "", fmt.Errorf("%w: %s needs a description",
				ErrBadDirective, directivePrefix+descDirective)
		}
	}
	return res, nil
}

func parsePredicate(fn *ast.FuncDecl) (*api.ScriptConfig, error) {
	var res *api.ScriptConfig
	for _, c := range fn.Doc.List {
		ref, ok := parseDirective(c.Text)
		if !ok || ref.kind != predicateDirective {
			continue
		}
		if res != nil {
			return nil, fmt.Errorf("%w: %s repeats",
				ErrBadDirective, directivePrefix+predicateDirective)
		}
		cfg := &api.ScriptConfig{Language: api.ScriptLangLua}
		if !parseScript(cfg, ref.args) {
			return nil, fmt.Errorf("%w: %s needs a script",
				ErrBadDirective, directivePrefix+predicateDirective)
		}
		res = cfg
	}
	return res, nil
}

func handlingIn(fn *ast.FuncDecl, decl *stepDecl) error {
	memoized, err := memoIn(fn)
	if err != nil {
		return err
	}
	if err := compIn(fn, decl); err != nil {
		return err
	}
	if memoized && decl.compensate != "" {
		return fmt.Errorf("%w: %s and %s are mutually exclusive",
			ErrBadDirective, directivePrefix+memoDirective,
			directivePrefix+compDirective,
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

func memoIn(fn *ast.FuncDecl) (bool, error) {
	var res bool
	for _, c := range fn.Doc.List {
		ref, ok := parseDirective(c.Text)
		if !ok || ref.kind != memoDirective {
			continue
		}
		if ref.args != "" {
			return false, fmt.Errorf("%w: %s takes no value",
				ErrBadDirective, directivePrefix+memoDirective)
		}
		if res {
			return false, fmt.Errorf("%w: %s repeats",
				ErrBadDirective, directivePrefix+memoDirective)
		}
		res = true
	}
	return res, nil
}

func compIn(fn *ast.FuncDecl, decl *stepDecl) error {
	for _, c := range fn.Doc.List {
		ref, ok := parseDirective(c.Text)
		if !ok || ref.kind != compDirective {
			continue
		}
		if decl.compensate != "" {
			return fmt.Errorf("%w: %s repeats", ErrBadDirective,
				directivePrefix+compDirective)
		}
		name, opts, err := ParseOptions(ref.args)
		if err != nil {
			return err
		}
		if !token.IsIdentifier(name) {
			return fmt.Errorf("%w: %s needs a function name",
				ErrBadDirective, directivePrefix+compDirective)
		}
		if err := compOptions(decl, opts); err != nil {
			return err
		}
		decl.compensate = name
	}
	return nil
}

func compOptions(decl *stepDecl, opts Options) error {
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
	return nil
}
