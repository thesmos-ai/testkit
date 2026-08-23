// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// outbox is the suite tier's under ADR-0028. `AUTO-PUBLISHER-DELIVERS` states
// delivery to a subscriber that was already listening, which is the `publisher`
// contract; what an outbox adds is that the record survives until somebody is,
// and no law carries that.
//
// So the durability row appends first and subscribes second, and the live row
// does the reverse. An outbox owes both.
package outboxtest_test

import (
	"context"
	"sync"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/outbox"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/outbox/outboxtest"
)

// TestContractContract runs the generated checks and this package's own.
func TestContractContract(t *testing.T) {
	t.Parallel()

	outboxtest.RunContract(t, inMemory("in-memory"), contractChecks)
}

// TestContractContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestContractContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	outboxtest.RunContract(t,
		inMemory("in-memory"),
		outboxtest.ContractSuite.Without(outboxtest.ContractSuite.Checks.Append.Smoke()),
	)
}

// TestContractChecksCanFail drives every row against its planted defect.
func TestContractChecksCanFail(t *testing.T) {
	t.Parallel()

	outboxtest.ProveContract(t, inMemory("in-memory"), contractChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) outboxtest.ContractHarness[*outboxtest.InMemory] {
	return outboxtest.ContractHarness[*outboxtest.InMemory]{
		Name: name, New: outboxtest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var contractChecks = outboxtest.ContractChecks{
	{
		Method: "Subscribe", Name: "keeps-a-record-until-somebody-listens",
		Claim: "Subscribe hands over a record appended before it attached",
		Run:   keepsARecordUntilSomebodyListens,
		ProvenBy: outboxtest.BrokenContract(
			"an outbox that keeps nothing for a subscriber yet to arrive",
			planted(keepsNoBacklog),
		),
		ProvenReason: "handed the backlog",
	},

	{
		Method: "Subscribe", Name: "delivers-to-an-attached-subscriber",
		Claim: "Subscribe delivers to a subscriber that was already attached",
		Run:   deliversToAnAttachedSubscriber,
		ProvenBy: outboxtest.BrokenContract(
			"an outbox that hands over the backlog and nothing after it",
			planted(deliversOnlyTheBacklog),
		),
		ProvenReason: "and reaches them",
	},

	{
		Method: "Append", Name: "reports-an-unreachable-subscriber",
		Claim: "Append reports a subscriber it can no longer reach",
		Run:   reportsAnUnreachableSubscriber,
		ProvenBy: outboxtest.BrokenContract(
			"an outbox that discards what a subscriber cannot take",
			planted(dropsSilently),
		),
		ProvenReason: "never reported as behind",
	},

	{
		Method: "Subscribe", Name: "refuses-a-backlog-it-cannot-hand-over",
		Claim: "Subscribe refuses a subscriber it cannot hand the backlog to",
		Run:   refusesABacklogItCannotHandOver,
		ProvenBy: outboxtest.BrokenContract(
			"an outbox that hands over as much of the backlog as fits",
			planted(truncatesTheBacklog),
		),
		ProvenReason: "refused rather than truncated",
	},
}

// --- Bodies -------------------------------------------------------------------

// keepsARecordUntilSomebodyListens is the durability half: the record is
// written with nobody listening, which is the whole reason an outbox exists.
func keepsARecordUntilSomebodyListens(
	tb testing.TB, s outbox.Contract, fx outboxtest.ContractFixture,
) {
	tb.Helper()
	testkit.NoError(tb, s.Append(tb.Context(), fx.Value()),
		"a record is appended with nobody listening")

	stream, err := s.Subscribe(tb.Context())
	testkit.NoError(tb, err, "a subscriber attaches afterwards")
	testkit.Equal(tb, <-stream, fx.Value(), "and is handed the backlog")
}

// deliversToAnAttachedSubscriber is the live half, which is `publisher`'s
// claim — an outbox owes it too, and a subject can keep a backlog without
// serving anybody currently attached.
func deliversToAnAttachedSubscriber(
	tb testing.TB, s outbox.Contract, fx outboxtest.ContractFixture,
) {
	tb.Helper()
	stream, err := s.Subscribe(tb.Context())
	testkit.NoError(tb, err, "a subscriber attaches")

	testkit.NoError(tb, s.Append(tb.Context(), fx.ValueOther()),
		"a record appended after the subscriber attached is accepted")
	testkit.Equal(tb, <-stream, fx.ValueOther(), "and reaches them")
}

// reportsAnUnreachableSubscriber makes the drop loud.
//
// A silent drop is the one failure this whole fixture is about, so the drop has
// to be reported — and a report nothing reaches is a report nothing proves. The
// reader never reads, which is what puts the subscriber far enough behind.
func reportsAnUnreachableSubscriber(
	tb testing.TB, s outbox.Contract, fx outboxtest.ContractFixture,
) {
	tb.Helper()
	_, err := s.Subscribe(tb.Context())
	testkit.NoError(tb, err, "a subscriber attaches")

	for range 32 {
		if err := s.Append(tb.Context(), fx.Value()); err != nil {
			testkit.ErrorIs(tb, err, outboxtest.ErrFull,
				"the append says why it could not be taken")
			return
		}
	}
	tb.Fatalf("a subscriber that never reads was never reported as behind")
}

// refusesABacklogItCannotHandOver keeps the durability promise honest.
//
// The backlog is loaded at subscribe time, so a log longer than a subscriber
// can hold is refused rather than truncated — which would lose exactly the
// records an outbox exists to keep.
func refusesABacklogItCannotHandOver(
	tb testing.TB, s outbox.Contract, fx outboxtest.ContractFixture,
) {
	tb.Helper()
	for range 32 {
		if err := s.Append(tb.Context(), fx.Value()); err != nil {
			tb.Fatalf("appending with nobody listening must not fail: %v", err)
		}
	}
	_, err := s.Subscribe(tb.Context())
	testkit.ErrorIs(tb, err, outboxtest.ErrFull,
		"a backlog too long to hand over is refused rather than truncated")
}

// --- Planted defects ----------------------------------------------------------

// fault names what one planted outbox gets wrong.
//
// One implementation with a switch rather than four, because all four get the
// same two decisions wrong in different combinations: what Subscribe hands over
// and what Append does with a subscriber it cannot reach.
type fault int

const (
	// keepsNoBacklog holds nothing for a subscriber yet to arrive, which is
	// a publisher wearing an outbox's name.
	keepsNoBacklog fault = iota

	// deliversOnlyTheBacklog hands over what it kept and never reaches a
	// subscriber already attached.
	deliversOnlyTheBacklog

	// dropsSilently discards a record a subscriber is too far behind to
	// take, and says the append succeeded.
	dropsSilently

	// truncatesTheBacklog hands over as much of the log as fits, which
	// loses exactly the records an outbox exists to keep.
	truncatesTheBacklog
)

// plantedBuffer is what a planted subscriber holds, small enough that a row
// appending 32 records overruns it: a bound nothing reaches proves nothing
// about what happens at it.
const plantedBuffer = 8

// planted builds the constructor for one broken outbox.
func planted(wrong fault) func() *plantedOutbox {
	return func() *plantedOutbox { return &plantedOutbox{wrong: wrong} }
}

type plantedOutbox struct {
	wrong fault
	mu    sync.Mutex
	log   []outbox.Value
	subs  []chan outbox.Value
}

// Append records and delivers, and never reports a subscriber it could not
// reach — which is dropsSilently's defect and invisible to the other three,
// because no other row leaves a subscriber behind.
func (o *plantedOutbox) Append(_ context.Context, v outbox.Value) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.log = append(o.log, v)
	if o.wrong == deliversOnlyTheBacklog {
		return nil
	}
	for _, ch := range o.subs {
		select {
		case ch <- v:
		default:
		}
	}
	return nil
}

// Subscribe hands over what the fault says it hands over, and never refuses.
//
// A stream nothing more is coming on is closed rather than left open: a check
// receiving from an open channel nobody will ever send to would hang the proof
// instead of failing it, and a proof that hangs reports nothing at all.
func (o *plantedOutbox) Subscribe(context.Context) (<-chan outbox.Value, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	ch := make(chan outbox.Value, plantedBuffer)
	if o.wrong != keepsNoBacklog {
		for _, v := range o.log[:min(len(o.log), plantedBuffer)] {
			ch <- v
		}
	}
	if o.wrong == keepsNoBacklog || o.wrong == deliversOnlyTheBacklog {
		close(ch)
		return ch, nil
	}
	o.subs = append(o.subs, ch)
	return ch, nil
}
