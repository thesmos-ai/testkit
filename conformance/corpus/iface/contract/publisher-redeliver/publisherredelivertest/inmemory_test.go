// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// publisher-redeliver is the model tier's under ADR-0028, and the one member of
// the mode family whose `AUTO-PUBLISHER-AT-LEAST-ONCE` runs its redelivery arm:
// the law re-offers the published message through Republish and counts the
// duplicate the mode permits. Its siblings prove the role's omission.
package publisherredelivertest_test

import (
	"context"
	"sync"
	"testing"

	"go.thesmos.sh/testkit"
	publisherredeliver "go.thesmos.sh/testkit/conformance/corpus/iface/contract/publisher-redeliver"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/publisher-redeliver/publisherredelivertest"
)

// TestContractContract runs the generated checks and this package's own.
func TestContractContract(t *testing.T) {
	t.Parallel()

	publisherredelivertest.RunContract(t, inMemory("in-memory"), contractChecks)
}

// TestContractContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestContractContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	publisherredelivertest.RunContract(t,
		inMemory("in-memory"),
		publisherredelivertest.ContractSuite.Without(
			publisherredelivertest.ContractSuite.Checks.Publish.Smoke(),
		),
	)
}

// TestContractChecksCanFail drives every row against its planted defect.
func TestContractChecksCanFail(t *testing.T) {
	t.Parallel()

	publisherredelivertest.ProveContract(t, contractChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(
	name string,
) publisherredelivertest.ContractHarness[*publisherredelivertest.InMemory] {
	return publisherredelivertest.ContractHarness[*publisherredelivertest.InMemory]{
		Name: name, New: publisherredelivertest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------
//
// Two of the three rows read from the stream, so no planted defect here may
// deliver nothing: a check waiting on a channel nobody will send to hangs the
// proof instead of failing it, and a proof that hangs reports nothing. Each one
// delivers the wrong thing rather than none.

var contractChecks = publisherredelivertest.ContractChecks{
	{
		Method: "Republish", Name: "duplicates-what-was-delivered",
		Claim: "Republish duplicates what was already delivered",
		Run:   duplicatesWhatWasDelivered,
		ProvenBy: publisherredelivertest.BrokenContract(
			"a publisher whose redelivery loses the message", planted(redeliversTheZero),
		),
		ProvenReason: "the duplicate at-least-once permits",
	},

	{
		Method: "Subscribe", Name: "receives-what-is-published-after",
		Claim: "Subscribe receives what is published after it attaches",
		Run:   receivesWhatIsPublishedAfter,
		ProvenBy: publisherredelivertest.BrokenContract(
			"a publisher that delivers an empty message", planted(publishesTheZero),
		),
		ProvenReason: "and reaches them",
	},

	{
		Method: "Publish", Name: "reports-an-unreachable-subscriber",
		Claim: "Publish reports a subscriber it can no longer reach",
		Run:   reportsAnUnreachableSubscriber,
		ProvenBy: publisherredelivertest.BrokenContract(
			"a publisher that discards what a subscriber cannot take",
			planted(dropsSilently),
		),
		ProvenReason: "never reported as behind",
	},
}

// --- Bodies -------------------------------------------------------------------

func duplicatesWhatWasDelivered(
	tb testing.TB, s publisherredeliver.Contract,
	fx publisherredelivertest.ContractFixture,
) {
	tb.Helper()
	stream, err := s.Subscribe(tb.Context())
	testkit.NoError(tb, err, "a subscriber attaches")

	testkit.NoError(tb, s.Publish(tb.Context(), fx.Value()), "the original lands")
	testkit.NoError(tb, s.Republish(tb.Context(), fx.Value()), "and so does the redelivery")
	testkit.Equal(tb, <-stream, fx.Value(), "the subscriber takes the original")
	testkit.Equal(tb, <-stream, fx.Value(), "and the duplicate at-least-once permits")
}

func receivesWhatIsPublishedAfter(
	tb testing.TB, s publisherredeliver.Contract,
	fx publisherredelivertest.ContractFixture,
) {
	tb.Helper()
	stream, err := s.Subscribe(tb.Context())
	testkit.NoError(tb, err, "a subscriber attaches")

	testkit.NoError(tb, s.Publish(tb.Context(), fx.ValueOther()),
		"a message published afterwards is accepted")
	testkit.Equal(tb, <-stream, fx.ValueOther(), "and reaches them")
}

// reportsAnUnreachableSubscriber makes the drop loud: a report nothing reaches
// is a report nothing proves, so the reader never reads.
func reportsAnUnreachableSubscriber(
	tb testing.TB, s publisherredeliver.Contract,
	fx publisherredelivertest.ContractFixture,
) {
	tb.Helper()
	_, err := s.Subscribe(tb.Context())
	testkit.NoError(tb, err, "a subscriber attaches")

	for range 32 {
		if err := s.Publish(tb.Context(), fx.Value()); err != nil {
			testkit.ErrorIs(tb, err, publisherredelivertest.ErrFull,
				"the publish says why it could not be taken")
			return
		}
	}
	tb.Fatalf("a subscriber that never reads was never reported as behind")
}

// --- Planted defects ----------------------------------------------------------

// fault names what one planted publisher gets wrong.
type fault int

const (
	// redeliversTheZero re-offers an empty message where the original
	// should have been, which is a redelivery path that lost the payload.
	redeliversTheZero fault = iota

	// publishesTheZero delivers an empty message to an attached subscriber,
	// which is the same bug one method over.
	publishesTheZero

	// dropsSilently discards what a subscriber is too far behind to take,
	// and says the publish succeeded.
	dropsSilently
)

// plantedBuffer is what a planted subscriber holds, small enough that the
// 32-message row overruns it.
const plantedBuffer = 8

// planted builds the constructor for one broken publisher.
func planted(wrong fault) func() *plantedPublisher {
	return func() *plantedPublisher { return &plantedPublisher{wrong: wrong} }
}

type plantedPublisher struct {
	wrong fault
	mu    sync.Mutex
	subs  []chan publisherredeliver.Value
}

func (p *plantedPublisher) Subscribe(
	context.Context,
) (<-chan publisherredeliver.Value, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	ch := make(chan publisherredeliver.Value, plantedBuffer)
	p.subs = append(p.subs, ch)
	return ch, nil
}

func (p *plantedPublisher) Publish(
	_ context.Context, v publisherredeliver.Value,
) error {
	if p.wrong == publishesTheZero {
		v = publisherredeliver.Value{}
	}
	p.send(v)
	return nil
}

func (p *plantedPublisher) Republish(
	_ context.Context, v publisherredeliver.Value,
) error {
	if p.wrong == redeliversTheZero {
		v = publisherredeliver.Value{}
	}
	p.send(v)
	return nil
}

// send never reports a subscriber it could not reach, which is dropsSilently's
// defect and invisible to the other two: no other row fills a buffer.
func (p *plantedPublisher) send(v publisherredeliver.Value) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, ch := range p.subs {
		select {
		case ch <- v:
		default:
		}
	}
}
