// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Two interfaces, two runs, one implementation answering to both — which is the
// arrangement rather than an accident: a merge needs a peer, and the peer is a
// contract of its own precisely so a merge cannot reach into it.
//
// crdtmerge is the model tier's — AUTO-CRDT-MERGE states it — so the suite
// generates the signature family alone. The assignment is right for a reason
// this fixture shows plainly: convergence is a statement about two merges in
// opposite orders, and there is no single call that makes it.
package crdtmergetest_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/crdtmerge"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/crdtmerge/crdtmergetest"
)

// TestMixedContract runs the generated checks and this package's own.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	crdtmergetest.RunMixed(t, inMemoryMixed("in-memory"), mixedChecks)
}

// TestMixedContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	crdtmergetest.RunMixed(t,
		inMemoryMixed("in-memory"),
		crdtmergetest.MixedSuite.Without(crdtmergetest.MixedSuite.Checks.Add.Smoke()),
	)
}

// TestMixedChecksCanFail drives every row against its planted defect.
func TestMixedChecksCanFail(t *testing.T) {
	t.Parallel()

	crdtmergetest.ProveMixed(t, mixedChecks)
}

// TestReplicaContract runs the peer, which is a contract in its own right — and
// one implementation answers to both, which is what lets a merge read through
// the interface rather than reaching into a type it happens to know.
func TestReplicaContract(t *testing.T) {
	t.Parallel()

	crdtmergetest.RunReplica(t, inMemoryReplica("in-memory"))
}

// --- Harnesses ---------------------------------------------------------------

// theirItem is what the peer holds, so a fold has something of its own to
// arrive and something of the subject's to survive.
const theirItem = "theirs"

func inMemoryMixed(name string) crdtmergetest.MixedHarness[*crdtmergetest.InMemory] {
	return crdtmergetest.MixedHarness[*crdtmergetest.InMemory]{
		Name: name, New: crdtmergetest.NewInMemory,
	}
}

