// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// A method returning an iterator returns a function, so what the signature can
// promise ends at the call: one check, about the call itself.
//
// Everything the stream is about happens when someone ranges it, and no
// generated check does — which is not a gap so much as the honest reading of a
// lazy return. The error is per element rather than per call, so even a
// cancelled context has nowhere to surface until the first yield.
package streamreadertest_test

import (
	"context"
	"iter"
	"slices"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/streamreader"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/streamreader/streamreadertest"
)

// TestStreamReaderContract runs the generated checks and this package's own.
func TestStreamReaderContract(t *testing.T) {
	t.Parallel()

	streamreadertest.RunStreamReader(t, inMemory("in-memory"), streamReaderChecks)
}

// TestStreamReaderContractWithoutSmoke drops a check through the typed index
// rather than a string, so a check that is renamed or stops being emitted
// breaks this compile instead of silently declining nothing.
func TestStreamReaderContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	streamreadertest.RunStreamReader(t,
		inMemory("in-memory"),
		streamreadertest.StreamReaderSuite.Without(
			streamreadertest.StreamReaderSuite.Checks.List.Smoke(),
		),
	)
}

// TestStreamReaderChecksCanFail drives every row against its planted defect.
func TestStreamReaderChecksCanFail(t *testing.T) {
	t.Parallel()

	streamreadertest.ProveStreamReader(t, streamReaderChecks)
}

// --- Harnesses ---------------------------------------------------------------

// held is what a seeded stream yields, in the order it was added.
//
// Two elements rather than one, because every claim below is about a SEQUENCE:
// a one-element stream agrees with itself in any order and stops after one
// element whether or not it was asked to.
var held = []streamreader.Value{
	{Key: "first", Body: "one"},
	{Key: "second", Body: "two"},
}

func inMemory(name string) streamreadertest.StreamReaderHarness[*streamreadertest.InMemory] {
	return streamreadertest.StreamReaderHarness[*streamreadertest.InMemory]{
		Name: name, New: seeded,
	}
}

func seeded() *streamreadertest.InMemory {
	s := streamreadertest.NewInMemory()
	for _, v := range held {
		s.Put(v)
	}
	return s
}

// --- The checks: claims, bodies and defects, by name --------------------------
//
// Nothing here is derivable: a generated check ends at the call, and the value
// it gets back is a function nobody has run yet.

var streamReaderChecks = streamreadertest.StreamReaderChecks{
	{
		Method: "List", Name: "yields-what-it-holds-in-order",
		Claim: "List yields what it holds, in order",
		Run:   yieldsWhatItHoldsInOrder,
		ProvenBy: streamreadertest.BrokenStreamReader(
			"a stream that yields what it holds the other way round",
			planted(yieldsOutOfOrder),
		),
		ProvenReason: "in the order they were added",
	},

	{
		Method: "List", Name: "stops-when-the-consumer-does",
		Claim: "List stops when the consumer does",
		Run:   stopsWhenTheConsumerDoes,
		ProvenBy: streamreadertest.BrokenStreamReader(
			"a stream that keeps yielding after the consumer broke out",
			planted(ignoresTheStop),
		),
		// A yield past the consumer's break is a runtime panic rather than a
		// wrong answer, and the harness records a panicking check as a failed
		// leg — which is what the row's own account of the failure predicts.
		ProvenReason: "panicked",
	},

	{
		Method: "List", Name: "cancelled-on-the-first-yield",
		Claim: "List reports a cancelled context on the first yield",
		Run:   cancelledOnTheFirstYield,
		ProvenBy: streamreadertest.BrokenStreamReader(
			"a stream that yields regardless of the context",
			planted(ignoresCancellation),
		),
		ProvenReason: "says so through its element error",
	},
}

// --- Bodies -------------------------------------------------------------------

func yieldsWhatItHoldsInOrder(
	tb testing.TB, s streamreader.StreamReader, _ streamreadertest.StreamReaderFixture,
) {
	tb.Helper()
	var got []string
	for v, err := range s.List(tb.Context()) {
		testkit.NoError(tb, err, "a healthy stream yields no per-element error")
		got = append(got, v.Key)
	}
	testkit.Equal(tb, got, []string{held[0].Key, held[1].Key},
		"in the order they were added")
}

// stopsWhenTheConsumerDoes is the shape's own law: a consumer may break out, so
// an implementation must not assume the sequence is drained. One that did would
// deadlock or panic here rather than return.
func stopsWhenTheConsumerDoes(
	tb testing.TB, s streamreader.StreamReader, _ streamreadertest.StreamReaderFixture,
) {
	tb.Helper()
	var seen int
	for range s.List(tb.Context()) {
		seen++
		break
	}
	testkit.Equal(tb, seen, 1, "the range stopped after one element")
}

// cancelledOnTheFirstYield is where the generated cancellation check would have
// gone, if the error were on the call rather than on the element.
func cancelledOnTheFirstYield(
	tb testing.TB, s streamreader.StreamReader, _ streamreadertest.StreamReaderFixture,
) {
	tb.Helper()
	ctx, cancel := context.WithCancel(tb.Context())
	cancel()
	for _, err := range s.List(ctx) {
		testkit.ErrorIs(tb, err, context.Canceled,
			"a cancelled stream says so through its element error")
		break
	}
}

// --- Planted defects ----------------------------------------------------------

// fault names what one planted stream gets wrong.
//
// All three are about the RANGE rather than the call, which is the point: a
// defect a generated check could catch would be a defect this row need not
// exist for.
type fault int

const (
	// yieldsOutOfOrder hands the elements back the other way round, which
	// is what ranging a Go map does.
	yieldsOutOfOrder fault = iota

	// ignoresTheStop keeps calling yield after the consumer broke out,
	// which the runtime turns into a panic.
	ignoresTheStop

	// ignoresCancellation yields cleanly whatever the context says, so a
	// cancelled range drains instead of stopping.
	ignoresCancellation
)

// planted builds the constructor for one broken stream, holding what the
// harness's own subject holds so a drain has the same thing to drain.
func planted(wrong fault) func() *plantedStream {
	return func() *plantedStream {
		return &plantedStream{wrong: wrong, held: slices.Clone(held)}
	}
}

type plantedStream struct {
	wrong fault
	held  []streamreader.Value
}

func (p *plantedStream) Add(_ context.Context, v streamreader.Value) error {
	p.held = append(p.held, v)
	return nil
}

func (p *plantedStream) List(ctx context.Context) iter.Seq2[streamreader.Value, error] {
	return func(yield func(streamreader.Value, error) bool) {
		out := slices.Clone(p.held)
		if p.wrong == yieldsOutOfOrder {
			slices.Reverse(out)
		}
		for _, v := range out {
			err := ctx.Err()
			if p.wrong == ignoresCancellation {
				err = nil
			}
			// The stop is read on every fault but one, and the one that
			// ignores it is why this is not a plain `if !yield(...) return`.
			if !yield(v, err) && p.wrong != ignoresTheStop {
				return
			}
		}
	}
}
