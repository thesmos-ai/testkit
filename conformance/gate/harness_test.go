// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gate_test

import (
	"strings"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/gate"
)

// TestEveryHarnessFieldIsExercisedOrArgued holds the consumer surface to
// the corpus.
//
// The harness is what a consumer fills in, and every field on it is a
// capability the product claims to offer. One no corpus test sets is
// offered and never exercised — the same silence the constructor census
// breaks, one layer up, and the one that let Oracle and Serial ship inert
// because every consumer test ran a single implementation.
//
// Both directions, for the reason the other censuses give: a registered
// row for a field now set is a stale excuse, and stale excuses are how a
// register stops being read.
func TestEveryHarnessFieldIsExercisedOrArgued(t *testing.T) {
	t.Parallel()

	fields, err := gate.HarnessFields("../corpus")
	testkit.NoError(t, err, "the harness census runs")
	testkit.True(t, len(fields) > 0, "the census found the generated harness fields")

	seen := map[string]bool{}
	for _, f := range fields {
		seen[f.Name] = true
		reason, argued := gate.UnsetHarnessFields[f.Name]
		switch {
		case f.Set && argued:
			t.Errorf("%s is set by a consumer test and still registered as unset — "+
				"delete the stale row: %s", f.Name, reason)
		case !f.Set && !argued:
			t.Errorf("the harness offers %s and no consumer test in the corpus sets it. "+
				"Either exercise it from a fixture, or say here why nothing does — a "+
				"capability nobody asks for reads the same as one nobody needs", f.Name)
		}
	}

	for name := range gate.UnsetHarnessFields {
		testkit.True(t, seen[name],
			"gate.UnsetHarnessFields names "+name+", which no generated harness declares — "+
				"delete the row with the field")
	}
}

// TestEveryUnsetHarnessRowSaysWhy holds the register's prose to a shape,
// for the reason its sibling over the constructors does: a one-word row
// is one nobody thought about.
func TestEveryUnsetHarnessRowSaysWhy(t *testing.T) {
	t.Parallel()

	for name, reason := range gate.UnsetHarnessFields {
		testkit.True(t, len(strings.Fields(reason)) >= 8,
			"the row for "+name+" is too short to be a reason: "+reason)
	}
}
