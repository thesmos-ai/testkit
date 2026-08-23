// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Two interfaces in one package, so two runs and two subjects — which is the
// arrangement, not an accident: the consumer needs something to consume, and
// the thing it consumes is a contract in its own right.
//
// Ingest takes a Source, which no literal can be written for, so every family
// the rules reached for it was refused; the generated header lists both. The
// rows below build the sources the claims need.
package streamconsumertest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/streamconsumer"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/streamconsumer/streamconsumertest"
)

// TestStreamConsumerContract runs the generated checks and this package's own.
func TestStreamConsumerContract(t *testing.T) {
	t.Parallel()

	streamconsumertest.RunStreamConsumer(t, inMemory("in-memory"), streamConsumerChecks)
}

// TestStreamConsumerChecksCanFail drives every consumer row against its planted
// defect.
func TestStreamConsumerChecksCanFail(t *testing.T) {
	t.Parallel()

	streamconsumertest.ProveStreamConsumer(t, inMemory("in-memory"), streamConsumerChecks)
}

// TestSourceContract runs the stream being consumed against its own contract,
// which is what makes the asymmetry in this fixture worth having: a produced
// stream is an iter.Seq2 return and generates almost nothing, while a consumed
// one is an interface parameter and generates a whole second harness.
func TestSourceContract(t *testing.T) {
	t.Parallel()

	streamconsumertest.RunSource(t, sliceSource("slice"), sourceChecks)
}

// TestSourceChecksCanFail drives the source row against its planted defect.
func TestSourceChecksCanFail(t *testing.T) {
	t.Parallel()

	streamconsumertest.ProveSource(t, sliceSource("slice"), sourceChecks)
}

// TestSourceContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
//
// Against Source rather than StreamConsumer: Ingest's only argument admits no
// literal, so StreamConsumer derives nothing and has no index entry to name.
func TestSourceContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	streamconsumertest.RunSource(t,
		sliceSource("slice"),
		streamconsumertest.SourceSuite.Without(streamconsumertest.SourceSuite.Checks.Next.Smoke()),
	)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(
	name string,
) streamconsumertest.StreamConsumerHarness[*streamconsumertest.InMemory] {
	return streamconsumertest.StreamConsumerHarness[*streamconsumertest.InMemory]{
		Name: name, New: streamconsumertest.NewInMemory,
	}
}

