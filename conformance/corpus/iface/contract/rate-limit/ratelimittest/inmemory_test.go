// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Two subjects, because the refusal has to be reachable from either side: the
// generous one spends its burst first, the spent one is refused on its first
// call. What is asserted is that the refusal comes, and that it says the rate
// was the reason.
package ratelimittest_test

import (
	"context"
	"testing"
	"time"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/clock"
	ratelimit "go.thesmos.sh/testkit/conformance/corpus/iface/contract/rate-limit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/rate-limit/ratelimittest"
)

// TestContractContract runs the generated checks and this package's own.
func TestContractContract(t *testing.T) {
	t.Parallel()

	ratelimittest.RunContract(t, generous("in-memory"), spent("in-memory, one token on a clock already moved"),
		contractChecks)
}

// TestContractContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestContractContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	ratelimittest.RunContract(t,
		generous("in-memory"),
		ratelimittest.ContractSuite.Without(ratelimittest.ContractSuite.Checks.Run.Smoke()),
	)
}

// TestContractChecksCanFail drives the row against its planted defect.
func TestContractChecksCanFail(t *testing.T) {
	t.Parallel()

	ratelimittest.ProveContract(t, generous("in-memory"), contractChecks)
}

// --- Harnesses ---------------------------------------------------------------

// The limiter's terms. The burst is what the generous subject may spend before
// it is refused, and the row walks one past it — so the two numbers are one
// decision and live together.
const (
	burst  = 10
	spends = burst + 1
	period = time.Second
)

// origin is where every test clock starts. Fixed rather than the wall, because
// a limiter is about elapsed time and a run that read the wall would refill by
// however long the test took.
var origin = time.Unix(0, 0)

func generous(name string) ratelimittest.ContractHarness[*ratelimittest.InMemory] {
	return ratelimittest.ContractHarness[*ratelimittest.InMemory]{
		Name: name,
		New: func() *ratelimittest.InMemory {
			return ratelimittest.NewInMemory(clock.NewTestClock(origin), burst, period)
		},
	}
}

// spent holds one token and a clock already moved past its refill, which is the
// state a limiter reaches after a quiet period rather than a busy one.
func spent(name string) ratelimittest.ContractHarness[*ratelimittest.InMemory] {
	return ratelimittest.ContractHarness[*ratelimittest.InMemory]{
		Name: name,
		New: func() *ratelimittest.InMemory {
			clk := clock.NewTestClock(origin)
			s := ratelimittest.NewInMemory(clk, 1, period)
			clk.Advance(2 * period)
			return s
		},
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var contractChecks = ratelimittest.ContractChecks{
	{
		Method: "Run", Name: "refuses-a-caller-with-nothing-left",
		Claim: "Run refuses a caller with nothing left",
		Run:   refusesACallerWithNothingLeft,
		ProvenBy: ratelimittest.BrokenContract(
			"a limiter that refuses for a reason of its own", newRefusesWithoutSaying,
		),
		ProvenReason: "says the rate was the reason",
	},
}

// --- Bodies -------------------------------------------------------------------

// refusesACallerWithNothingLeft spends until refused, which both subjects
// reach: the generous one after its burst, the spent one on its first call.
func refusesACallerWithNothingLeft(
	tb testing.TB, s ratelimit.Contract, fx ratelimittest.ContractFixture,
) {
	tb.Helper()
	for range spends {
		err := s.Run(tb.Context(), fx.Key())
		if err == nil {
			continue
		}
		testkit.ErrorIs(tb, err, ratelimittest.ErrLimited,
			"a refusal says the rate was the reason")
		return
	}
	tb.Fatalf("a limiter that never refuses bounds nothing")
}

// --- Planted defects ----------------------------------------------------------

// refusesWithoutSaying bounds the caller and reports something else, which
// leaves a client unable to tell "slow down" from "this is broken" — and so
// unable to decide whether retrying is worth anything.
//
// A limiter that never refused would red this row too, at the loop's own
// Fatalf. This one is the sharper defect: it refuses correctly and only the
// REASON is wrong, which is what ProvenReason pins.
type refusesWithoutSaying struct{ spent int }

func newRefusesWithoutSaying() *refusesWithoutSaying { return &refusesWithoutSaying{} }

func (r *refusesWithoutSaying) Run(_ context.Context, _ string) error {
	r.spent++
	if r.spent > burst {
		return context.DeadlineExceeded
	}
	return nil
}
