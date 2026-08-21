package generator

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

// FileMode is the permission mode of generated files
const FileMode = 0o644

var ErrLoadFailed = errors.New("failed to load packages")

// Generate writes a generated file for every loaded package containing
// Argyll directives, returning the paths it wrote
func Generate(dir string, patterns ...string) ([]string, error) {
	pkgs, err := Load(dir, patterns...)
	if err != nil {
		return nil, err
	}
	var written []string
	for _, p := range pkgs {
		path, ok := generatedPath(p)
		if !ok {
			continue
		}
		src, err := Render(p)
		if err != nil {
			return nil, err
		}
		if src == nil {
			if err := removeStale(path); err != nil {
				return nil, err
			}
			continue
		}
		if same(path, src) {
			continue
		}
		if err := os.WriteFile(path, src, FileMode); err != nil {
			return nil, err
		}
		written = append(written, path)
	}
	return written, nil
}

// Load type checks the packages matching patterns, relative to dir
func Load(dir string, patterns ...string) ([]*packages.Package, error) {
	cfg := &packages.Config{
		Dir: dir,
		Mode: packages.NeedName | packages.NeedFiles |
			packages.NeedCompiledGoFiles | packages.NeedImports |
			packages.NeedDeps | packages.NeedTypes | packages.NeedSyntax |
			packages.NeedTypesInfo,
	}
	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		return nil, errors.Join(ErrLoadFailed, err)
	}
	for _, p := range pkgs {
		if len(p.Syntax) == 0 && len(p.Errors) > 0 {
			return nil, errors.Join(ErrLoadFailed, p.Errors[0])
		}
	}
	return pkgs, nil
}

func generatedPath(p *packages.Package) (string, bool) {
	if strings.HasSuffix(p.Name, "_test") {
		return "", false
	}
	for _, f := range p.GoFiles {
		return filepath.Join(filepath.Dir(f), GeneratedFile), true
	}
	return "", false
}

func removeStale(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func same(path string, src []byte) bool {
	old, err := os.ReadFile(path)
	return err == nil && bytes.Equal(old, src)
}
