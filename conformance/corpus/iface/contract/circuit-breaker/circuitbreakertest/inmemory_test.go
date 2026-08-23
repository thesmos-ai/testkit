// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// The lever is the key, so the claim is statable through the interface after
// all: a breaker guards something the caller names, and asking for the unwell
// one is how a run induces the failure.
package circuitbreakertest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	circuitbreaker "go.thesmos.sh/testkit/conformance/corpus/iface/contract/circuit-breaker"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/circuit-breaker/circuitbreakertest"
)

// TestContractContract runs the generated checks and this package's own.
func TestContractContract(t *testing.T) {
	t.Parallel()

	circuitbreakertest.RunContract(t, inMemory("in-memory"), contractChecks)
}

// TestContractContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestContractContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	circuitbreakertest.RunContract(t,
		inMemory("in-memory"),
		circuitbreakertest.ContractSuite.Without(circuitbreakertest.ContractSuite.Checks.Run.Smoke()),
	)
}

// TestContractChecksCanFail drives the row against its planted defect.
func TestContractChecksCanFail(t *testing.T) {
	t.Parallel()

	circuitbreakertest.ProveContract(t, inMemory("in-memory"), contractChecks)
}

// --- Harnesses ---------------------------------------------------------------

// threshold is how many failures the breaker tolerates before it opens. It is
// the subject's to declare and the row's to walk to, so both name it here.
const threshold = 3

func inMemory(
	name string,
) circuitbreakertest.ContractHarness[*circuitbreakertest.InMemory] {
	return circuitbreakertest.ContractHarness[*circuitbreakertest.InMemory]{
		Name: name,
		New:  func() *circuitbreakertest.InMemory { return circuitbreakertest.NewInMemory(threshold) },
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var contractChecks = circuitbreakertest.ContractChecks{
	{
		Method: "Run", Name: "stops-calling-a-failing-downstream",
		Claim: "Run stops calling a downstream that keeps failing",
		Run:   stopsCallingAFailingDownstream,
		ProvenBy: circuitbreakertest.BrokenContract(
			"a breaker that never opens", newNeverOpens,
		),
		ProvenReason: "refused by the breaker",
	},
}

// --- Bodies -------------------------------------------------------------------

func stopsCallingAFailingDownstream(
	tb testing.TB, s circuitbreaker.Contract, fx circuitbreakertest.ContractFixture,
) {
	tb.Helper()
	for range threshold {
		testkit.ErrorIs(tb, s.Run(tb.Context(), circuitbreakertest.UnwellKey),
			circuitbreakertest.ErrDownstream,
			"a call under the threshold reaches the downstream")
	}
	testkit.ErrorIs(tb, s.Run(tb.Context(), circuitbreakertest.UnwellKey),
		circuitbreakertest.ErrOpen,
		"and the call after it is refused by the breaker")

	testkit.NoError(tb, s.Run(tb.Context(), fx.Key()),
		"while a healthy downstream is still reachable")
}

// --- Planted defects ----------------------------------------------------------

// neverOpens forwards every call and reports the downstream's own failure
// forever, which is a breaker with the counter but no threshold — and which
// looks correct for as long as anybody only checks that a failure is reported.
type neverOpens struct{}

func newNeverOpens() neverOpens { return neverOpens{} }

func (neverOpens) Run(_ context.Context, key string) error {
	if key == circuitbreakertest.UnwellKey {
		return circuitbreakertest.ErrDownstream
	}
	return nil
}
