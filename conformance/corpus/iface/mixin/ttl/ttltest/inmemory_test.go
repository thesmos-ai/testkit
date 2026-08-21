// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// `ttl expired=ErrExpired` names the sentinel a lapsed read owes, and the row
// below is the only place it can be reached: the generated Read/miss covers the
// key nothing wrote, and this covers the one whose lifetime passed.
package ttltest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
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

	ttltest.ProveMixed(t, mixedChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) ttltest.MixedHarness[*ttltest.InMemory] {
	return ttltest.MixedHarness[*ttltest.InMemory]{
		Name: name, New: ttltest.NewInMemory,
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
