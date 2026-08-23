// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// pool is the model tier's under ADR-0028: `AUTO-POOL-BALANCED` and
// `AUTO-POOL-LEAK-FREE` state it, and both are claims about a sequence rather
// than about a call.
//
// The row below is the bound, stated through the interface: it puts one value
// in and asks for two.
package pooltest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/pool"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/pool/pooltest"
	"go.thesmos.sh/testkit/engine/model"
	"go.thesmos.sh/testkit/engine/suite"
)

// TestContractContract runs the generated checks and this package's own.
func TestContractContract(t *testing.T) {
	t.Parallel()

	pooltest.RunContract(t, inMemory("in-memory"), contractChecks)
}

// TestContractContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestContractContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	pooltest.RunContract(t,
		inMemory("in-memory"),
		pooltest.ContractSuite.Without(pooltest.ContractSuite.Checks.Get.Smoke()),
	)
}

// TestContractChecksCanFail drives the row against its planted defect.
func TestContractChecksCanFail(t *testing.T) {
	t.Parallel()

	pooltest.ProveContract(t, inMemory("in-memory"), contractChecks)
}

// --- Harnesses ---------------------------------------------------------------

// inMemory is a pool holding exactly one value.
//
// One, because that is the bound every claim here is about: the row below
// asks for a second and must be refused, and the model tier's cycle takes
// the value and hands it back, which states nothing at all against a pool
// that had none to take. The constructor is variadic, so the harness closes
// over it rather than naming it.
func inMemory(name string) pooltest.ContractHarness[*pooltest.InMemory] {
	return pooltest.ContractHarness[*pooltest.InMemory]{
		Name: name,
		New:  func() *pooltest.InMemory { return pooltest.NewInMemory(held) },
		// The balance laws read the pool's own accounting. Which method
		// carries it, and which of its fields are the two counts and the
		// held total, is a fact about this declaration rather than about
		// its shape — so the harness answers it.
		Provide: map[suite.Capability]any{
			"stats":    poolStats,
			"balanced": poolBalanced,
		},
	}
}

// poolStats is the accounting AUTO-POOL-BALANCED compares.
//
// Zeros on a refused observation, which the law reads as a pool at rest.
// The alternative is failing the law for a Stats that errored, and a
// refusal to answer is not an imbalance: a pool whose Stats genuinely
// errors fails Stats/smoke, which is where that belongs.
func poolStats(rt *model.T, p pool.Contract) (gets, puts, outstanding int) {
	s, err := p.Stats(rt.Context())
	if err != nil {
		return 0, 0, 0
	}
	return s.Gets, s.Puts, s.Outstanding
}

// poolBalanced is what AUTO-POOL-LEAK-FREE asks at quiescence: nothing is
// still held. Read through poolStats so the two rows cannot disagree about
// where the number comes from.
func poolBalanced(rt *model.T, p pool.Contract) bool {
	_, _, outstanding := poolStats(rt, p)
	return outstanding == 0
}

// --- The checks: claims, bodies and defects, by name --------------------------

// held is the one value the pool starts with, named so the row and the
// harness cannot disagree about how many are in there.
var held = pool.Value{Key: "held", Body: "the pool's one value"}

var contractChecks = pooltest.ContractChecks{
	{
		Method: "Get", Name: "hands-out-what-it-holds",
		Claim: "Get hands out what it holds and no more",
		Run:   handsOutWhatItHolds,
		ProvenBy: pooltest.BrokenContract(
			"a pool with no bound", newUnboundedPool,
		),
		ProvenReason: "the pool it came from is then empty",
	},
}

// --- Bodies -------------------------------------------------------------------

// handsOutWhatItHolds is the pool's bound, stated through the interface.
//
// The refusal in the middle is what makes it a check rather than a smoke:
// the pool holds one value, so a subject that manufactures on demand — or
// one whose methods return nil and nothing else — answers the second Get
// too. It ends by returning the value and taking it again, because a pool
// that refused everything would satisfy the refusal and hand out nothing.
func handsOutWhatItHolds(
	tb testing.TB, s pool.Contract, _ pooltest.ContractFixture,
) {
	tb.Helper()
	got, err := s.Get(tb.Context())
	testkit.NoError(tb, err, "the value it holds is available")

	_, err = s.Get(tb.Context())
	testkit.Error(tb, err, "and the pool it came from is then empty")

	testkit.NoError(tb, s.Put(tb.Context(), got), "returning it succeeds")
	_, err = s.Get(tb.Context())
	testkit.NoError(tb, err, "and the pool can hand it out again")
}

// --- Planted defects ----------------------------------------------------------

// unboundedPool hands out a fresh value however often it is asked, which is a
// constructor rather than a pool.
//
// The implementation every check on Get exists to reject: it satisfies any
// assertion that a Get succeeded, and provides no limit at all.
type unboundedPool struct{}

func newUnboundedPool() unboundedPool { return unboundedPool{} }

func (unboundedPool) Get(context.Context) (pool.Value, error) { return pool.Value{}, nil }
func (unboundedPool) Put(context.Context, pool.Value) error   { return nil }

// Stats lies the way the rest of the stand-in does: permanently balanced
// numbers from a pool that bounds nothing.
func (unboundedPool) Stats(context.Context) (pool.Stats, error) { return pool.Stats{}, nil }
