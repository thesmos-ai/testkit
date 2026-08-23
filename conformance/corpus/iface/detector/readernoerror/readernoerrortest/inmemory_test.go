// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// A read with no error slot earns one check, and the missing four are the point
// of the fixture rather than a gap.
//
// Cancellation and an expired deadline are claims about what a method *reports*,
// and this one has nowhere to report them; the zero-value check compares a
// result against an error that does not exist. Nothing on the interface writes
// either, so the miss is refused too — leaving the smoke call, which is about
// not crashing, and the two rows below.
package readernoerrortest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/readernoerror"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/readernoerror/readernoerrortest"
)

// TestReaderNoErrorContract runs the generated check and this package's own.
func TestReaderNoErrorContract(t *testing.T) {
	t.Parallel()

	readernoerrortest.RunReaderNoError(t, inMemory("in-memory"), readerNoErrorChecks)
}

// TestReaderNoErrorContractWithoutSmoke drops a check through the typed index
// rather than a string, so a check that is renamed or stops being emitted
// breaks this compile instead of silently declining nothing.
func TestReaderNoErrorContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	readernoerrortest.RunReaderNoError(t,
		inMemory("in-memory"),
		readernoerrortest.ReaderNoErrorSuite.Without(
			readernoerrortest.ReaderNoErrorSuite.Checks.Lookup.Smoke(),
		),
	)
}

// TestReaderNoErrorChecksCanFail drives every row against its planted defect.
func TestReaderNoErrorChecksCanFail(t *testing.T) {
	t.Parallel()

	readernoerrortest.ProveReaderNoError(t, inMemory("in-memory"), readerNoErrorChecks)
}

// --- Harnesses ---------------------------------------------------------------

// seededBody is what the seeded key holds. Non-empty on purpose: with no error
// slot and no flag, the VALUE is the only thing either row can read, so a zero
// here would make both of them vacuous.
const seededBody = "seeded"

func inMemory(name string) readernoerrortest.ReaderNoErrorHarness[*readernoerrortest.InMemory] {
	return readernoerrortest.ReaderNoErrorHarness[*readernoerrortest.InMemory]{
		Name: name, New: seeded,
	}
}

func seeded() *readernoerrortest.InMemory {
	s := readernoerrortest.NewInMemory()
	s.Put(readernoerror.Value{
		Key:  readernoerrortest.DefaultReaderNoErrorFixture().Key(),
		Body: seededBody,
	})
	return s
}

// --- The checks: claims, bodies and defects, by name --------------------------

var readerNoErrorChecks = readernoerrortest.ReaderNoErrorChecks{
	{
		Method: "Lookup", Name: "hit-reads-what-was-seeded",
		Claim: "Lookup reads back what was seeded",
		Run:   hitReadsWhatWasSeeded,
		ProvenBy: readernoerrortest.BrokenReaderNoError(
			"a reader that answers the zero for a key it holds",
			planted(missesItsOwnKey),
		),
		ProvenReason: "a present key reads as what was written",
	},

	{
		Method: "Lookup", Name: "miss-reads-as-the-zero",
		Claim: "Lookup reads as the zero for a key nothing wrote",
		Run:   missReadsAsTheZero,
		ProvenBy: readernoerrortest.BrokenReaderNoError(
			"a reader that answers its last record whatever it is asked",
			planted(answersOnAMiss),
		),
		ProvenReason: "reads as the zero rather than as anything held",
	},
}

// --- Bodies -------------------------------------------------------------------

func hitReadsWhatWasSeeded(
	tb testing.TB, s readernoerror.ReaderNoError,
	fx readernoerrortest.ReaderNoErrorFixture,
) {
	tb.Helper()
	testkit.Equal(tb, s.Lookup(tb.Context(), fx.Key()).Body, seededBody,
		"a present key reads as what was written")
}

// missReadsAsTheZero is the whole of what this shape can say about absence, and
// a claim only a seeding consumer can make: without a writer the generator
// cannot tell an unwritten key from any other.
func missReadsAsTheZero(
	tb testing.TB, s readernoerror.ReaderNoError,
	fx readernoerrortest.ReaderNoErrorFixture,
) {
	tb.Helper()
	testkit.Equal(tb, s.Lookup(tb.Context(), fx.KeyOther()), readernoerror.Value{},
		"an absent key reads as the zero rather than as anything held")
}

// --- Planted defects ----------------------------------------------------------

// fault names what one planted reader gets wrong.
//
// With no error slot and no flag, the value IS the answer — so both defects are
// about handing back the wrong one, and neither is visible to any check that
// reads a return other than the first.
type fault int

const (
	// missesItsOwnKey answers the zero for a key it holds, which with no
	// flag beside it a caller reads as absence.
	missesItsOwnKey fault = iota

	// answersOnAMiss hands back its last record whatever it was asked,
	// which is a cache returning a stale entry under a key nobody wrote.
	answersOnAMiss
)

// planted builds the constructor for one broken reader, holding what the
// harness's own subject holds so a hit has the same thing to hit.
func planted(wrong fault) func() plantedReader {
	return func() plantedReader {
		return plantedReader{
			wrong: wrong,
			key:   readernoerrortest.DefaultReaderNoErrorFixture().Key(),
		}
	}
}

type plantedReader struct {
	wrong fault
	key   string
}

func (p plantedReader) Lookup(_ context.Context, key string) readernoerror.Value {
	held := readernoerror.Value{Key: p.key, Body: seededBody}
	if key != p.key {
		if p.wrong == answersOnAMiss {
			return held
		}
		return readernoerror.Value{}
	}
	if p.wrong == missesItsOwnKey {
		return readernoerror.Value{}
	}
	return held
}
