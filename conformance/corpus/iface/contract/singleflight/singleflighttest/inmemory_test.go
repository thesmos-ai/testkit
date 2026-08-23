// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// `singleflight` is the model tier's: that concurrent callers share one compute
// is a claim about an interleaving. The compute below is the row's rather than
// the fixture's — counting the calls is the whole claim, and a derived func
// literal has no counter in it.
package singleflighttest_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/singleflight"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/singleflight/singleflighttest"
)

// TestContractContract runs the generated checks and this package's own.
func TestContractContract(t *testing.T) {
	t.Parallel()

	singleflighttest.RunContract(t, inMemory("in-memory"), contractChecks)
}

// TestContractContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestContractContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	singleflighttest.RunContract(t,
		inMemory("in-memory"),
		singleflighttest.ContractSuite.Without(singleflighttest.ContractSuite.Checks.Run.Smoke()),
	)
}

// TestContractChecksCanFail drives the row against its planted defect.
func TestContractChecksCanFail(t *testing.T) {
	t.Parallel()

	singleflighttest.ProveContract(t, inMemory("in-memory"), contractChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) singleflighttest.ContractHarness[*singleflighttest.InMemory] {
	return singleflighttest.ContractHarness[*singleflighttest.InMemory]{
		Name: name, New: singleflighttest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var contractChecks = singleflighttest.ContractChecks{
	{
		Method: "Run", Name: "computes-once-per-key",
		Claim: "Run computes once per key and shares the answer",
		Run:   computesOncePerKey,
		ProvenBy: singleflighttest.BrokenContract(
			"a subject that runs the compute for every caller", newRunsEveryCaller,
		),
		ProvenReason: "ran the compute once",
	},
}

// --- Bodies -------------------------------------------------------------------

func computesOncePerKey(
	tb testing.TB, s singleflight.Contract, fx singleflighttest.ContractFixture,
) {
	tb.Helper()
	var calls atomic.Int64
	compute := func() string {
		calls.Add(1)
		return computed
	}

	var wg sync.WaitGroup
	for range callers {
		wg.Go(func() {
			got, err := s.Run(tb.Context(), fx.Key(), compute)
			testkit.NoError(tb, err, "a caller is answered")
			testkit.Equal(tb, got, computed, "with the shared answer")
		})
	}
	wg.Wait()

	testkit.Equal(tb, calls.Load(), int64(1),
		"four callers for one key ran the compute once")
}

// --- Planted defects ----------------------------------------------------------

const (
	// callers is how many ask at once. More than one, because with a single
	// caller sharing and not sharing look identical.
	callers = 4

	// computed is what the compute answers, so every caller has the same
	// thing to compare against.
	computed = "computed"
)

// runsEveryCaller answers each caller correctly and computes once per call,
// which is the whole point of the contract missed — and invisible to anything
// that only checks the ANSWER, since every caller gets the right one.
type runsEveryCaller struct{ flights atomic.Int64 }

func newRunsEveryCaller() *runsEveryCaller { return &runsEveryCaller{} }

func (r *runsEveryCaller) Run(
	_ context.Context, _ string, compute func() string,
) (string, error) {
	r.flights.Add(1)
	return compute(), nil
}

func (r *runsEveryCaller) Flights(context.Context) (int, error) {
	return int(r.flights.Load()), nil
}
