// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// answeringwriter is the detector axis's: a write that answers the state it
// stored, at the same named type it took. The suite generates the signature
// family, and the row below is what the shape's own claim comes to.
package answeringwritertest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/answeringwriter"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/answeringwriter/answeringwritertest"
)

// TestAnsweringWriterContract runs the generated checks and this package's own.
func TestAnsweringWriterContract(t *testing.T) {
	t.Parallel()

	answeringwritertest.RunAnsweringWriter(t, inMemory("in-memory"), answeringWriterChecks)
}

// TestAnsweringWriterChecksCanFail drives every planted defect through the
// check it is evidence for.
func TestAnsweringWriterChecksCanFail(t *testing.T) {
	t.Parallel()

	answeringwritertest.ProveAnsweringWriter(t, inMemory("in-memory"), answeringWriterChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) answeringwritertest.AnsweringWriterHarness[*answeringwritertest.InMemory] {
	return answeringwritertest.AnsweringWriterHarness[*answeringwritertest.InMemory]{
		Name: name, New: answeringwritertest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var answeringWriterChecks = answeringwritertest.AnsweringWriterChecks{
	{
		Method: "Put", Name: "answers-what-it-stored",
		Claim: "Put answers the value it stored",
		Run:   answersWhatItStored,
		ProvenBy: answeringwritertest.BrokenAnsweringWriter(
			"a writer that answers the zero value", newAnswersZero,
		),
		ProvenReason: "the answer is what went in",
	},
}

// --- Bodies -------------------------------------------------------------------

// answersWhatItStored is the whole content of this shape: the answer is the
// state, so a caller reading the return and one reading the store must not
// disagree.
func answersWhatItStored(
	tb testing.TB, s answeringwriter.AnsweringWriter, fx answeringwritertest.AnsweringWriterFixture,
) {
	tb.Helper()
	got, err := s.Put(tb.Context(), fx.Value())
	testkit.NoError(tb, err, "the write is accepted")
	testkit.Equal(tb, got, fx.Value(), "the answer is what went in")
}

// --- Planted defects ----------------------------------------------------------

// answersZero accepts the write and answers nothing, which is a writer whose
// return says the store is empty while the store is not.
type answersZero struct{}

func newAnswersZero() answersZero { return answersZero{} }

func (answersZero) Put(context.Context, answeringwriter.Value) (answeringwriter.Value, error) {
	return answeringwriter.Value{}, nil
}
