// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gate_test

import (
	"path/filepath"
	"strings"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/gate"
)

// TestEveryAdapterDebtIsRealAndRegistered holds the language axis to the
// register in both directions: a lang fixture without a model tier must
// argue its absence in gate.AdapterDebt, and a fixture that gained one must
// delete its row — a stale excuse reads as debt that was already paid.
func TestEveryAdapterDebtIsRealAndRegistered(t *testing.T) {
	t.Parallel()

	pkgs, err := gate.Walk(corpusRoot)
	testkit.NoError(t, err, "the corpus walk succeeds")

	seen := map[string]bool{}
	for _, p := range pkgs {
		if p.Axis != "lang" || strings.HasSuffix(p.Name, "test") {
			// The <name>test packages are the fixtures' outputs, not
			// fixtures of their own.
			continue
		}
		seen[p.Name] = true
		matches, gErr := filepath.Glob(filepath.Join(corpusRoot, p.Dir, "*", "iface_model.gen.go"))
		testkit.NoError(t, gErr, "the fixture's generated output is listable")
		modeled := len(matches) > 0
		reason, argued := gate.AdapterDebt[p.Name]
		switch {
		case modeled && argued:
			t.Errorf("%s carries a model tier and is still registered — "+
				"the debt was paid, delete the row: %s", p.Name, reason)
		case !modeled && !argued:
			t.Errorf("%s has no model tier and no row in gate.AdapterDebt — "+
				"an absence without a verdict is a judgment nobody made", p.Name)
		}
	}

	for name := range gate.AdapterDebt {
		testkit.True(t, seen[name],
			name+" is registered but corpus/iface/lang holds no such fixture — the row outlived its debt")
	}
	// The walk found the axis at all: a rename that emptied it would
	// otherwise pass every row check vacuously.
	testkit.True(t, len(seen) > 0, "the language axis exists")
}
