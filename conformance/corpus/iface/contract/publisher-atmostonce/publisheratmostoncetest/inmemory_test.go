// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// publisher is the model tier's under ADR-0028: `AUTO-PUBLISHER-DELIVERS` and
// the three delivery-guarantee laws state it.
//
// Delivery is what this contract shares with `outbox`, and the difference is
// what neither signature shows: an outbox holds a record until somebody reads
// it, a publisher delivers to whoever is listening. The suite tier owns the
// outbox half because no law states durability; this half is already covered.
package publisheratmostoncetest_test

import (
	"context"
	"sync"
	"testing"

	"go.thesmos.sh/testkit"
	publisheratmostonce "go.thesmos.sh/testkit/conformance/corpus/iface/contract/publisher-atmostonce"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/publisher-atmostonce/publisheratmostoncetest"
)

// TestContractContract runs the generated checks and this package's own.
func TestContractContract(t *testing.T) {
	t.Parallel()

	publisheratmostoncetest.RunContract(t, inMemory("in-memory"), contractChecks)
}

// TestContractContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestContractContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	publisheratmostoncetest.RunContract(t,
		inMemory("in-memory"),
		publisheratmostoncetest.ContractSuite.Without(
			publisheratmostoncetest.ContractSuite.Checks.Publish.Smoke(),
		),
	)
}

// TestContractChecksCanFail drives every row against its planted defect.
func TestContractChecksCanFail(t *testing.T) {
	t.Parallel()

	publisheratmostoncetest.ProveContract(t, contractChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(
	name string,
) publisheratmostoncetest.ContractHarness[*publisheratmostoncetest.InMemory] {
	return publisheratmostoncetest.ContractHarness[*publisheratmostoncetest.InMemory]{
		Name: name, New: publisheratmostoncetest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var contractChecks = publisheratmostoncetest.ContractChecks{
	{
		Method: "Subscribe", Name: "receives-what-is-published-after",
		Claim: "Subscribe receives what is published after it attaches",
		Run:   receivesWhatIsPublishedAfter,
		ProvenBy: publisheratmostoncetest.BrokenContract(
			"a publisher that delivers an empty message", planted(publishesTheZero),
		),
		ProvenReason: "and reaches them",
	},

	{
		Method: "Publish", Name: "reports-an-unreachable-subscriber",
		Claim: "Publish reports a subscriber it can no longer reach",
		Run:   reportsAnUnreachableSubscriber,
		ProvenBy: publisheratmostoncetest.BrokenContract(
			"a publisher that discards what a subscriber cannot take",
			planted(dropsSilently),
		),
		ProvenReason: "never reported as behind",
	},
}

// --- Bodies -------------------------------------------------------------------

func receivesWhatIsPublishedAfter(
	tb testing.TB, s publisheratmostonce.Contract,
	fx publisheratmostoncetest.ContractFixture,
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
	tb testing.TB, s publisheratmostonce.Contract,
	fx publisheratmostoncetest.ContractFixture,
) {
	tb.Helper()
	_, err := s.Subscribe(tb.Context())
	testkit.NoError(tb, err, "a subscriber attaches")

	for range 32 {
		if err := s.Publish(tb.Context(), fx.Value()); err != nil {
			testkit.ErrorIs(tb, err, publisheratmostoncetest.ErrFull,
				"the publish says why it could not be taken")
			return
		}
	}
	tb.Fatalf("a subscriber that never reads was never reported as behind")
}

// --- Planted defects ----------------------------------------------------------

// fault names what one planted publisher gets wrong.
//
// Neither delivers nothing: the first row reads from the stream, and a check
// waiting on a channel nobody will send to hangs the proof instead of failing
// it. Delivering the wrong thing fails; delivering nothing does not report.
type fault int

const (
	// publishesTheZero delivers an empty message to an attached subscriber,
	// which is a delivery path that lost the payload.
	publishesTheZero fault = iota

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
	subs  []chan publisheratmostonce.Value
}

func (p *plantedPublisher) Subscribe(
	context.Context,
) (<-chan publisheratmostonce.Value, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	ch := make(chan publisheratmostonce.Value, plantedBuffer)
	p.subs = append(p.subs, ch)
	return ch, nil
}

// Publish never reports a subscriber it could not reach, which is
// dropsSilently's defect and invisible to the other row, which fills nothing.
func (p *plantedPublisher) Publish(
	_ context.Context, v publisheratmostonce.Value,
) error {
	if p.wrong == publishesTheZero {
		v = publisheratmostonce.Value{}
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, ch := range p.subs {
		select {
		case ch <- v:
		default:
		}
	}
	return nil
}
