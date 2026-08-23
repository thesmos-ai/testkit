// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// The plain reader shape: a keyed read with an error slot, which is the one
// signature the rules have most to say about. What is left for the row is the
// hit — nothing on this interface writes, so only a seeding consumer can put
// something there to find.
package readertest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/reader"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/reader/readertest"
)

// TestReaderContract runs the generated checks and this package's own.
func TestReaderContract(t *testing.T) {
	t.Parallel()

	readertest.RunReader(t, inMemory("in-memory"), readerChecks)
}

// TestReaderContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestReaderContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	readertest.RunReader(t,
		inMemory("in-memory"),
		readertest.ReaderSuite.Without(readertest.ReaderSuite.Checks.Get.Smoke()),
	)
}

// TestReaderChecksCanFail drives the row against its planted defect.
func TestReaderChecksCanFail(t *testing.T) {
	t.Parallel()

	readertest.ProveReader(t, inMemory("in-memory"), readerChecks)
}

// --- Harnesses ---------------------------------------------------------------

// seededBody is what the seeded key holds.
const seededBody = "seeded"

// inMemory folds the seed into the constructor, which is where a seeded subject
// is built now: a factory may make any starting state, and it runs before
// anything wraps it.
func inMemory(name string) readertest.ReaderHarness[*readertest.InMemory] {
	return readertest.ReaderHarness[*readertest.InMemory]{Name: name, New: seeded}
}

func seeded() *readertest.InMemory {
	s := readertest.NewInMemory()
	s.Put(reader.Value{Key: readertest.DefaultReaderFixture().Key(), Body: seededBody})
	return s
}

// --- The checks: claims, bodies and defects, by name --------------------------

var readerChecks = readertest.ReaderChecks{
	{
		Method: "Get", Name: "hit-returns-seeded",
		Claim: "Get returns what was seeded under a key that was written",
		Run:   hitReturnsSeeded,
		ProvenBy: readertest.BrokenReader(
			"a reader that finds the key and answers nothing", newAnswersTheZero,
		),
		ProvenReason: "carries what was written",
	},
}

// --- Bodies -------------------------------------------------------------------

func hitReturnsSeeded(tb testing.TB, s reader.Reader, fx readertest.ReaderFixture) {
	tb.Helper()
	got, err := s.Get(tb.Context(), fx.Key())
	testkit.NoError(tb, err, "a seeded key is found")
	testkit.Equal(tb, got.Body, seededBody, "and carries what was written")
}

// --- Planted defects ----------------------------------------------------------

// answersTheZero reports the key found and hands back an empty value, which is
// the failure the derived miss check cannot see: it asks what happens for a key
// nothing wrote, and this one is wrong about a key something did.
type answersTheZero struct{ key string }

func newAnswersTheZero() answersTheZero {
	return answersTheZero{key: readertest.DefaultReaderFixture().Key()}
}

func (a answersTheZero) Get(_ context.Context, key string) (reader.Value, error) {
	if key != a.key {
		return reader.Value{}, reader.ErrNotFound
	}
	return reader.Value{}, nil
}
