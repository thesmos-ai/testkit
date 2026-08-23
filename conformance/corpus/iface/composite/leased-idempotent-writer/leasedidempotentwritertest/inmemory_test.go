// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// The two classifications make opposite demands of Acquire, and this fixture
// exists because a generator reading them independently produces a suite that
// fails a correct implementation.
//
// `lease` is the model tier's under ADR-0028 and so is `idempotent`, so neither
// generates a check today — which means nothing here would notice the conflict.
// What the extension point can carry is what a single subject can be asked, and
// every row below plants the implementation it exists to reject. The rest —
// that a release frees one key and not the others — needs a second holder and
// is a package test, because a lease is refused only when somebody else has it.
package leasedidempotentwritertest_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"go.thesmos.sh/testkit"
	leasedidempotentwriter "go.thesmos.sh/testkit/conformance/corpus/iface/composite/leased-idempotent-writer"
	"go.thesmos.sh/testkit/conformance/corpus/iface/composite/leased-idempotent-writer/leasedidempotentwritertest"
)

// TestLeasedWriterContract runs the generated checks and this package's own
// against both subjects.
func TestLeasedWriterContract(t *testing.T) {
	t.Parallel()

	leasedidempotentwritertest.RunLeasedWriter(t,
		inMemory("in-memory"),
		contended("in-memory, contended"),
		leasedWriterChecks,
	)
}

// TestLeasedWriterContractWithoutSmoke drops a check through the typed index
// rather than a string, so a check that is renamed or stops being emitted
// breaks this compile instead of silently declining nothing.
func TestLeasedWriterContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	leasedidempotentwritertest.RunLeasedWriter(t,
		inMemory("in-memory"),
		leasedidempotentwritertest.LeasedWriterSuite.Without(
			leasedidempotentwritertest.LeasedWriterSuite.Checks.Acquire.Smoke(),
		),
	)
}

// TestLeasedWriterChecksCanFail drives every row against its planted defect.
//
// Without it the rows above are NoErrors, which a subject whose methods return
// nil satisfies — and the composite's whole question would be asked of nothing.
func TestLeasedWriterChecksCanFail(t *testing.T) {
	t.Parallel()

	leasedidempotentwritertest.ProveLeasedWriter(t, inMemory("in-memory"), leasedWriterChecks)
}

// --- Harnesses ---------------------------------------------------------------

// contendedKey is the key the contended subject's registry already holds.
//
// Distinct from the fixture's derived key, so the harness can still seed every
// subject through Acquire: a contended subject that refused the seed would fail
// every check before reaching the one it exists for.
const contendedKey = "held-by-another"

// inMemory is the lone holder, in a registry nobody else contends over.
func inMemory(
	name string,
) leasedidempotentwritertest.LeasedWriterHarness[*leasedidempotentwritertest.InMemory] {
	return leasedidempotentwritertest.LeasedWriterHarness[*leasedidempotentwritertest.InMemory]{
		Name: name, New: leasedidempotentwritertest.NewInMemory,
	}
}

// contended is a holder whose registry another already has the key in.
//
// A lease is refused only when somebody else holds it, and a row receives one
// subject — so the refusal is reached by seating an incumbent first.
func contended(
	name string,
) leasedidempotentwritertest.LeasedWriterHarness[*leasedidempotentwritertest.InMemory] {
	return leasedidempotentwritertest.LeasedWriterHarness[*leasedidempotentwritertest.InMemory]{
		Name: name, Start: seatAnIncumbent,
	}
}

// seatAnIncumbent hands back a holder contending against one that already took
// contendedKey. Start rather than New, because the incumbent's acquire needs
// the test's context and its failure is the test's to report.
func seatAnIncumbent(tb testing.TB) *leasedidempotentwritertest.InMemory {
	tb.Helper()
	r := leasedidempotentwritertest.NewRegistry()
	if err := r.Holder().Acquire(tb.Context(), contendedKey); err != nil {
		tb.Fatalf("seat the incumbent that makes the contention real: %v", err)
	}
	return r.Holder()
}

// --- The checks: claims, bodies and defects, by name --------------------------
//
// None of these could have been generated: both classifications are the model
// tier's, and what a single subject can be asked about their interaction is a
// claim about this domain rather than about the interface's shape.

var leasedWriterChecks = leasedidempotentwritertest.LeasedWriterChecks{
	{
		Method: "Acquire", Name: "loses-a-key-another-took",
		Claim: "Acquire loses a key another holder took",
		Run:   losesAKeyAnotherTook,
		ProvenBy: leasedidempotentwritertest.BrokenLeasedWriter(
			"a lease that refuses without saying who to", newVagueLease,
		),
		ProvenReason: "says who to",
	},

	{
		Method: "Acquire", Name: "repeats-without-unbalancing",
		Claim: "Acquire repeats without unbalancing the lease",
		Run:   repeatsWithoutUnbalancing,
		ProvenBy: leasedidempotentwritertest.BrokenLeasedWriter(
			"a lease with no idempotence", newPlainLease,
		),
		ProvenReason: "re-acquiring is a no-op",
	},

	{
		Method: "Release", Name: "tolerates-an-unheld-key",
		Claim: "Release tolerates a key nobody holds",
		Run:   toleratesAnUnheldKey,
		ProvenBy: leasedidempotentwritertest.BrokenLeasedWriter(
			"a lease strict about releasing", newStrictLease,
		),
		ProvenReason: "releasing what was never held",
	},
}

