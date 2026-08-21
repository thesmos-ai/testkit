// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// `deleteremoves gone=ErrGone` names the sentinel a read owes once the delete
// has run, and the row below is the sequence that reaches it: a generated check
// meets a fresh subject and has nothing deleted to read.
package deleteremovestest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/deleteremoves"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/deleteremoves/deleteremovestest"
)

// TestMixedContract runs the generated checks and this package's own.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	deleteremovestest.RunMixed(t, inMemory("in-memory"), mixedChecks)
}

// TestMixedContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	deleteremovestest.RunMixed(t,
		inMemory("in-memory"),
		deleteremovestest.MixedSuite.Without(deleteremovestest.MixedSuite.Checks.Put.Smoke()),
	)
}

// TestMixedChecksCanFail drives the row against its planted defect.
func TestMixedChecksCanFail(t *testing.T) {
	t.Parallel()

	deleteremovestest.ProveMixed(t, mixedChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) deleteremovestest.MixedHarness[*deleteremovestest.InMemory] {
	return deleteremovestest.MixedHarness[*deleteremovestest.InMemory]{
		Name: name, New: deleteremovestest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var mixedChecks = deleteremovestest.MixedChecks{
	{
		Method: "Read", Name: "delete-removes-what-put-wrote",
		Claim: "Read reports the declared sentinel once Delete has run",
		Run:   deleteRemovesWhatPutWrote,
		ProvenBy: deleteremovestest.BrokenMixed(
			"a store whose delete only forgets the value", newKeepsTheKey,
		),
		ProvenReason: "report the declared sentinel",
	},
}

// --- Bodies -------------------------------------------------------------------

func deleteRemovesWhatPutWrote(
	tb testing.TB, s deleteremoves.Mixed, fx deleteremovestest.MixedFixture,
) {
	tb.Helper()
	testkit.NoError(tb, s.Put(tb.Context(), fx.Key(), fx.Value()), "the key is written")

	got, err := s.Read(tb.Context(), fx.Key())
	testkit.NoError(tb, err, "a written key is found")
	testkit.Equal(tb, got, fx.Value(), "and carries what was written")

	testkit.NoError(tb, s.Delete(tb.Context(), fx.Key()), "the key is removed")

	_, err = s.Read(tb.Context(), fx.Key())
	testkit.ErrorIs(tb, err, deleteremoves.ErrGone,
		"and reads after it report the declared sentinel")
}

// --- Planted defects ----------------------------------------------------------

// keepsTheKey clears the value on delete and leaves the key present, so a read
// afterwards succeeds with nothing in it rather than reporting the sentinel.
// The distinction is the whole of what `gone=` declares.
type keepsTheKey struct{ held map[string]string }

func newKeepsTheKey() *keepsTheKey { return &keepsTheKey{held: map[string]string{}} }

func (k *keepsTheKey) Put(_ context.Context, key, value string) error {
	k.held[key] = value
	return nil
}

func (k *keepsTheKey) Delete(_ context.Context, key string) error {
	k.held[key] = ""
	return nil
}

func (k *keepsTheKey) Read(_ context.Context, key string) (string, error) {
	value, held := k.held[key]
	if !held {
		return "", deleteremoves.ErrGone
	}
	return value, nil
}
