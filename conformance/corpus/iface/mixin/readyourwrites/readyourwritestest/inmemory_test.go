// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// readyourwrites is the model tier's: the law drives a generated sequence and
// checks that every read after a write sees it. The row below is the smallest
// case, and the one a single subject settles — a caller reading back the write
// it just made.
package readyourwritestest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/readyourwrites"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/readyourwrites/readyourwritestest"
)

// TestMixedContract runs the generated checks and this package's own.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	readyourwritestest.RunMixed(t, inMemory("in-memory"), mixedChecks)
}

// TestMixedContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	readyourwritestest.RunMixed(t,
		inMemory("in-memory"),
		readyourwritestest.MixedSuite.Without(readyourwritestest.MixedSuite.Checks.Store.Smoke()),
	)
}

// TestMixedChecksCanFail drives the row against its planted defect.
func TestMixedChecksCanFail(t *testing.T) {
	t.Parallel()

	readyourwritestest.ProveMixed(t, mixedChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) readyourwritestest.MixedHarness[*readyourwritestest.InMemory] {
	return readyourwritestest.MixedHarness[*readyourwritestest.InMemory]{
		Name: name, New: readyourwritestest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var mixedChecks = readyourwritestest.MixedChecks{
	{
		Method: "Get", Name: "reads-back-what-store-wrote",
		Claim: "Get returns what Store wrote",
		Run:   readsBackWhatStoreWrote,
		ProvenBy: readyourwritestest.BrokenMixed(
			"a store that does not see its own write", newDoesNotSeeItsOwnWrite,
		),
		ProvenReason: "the written key is present",
	},
}

// --- Bodies -------------------------------------------------------------------

func readsBackWhatStoreWrote(
	tb testing.TB, s readyourwrites.Mixed, fx readyourwritestest.MixedFixture,
) {
	tb.Helper()
	written := fx.Value()
	// Store answers the state it wrote beside its error, which is what
	// makes this an answering writer rather than a plain one.
	stored, err := s.Store(tb.Context(), written)
	testkit.NoError(tb, err, "the value is stored")
	testkit.Equal(tb, stored.Key, written.Key, "under the key it was given")

	got, err := s.Get(tb.Context(), written.Key)
	testkit.NoError(tb, err, "the written key is present")
	testkit.Equal(tb, got.Key, written.Key,
		"and Get answers under the key it was stored with")
}

// --- Planted defects ----------------------------------------------------------

// doesNotSeeItsOwnWrite acknowledges the write and answers a miss for it, which
// is the mixin's own failure: a read routed somewhere the write has not
// reached. The acknowledgement is correct, so nothing before the read can tell.
type doesNotSeeItsOwnWrite struct{}

func newDoesNotSeeItsOwnWrite() doesNotSeeItsOwnWrite { return doesNotSeeItsOwnWrite{} }

func (doesNotSeeItsOwnWrite) Store(
	_ context.Context, v readyourwrites.Value,
) (readyourwrites.Value, error) {
	return v, nil
}

func (doesNotSeeItsOwnWrite) Get(
	_ context.Context, _ string,
) (readyourwrites.Value, error) {
	return readyourwrites.Value{}, readyourwritestest.ErrNotFound
}

// TestMixedLawsCanSaturate drives each bound law against defects worn on
// its own methods, with that law as the run's only oracle.
//
// Binding a law is necessary; this is what makes it sufficient. A law
// every worn defect survives is bound and unsaturatable, which reads as
// coverage in the report and is not.
func TestMixedLawsCanSaturate(t *testing.T) {
	t.Parallel()

	readyourwritestest.MixedModelSaturation(t, func() readyourwritestest.Mixed { return readyourwritestest.NewInMemory() })
}
