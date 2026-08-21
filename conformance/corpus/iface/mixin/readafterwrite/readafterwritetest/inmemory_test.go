// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// readafterwrite is the model tier's: the law compares a read against the write
// that preceded it across a generated sequence. The row below is the smallest
// case of it — one write, one read — which is what a single subject settles.
package readafterwritetest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/readafterwrite"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/readafterwrite/readafterwritetest"
)

// TestMixedContract runs the generated checks and this package's own.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	readafterwritetest.RunMixed(t, inMemory("in-memory"), mixedChecks)
}

// TestMixedContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	readafterwritetest.RunMixed(t,
		inMemory("in-memory"),
		readafterwritetest.MixedSuite.Without(readafterwritetest.MixedSuite.Checks.Write.Smoke()),
	)
}

// TestMixedChecksCanFail drives the row against its planted defect.
func TestMixedChecksCanFail(t *testing.T) {
	t.Parallel()

	readafterwritetest.ProveMixed(t, mixedChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) readafterwritetest.MixedHarness[*readafterwritetest.InMemory] {
	return readafterwritetest.MixedHarness[*readafterwritetest.InMemory]{
		Name: name, New: readafterwritetest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var mixedChecks = readafterwritetest.MixedChecks{
	{
		Method: "Read", Name: "reads-back-what-write-wrote",
		Claim: "Read returns what Write wrote",
		Run:   readsBackWhatWriteWrote,
		ProvenBy: readafterwritetest.BrokenMixed(
			"a store whose reads lag one write behind", newLagsOneWrite,
		),
		ProvenReason: "carries what was written",
	},
}

// --- Bodies -------------------------------------------------------------------

func readsBackWhatWriteWrote(
	tb testing.TB, s readafterwrite.Mixed, fx readafterwritetest.MixedFixture,
) {
	tb.Helper()
	testkit.NoError(tb, s.Write(tb.Context(), fx.Key(), fx.Value()), "the key is written")

	got, err := s.Read(tb.Context(), fx.Key())
	testkit.NoError(tb, err, "a written key is found")
	testkit.Equal(tb, got, fx.Value(), "and carries what was written")
}

// --- Planted defects ----------------------------------------------------------

// lagsOneWrite acknowledges the write and serves the value it held before it,
// which is a replica read that went to a follower. It is the honest near miss
// for this mixin: the key exists, the read succeeds, and the answer is stale.
type lagsOneWrite struct {
	committed map[string]string
	pending   map[string]string
}

func newLagsOneWrite() *lagsOneWrite {
	return &lagsOneWrite{committed: map[string]string{}, pending: map[string]string{}}
}

func (l *lagsOneWrite) Write(_ context.Context, key, value string) error {
	if held, ok := l.pending[key]; ok {
		l.committed[key] = held
	}
	l.pending[key] = value
	return nil
}

func (l *lagsOneWrite) Read(_ context.Context, key string) (string, error) {
	if _, ever := l.pending[key]; !ever {
		return "", readafterwritetest.ErrNotFound
	}
	return l.committed[key], nil
}

// TestMixedLawsCanSaturate drives each bound law against defects worn on
// its own methods, with the law as the run's only oracle.
//
// Binding a law is necessary; this is what makes it sufficient. A law
// every worn defect survives is bound and unsaturatable, which reads as
// coverage in the report and is not.
func TestMixedLawsCanSaturate(t *testing.T) {
	t.Parallel()

	readafterwritetest.MixedModelSaturation(t, func() readafterwritetest.Mixed {
		return readafterwritetest.NewInMemory()
	})
}