// sliceSource is a source holding one element, which is enough for the flag
// claim: exhaustion needs something to be exhausted.
func sliceSource(name string) streamconsumertest.SourceHarness[streamconsumer.Source] {
	return streamconsumertest.SourceHarness[streamconsumer.Source]{
		Name: name,
		New: func() streamconsumer.Source {
			return streamconsumertest.NewSliceSource(streamconsumer.Value{Key: "a", Body: "one"})
		},
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var streamConsumerChecks = streamconsumertest.StreamConsumerChecks{
	{
		Method: "Ingest", Name: "drains-and-counts",
		Claim: "Ingest drains the source and counts what it took",
		Run:   drainsAndCounts,
		ProvenBy: streamconsumertest.BrokenStreamConsumer(
			"a consumer that stops after the first element", consuming(countsOnlyTheFirst),
		),
		ProvenReason: "every element is counted",
	},

	{
		Method: "Ingest", Name: "refuses-a-missing-source",
		Claim: "Ingest refuses a source that is not there",
		Run:   refusesAMissingSource,
		ProvenBy: streamconsumertest.BrokenStreamConsumer(
			"a consumer that reads a nil source as an empty one", consuming(treatsNilAsEmpty),
		),
		ProvenReason: "a failed ingest rather than an empty one",
	},

	{
		Method: "Ingest", Name: "stops-on-a-failing-source",
		Claim: "Ingest stops on a source that fails mid-drain",
		Run:   stopsOnAFailingSource,
		ProvenBy: streamconsumertest.BrokenStreamConsumer(
			"a consumer that counts what it took before the failure",
			consuming(countsAPartialDrain),
		),
		ProvenReason: "a partial drain is not a count",
	},
}

var sourceChecks = streamconsumertest.SourceChecks{
	{
		Method: "Next", Name: "reports-exhaustion-through-its-flag",
		Claim: "Next reports exhaustion through its flag",
		Run:   reportsExhaustionThroughItsFlag,
		ProvenBy: streamconsumertest.BrokenSource(
			"a source that serves its last element again at exhaustion", newRepeatingSource,
		),
		ProvenReason: "the zero rather than the last element again",
	},
}

// --- Bodies -------------------------------------------------------------------

func drainsAndCounts(
	tb testing.TB, s streamconsumer.StreamConsumer,
	_ streamconsumertest.StreamConsumerFixture,
) {
	tb.Helper()
	got, err := s.Ingest(tb.Context(), streamconsumertest.NewSliceSource(
		streamconsumer.Value{Key: "a", Body: "one"},
		streamconsumer.Value{Key: "b", Body: "two"},
	))
	testkit.NoError(tb, err, "a readable source is ingested")
	testkit.Equal(tb, got, 2, "and every element is counted")
}

// refusesAMissingSource keeps a construction failure from reading as an empty
// stream.
//
// A nil source reaches production through a caller whose own construction
// failed, and draining it is a panic rather than a count of zero.
func refusesAMissingSource(
	tb testing.TB, s streamconsumer.StreamConsumer,
	_ streamconsumertest.StreamConsumerFixture,
) {
	tb.Helper()
	got, err := s.Ingest(tb.Context(), nil)
	testkit.ErrorIs(tb, err, streamconsumertest.ErrNoSource,
		"a nil source is a failed ingest rather than an empty one")
	testkit.Equal(tb, got, 0, "and carries the zero count beside it")
}

// stopsOnAFailingSource keeps a partial drain from reading as a count.
//
// A source that fails partway is the ordinary network case, and the count that
// comes back with the error is what tells a caller whether to resume or
// restart.
func stopsOnAFailingSource(
	tb testing.TB, s streamconsumer.StreamConsumer,
	_ streamconsumertest.StreamConsumerFixture,
) {
	tb.Helper()
	got, err := s.Ingest(tb.Context(), &failingSource{})
	testkit.ErrorIs(tb, err, streamconsumertest.ErrSourceFailed,
		"the source's failure is reported")
	testkit.Equal(tb, got, 0,
		"with nothing counted beside it, since a partial drain is not a count")
}

func reportsExhaustionThroughItsFlag(
	tb testing.TB, s streamconsumer.Source, _ streamconsumertest.SourceFixture,
) {
	tb.Helper()
	v, ok, err := s.Next(tb.Context())
	testkit.NoError(tb, err, "the first element reads cleanly")
	testkit.True(tb, ok, "and the flag says there was one")
	testkit.Equal(tb, v.Key, "a", "carrying what the source held")

	v, ok, err = s.Next(tb.Context())
	testkit.NoError(tb, err, "exhaustion is not a failure")
	testkit.False(tb, ok, "the flag says the stream is done")
	testkit.Equal(tb, v, streamconsumer.Value{},
		"and the value slot is the zero rather than the last element again")
}

// --- Planted defects ----------------------------------------------------------

// failingSource yields one element and then fails, which is the case a count
// returned beside an error would misreport. It is an input rather than a
// planted defect: the subject under proof is the consumer reading it.
//
// A pointer receiver because the state has to advance: a value receiver would
// serve the first element forever and the ingest would never terminate.
type failingSource struct{ served bool }

func (f *failingSource) Next(context.Context) (streamconsumer.Value, bool, error) {
	if !f.served {
		f.served = true
		return streamconsumer.Value{Key: "a"}, true, nil
	}
	return streamconsumer.Value{}, false, streamconsumertest.ErrSourceFailed
}

// fault names what one planted consumer gets wrong.
//
// One implementation with three branches rather than three consumers: the drain
// loop is the same in every case, and what differs is which of its three
// answers is wrong.
type fault int

const (
	// countsOnlyTheFirst drains the whole source and reports having taken
	// one element from it.
	countsOnlyTheFirst fault = iota

	// treatsNilAsEmpty reads a missing source as a stream with nothing in
	// it, which is the answer a caller cannot tell from success.
	treatsNilAsEmpty

	// countsAPartialDrain returns what it took before the failure, which
	// reads as a count and is a position.
	countsAPartialDrain
)

// consuming builds the constructor for one broken consumer.
func consuming(wrong fault) func() plantedConsumer {
	return func() plantedConsumer { return plantedConsumer{wrong: wrong} }
}

type plantedConsumer struct{ wrong fault }

func (c plantedConsumer) Ingest(
	ctx context.Context, src streamconsumer.Source,
) (int, error) {
	if src == nil {
		if c.wrong == treatsNilAsEmpty {
			return 0, nil
		}
		return 0, streamconsumertest.ErrNoSource
	}
	taken := 0
	for {
		_, ok, err := src.Next(ctx)
		if err != nil {
			if c.wrong == countsAPartialDrain {
				return taken, err
			}
			return 0, err
		}
		if !ok {
			break
		}
		taken++
	}
	if c.wrong == countsOnlyTheFirst && taken > 0 {
		return 1, nil
	}
	return taken, nil
}

// repeatingSource serves its last element again once exhausted, with the flag
// correctly false — the near miss that makes a caller reading the value slot
// before the flag process the tail twice.
type repeatingSource struct{ last streamconsumer.Value }

func newRepeatingSource() *repeatingSource { return &repeatingSource{} }

func (s *repeatingSource) Next(context.Context) (streamconsumer.Value, bool, error) {
	if s.last == (streamconsumer.Value{}) {
		s.last = streamconsumer.Value{Key: "a", Body: "one"}
		return s.last, true, nil
	}
	return s.last, false, nil
}
