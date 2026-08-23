// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// `snapshotisolation` is the model tier's: an anomaly is a claim about
// concurrent transactions, which needs generated interleavings. The row below
// is what makes those readable at all — that a history handed to a caller is a
// copy.
package snapshotisolationtest_test

import (
	"context"
	"slices"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/snapshotisolation"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/snapshotisolation/snapshotisolationtest"
	"go.thesmos.sh/testkit/engine/model"
	"go.thesmos.sh/testkit/engine/model/law"
	"go.thesmos.sh/testkit/engine/suite"
)

// TestMixedContract runs the generated checks and this package's own.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	snapshotisolationtest.RunMixed(t, inMemory("in-memory"), mixedChecks)
}

// TestMixedContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	snapshotisolationtest.RunMixed(t,
		inMemory("in-memory"),
		snapshotisolationtest.MixedSuite.Without(snapshotisolationtest.MixedSuite.Checks.Record.Smoke()),
	)
}

// TestMixedChecksCanFail drives the row against its planted defect.
func TestMixedChecksCanFail(t *testing.T) {
	t.Parallel()

	snapshotisolationtest.ProveMixed(t, inMemory("in-memory"), mixedChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) snapshotisolationtest.MixedHarness[*snapshotisolationtest.InMemory] {
	return snapshotisolationtest.MixedHarness[*snapshotisolationtest.InMemory]{
		Name: name, New: snapshotisolationtest.NewInMemory,
		// The anomaly checks read transactions, and this interface reports
		// operations. Which field groups them, which one says read from
		// write and which carries the version, is what the declaration
		// knows and the shape does not.
		Provide: map[suite.Capability]any{"history": transactions},
	}
}

// transactions folds the recorded operations into the transactions the
// anomaly checks walk.
//
// Grouped by Txn and ordered by it, because the checkers index into the
// slice and a map's order would make one iteration's verdict differ from
// the next on the same history. Nothing is marked aborted: this interface
// records no outcome, so every recorded transaction reads as committed —
// which is the stricter reading, since an aborted transaction contributes
// no edges and could only make an anomaly disappear.
func transactions(rt *model.T, s snapshotisolation.Mixed) []law.Txn[string] {
	entries, err := s.History(rt.Context())
	if err != nil {
		return nil
	}
	byTxn := map[int]*law.Txn[string]{}
	var ids []int
	for _, e := range entries {
		t, seen := byTxn[e.Txn]
		if !seen {
			t = &law.Txn[string]{ID: e.Txn}
			byTxn[e.Txn] = t
			ids = append(ids, e.Txn)
		}
		op := law.TxnOp[string]{Key: e.Key, Version: e.Version}
		if e.Write {
			t.Writes = append(t.Writes, op)
			continue
		}
		t.Reads = append(t.Reads, op)
	}
	slices.Sort(ids)
	out := make([]law.Txn[string], 0, len(ids))
	for _, id := range ids {
		out = append(out, *byTxn[id])
	}
	return out
}

// --- The checks: claims, bodies and defects, by name --------------------------

var mixedChecks = snapshotisolationtest.MixedChecks{
	{
		Method: "History", Name: "hands-back-a-copy",
		Claim: "History hands back a copy, not the backing array",
		Run:   handsBackACopy,
		ProvenBy: snapshotisolationtest.BrokenMixed(
			"a subject that hands out its own backing array", newSharesItsBacking,
		),
		ProvenReason: "the caller's edit did not reach the subject",
	},
}

// --- Bodies -------------------------------------------------------------------

// handsBackACopy mutates what History returned and asks again.
//
// The edit must not reach the subject: an anomaly check walks this history
// while the subject may still be recording, and a shared backing array would
// let the checker corrupt the thing it is checking.
func handsBackACopy(
	tb testing.TB, s snapshotisolation.Mixed, fx snapshotisolationtest.MixedFixture,
) {
	tb.Helper()
	testkit.NoError(tb, s.Record(tb.Context(), fx.Entry()), "an entry is recorded")

	got, err := s.History(tb.Context())
	testkit.NoError(tb, err, "the history is readable")
	testkit.Equal(tb, len(got), 1, "the recorded entry is there")

	got[0].Txn = tamperedTxn
	again, err := s.History(tb.Context())
	testkit.NoError(tb, err, "the history is still readable")
	testkit.NotEqual(tb, again[0].Txn, tamperedTxn,
		"the caller's edit did not reach the subject")
}

// --- Planted defects ----------------------------------------------------------

// tamperedTxn is the value the row writes into the slice it was handed. Its
// number does not matter, only that no correct subject would ever answer it.
const tamperedTxn = 99

// sharesItsBacking returns the slice it stores rather than a copy, which is
// what every Go collection does unless somebody decided otherwise — and which
// no check that only READS the history can see.
type sharesItsBacking struct{ history []snapshotisolation.Entry }

func newSharesItsBacking() *sharesItsBacking { return &sharesItsBacking{} }

func (s *sharesItsBacking) Record(_ context.Context, e snapshotisolation.Entry) error {
	s.history = append(s.history, e)
	return nil
}

func (s *sharesItsBacking) History(context.Context) ([]snapshotisolation.Entry, error) {
	return s.history, nil
}

func (s *sharesItsBacking) Get(
	_ context.Context, key string,
) (snapshotisolation.Entry, error) {
	at := slices.IndexFunc(s.history, func(e snapshotisolation.Entry) bool {
		return e.Key == key
	})
	if at < 0 {
		return snapshotisolation.Entry{}, snapshotisolationtest.ErrNotFound
	}
	return s.history[at], nil
}
