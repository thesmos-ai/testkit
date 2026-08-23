// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Internal because the template tree is embedded unexported, and the
// census's whole point is to read the templates that actually ship
// rather than a second list of them.
package model

import (
	"errors"
	"sort"
	"strings"
	"testing"
	"text/template"

	langgo "go.thesmos.sh/eidos/lang/golang"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/internal/projection"
)

// legTemplates parses the leg subtree under the plugin's own function
// map, so this parse and a generation run's cannot diverge.
//
// Test scaffolding on purpose: production rendering is the backend's,
// and a second parse path with no production consumer would be a second
// thing to keep in step.
func legTemplates(t *testing.T) *template.Template {
	t.Helper()

	parsed, err := template.New("legs").
		Funcs(New().TemplateFuncs(langgo.Language)).
		Funcs(legPlaceholders()).
		ParseFS(goTemplatesFS, "templates/golang/*.tmpl")
	testkit.NoError(t, err, "the leg templates parse under their own function map")
	return parsed
}

// legPlaceholders stands in for the helpers the backend owns, so this
// harness can PARSE every leg template.
//
// It cannot execute them and does not try: renderExpr and external are
// how an emitted reference registers its import, and reimplementing them
// here would pin the census against a second import machinery rather
// than the one that ships. What this harness answers is the census, and
// a name is all a census needs.
func legPlaceholders() template.FuncMap {
	invoked := func(...any) (string, error) {
		return "", errors.New("model: a backend helper cannot run in the parse-only harness")
	}
	return template.FuncMap{
		"lower":            invoked,
		"renderTypeParams": invoked,
		"renderExpr":       invoked,
		"external":         invoked,
		"render":           invoked,
		"renderType":       invoked,
		"renderParams":     invoked,
		"renderReturns":    invoked,
	}
}

// Every emit kind this tier's dispatch can produce has a template that
// spells it.
//
// The composition root holds RowKinds from both tiers against the
// projection's declared set, which catches a kind NEITHER tier claims.
// It cannot catch a kind this tier claims and renders nothing for — a
// false claim and a true one look the same from there, and the root's
// census reads as coverage either way. This is the direction that was
// missing: the claim now has to be backed by a template in the shipped
// tree.
//
// One direction only. The reverse — every leg template is named by some
// dispatch — is not stated here because the bodies and the fragments
// they compose from share the prefix, and a rule that told them apart by
// name would be guessing. A fragment nothing includes fails the render
// by name, which is where that half already lives.
func TestEveryLegKindHasATemplate(t *testing.T) {
	t.Parallel()

	parsed := legTemplates(t)
	defined := map[string]bool{}
	for _, tmpl := range parsed.Templates() {
		if name := tmpl.Name(); strings.HasPrefix(name, "model.leg.") {
			defined[name] = true
		}
	}

	var missing []string
	for body, kinds := range LegTemplates() {
		testkit.Contains(t, projection.BodyKinds(), body,
			"this tier claims to render "+string(body)+", which projection does not declare")
		testkit.True(t, len(kinds) > 0,
			"body kind "+string(body)+" is claimed with no emit kind, so its rows render nothing")
		for _, k := range kinds {
			if !defined[string(k)] {
				missing = append(missing, string(k)+" (for "+string(body)+")")
			}
		}
	}

	sort.Strings(missing)
	testkit.Len(t, missing, 0,
		"dispatched with no template to spell it, so the row renders nothing while the "+
			"root's census still reads this tier as covering its body kind: "+
			strings.Join(missing, ", "))
}
