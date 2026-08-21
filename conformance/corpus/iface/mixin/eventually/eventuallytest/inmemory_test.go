// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package eventuallytest_test

import (
	"context"
	"errors"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/eventually"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/eventually/eventuallytest"
)

// eventually is the model tier's — AUTO-EVENTUAL-CONVERGENCE states it — so the
// suite generates the signature family alone.
//
// eidos held it out of the relational set deliberately: "observable eventually"
// raises *observable by what*, the same question `sideeffect` raises, and
// whether it joins that vocabulary is an open call. This fixture answers it
// structurally instead — Settle is the seam, so convergence is driven rather
// than waited for, and no clock is involved.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	eventuallytest.RunMixed(t,
		eventuallytest.MixedHarness[*eventuallytest.InMemory]{Name: "in-memory", New: eventuallytest.NewInMemory},
	)
}

// Dropping a check is written against the typed index rather than a string, so
// a check that is renamed or stops being emitted breaks this compile instead of
// silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	eventuallytest.RunMixed(t,
		eventuallytest.MixedHarness[*eventuallytest.InMemory]{Name: "in-memory", New: eventuallytest.NewInMemory},
		eventuallytest.MixedSuite.Without(eventuallytest.MixedSuite.Checks.Publish.Smoke()),
	)
}

// unreadablePeer answers no items — the peer whose failure Sync must carry
// out rather than half-apply.
type unreadablePeer struct{ eventually.Mixed }

func (unreadablePeer) Items(context.Context) ([]string, error) {
	return nil, errors.New("eventuallytest_test: unreadable")
}

// A peer that cannot be read is a sync that reports, not one that guesses:
// nothing lands from a partial exchange.
//
// Written outside the run because Sync takes another Mixed, which no literal
// can be written for — the header refuses both its families and says so.
func TestSyncCarriesThePeersFailure(t *testing.T) {
	t.Parallel()

	replica := eventuallytest.NewInMemory()
	testkit.Error(t, replica.Sync(t.Context(), unreadablePeer{}),
		"the unreadable peer's failure is the sync's answer")

	items, err := replica.Items(t.Context())
	testkit.NoError(t, err, "the replica is still readable")
	testkit.Len(t, items, 0, "and nothing landed from the failed exchange")
}
