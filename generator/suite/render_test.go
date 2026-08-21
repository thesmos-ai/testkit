// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Internal for the reason derive_stamps_test.go is: the renderer and
// its view are the unexported seam the emitter drives.
package suite

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"text/template"

	langgo "go.thesmos.sh/eidos/lang/golang"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/internal/projection"
)

// bodyTemplates parses the body subtree exactly as the backend will:
// the prefixed function map read back from the plugin itself, so this
// parse and the run's cannot diverge. Test scaffolding on purpose —
// production rendering is the backend's, and a parse path with no
// production consumer does not belong in production code.
func bodyTemplates() (*template.Template, error) {
	t, err := template.New("body").
		Funcs(New().TemplateFuncs(langgo.Language)).
		Funcs(backendPlaceholders()).
		ParseFS(goTemplatesFS, "templates/golang/body/*.tmpl")
	if err != nil {
		return nil, fmt.Errorf("suite: parse the body templates: %w", err)
	}
	return t, nil
}

// The three smoke arms are pinned by the pipeline golden, not here.
//
// Each reaches the backend for its primitive and its context type — an
// emitted reference registers its import through renderExpr, and this
// harness cannot run one. Reimplementing them would pin bytes against a
// second import machinery rather than against the one that ships, which
// is the failure the placeholder below exists to make loud rather than
// plausible.
//
// What moved: TestGeneratedHarnessBuildsAndItsProofsRun renders the
// bodies through the real backend, compiles them, and runs them. It is
// a stronger assertion than a byte comparison and a weaker one about
// SHAPE, which is the trade — the arms a corpus fixture does not
// exercise are covered by the corpus, and the corpus gate reads every
// one of them.

func TestBodyTemplateCensus(t *testing.T) {
	t.Parallel()

	parsed, err := bodyTemplates()
	testkit.NoError(t, err, "the tree parses under its own function map")

	kinds := map[string]bool{}
	for _, k := range projection.BodyKinds() {
		kinds[string(k)] = true
	}
	defined := 0
	for _, tmpl := range parsed.Templates() {
		name := tmpl.Name()
		if !strings.HasPrefix(name, projection.BodyKindPrefix) {
			continue
		}
		defined++
		testkit.True(t, kinds[name], name+" names a registered body variant — an orphan template renders nothing")
	}
	// One direction only until the emission set completes: every
	// defined template names a variant. The equality gate — every
	// variant owns a template — arms when the last body lands, and
	// the count below is what will flip it.
	testkit.True(t, defined >= 1, "the smoke family is defined")
}

// backendPlaceholders stands in for the helpers the backend owns, so
// this harness can PARSE every body template.
//
// It cannot execute the ones that use them, and deliberately does not
// try: renderExpr and external are how an emitted reference registers
// its import, and a harness that reimplemented them would be pinning
// bytes against a second import machinery rather than against the one
// that runs. The bodies that reach for them are pinned by the pipeline
// golden instead; what this harness still answers is the census —
// every defined template names a registered variant.
func backendPlaceholders() template.FuncMap {
	invoked := func(...any) (string, error) {
		return "", errors.New("suite: a backend helper cannot run in the parse-only harness")
	}
	return template.FuncMap{
		"renderExpr":    invoked,
		"external":      invoked,
		"renderParams":  invoked,
		"renderReturns": invoked,
	}
}
