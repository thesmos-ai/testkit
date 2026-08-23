// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model_test

import (
	"io/fs"
	"strings"
	"testing"

	"go.thesmos.sh/eidos/lang/golang"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/core/tiers"
	"go.thesmos.sh/testkit/generator/model"
)

// Layout composes every filename from a declared suffix and reads the language
// off the backend, so a plugin keyed on anything but eidos's own constant
// answers nothing for every real run.
func TestGoOutputs(t *testing.T) {
	t.Parallel()

	t.Run("declares the bindings and their proof", func(t *testing.T) {
		t.Parallel()
		testkit.Len(t, model.GoOutputs(), 2, "the bindings and the companion")
		testkit.Equal(t, model.GoOutputs()[0].Tag, "", "the empty tag is the primary")
	})

	t.Run("gives the companion the test suffix Layout keys on", func(t *testing.T) {
		t.Parallel()
		// The `_test.go` ending earns the external-test-package shift, which
		// is what makes the companion drive the bindings across the package
		// boundary the way a consumer does — and why the reference
		// constructor it drives is exported.
		testkit.Assert(t, model.GoTestSuffix).HasSuffix("_test.go",
			"the shift is keyed on the suffix, not the tag")
	})

	t.Run("answers nothing for another language", func(t *testing.T) {
		t.Parallel()
		testkit.Assert(t, model.New().Outputs("rust")).IsEmpty("only Go is served")
	})
}

// A template the backend cannot find renders nothing and fails nowhere, so the
// file simply comes out short.
func TestGoTemplates(t *testing.T) {
	t.Parallel()

	t.Run("ships the bindings template", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, hasTemplate(t, "model.bindings.tmpl"), "the file's spine must ship")
	})

	t.Run("ships a template per shape the corpus arms", func(t *testing.T) {
		t.Parallel()
		// Not one per detector: an action for a shape with no template renders
		// nothing, the actions list gains a bare comma, and the corpus fails to
		// compile — loudly. The set grows fixture by fixture; this pins that
		// what exists keeps existing.
		for _, name := range []string{
			"action/model.action.reader.tmpl",
			"action/model.action.writer.tmpl",
			"action/model.action.aggregator.tmpl",
			"action/model.action.lifecycle.tmpl",
			"action/model.action.pure.tmpl",
			"action/model.action.predicate.tmpl",
		} {
			testkit.True(t, hasTemplate(t, name), name+" must ship")
		}
	})

	t.Run("every shipped action template names a catalogued shape", func(t *testing.T) {
		t.Parallel()
		// The template's name is the dispatch key: `model.action.<shape>`. One
		// spelled outside the detector vocabulary can never be selected, which
		// is a template that silently stopped shipping.
		for _, name := range templateNames(t) {
			rest, matched := strings.CutPrefix(name, "action/model.action.")
			if !matched {
				continue
			}
			shape := strings.TrimSuffix(rest, ".tmpl")
			_, known := tiers.ActionFor(shape)
			// The pseudo-spellings are the generator's own refinements — the
			// slice-returning aggregator drains, a parameterised pure or
			// predicate call draws its arguments, and the two-phase composite
			// is the contract-role re-point that consumes its terminal
			// siblings — each selected beside the collector precedent.
			pseudo := map[string]bool{
				tiers.ShapeCollector: true,
				"purevar":            true,
				"predicatevar":       true,
				"twophase":           true,
				// The two-role cycle is the other contract-role re-point
				// that consumes its sibling: the pool's get takes a value
				// and its put hands it back, and the claims are about the
				// round trip rather than either call.
				"cycle": true,
			}
			testkit.True(t, known || pseudo[shape],
				name+" names the "+shape+" shape, which the catalogue drives")
		}
	})
}

// hasTemplate reports whether the named template ships in the plugin's Go tree.
func hasTemplate(t *testing.T, name string) bool {
	t.Helper()
	tree, ok := model.New().Templates(golang.Language)
	if !ok {
		t.Fatal("the plugin reports no Go template tree")
	}
	_, err := fs.Stat(tree, name)
	return err == nil
}

// templateNames returns every template the tree carries, at any depth.
func templateNames(t *testing.T) []string {
	t.Helper()
	tree, ok := model.New().Templates(golang.Language)
	if !ok {
		t.Fatal("the plugin reports no Go template tree")
	}
	var out []string
	if err := fs.WalkDir(tree, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			out = append(out, p)
		}
		return nil
	}); err != nil {
		t.Fatalf("walking the template tree: %v", err)
	}
	return out
}
