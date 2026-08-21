// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Internal for the reason derive_stamps_test.go is: the fixtures
// populate the unexported stamp projections through the real keys on
// real bags, which only this package's own constructors reach.
package suite

import (
	"testing"

	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/writer"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/core/lawid"
	vocab "go.thesmos.sh/testkit/engine/suite"
	"go.thesmos.sh/testkit/generator/internal/projection"
	"go.thesmos.sh/testkit/generator/internal/subject"
)

func TestLawDefectRulesPlantFromTheStamps(t *testing.T) {
	t.Parallel()

	store := lawStore()

	t.Run("W1: the discarded write proves every agreement row", func(t *testing.T) {
		t.Parallel()
		writerStore := Iface{
			Name:      "Store",
			Token:     "store",
			Qualifier: "store",
			Methods:   []subject.Method{stampMethod("Put", writer.Name)},
		}
		defect, proven := observationDefect(writerStore)
		testkit.True(t, proven, "a writer exists to acknowledge and drop")
		testkit.Equal(
			t,
			defect,
			projection.Defect(projection.AnswersAnyway{
				Clause: projection.Clause{Text: "Put reports success and keeps nothing"},
				Option: projection.OptionName("Store", "Put"),
			}),
			"the defect rides the writer's own stub option",
		)
	})

	t.Run("P1: the poison heals through the stamped sentinel", func(t *testing.T) {
		t.Parallel()
		defect, proven := lawDefect(store, lawid.PoisonConsistent)
		testkit.True(t, proven, "the sentinel that licensed the law plants its defect")
		testkit.Equal(t, defect, projection.Defect(projection.SentinelOnce{Sentinel: projection.Expr("kv.ErrClosed")}),
			"the healed sentinel is the declared one, never another")
	})

	t.Run("the after-close defect outlives through a stamped method", func(t *testing.T) {
		t.Parallel()
		defect, proven := lawDefect(store, lawid.LifecycleAfterClose)
		testkit.True(t, proven, "a stamped carrier supplies the outliving method")
		testkit.Equal(
			t,
			defect,
			projection.Defect(projection.PartialOutlive{Option: projection.OptionName("Store", "Put")}),
			"one method kept alive past Close, by its stub option",
		)
	})

	t.Run("the residue has no rule and says so", func(t *testing.T) {
		t.Parallel()
		_, proven := lawDefect(store, lawid.CursorCloseIdempotent)
		testkit.False(t, proven, "a domain-composite defect ships Argued, never a proof nobody derived")
	})

	t.Run("no writer, no observation defect", func(t *testing.T) {
		t.Parallel()
		_, proven := observationDefect(Iface{Name: "Catalog", Token: "catalog", Qualifier: "catalog"})
		testkit.False(t, proven, "nothing writes, nothing can be discarded")
	})
}

func TestProofRulesFlipTheRows(t *testing.T) {
	t.Parallel()

	byID, _ := lawPlansByID(t, lawStore())

	t.Run("a ruled law row is Proven with its defect", func(t *testing.T) {
		t.Parallel()
		p := byID["model/store/AUTO-LIFECYCLE-AFTER-CLOSE"]
		testkit.Equal(t, p.Falsifiable.State, vocab.FalsifiableProven, "the mechanical rule proves the row")
		testkit.Equal(
			t,
			p.Defect,
			projection.Defect(projection.PartialOutlive{Option: projection.OptionName("Store", "Put")}),
			"and the defect is the rule's",
		)
	})

	t.Run("an unruled row stays Argued with the pending reason", func(t *testing.T) {
		t.Parallel()
		p := byID["model/store/AUTO-TTL-EXPIRY"]
		testkit.Equal(t, p.Falsifiable.State, vocab.FalsifiableArgued,
			"the ttl proof (F1 strip-role-field) waits on its defect variant joining the closed set")
		testkit.Equal(t, p.Defect, nil, "no defect nobody derived")
	})

	t.Run("the linearizable row rides the discarded write", func(t *testing.T) {
		t.Parallel()
		p := byID["model/store/AUTO-LINEARIZABLE"]
		testkit.Equal(t, p.Falsifiable.State, vocab.FalsifiableProven, "an agreement row with a writer proves by W1")
	})
}
