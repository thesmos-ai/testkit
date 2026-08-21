// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	vocab "go.thesmos.sh/testkit/engine/suite"
	"go.thesmos.sh/testkit/generator/internal/projection"
	"go.thesmos.sh/testkit/generator/model"
)

// A row carries everything the harness generator's own rows carry.
//
// They land in one []suite.Check and a consumer reading the report
// cannot tell which generator wrote which, so a row that carried less
// would be a row the runner treats differently.
func TestCheckRowsCarryTheWholePlan(t *testing.T) {
	t.Parallel()

	rows := model.CheckRows("mixed", []projection.CheckPlan{{
		ID:          projection.IDPlan{Family: vocab.FamilyModel, Seg: "ttl-expiry"},
		Class:       vocab.ClassClocked,
		Claim:       "an entry stops being readable once its lifetime has run out",
		Falsifiable: vocab.Proven(),
		Needs:       []projection.NeedPlan{{Capability: vocab.CapClock}},
	}})

	testkit.Len(t, rows, 1, "one row per plan")
	r := rows[0]
	testkit.Equal(t, r.Accessor, "TTLExpiry", "the index method, family fixed and segment varying")
	testkit.Equal(t, r.AssertName, "mixedAssertTTLExpiry", "the token and the segment")
	testkit.Equal(t, r.Claim, "an entry stops being readable once its lifetime has run out",
		"carried verbatim — the plan's claim is what the census measured")
	testkit.True(t, r.Proven, "a plan with a defect behind it may be stamped Proven")
	testkit.Equal(t, r.NeedsCtor, "NeedsClock", "and the capability it demands of the harness")
}

// A capability this tier cannot spell is spelled as none.
//
// The row names a runtime constructor, and the runtime is where the
// pairing of a capability with its value is decided. A plan carrying a
// capability with no constructor here would otherwise render as the
// nearest one — a row demanding a clock where its law wanted a poisoned
// subject, which fails naming the wrong field.
func TestCheckRowsSpellOnlyTheCapabilitiesTheyKnow(t *testing.T) {
	t.Parallel()

	rows := model.CheckRows("mixed", []projection.CheckPlan{{
		ID:    projection.IDPlan{Family: vocab.FamilyModel, Seg: "poison-consistent"},
		Class: vocab.ClassPoison,
		Needs: []projection.NeedPlan{{Capability: vocab.CapInduce, Value: "kv.ErrClosed"}},
	}})

	testkit.Equal(t, rows[0].NeedsCtor, "",
		"an induction has no constructor here, so the row demands nothing "+
			"rather than demanding the wrong thing")
}

// An argued row is not stamped proven.
func TestCheckRowsKeepTheFalsifiabilityVerdict(t *testing.T) {
	t.Parallel()

	rows := model.CheckRows("mixed", []projection.CheckPlan{{
		ID:          projection.IDPlan{Family: vocab.FamilyModel, Seg: "laws"},
		Falsifiable: vocab.Argued("no defect template plants this yet"),
	}})

	testkit.False(t, rows[0].Proven,
		"claiming proof without the evidence is refused, in both directions")
}

// Nothing is dropped, because the index names every row this tier owns.
func TestCheckRowsDropNothing(t *testing.T) {
	t.Parallel()

	plans := []projection.CheckPlan{
		{ID: projection.IDPlan{Family: vocab.FamilyModel, Seg: "laws"}},
		{ID: projection.IDPlan{Family: vocab.FamilyModel, Seg: "agrees"}},
		{ID: projection.IDPlan{Family: vocab.FamilySim, Seg: "recovery"}},
	}

	rows := model.CheckRows("mixed", plans)
	testkit.Len(t, rows, len(plans),
		"a plan dropped here leaves the index naming a check nothing emits")
	testkit.Equal(t, rows[2].AssertName, "mixedAssertRecovery", "the sim family is spelled the same way")

	testkit.Len(t, model.CheckRows("mixed", nil), 0, "and no plans is no rows")
}
