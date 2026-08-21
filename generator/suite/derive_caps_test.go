// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Internal for the reason derive_stamps_test.go is: the fixtures
// populate the unexported stamp projections through the real keys on
// real bags, which only this package's own constructors reach.
package suite

import (
	"testing"

	"go.thesmos.sh/testkit"
	vocab "go.thesmos.sh/testkit/engine/suite"
	"go.thesmos.sh/testkit/generator/internal/projection"
)

func TestCapabilityDoorsFollowTheClass(t *testing.T) {
	t.Parallel()

	store := lawStore()

	t.Run("a clocked class demands the clock", func(t *testing.T) {
		t.Parallel()
		needs := capsFor(store, vocab.ClassClocked)
		testkit.Equal(t, needs, []projection.NeedPlan{{Capability: vocab.CapClock}},
			"a check that moves time cannot run on the wall clock")
	})

	t.Run("the poison class demands induction at its sentinel", func(t *testing.T) {
		t.Parallel()
		needs := capsFor(store, vocab.ClassPoison)
		testkit.Equal(
			t,
			needs,
			[]projection.NeedPlan{{Capability: vocab.CapInduce, Value: projection.Expr("kv.ErrClosed")}},
			"the door's value is the declaration that licensed the law",
		)
	})

	t.Run("an ungated class walks in unaided", func(t *testing.T) {
		t.Parallel()
		testkit.Len(t, capsFor(store, vocab.ClassLifecycle), 0,
			"the harness surface never grows on speculation")
	})

	t.Run("the derived rows carry their doors", func(t *testing.T) {
		t.Parallel()
		byID, _ := lawPlansByID(t, store)
		testkit.Equal(t, byID["model/store/AUTO-TTL-EXPIRY"].Needs,
			[]projection.NeedPlan{{Capability: vocab.CapClock}},
			"the clocked row declares the clock door")
		testkit.Equal(t, byID["model/store/AUTO-POISON-CONSISTENT"].Needs,
			[]projection.NeedPlan{{Capability: vocab.CapInduce, Value: projection.Expr("kv.ErrClosed")}},
			"the poison row declares induction at the stamped sentinel")
	})

	t.Run("the harness projects exactly the declared doors", func(t *testing.T) {
		t.Parallel()
		plans, _ := Laws{}.Derive(store)
		h := projection.HarnessOf(store.Name, plans)
		testkit.True(t, h.Clock, "a clocked row opened the clock field")
		testkit.True(t, h.Induce, "a poison row opened the induction map")
		testkit.False(t, h.Recover, "no sim row, no recover seam — the family waits on its license")
		testkit.False(t, h.Seeded, "nothing seeded, no seed constructor")
	})
}
