// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gate_test

import (
	"strings"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/gate"
)

// TestEveryShippedConstructorIsEmittedOrArgued is the dead-machinery census.
//
// The engine ships constructors for a generator to call. One nothing calls
// is either a gap nobody has noticed or a decision nobody wrote down, and
// from outside the two look identical — which is how [action.EvictingReader]
// sat unused from the day it was written until somebody happened to read
// the file, and how the delivery oracles beside it did the same.
//
// So: every constructor is emitted by the corpus, or registered here with
// the verdict that keeps its absence honest. Both directions, for the
// reason the unarmed-door census gives — an argued row for something now
// emitted is a stale excuse, and stale excuses are how a register stops
// being read.
//
// No counts. The set derives from the engine's own source and the emission
// from the bytes the corpus ships, so neither can drift from a number
// somebody remembered to update.
func TestEveryShippedConstructorIsEmittedOrArgued(t *testing.T) {
	t.Parallel()

	shipped, err := gate.ShippedConstructors("../../engine", "../corpus")
	testkit.NoError(t, err, "the constructor census runs")
	testkit.True(t, len(shipped) > 0, "the census found the engine's constructors")

	seen := map[string]bool{}
	for _, s := range shipped {
		key := s.Key()
		seen[key] = true
		reason, argued := gate.UnemittedConstructors[key]
		switch {
		case s.Emitted && argued:
			t.Errorf("%s is emitted by the corpus and still registered as absent — "+
				"delete the stale row: %s", key, reason)
		case !s.Emitted && !argued:
			t.Errorf("%s is shipped by the engine, called by no generated file, and "+
				"registered nowhere. Either a generator should emit it, or say here "+
				"why nothing does — an unexplained absence reads the same as a gap", key)
		}
	}

	for key := range gate.UnemittedConstructors {
		testkit.True(t, seen[key],
			"gate.UnemittedConstructors names "+key+", which the engine no longer "+
				"ships — delete the row with the constructor")
	}
}

// TestEveryUnemittedRowSaysWhy holds the register's prose to a shape.
//
// A row reading "not used" records the absence and none of the reasoning,
// which is what the register exists for. The bar is low on purpose — this
// cannot judge whether a reason is good — but a one-word row is one nobody
// thought about, and that is worth catching.
func TestEveryUnemittedRowSaysWhy(t *testing.T) {
	t.Parallel()

	for key, reason := range gate.UnemittedConstructors {
		testkit.True(t, len(strings.Fields(reason)) >= 8,
			"the row for "+key+" is too short to be a reason: "+reason)
	}
}