// --- Bodies -------------------------------------------------------------------

// losesAKeyAnotherTook gets the same (tb, s, fx) triple every generated
// assertion takes.
//
// True of the contended subject and vacuous for the lone one, which is the
// shape a two-subject claim takes when only one of them can be in the losing
// state. It names contendedKey rather than drawing from fx because the losing
// state is the one thing the fixture cannot seed.
func losesAKeyAnotherTook(
	tb testing.TB,
	s leasedidempotentwriter.LeasedWriter,
	_ leasedidempotentwritertest.LeasedWriterFixture,
) {
	tb.Helper()
	if err := s.Acquire(tb.Context(), contendedKey); err != nil {
		testkit.ErrorIs(tb, err, leasedidempotentwriter.ErrHeld,
			"an acquire that loses says who to")
		return
	}
	testkit.NoError(tb, s.Release(tb.Context(), contendedKey),
		"and one that wins can give it back")
}

// repeatsWithoutUnbalancing is the whole of the composite.
//
// The row acquires first, so these are the repeats `idempotent` asks for — and
// one release still has to settle them, which is what `lease` asks for. The
// implementation it rejects is a plain lease: one that refuses the second
// acquire, which is correct for the contract alone and wrong for the pair.
func repeatsWithoutUnbalancing(
	tb testing.TB,
	s leasedidempotentwriter.LeasedWriter,
	fx leasedidempotentwritertest.LeasedWriterFixture,
) {
	tb.Helper()
	ctx, key := tb.Context(), fx.Key()

	testkit.NoError(tb, s.Acquire(ctx, key), "the key is taken")
	testkit.NoError(tb, s.Acquire(ctx, key), "re-acquiring is a no-op")
	testkit.NoError(tb, s.Acquire(ctx, key), "however often it happens")

	testkit.NoError(tb, s.Release(ctx, key), "one release settles them")
	testkit.NoError(tb, s.Acquire(ctx, key),
		"and the key is free again rather than still held")
}

// toleratesAnUnheldKey holds the shutdown path usable.
//
// A caller deferring Release and returning early on a failed Acquire is
// ordinary Go, and the implementation this rejects is one strict about it —
// which reports a failure to give up something never taken, on every path that
// did not get the lease.
func toleratesAnUnheldKey(
	tb testing.TB,
	s leasedidempotentwriter.LeasedWriter,
	fx leasedidempotentwritertest.LeasedWriterFixture,
) {
	tb.Helper()
	// KeyOther rather than Key: the fixture's second value is different by
	// construction, which is what makes "nobody holds it" true without this
	// body having to know what the run seeded.
	testkit.NoError(tb, s.Release(tb.Context(), fx.KeyOther()),
		"releasing what was never held is not a failure")
}

// --- Planted defects ----------------------------------------------------------
//
// Each is a whole lease rather than the generated double, because what these
// break is the interaction between two methods — a double delegating to a
// correct store everywhere but one method cannot express "no idempotence".

// plainLease is the lease contract without the mixin: correct on its own, and
// wrong for a method carrying both. It refuses the second acquire, which is
// what repeats-without-unbalancing exists to reject.
type plainLease struct {
	mu   sync.Mutex
	held map[string]bool
}

func newPlainLease() *plainLease { return &plainLease{held: map[string]bool{}} }

func (s *plainLease) Acquire(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.held[key] {
		return leasedidempotentwriter.ErrHeld
	}
	s.held[key] = true
	return nil
}

func (s *plainLease) Release(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.held, key)
	return nil
}

// strictLease refuses to release a key it does not hold, which takes down every
// caller whose deferred shutdown runs after a failed acquire.
type strictLease struct{ plainLease }

func newStrictLease() *strictLease {
	s := &strictLease{}
	s.held = map[string]bool{}
	return s
}

func (s *strictLease) Release(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.held[key] {
		return leasedidempotentwriter.ErrHeld
	}
	delete(s.held, key)
	return nil
}

// vagueLease loses every race and will not say to whom, which leaves a caller
// unable to tell a lost lease from a broken store — the near miss
// loses-a-key-another-took exists to reject.
type vagueLease struct{}

func newVagueLease() *vagueLease { return &vagueLease{} }

func (vagueLease) Acquire(context.Context, string) error {
	return errors.New("vaguelease: cannot have it")
}

func (vagueLease) Release(context.Context, string) error { return nil }
