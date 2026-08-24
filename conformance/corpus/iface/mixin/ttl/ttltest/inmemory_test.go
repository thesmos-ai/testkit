// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// `ttl expired=ErrExpired` names the sentinel a lapsed read owes, and the row
// below is the only place it can be reached: the generated Read/miss covers the
// key nothing wrote, and this covers the one whose lifetime passed.
package ttltest_test

import (
	"context"
	"testing"
	"time"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/clock"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/ttl"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/ttl/ttltest"
)

// TestMixedContract runs the generated checks and this package's own.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	ttltest.RunMixed(t, inMemory("in-memory"), mixedChecks)
}

// TestMixedContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	ttltest.RunMixed(t,
		inMemory("in-memory"),
		ttltest.MixedSuite.Without(ttltest.MixedSuite.Checks.Put.Smoke()),
	)
}

// TestMixedChecksCanFail drives the row against its planted defect.
func TestMixedChecksCanFail(t *testing.T) {
	t.Parallel()

	ttltest.ProveMixed(t, inMemory("in-memory"), mixedChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) ttltest.MixedHarness[*ttltest.InMemory] {
	return ttltest.MixedHarness[*ttltest.InMemory]{
		Name: name, New: ttltest.NewInMemory,
		// The crash seam. The map outlives the instance holding it, which
		// is what makes a rebuild over it mean anything: an acknowledged
		// write is still there when the process that took it is not.
		Recover: ttltest.Reopen,
		// The aging check advances a clock, and one the subject does not
		// read is one it can advance forever without the subject
		// noticing — so the door is a constructor rather than a setter.
		OnClock: ttltest.NewInMemoryOn,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var mixedChecks = ttltest.MixedChecks{
	{
		Method: "Read", Name: "lapsed-reads-report-the-sentinel",
		Claim: "Read reports the declared sentinel for an entry whose lifetime has passed",
		Run:   lapsedReadsReportTheSentinel,
		ProvenBy: ttltest.BrokenMixed(
			"a store whose entries never lapse", newNeverExpires,
		),
		ProvenReason: "a lapsed read reports the sentinel",
	},
}

// --- Bodies -------------------------------------------------------------------

// lapsedReadsReportTheSentinel writes a live entry first, so the sentinel below
// is about the lapse rather than about an empty store.
//
// The lever is StaleKey: an entry stamped a lifetime ago is exactly what an
// elapsed one looks like, and the expiry arm is the whole claim the directive
// makes.
func lapsedReadsReportTheSentinel(
	tb testing.TB, s ttl.Mixed, fx ttltest.MixedFixture,
) {
	tb.Helper()
	written := fx.Value()
	testkit.NoError(tb, s.Put(tb.Context(), written), "the entry stores")

	got, err := s.Read(tb.Context(), written.Key)
	testkit.NoError(tb, err, "and is within its lifetime")
	testkit.Equal(tb, got.Key, written.Key, "answering under the key it was stored with")

	testkit.NoError(tb, s.Put(tb.Context(),
		ttl.Value{Key: ttltest.StaleKey, Body: "elapsed"}), "the stale entry stores")
	_, err = s.Read(tb.Context(), ttltest.StaleKey)
	testkit.ErrorIs(tb, err, ttl.ErrExpired, "and a lapsed read reports the sentinel")
}

// --- Planted defects ----------------------------------------------------------

// neverExpires keeps everything and serves it forever, which is a cache with
// the eviction pass never scheduled. Every generated check passes against it:
// writes land, live reads answer, and an absent key still misses.
type neverExpires struct{ held map[string]ttl.Value }

func newNeverExpires() *neverExpires { return &neverExpires{held: map[string]ttl.Value{}} }

func (n *neverExpires) Put(_ context.Context, v ttl.Value) error {
	n.held[v.Key] = v
	return nil
}

func (n *neverExpires) Read(_ context.Context, key string) (ttl.Value, error) {
	v, held := n.held[key]
	if !held {
		return ttl.Value{}, ttl.ErrExpired
	}
	return v, nil
}

// --- The Start-shaped clock door ------------------------------------------------

// TestMixedOnAStartedClock runs the same checks against a store built
// through the harness's Start-shaped clock door.
//
// The same implementation, reached the other way. OnClock is for a
// clocked constructor that cannot fail; StartOnClock is for one needing
// the test's lifetime — a background sweeper to stop, a file to remove —
// and it is handed the testing.TB to register that on. Both doors ship
// and only one was ever opened.
func TestMixedOnAStartedClock(t *testing.T) {
	t.Parallel()

	ttltest.RunMixed(t, ttltest.MixedHarness[*ttltest.InMemory]{
		Name: "in-memory on a started clock",
		// Start beside it, because one of the two base doors is always
		// required: the clock door refines how a clocked check builds the
		// subject, it does not replace the constructor.
		Start: func(tb testing.TB) *ttltest.InMemory {
			tb.Helper()
			return startOnClock(tb, clock.NewTestClock(epoch()))
		},
		StartOnClock: startOnClock,
		// The Start-shaped sibling of Recover, for a reopen that also
		// needs the test's lifetime. Its plain form is exercised by the
		// harness above; this is the other door onto the same seam.
		StartRecover: startRecover,
	}, mixedChecks)
}

// startRecover rebuilds over the medium the prior instance left behind
// and registers the new one's teardown.
func startRecover(tb testing.TB, prior *ttltest.InMemory) *ttltest.InMemory {
	tb.Helper()
	s := ttltest.Reopen(prior)
	tb.Cleanup(func() { _ = s })
	return s
}

// epoch is the instant a subject built outside a clocked check starts
// at. The clocked checks hand their own clock over instead.
func epoch() time.Time { return time.Unix(0, 0).UTC() }

// startOnClock builds the store on the run's clock and registers its
// teardown, which is the whole difference from OnClock.
func startOnClock(tb testing.TB, clk *clock.TestClock) *ttltest.InMemory {
	tb.Helper()
	s := ttltest.NewInMemoryOn(clk)
	tb.Cleanup(func() { _ = s })
	return s
}
