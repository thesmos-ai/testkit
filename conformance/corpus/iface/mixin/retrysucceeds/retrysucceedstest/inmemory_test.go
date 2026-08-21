// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// retrysucceeds is the suite tier's under ADR-0028 and generates no check — the
// header records the gap — because the mixin names no attempt count.
//
// "Succeeds within the declared attempts" is not a statement until a number is
// declared, the same reason `timeout` is gated on its `duration` rather than on
// the mixin. Without one, the only generatable check is "call it and expect
// success", which is the smoke check under another name and would fail this
// subject, whose first attempts are meant to fail.
package retrysucceedstest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/retrysucceeds"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/retrysucceeds/retrysucceedstest"
)

// TestMixedContract runs the generated checks and this package's own.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	retrysucceedstest.RunMixed(t, inMemory("in-memory"), mixedChecks)
}

// TestMixedContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	retrysucceedstest.RunMixed(t,
		inMemory("in-memory"),
		retrysucceedstest.MixedSuite.Without(retrysucceedstest.MixedSuite.Checks.Call.Smoke()),
	)
}

// TestMixedChecksCanFail drives the row against its planted defect.
func TestMixedChecksCanFail(t *testing.T) {
	t.Parallel()

	retrysucceedstest.ProveMixed(t, mixedChecks)
}

// --- Harnesses ---------------------------------------------------------------

// maxAttempts bounds the caller's loop. A bound rather than an open loop: the
// mixin promises the retry terminates, and a test that trusted it would hang
// against a subject that does not.
const maxAttempts = 8

func inMemory(name string) retrysucceedstest.MixedHarness[*retrysucceedstest.InMemory] {
	return retrysucceedstest.MixedHarness[*retrysucceedstest.InMemory]{
		Name: name, New: retrysucceedstest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var mixedChecks = retrysucceedstest.MixedChecks{
	{
		Method: "Call", Name: "succeeds-within-the-retries",
		Claim: "Call succeeds once a caller retries it",
		Run:   succeedsWithinTheRetries,
		ProvenBy: retrysucceedstest.BrokenMixed(
			"a subject that never needed retrying", newSucceedsFirstTime,
		),
		ProvenReason: "more than one attempt was needed",
	},
}

// --- Bodies -------------------------------------------------------------------

// succeedsWithinTheRetries is what a caller of this interface writes, and what
// the mixin promises will terminate.
//
// It asserts the attempt count as well as the success, because a subject that
// answered on the first call satisfies the loop while making the mixin's claim
// about nothing.
func succeedsWithinTheRetries(
	tb testing.TB, s retrysucceeds.Mixed, fx retrysucceedstest.MixedFixture,
) {
	tb.Helper()
	testkit.NoError(tb, retryUntilSuccess(tb.Context(), s, fx.Key()),
		"a bounded retry loop gets an answer")

	n, err := s.Attempts(tb.Context())
	testkit.NoError(tb, err, "the attempt count is readable")
	testkit.True(tb, n > 1, "and more than one attempt was needed")
}

// retryUntilSuccess is the caller's loop, bounded.
func retryUntilSuccess(
	ctx context.Context, subject retrysucceeds.Mixed, key string,
) error {
	var err error
	for range maxAttempts {
		if err = subject.Call(ctx, key); err == nil {
			return nil
		}
	}
	return err
}

// --- Planted defects ----------------------------------------------------------

// succeedsFirstTime answers cleanly on the first call, which passes the loop
// above and leaves the retry unexercised — the reason the row reads the attempt
// count rather than stopping at the error.
type succeedsFirstTime struct{ calls int }

func newSucceedsFirstTime() *succeedsFirstTime { return &succeedsFirstTime{} }

func (s *succeedsFirstTime) Call(context.Context, string) error {
	s.calls++
	return nil
}

func (s *succeedsFirstTime) Attempts(context.Context) (int, error) {
	return s.calls, nil
}

// TestMixedLawsCanSaturate drives each bound law against defects worn on
// its own methods, with that law as the run's only oracle.
//
// Binding a law is necessary; this is what makes it sufficient. A law
// every worn defect survives is bound and unsaturatable, which reads as
// coverage in the report and is not.
func TestMixedLawsCanSaturate(t *testing.T) {
	t.Parallel()

	retrysucceedstest.MixedModelSaturation(t, func() retrysucceedstest.Mixed { return retrysucceedstest.NewInMemory() })
}
