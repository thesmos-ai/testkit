// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// streamreflectsmutations is the model tier's — AUTO-STREAM-REFLECTS-MUTATIONS
// states it — so the suite generates the signature family alone, even though
// eidos now lets the mixin name its mutator through `mutate=Add`.
//
// Stream returns a function, so what the signature can promise ends at the
// call: one check, about not crashing. Everything the mixin is about happens
// while someone is mid-range.
package streamreflectsmutationstest_test

import (
	"context"
	"iter"
	"slices"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/streamreflectsmutations"
	sm "go.thesmos.sh/testkit/conformance/corpus/iface/mixin/streamreflectsmutations/streamreflectsmutationstest"
)

// TestMixedContract runs the generated checks and this package's own.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	sm.RunMixed(t, inMemory("in-memory"), mixedChecks)
}

// TestMixedContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	sm.RunMixed(t, inMemory("in-memory"), sm.MixedSuite.Without(sm.MixedSuite.Checks.Add.Smoke()))
}

// TestMixedChecksCanFail drives every row against its planted defect.
func TestMixedChecksCanFail(t *testing.T) {
	t.Parallel()

	sm.ProveMixed(t, inMemory("in-memory"), mixedChecks)
}

// --- Harnesses ---------------------------------------------------------------

// The two items the mid-range row uses: one present before the range starts and
// one added while it is running, which is the whole of what the mixin claims.
const (
	beforeTheRange = "a"
	duringTheRange = "b"
)

func inMemory(name string) sm.MixedHarness[*sm.InMemory] {
	return sm.MixedHarness[*sm.InMemory]{Name: name, New: sm.NewInMemory}
}

// --- The checks: claims, bodies and defects, by name --------------------------
//
// Every row ranges the sequence, which is the one thing no generated check
// does: a check that only CALLED Stream would assert that a closure was built.

var mixedChecks = sm.MixedChecks{
	{
		Method: "Stream", Name: "yields-an-item-added-mid-range",
		Claim: "Stream yields an item added while it is running",
		Run:   yieldsAnItemAddedMidRange,
		ProvenBy: sm.BrokenMixed(
			"a stream over a copy taken when it was called", planted(snapshotsAtTheCall),
		),
		ProvenReason: "the run that added it sees it",
	},

	{
		Method: "Stream", Name: "stops-when-the-consumer-does",
		Claim: "Stream stops when the consumer does",
		Run:   stopsWhenTheConsumerDoes,
		ProvenBy: sm.BrokenMixed(
			"a stream that keeps yielding after the consumer broke out",
			planted(ignoresTheStop),
		),
		// A yield past the consumer's break is a runtime panic rather than a
		// wrong answer, and the harness records a panicking check as a failed
		// leg — which is what the row's own account of the failure predicts.
		ProvenReason: "panicked",
	},

	{
		Method: "Stream", Name: "cancellation-through-the-sequence",
		Claim: "Stream reports a cancelled context through the sequence",
		Run:   cancellationThroughTheSequence,
		ProvenBy: sm.BrokenMixed(
			"a stream that ranges whatever the context says", planted(ignoresCancellation),
		),
		ProvenReason: "a cancelled caller is answered once",
	},
}

// --- Bodies -------------------------------------------------------------------

// yieldsAnItemAddedMidRange is the mixin's whole claim, and one the signature
// cannot make.
func yieldsAnItemAddedMidRange(
	tb testing.TB, s streamreflectsmutations.Mixed, _ sm.MixedFixture,
) {
	tb.Helper()
	testkit.NoError(tb, s.Add(tb.Context(), beforeTheRange), "the first item is added")

	var seen []string
	for item, err := range s.Stream(tb.Context()) {
		testkit.NoError(tb, err, "the sequence yields without failing")
		seen = append(seen, item)
		if len(seen) == 1 {
			testkit.NoError(tb, s.Add(tb.Context(), duringTheRange),
				"an item added mid-range is accepted")
		}
	}
	testkit.Contains(tb, seen, duringTheRange, "and the run that added it sees it")
}

// stopsWhenTheConsumerDoes keeps a live stream bounded. A sequence that ignored
// the consumer's break would run to the end of a collection the caller stopped
// caring about — which for a stream over a live store is unbounded.
func stopsWhenTheConsumerDoes(
	tb testing.TB, s streamreflectsmutations.Mixed, fx sm.MixedFixture,
) {
	tb.Helper()
	for range 3 {
		testkit.NoError(tb, s.Add(tb.Context(), fx.Item()), "an item is added")
	}

	taken := 0
	for range s.Stream(tb.Context()) {
		taken++
		break
	}
	testkit.Equal(tb, taken, 1, "the range stopped where the consumer did")
}

// cancellationThroughTheSequence is the only place a cancellation can surface:
// the signature returns no error, so the sequence's own error slot carries it.
func cancellationThroughTheSequence(
	tb testing.TB, s streamreflectsmutations.Mixed, _ sm.MixedFixture,
) {
	tb.Helper()
	ctx, cancel := context.WithCancel(tb.Context())
	cancel()

	var errs []error
	for _, err := range s.Stream(ctx) {
		errs = append(errs, err)
	}
	testkit.Len(tb, errs, 1, "a cancelled caller is answered once")
	testkit.ErrorIs(tb, errs[0], context.Canceled, "with the cancellation")
}

// --- Planted defects ----------------------------------------------------------

// fault names what one planted stream gets wrong.
//
// All three are about the RANGE rather than the call, which is the point: a
// defect a generated check could catch would be a defect this row need not
// exist for.
type fault int

const (
	// snapshotsAtTheCall copies the collection when Stream is called and
	// ranges the copy, so a mutation mid-range is invisible — the ordinary
	// way to make a range safe, and the one thing this mixin forbids.
	snapshotsAtTheCall fault = iota

	// ignoresTheStop keeps calling yield after the consumer broke out,
	// which the runtime turns into a panic.
	ignoresTheStop

	// ignoresCancellation ranges what it holds whatever the context says,
	// so a cancelled caller is answered with a drain instead of a refusal.
	ignoresCancellation
)

// planted builds the constructor for one broken stream.
func planted(wrong fault) func() *plantedStream {
	return func() *plantedStream { return &plantedStream{wrong: wrong} }
}

type plantedStream struct {
	wrong fault
	items []string
}

func (p *plantedStream) Add(_ context.Context, item string) error {
	p.items = append(p.items, item)
	return nil
}

func (p *plantedStream) Remove(_ context.Context, item string) error {
	p.items = slices.DeleteFunc(p.items, func(held string) bool { return held == item })
	return nil
}

func (p *plantedStream) Stream(ctx context.Context) iter.Seq2[string, error] {
	return func(yield func(string, error) bool) {
		if err := ctx.Err(); err != nil && p.wrong != ignoresCancellation {
			yield("", err)
			return
		}
		if p.wrong == snapshotsAtTheCall {
			for _, item := range slices.Clone(p.items) {
				if !yield(item, nil) {
					return
				}
			}
			return
		}
		// Indexed rather than ranged, so an item appended mid-range is
		// reached — which is what the mixin asks for and what makes
		// ignoresTheStop's overrun unbounded.
		for i := 0; i < len(p.items); i++ {
			if !yield(p.items[i], nil) && p.wrong != ignoresTheStop {
				return
			}
		}
	}
}
