// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// An embed from outside the run is still part of the contract.
//
// Close has no declaration in embeddedforeign — it arrives from io.Closer,
// projected off the type-checker — so a run that flattened only the source
// would hold an implementation to Read alone and report success. The proof is
// that the harness runs Close/smoke at all, which is why this file names no
// check for Close itself: the generated one is the claim.
package embeddedforeigntest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/embeddedforeign"
	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/embeddedforeign/embeddedforeigntest"
)

// TestStreamContract runs the generated checks and this package's own.
func TestStreamContract(t *testing.T) {
	t.Parallel()

	embeddedforeigntest.RunStream(t, inMemory("in-memory"), streamChecks)
}

// TestStreamContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestStreamContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	embeddedforeigntest.RunStream(t,
		inMemory("in-memory"),
		embeddedforeigntest.StreamSuite.Without(embeddedforeigntest.StreamSuite.Checks.Close.Smoke()),
	)
}

// TestStreamChecksCanFail drives the row against its planted defect.
func TestStreamChecksCanFail(t *testing.T) {
	t.Parallel()

	embeddedforeigntest.ProveStream(t, inMemory("in-memory"), streamChecks)
}

// --- Harnesses ---------------------------------------------------------------

// streamedBody is what the seeded key holds.
const streamedBody = "streamed"

// inMemory seeds the subject: Stream declares no writer, so the reader's hit
// path is unreachable without a seeded constructor.
func inMemory(name string) embeddedforeigntest.StreamHarness[*embeddedforeigntest.InMemory] {
	return embeddedforeigntest.StreamHarness[*embeddedforeigntest.InMemory]{
		Name: name, New: seeded,
	}
}

func seeded() *embeddedforeigntest.InMemory {
	s := embeddedforeigntest.NewInMemory()
	s.Put(embeddedforeigntest.DefaultStreamFixture().Key(), streamedBody)
	return s
}

// --- The checks: claims, bodies and defects, by name --------------------------

var streamChecks = embeddedforeigntest.StreamChecks{
	{
		Method: "Read", Name: "returns-what-was-seeded",
		Claim: "Read returns what was seeded",
		Run:   returnsWhatWasSeeded,
		ProvenBy: embeddedforeigntest.BrokenStream(
			"a stream that answers the zero for a key it holds", newAnswersTheZero,
		),
		ProvenReason: "carries what was written",
	},
}

// --- Bodies -------------------------------------------------------------------

func returnsWhatWasSeeded(
	tb testing.TB, s embeddedforeign.Stream, fx embeddedforeigntest.StreamFixture,
) {
	tb.Helper()
	got, err := s.Read(tb.Context(), fx.Key())
	testkit.NoError(tb, err, "a seeded key is found")
	testkit.Equal(tb, got, streamedBody, "and carries what was written")
}

// --- Planted defects ----------------------------------------------------------

// answersTheZero finds the key and hands back nothing, which a check reading
// only the error takes for a real record.
//
// It has to implement Close too, and that is the fixture's point: the embed
// from io.Closer is part of the contract, so a defect standing in for a subject
// owes it as much as the real one does.
type answersTheZero struct{ key string }

func newAnswersTheZero() *answersTheZero {
	return &answersTheZero{key: embeddedforeigntest.DefaultStreamFixture().Key()}
}

func (a *answersTheZero) Read(_ context.Context, key string) (string, error) {
	if key != a.key {
		return "", embeddedforeigntest.ErrNotFound
	}
	return "", nil
}

func (*answersTheZero) Close() error { return nil }
