// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package projection_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/core/lawid"
	"go.thesmos.sh/testkit/engine/suite"
	"go.thesmos.sh/testkit/generator/internal/projection"
)

// plan is one check's identity, which is all the index reads.
func plan(id projection.IDPlan) projection.CheckPlan {
	return projection.CheckPlan{ID: id}
}

// methodPlan is a method-scoped check on one segment.
func methodPlan(method, seg string) projection.CheckPlan {
	return plan(projection.IDPlan{Method: method, Seg: seg})
}

// familyPlan is a family-scoped check, qualified as the grammar
// requires.
func familyPlan(family, seg string) projection.CheckPlan {
	return plan(projection.IDPlan{Family: family, Qualifier: "log", Seg: seg})
}

// The index groups by what a consumer knows: the method they called,
// or the tier that ran.
//
// Grouping by deriver would be the emitter's own convenience and would
// leak an implementation detail into the drop surface — a consumer
// dropping ix.Get.ReportsAMiss() should not have to know that a
// detector rule rather than the signature rules licensed it.
func TestIndexGroupsByScope(t *testing.T) {
	t.Parallel()

	t.Run("two derivers landing on one method share its group", func(t *testing.T) {
		t.Parallel()

		got, err := projection.IndexOf(projection.Inventory{
			Iface: "Store", Token: "store",
			Checks: []projection.CheckPlan{
				methodPlan("Get", suite.SegSmoke),
				methodPlan("Get", suite.SegMiss),
			},
		})
		testkit.NoError(t, err, "both segments are worded")
		testkit.Len(t, got.Groups, 1, "one method, one group")
		testkit.Equal(t, got.Groups[0].Field, "Get", "the group is the method")
		testkit.Equal(t, len(got.Groups[0].Accessors), 2, "both checks index under it")
	})

	t.Run("the family groups follow the methods", func(t *testing.T) {
		t.Parallel()

		got, err := projection.IndexOf(projection.Inventory{
			Iface: "Log", Token: "log",
			Checks: []projection.CheckPlan{
				familyPlan(suite.FamilyModel, suite.SegDifferential),
				methodPlan("Append", suite.SegSmoke),
			},
		})
		testkit.NoError(t, err, "both scopes are worded")
		testkit.Len(t, got.Groups, 2, "one method group, one family group")
		testkit.Equal(t, got.Groups[0].Field, "Append", "methods lead, in emission order")
		testkit.Equal(t, got.Groups[1].Field, "Model", "the family trails")
	})
}

// The accessor is the segment's own name, Pascal-cased.
//
// Derived rather than tabulated: the segment vocabulary grows upstream,
// and a closed table would refuse a new segment instead of spelling the
// obvious identifier for it.
//
//nolint:thelper // the case body is the test, not a helper; see core/lawid
func TestSegmentAccessorsAreDerived(t *testing.T) {
	t.Parallel()

	type segCase struct {
		name string
		seg  string
		want string
	}

	testkit.TableTest(t, []segCase{
		{"a one-word segment is the word", suite.SegSmoke, "Smoke"},
		{"a kebab segment joins its parts", suite.SegZeroValue, "ZeroOnError"},
		{"a run-together segment keeps the platform's casing", suite.SegNilContext, "NilContext"},
		{"the miss segment is the runtime's own", suite.SegMiss, "Miss"},
		{"cancel is the one verb", suite.SegCancel, "Cancels"},
	}, func(t *testing.T, tc segCase) {
		got, err := projection.IndexOf(projection.Inventory{
			Iface: "Store", Token: "store",
			Checks: []projection.CheckPlan{methodPlan("Get", tc.seg)},
		})
		testkit.NoError(t, err, "the segment is worded")
		testkit.Equal(t, got.Groups[0].Accessors[0].Name, tc.want, tc.name)
	})
}

// A law-scoped accessor is the law's own spelling, and it carries the
// identifier the emitted body names the law through.
//
// Both come from lawid rather than from a table here, because they are
// facts about the law: an accessor invented beside the emitter is one
// the law's own package cannot be held to.
func TestLawAccessorsComeFromTheLawsHome(t *testing.T) {
	t.Parallel()

	got, err := projection.IndexOf(projection.Inventory{
		Iface: "Log", Token: "log",
		Checks: []projection.CheckPlan{familyPlan(suite.FamilyModel, lawid.CursorCloseIdempotent)},
	})
	testkit.NoError(t, err, "a worded law indexes")

	acc := got.Groups[0].Accessors[0]
	want, _ := lawid.AccessorOf(lawid.CursorCloseIdempotent)
	testkit.Equal(t, acc.Name, want, "the accessor is the law's own")
	testkit.Equal(t, acc.LawConst, "CursorCloseIdempotent",
		"and it carries the identifier the emitted body spells, not the AUTO- text")
}

// An unworded law is refused by name rather than indexed as prose.
//
// The refusal is the whole point: a family accessor derived from an
// AUTO- string would emit `ix.Model.AUTOSomethingOrOther()`, which
// compiles, ships, and tells the next reader nothing.
//
// Driven with a law the live registry carries and this tree has not
// worded yet, rather than an invented identifier — an invented one
// would prove only that unknown strings are refused, and the case that
// matters is the registered law that accretes ahead of its wording.
func TestIndexRefusesAnUnwordedLaw(t *testing.T) {
	t.Parallel()

	var unworded string
	for _, id := range lawid.All() {
		if _, worded := lawid.AccessorOf(id); !worded {
			unworded = id
			break
		}
	}
	if unworded == "" {
		t.Skip("every registered law is worded, so this refusal has no case; expires 2026-11-30")
	}

	_, err := projection.IndexOf(projection.Inventory{
		Iface: "Log", Token: "log",
		Checks: []projection.CheckPlan{familyPlan(suite.FamilyModel, unworded)},
	})
	testkit.Error(t, err, "an unworded law names itself in the refusal")
}

// An accessor carries the identifier its ID composes from, never the
// slug.
//
// The emitted body reads `suite.MethodID(logAppend, suite.SegSmoke)`,
// so the segment reaches the template as its constant. A literal there
// would be the one place the ID grammar has two homes.
func TestAccessorsCarryTheirSegmentConstant(t *testing.T) {
	t.Parallel()

	got, err := projection.IndexOf(projection.Inventory{
		Iface: "Log", Token: "log",
		Checks: []projection.CheckPlan{methodPlan("Append", suite.SegSmoke)},
	})
	testkit.NoError(t, err, "a declared segment indexes")
	testkit.Equal(t, got.Groups[0].Accessors[0].SegConst, "SegSmoke",
		"the accessor names the engine's constant, not the slug")
}
