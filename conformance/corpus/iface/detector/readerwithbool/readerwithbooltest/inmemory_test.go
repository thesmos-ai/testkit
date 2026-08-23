// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// The comma-ok read: absence is a flag, not a failure, so no error slot exists
// and the signature earns only the check that asks about crashing.
//
// The claim worth making — that the flag and the value agree — is stated here.
// A subject returning a populated value beside false, or the zero beside true,
// satisfies every generated check and is broken.
package readerwithbooltest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/readerwithbool"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/readerwithbool/readerwithbooltest"
)

// TestReaderWithBoolContract runs the generated check and this package's own.
func TestReaderWithBoolContract(t *testing.T) {
	t.Parallel()

	readerwithbooltest.RunReaderWithBool(t, inMemory("in-memory"), readerWithBoolChecks)
}

// TestReaderWithBoolContractWithoutSmoke drops a check through the typed index
// rather than a string, so a check that is renamed or stops being emitted
// breaks this compile instead of silently declining nothing.
func TestReaderWithBoolContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	readerwithbooltest.RunReaderWithBool(t,
		inMemory("in-memory"),
		readerwithbooltest.ReaderWithBoolSuite.Without(
			readerwithbooltest.ReaderWithBoolSuite.Checks.Load.Smoke(),
		),
	)
}

// TestReaderWithBoolChecksCanFail drives every row against its planted defect.
func TestReaderWithBoolChecksCanFail(t *testing.T) {
	t.Parallel()

	readerwithbooltest.ProveReaderWithBool(t, inMemory("in-memory"), readerWithBoolChecks)
}

// --- Harnesses ---------------------------------------------------------------

// seededBody is what the seeded key holds, so "the value comes with the flag"
// compares against something rather than against a zero either way.
const seededBody = "seeded"

func inMemory(name string) readerwithbooltest.ReaderWithBoolHarness[*readerwithbooltest.InMemory] {
	return readerwithbooltest.ReaderWithBoolHarness[*readerwithbooltest.InMemory]{
		Name: name, New: seeded,
	}
}

func seeded() *readerwithbooltest.InMemory {
	s := readerwithbooltest.NewInMemory()
	s.Put(readerwithbool.Value{
		Key:  readerwithbooltest.DefaultReaderWithBoolFixture().Key(),
		Body: seededBody,
	})
	return s
}

// --- The checks: claims, bodies and defects, by name --------------------------
//
// Each planted defect gets the FLAG right and the value wrong, which is the
// disagreement this shape allows: a check reading only the flag passes both.

var readerWithBoolChecks = readerwithbooltest.ReaderWithBoolChecks{
	{
		Method: "Load", Name: "flag-agrees-on-a-hit",
		Claim: "Load agrees with its own flag on a hit",
		Run:   flagAgreesOnAHit,
		ProvenBy: readerwithbooltest.BrokenReaderWithBool(
			"a reader answering the zero beside a true flag", planted(zeroBesideTrue),
		),
		ProvenReason: "the value comes with the flag",
	},

	{
		Method: "Load", Name: "flag-agrees-on-a-miss",
		Claim: "Load agrees with its own flag on a miss",
		Run:   flagAgreesOnAMiss,
		ProvenBy: readerwithbooltest.BrokenReaderWithBool(
			"a reader answering a record beside a false flag", planted(recordBesideFalse),
		),
		ProvenReason: "the value slot is the zero",
	},
}

// --- Bodies -------------------------------------------------------------------

func flagAgreesOnAHit(
	tb testing.TB, s readerwithbool.ReaderWithBool,
	fx readerwithbooltest.ReaderWithBoolFixture,
) {
	tb.Helper()
	got, ok := s.Load(tb.Context(), fx.Key())
	testkit.True(tb, ok, "a seeded key is present")
	testkit.Equal(tb, got.Body, seededBody, "and the value comes with the flag")
}

// flagAgreesOnAMiss asks both halves, because either alone is satisfied by a
// broken subject: false beside a populated value, or true beside the zero.
func flagAgreesOnAMiss(
	tb testing.TB, s readerwithbool.ReaderWithBool,
	fx readerwithbooltest.ReaderWithBoolFixture,
) {
	tb.Helper()
	got, ok := s.Load(tb.Context(), fx.KeyOther())
	testkit.False(tb, ok, "an unwritten key is absent")
	testkit.Equal(tb, got, readerwithbool.Value{}, "and the value slot is the zero")
}

// --- Planted defects ----------------------------------------------------------

// fault names which way one planted reader's two returns disagree.
type fault int

const (
	// zeroBesideTrue reports a key present and hands back nothing, which a
	// caller trusting the flag reads as a real record with empty fields.
	zeroBesideTrue fault = iota

	// recordBesideFalse reports a key absent and leaves the last record in
	// the value slot, which a caller reading before the flag processes
	// twice.
	recordBesideFalse
)

// planted builds the constructor for one broken reader, holding what the
// harness's own subject holds so a hit has the same thing to hit.
func planted(wrong fault) func() plantedReader {
	return func() plantedReader {
		return plantedReader{
			wrong: wrong,
			key:   readerwithbooltest.DefaultReaderWithBoolFixture().Key(),
		}
	}
}

type plantedReader struct {
	wrong fault
	key   string
}

func (p plantedReader) Load(
	_ context.Context, key string,
) (readerwithbool.Value, bool) {
	held := readerwithbool.Value{Key: p.key, Body: seededBody}
	if key != p.key {
		if p.wrong == recordBesideFalse {
			return held, false
		}
		return readerwithbool.Value{}, false
	}
	if p.wrong == zeroBesideTrue {
		return readerwithbool.Value{}, true
	}
	return held, true
}
