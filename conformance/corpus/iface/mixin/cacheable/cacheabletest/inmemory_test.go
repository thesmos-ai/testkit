// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// cacheable is the model tier's — AUTO-CACHEABLE states it — so the suite
// generates the signature family alone.
//
// The assignment is the right one here for a reason visible in the subject: a
// cached read and an uncached one return the same value, so no single call can
// tell them apart. Observing it needs a reference to compare against, which is
// what the model tier has and this one does not.
//
// Nothing on this interface writes, so the reader shape's miss check was
// refused; the seed lives in the constructor instead, which is where a seeded
// subject is built now.
package cacheabletest_test

import (
	"context"
	"strconv"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/cacheable"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/cacheable/cacheabletest"
)

// TestMixedContract runs the generated checks and this package's own.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	cacheabletest.RunMixed(t, inMemory("in-memory"), mixedChecks)
}

// TestMixedContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	cacheabletest.RunMixed(t,
		inMemory("in-memory"),
		cacheabletest.MixedSuite.Without(cacheabletest.MixedSuite.Checks.Get.Smoke()),
	)
}

// TestMixedChecksCanFail drives every row against its planted defect.
func TestMixedChecksCanFail(t *testing.T) {
	t.Parallel()

	cacheabletest.ProveMixed(t, mixedChecks)
}

// --- Harnesses ---------------------------------------------------------------

// cachedBody is what the seeded key holds.
const cachedBody = "cached"

func inMemory(name string) cacheabletest.MixedHarness[*cacheabletest.InMemory] {
	return cacheabletest.MixedHarness[*cacheabletest.InMemory]{Name: name, New: seeded}
}

func seeded() *cacheabletest.InMemory {
	s := cacheabletest.NewInMemory()
	s.Put(cacheabletest.DefaultMixedFixture().Key(), cachedBody)
	return s
}

// --- The checks: claims, bodies and defects, by name --------------------------

var mixedChecks = cacheabletest.MixedChecks{
	{
		Method: "Get", Name: "returns-what-was-seeded",
		Claim: "Get returns what was seeded",
		Run:   returnsWhatWasSeeded,
		ProvenBy: cacheabletest.BrokenMixed(
			"a reader that answers the zero for a key it holds", planted(answersTheZero),
		),
		ProvenReason: "carries what was written",
	},

	{
		Method: "Get", Name: "repeated-read-agrees",
		Claim: "Get answers a repeated read from the cache",
		Run:   repeatedReadAgrees,
		ProvenBy: cacheabletest.BrokenMixed(
			"a reader that recomputes its answer on every call",
			planted(answersFreshEachTime),
		),
		ProvenReason: "with the same answer",
	},
}

// --- Bodies -------------------------------------------------------------------

func returnsWhatWasSeeded(
	tb testing.TB, s cacheable.Mixed, fx cacheabletest.MixedFixture,
) {
	tb.Helper()
	got, err := s.Get(tb.Context(), fx.Key())
	testkit.NoError(tb, err, "a seeded key is found")
	testkit.Equal(tb, got, cachedBody, "and carries what was written")
}

// repeatedReadAgrees is the mixin's own shape: the second read must agree with
// the first without consulting what backs it. Both answers look alike, which is
// why one read cannot state this and two can.
func repeatedReadAgrees(
	tb testing.TB, s cacheable.Mixed, fx cacheabletest.MixedFixture,
) {
	tb.Helper()
	first, err := s.Get(tb.Context(), fx.Key())
	testkit.NoError(tb, err, "the first read finds the seeded key")
	second, err := s.Get(tb.Context(), fx.Key())
	testkit.NoError(tb, err, "and so does the second")
	testkit.Equal(tb, second, first, "with the same answer")
}

// --- Planted defects ----------------------------------------------------------

// fault names what one planted reader gets wrong.
type fault int

const (
	// answersTheZero hands back nothing for a key it holds.
	answersTheZero fault = iota

	// answersFreshEachTime recomputes on every call and lands somewhere
	// different — a reader consulting what backs it rather than what it
	// cached, which is the one thing a cache must not do.
	answersFreshEachTime
)

// planted builds the constructor for one broken reader, holding what the
// harness's own subject holds so a read has the same thing to find.
func planted(wrong fault) func() *plantedReader {
	return func() *plantedReader {
		return &plantedReader{wrong: wrong, key: cacheabletest.DefaultMixedFixture().Key()}
	}
}

type plantedReader struct {
	wrong fault
	key   string
	reads int
}

func (p *plantedReader) Get(_ context.Context, key string) (string, error) {
	if key != p.key {
		return "", cacheabletest.ErrNotFound
	}
	switch p.wrong {
	case answersTheZero:
		return "", nil
	case answersFreshEachTime:
		p.reads++
		return cachedBody + "-" + strconv.Itoa(p.reads), nil
	}
	return cachedBody, nil
}
