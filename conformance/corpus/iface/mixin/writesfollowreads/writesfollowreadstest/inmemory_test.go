// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// `writesfollowreads` is the model tier's: that a write lands after everything
// the same session read is a claim about an interleaving, which needs a
// generated history. The row below is what one write and one read settle.
package writesfollowreadstest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/writesfollowreads"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/writesfollowreads/writesfollowreadstest"
)

// TestMixedContract runs the generated checks and this package's own.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	writesfollowreadstest.RunMixed(t, inMemory("in-memory"), mixedChecks)
}

// TestMixedContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	writesfollowreadstest.RunMixed(t,
		inMemory("in-memory"),
		writesfollowreadstest.MixedSuite.Without(writesfollowreadstest.MixedSuite.Checks.Store.Smoke()),
	)
}

// TestMixedChecksCanFail drives the row against its planted defect.
func TestMixedChecksCanFail(t *testing.T) {
	t.Parallel()

	writesfollowreadstest.ProveMixed(t, mixedChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) writesfollowreadstest.MixedHarness[*writesfollowreadstest.InMemory] {
	return writesfollowreadstest.MixedHarness[*writesfollowreadstest.InMemory]{
		Name: name, New: writesfollowreadstest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var mixedChecks = writesfollowreadstest.MixedChecks{
	{
		Method: "Get", Name: "reads-back-what-store-wrote",
		Claim: "Get returns what Store wrote",
		Run:   readsBackWhatStoreWrote,
		ProvenBy: writesfollowreadstest.BrokenMixed(
			"a store that acknowledges a write it never applied", newAcknowledgesWithoutWriting,
		),
		ProvenReason: "the written key is present",
	},
}

// --- Bodies -------------------------------------------------------------------

func readsBackWhatStoreWrote(
	tb testing.TB, s writesfollowreads.Mixed, fx writesfollowreadstest.MixedFixture,
) {
	tb.Helper()
	written := fx.Value()
	// Store answers the state it wrote beside its error.
	stored, err := s.Store(tb.Context(), written)
	testkit.NoError(tb, err, "the value is stored")
	testkit.Equal(tb, stored.Key, written.Key, "under the key it was given")

	got, err := s.Get(tb.Context(), written.Key)
	testkit.NoError(tb, err, "the written key is present")
	testkit.Equal(tb, got.Key, written.Key,
		"and Get answers under the key it was stored with")
}

// --- Planted defects ----------------------------------------------------------

// acknowledgesWithoutWriting answers a correct receipt and keeps nothing, which
// is a write buffered somewhere that never flushed. The acknowledgement is the
// only thing a caller sees before the read, so nothing earlier can tell.
type acknowledgesWithoutWriting struct{}

func newAcknowledgesWithoutWriting() acknowledgesWithoutWriting {
	return acknowledgesWithoutWriting{}
}

func (acknowledgesWithoutWriting) Store(
	_ context.Context, v writesfollowreads.Value,
) (writesfollowreads.Value, error) {
	return v, nil
}

func (acknowledgesWithoutWriting) Get(
	_ context.Context, _ string,
) (writesfollowreads.Value, error) {
	return writesfollowreads.Value{}, writesfollowreadstest.ErrNotFound
}
