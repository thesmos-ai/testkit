// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gate

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"
)

// HarnessField is one field the generated harness offers a consumer, and
// whether any consumer test in the corpus sets it.
type HarnessField struct {
	// Name is the field as a consumer spells it in the literal.
	Name string

	// Set reports that some corpus consumer test assigns it.
	Set bool
}

// HarnessFields reads every exported field off the generated harnesses
// and reports which the corpus's hand-written tests actually set.
//
// The harness is the whole consumer-facing surface: a field nobody sets
// is a capability the product ships and nothing exercises, which is the
// same silence the constructor census exists to break. Two of them —
// Oracle and Serial — only mean anything with more than one
// implementation in a run, so a corpus that always ran one would never
// have noticed they were inert.
//
// Read off the generated files rather than from a list, because the
// harness is generated: a field added to the template appears here the
// day it ships, unregistered and unset, and this reports it.
//
// The setters are the hand-written test files, which is what a consumer
// writes. Generated companions are excluded: the lowering assigns these
// fields itself, and counting that would let the generator vouch for its
// own surface.
func HarnessFields(corpusRoot string) ([]HarnessField, error) {
	declared, err := harnessFieldNames(corpusRoot)
	if err != nil {
		return nil, err
	}
	set, err := harnessFieldsSet(corpusRoot)
	if err != nil {
		return nil, err
	}
	out := make([]HarnessField, 0, len(declared))
	for _, name := range declared {
		out = append(out, HarnessField{Name: name, Set: set[name]})
	}
	return out, nil
}

// harnessFieldNames is every exported field of every `<Iface>Harness`
// the corpus generates, each once.
func harnessFieldNames(corpusRoot string) ([]string, error) {
	var out []string
	err := walkGoFiles(corpusRoot, func(file *ast.File) {
		for _, decl := range file.Decls {
			gen, isGen := decl.(*ast.GenDecl)
			if !isGen || gen.Tok != token.TYPE {
				continue
			}
			for _, spec := range gen.Specs {
				ts, isType := spec.(*ast.TypeSpec)
				if !isType || !strings.HasSuffix(ts.Name.Name, "Harness") {
					continue
				}
				st, isStruct := ts.Type.(*ast.StructType)
				if !isStruct {
					continue
				}
				for _, f := range st.Fields.List {
					for _, n := range f.Names {
						if n.IsExported() && !slices.Contains(out, n.Name) {
							out = append(out, n.Name)
						}
					}
				}
			}
		}
	}, func(path string) bool { return strings.HasSuffix(path, "iface_suite.gen.go") })
	if err != nil {
		return nil, err
	}
	slices.Sort(out)
	return out, nil
}

// harnessFieldsSet is every harness field a hand-written consumer test
// assigns, read off composite literals rather than by text so a field
// named in a comment does not vouch for itself.
func harnessFieldsSet(corpusRoot string) (map[string]bool, error) {
	out := map[string]bool{}
	err := walkGoFiles(corpusRoot, func(file *ast.File) {
		ast.Inspect(file, func(n ast.Node) bool {
			lit, isLit := n.(*ast.CompositeLit)
			if !isLit {
				return true
			}
			for _, elt := range lit.Elts {
				kv, isKV := elt.(*ast.KeyValueExpr)
				if !isKV {
					continue
				}
				if key, isIdent := kv.Key.(*ast.Ident); isIdent {
					out[key.Name] = true
				}
			}
			return true
		})
	}, handWrittenTest)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// handWrittenTest reports a test file a person wrote, which is the
// consumer surface this census is about.
//
// Not by one filename: the fixtures name their second implementation
// after what it is, and a census keyed on inmemory_test.go reported the
// fields only that file sets. Generated companions are excluded because
// the generator assigns these fields itself, and counting that would let
// it vouch for its own surface.
func handWrittenTest(path string) bool {
	return strings.HasSuffix(path, "_test.go") && !strings.HasSuffix(path, ".gen_test.go")
}

// walkGoFiles parses every file under root the predicate accepts.
func walkGoFiles(root string, visit func(*ast.File), want func(string) bool) error {
	fset := token.NewFileSet()
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		switch {
		case walkErr != nil:
			return walkErr
		case d.IsDir(), !want(path):
			return nil
		}
		parsed, parseErr := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if parseErr != nil {
			return fmt.Errorf("gate: parse %s: %w", path, parseErr)
		}
		visit(parsed)
		return nil
	})
	if err != nil {
		return fmt.Errorf("gate: walk %s: %w", root, err)
	}
	return nil
}

// UnsetHarnessFields registers every harness field no consumer test in
// the corpus sets, with the verdict that keeps its absence honest.
//
// A field here is one the product offers and the corpus never asks for.
// From outside that reads the same as a field nobody needs, which is how
// Oracle and Serial sat inert: both only mean anything in a run with more
// than one implementation, and every consumer test ran exactly one.
//
//nolint:gochecknoglobals // a debt register, read-only, test-facing.
var UnsetHarnessFields = map[string]string{}
