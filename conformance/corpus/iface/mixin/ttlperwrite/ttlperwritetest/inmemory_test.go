// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// The per-write lifetime is the point of this fixture. Its sibling `ttl`
// fixes one lifetime for every write through `duration=`; here each entry
// carries its own, which is the shape that gives a defect a field to
// reach for.
package ttlperwritetest_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/clock"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/ttlperwrite"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/ttlperwrite/ttlperwritetest"
	"go.thesmos.sh/testkit/engine/suite"
)

// TestMixedContract runs the generated checks and this package's own.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	ttlperwritetest.RunMixed(t, inMemory("in-memory"), mixedChecks)
}

// TestMixedChecksCanFail drives each row against its planted defect.
func TestMixedChecksCanFail(t *testing.T) {
	t.Parallel()

	ttlperwritetest.ProveMixed(t, inMemory("in-memory"), mixedChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) ttlperwritetest.MixedHarness[*ttlperwritetest.InMemory] {
	return ttlperwritetest.MixedHarness[*ttlperwritetest.InMemory]{
		Name: name, New: ttlperwritetest.NewInMemory,
		// The generated expiry row moves time, and so does the check
		// below; both build the subject through this.
		OnClock: ttlperwritetest.NewInMemoryOn,
		// The map and the clock both outlive the instance holding them,
		// which is what makes a rebuild over them mean anything: an entry
		// written before the crash still expires when it was going to.
		Recover: ttlperwritetest.Reopen,
	}
}

// --- The checks -----------------------------------------------------------

// The claim this fixture exists for, and the one the generated row cannot
// state.
//
// AUTO-TTL-EXPIRY holds every value it draws to ONE lifetime — the longest
// the pool carries — because the law advances once and then asks. A store
// that ignored the field and gave every entry that longest lifetime passes
// it. What separates such a store from a correct one is two entries written
// together and expiring apart, which needs both lifetimes inside one run
// and so needs a body of its own.
var mixedChecks = ttlperwritetest.MixedChecks{
	{
		Name: "each-entry-expires-on-its-own-lifetime",
		Claim: "two entries written together stop being readable at different " +
			"times, each at the lifetime it asked for",
		Class:   suite.ClassClocked,
		Needs:   suite.Caps{suite.CapClock: nil},
		RunWith: eachEntryExpiresOnItsOwn,
		ProvenBy: ttlperwritetest.MixedHarness[*longestWins]{
			Name:    "a Mixed that stretches every lifetime to the longest one it has been handed",
			New:     func() *longestWins { return newLongestWins(clock.NewTestClock(epoch)) },
			OnClock: newLongestWins,
		},
		// The red has to be the short entry outliving its own lifetime.
		// A defect that also broke the long one would redden this while
		// breaking the claim the generated row already owns, and the
		// evidence would be for the wrong check.
		ProvenReason: "and should have expired",
	},
}

// epoch is where every clock in this file starts, so a duration printed in
// a failure is time since the writes rather than since 1970.
var epoch = time.Unix(0, 0).UTC()

// eachEntryExpiresOnItsOwn writes the fixture's two entries, advances past
// the shorter lifetime alone, and asks for both.
func eachEntryExpiresOnItsOwn(
	tb testing.TB, sub ttlperwritetest.MixedSubject, fx ttlperwritetest.MixedFixture,
) {
	tb.Helper()

	// Sorted rather than assumed. Which of the fixture's two entries
	// carries the longer lifetime is the generator's choice, and a check
	// that advanced past the wrong one would assert the reverse of its
	// claim without failing to compile.
	short, long := fx.Entry(), fx.EntryOther()
	if short.Lifetime > long.Lifetime {
		short, long = long, short
	}
	if short.Lifetime == long.Lifetime {
		tb.Fatalf("this check needs two different lifetimes and the fixture "+
			"carries %v twice; nothing here can expire apart", short.Lifetime)
	}

	clk := clock.NewTestClock(epoch)
	s := sub.OnClock(tb, clk)

	// The long-lived entry first, and the order is load-bearing. A store
	// that carries a lifetime over from the previous write looks correct
	// while lifetimes only grow; writing the short one last is what makes
	// it answer for the value in front of it.
	ctx := tb.Context()
	testkit.NoError(tb, s.Put(ctx, long), "the long-lived entry must be storable")
	testkit.NoError(tb, s.Put(ctx, short), "the short-lived entry must be storable")

	// Past the short lifetime and short of the long one, so exactly one of
	// the two is owed an answer. The millisecond of slack is the generated
	// row's, and for its reason: an entry whose lifetime has run out to the
	// nanosecond is a rounding argument rather than a claim.
	clk.Advance(short.Lifetime + time.Millisecond)

	if _, err := s.Read(ctx, short.Key); !errors.Is(err, ttlperwrite.ErrExpired) {
		tb.Errorf("%s asked for %v and should have expired after it; Read answered %v",
			short.Key, short.Lifetime, err)
	}
	got, err := s.Read(ctx, long.Key)
	testkit.NoError(tb, err, fmt.Sprintf("%s asked for %v and has %v of it left to run",
		long.Key, long.Lifetime, long.Lifetime-short.Lifetime))
	testkit.Equal(tb, got.Body, long.Body,
		"the entry still inside its lifetime must read back what was written")
}

// longestWins is the implementation the check is proven against: correct
// but for stretching every lifetime to the longest one it has been handed.
//
// It stores through the real fake, so everything else about it — the
// context handling, the miss sentinel, what a read answers — is right, and
// the only claim it can redden is the one it was built to break. A double
// with nothing behind it would keep no entries at all, and a check about
// which of two entries survives has nothing to observe.
type longestWins struct {
	inner   *ttlperwritetest.InMemory
	longest time.Duration
}

func newLongestWins(clk clock.Clock) *longestWins {
	return &longestWins{inner: ttlperwritetest.NewInMemoryOn(clk)}
}

func (s *longestWins) Put(ctx context.Context, e ttlperwrite.Entry) error {
	s.longest = max(s.longest, e.Lifetime)
	e.Lifetime = s.longest
	return s.inner.Put(ctx, e)
}

func (s *longestWins) Read(ctx context.Context, key string) (ttlperwrite.Entry, error) {
	return s.inner.Read(ctx, key)
}