func inMemoryReplica(name string) crdtmergetest.ReplicaHarness[*crdtmergetest.InMemory] {
	return crdtmergetest.ReplicaHarness[*crdtmergetest.InMemory]{
		Name: name, New: crdtmergetest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------
//
// Every row is about Merge, and the derived peer is nil — an interface
// parameter admits no literal — so a check wanting a real one builds it. That
// is what the row table exists for.

var mixedChecks = crdtmergetest.MixedChecks{
	{
		Method: "Merge", Name: "folds-a-peer-in",
		Claim: "Merge folds a peer in through its own interface",
		Run:   foldsAPeerIn,
		ProvenBy: crdtmergetest.BrokenMixed(
			"a replica that takes the peer's items for its own",
			planted(replacesRatherThanFolds),
		),
		ProvenReason: "a merge in name only",
	},

	{
		Method: "Merge", Name: "tolerates-a-missing-peer",
		Claim: "Merge tolerates a peer that is not there",
		Run:   toleratesAMissingPeer,
		ProvenBy: crdtmergetest.BrokenMixed(
			"a replica that reports a peer it was never given",
			planted(refusesAMissingPeer),
		),
		ProvenReason: "merging with no peer changes nothing",
	},

	{
		Method: "Merge", Name: "reports-an-unreadable-peer",
		Claim: "Merge reports a peer that cannot be read",
		Run:   reportsAnUnreadablePeer,
		ProvenBy: crdtmergetest.BrokenMixed(
			"a replica that merges over a peer it could not read",
			planted(swallowsAnUnreadablePeer),
		),
		ProvenReason: "reported rather than merged over",
	},
}

// --- Bodies -------------------------------------------------------------------

func foldsAPeerIn(tb testing.TB, s crdtmerge.Mixed, fx crdtmergetest.MixedFixture) {
	tb.Helper()
	testkit.NoError(tb, s.Add(tb.Context(), fx.Item()), "the subject has an item")

	other := crdtmergetest.NewInMemory()
	testkit.NoError(tb, other.Add(tb.Context(), theirItem), "the peer has one too")
	testkit.NoError(tb, s.Merge(tb.Context(), other), "merging succeeds")

	got, err := s.Items(tb.Context())
	testkit.NoError(tb, err, "listing succeeds")
	testkit.Assert(tb, got).Contains(theirItem, "the peer's item arrived")
	testkit.Assert(tb, got).Contains(fx.Item(),
		"and a merge that discarded what was there would be a merge in name only")
}

// toleratesAMissingPeer keeps a failed dial from taking the replica down: a nil
// peer reaches production through a replica that could not reach its partner,
// and merging with nothing is a no-op rather than a panic.
func toleratesAMissingPeer(
	tb testing.TB, s crdtmerge.Mixed, _ crdtmergetest.MixedFixture,
) {
	tb.Helper()
	testkit.NoError(tb, s.Merge(tb.Context(), nil),
		"merging with no peer changes nothing and reports nothing")
}

// reportsAnUnreadablePeer keeps convergence honest. It is a claim about two
// replicas that both answered, so a merge that swallowed an unreachable peer
// would report agreement with something it never read.
func reportsAnUnreadablePeer(
	tb testing.TB, s crdtmerge.Mixed, _ crdtmergetest.MixedFixture,
) {
	tb.Helper()
	testkit.ErrorIs(tb, s.Merge(tb.Context(), failingReplica{}), errPeerUnreadable,
		"the peer's failure is reported rather than merged over")
}

// --- Planted defects ----------------------------------------------------------

// errPeerUnreadable is what failingReplica reports.
var errPeerUnreadable = errors.New("crdtmergetest_test: peer unreadable")

// errNoPeer is what refusesAMissingPeer answers with. It is an input to one row
// rather than a claim of its own, so its identity does not matter: the row
// asserts only that a merge with no peer reported nothing.
var errNoPeer = errors.New("crdtmergetest_test: the planted defect wants a peer")

// failingReplica is a peer whose contents cannot be read. It is an INPUT rather
// than a planted defect — the subject under proof is the replica merging it.
type failingReplica struct{}

func (failingReplica) Items(context.Context) ([]string, error) { return nil, errPeerUnreadable }

// fault names what one planted replica gets wrong about a merge.
//
// All three are about what Merge does with the peer it was handed, which is the
// only thing this fixture's rows are about: the generated family asks whether
// Add and Items survive a call, and none of it reaches the peer at all.
type fault int

const (
	// replacesRatherThanFolds takes the peer's items for its own, which
	// converges two replicas onto whichever merged last.
	replacesRatherThanFolds fault = iota

	// refusesAMissingPeer reports a peer it was never given, which takes
	// down every replica whose partner was unreachable.
	refusesAMissingPeer

	// swallowsAnUnreadablePeer merges over a peer it could not read and
	// reports success, so a replica claims agreement with a silence.
	swallowsAnUnreadablePeer
)

// planted builds the constructor for one broken replica.
func planted(wrong fault) func() *plantedReplica {
	return func() *plantedReplica { return &plantedReplica{wrong: wrong} }
}

type plantedReplica struct {
	wrong fault
	items []string
}

func (r *plantedReplica) Add(_ context.Context, item string) error {
	if !slices.Contains(r.items, item) {
		r.items = append(r.items, item)
	}
	return nil
}

func (r *plantedReplica) Items(context.Context) ([]string, error) {
	return slices.Clone(r.items), nil
}

func (r *plantedReplica) Merge(ctx context.Context, peer crdtmerge.Replica) error {
	if peer == nil {
		if r.wrong == refusesAMissingPeer {
			return errNoPeer
		}
		return nil
	}
	theirs, err := peer.Items(ctx)
	if err != nil {
		if r.wrong == swallowsAnUnreadablePeer {
			return nil
		}
		return err
	}
	if r.wrong == replacesRatherThanFolds {
		r.items = slices.Clone(theirs)
		return nil
	}
	for _, item := range theirs {
		if err := r.Add(ctx, item); err != nil {
			return err
		}
	}
	return nil
}
