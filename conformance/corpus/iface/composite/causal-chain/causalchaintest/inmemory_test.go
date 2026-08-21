// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// causal-chain is the model tier's: `AUTO-REPLAY-CAUSAL-ORDERING` walks the
// replay against the dependency graph, whose identifiers and edges are the
// domain's rather than anything derivation can invent.
//
// What the suite tier states is the row below: an effect cannot precede its
// cause. The derived entry leaves the cause list at its zero, which is why a
// log admitting only settled causes accepts it — the dangling entry is the
// row's to build.
package causalchaintest_test

import (
	"context"
	"slices"
	"testing"

	"go.thesmos.sh/testkit"
	causalchain "go.thesmos.sh/testkit/conformance/corpus/iface/composite/causal-chain"
	"go.thesmos.sh/testkit/conformance/corpus/iface/composite/causal-chain/causalchaintest"
)

// TestLogContract runs the generated checks and this package's own.
func TestLogContract(t *testing.T) {
	t.Parallel()

	causalchaintest.RunLog(t, inMemory("in-memory"), logChecks)
}

// TestLogContractWithoutSmoke drops a check through the typed index rather than
// a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestLogContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	causalchaintest.RunLog(t,
		inMemory("in-memory"),
		causalchaintest.LogSuite.Without(causalchaintest.LogSuite.Checks.Append.Smoke()),
	)
}

// TestLogChecksCanFail drives the row against its planted defect.
func TestLogChecksCanFail(t *testing.T) {
	t.Parallel()

	causalchaintest.ProveLog(t, logChecks)
}

// --- Harnesses ---------------------------------------------------------------

// The two identifiers the row builds its edge from. Named here because the
// planted defect has to admit and refuse the same pair the row does.
const (
	causeID  = "b6-cause"
	effectID = "b6-effect"
)

func inMemory(name string) causalchaintest.LogHarness[*causalchaintest.InMemory] {
	return causalchaintest.LogHarness[*causalchaintest.InMemory]{
		Name: name, New: causalchaintest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var logChecks = causalchaintest.LogChecks{
	{
		Method: "Append", Name: "refuses-an-unlanded-cause",
		Claim: "Append refuses an entry whose cause has not landed",
		Run:   refusesAnUnlandedCause,
		ProvenBy: causalchaintest.BrokenLog(
			"a log that takes every entry it is offered", newAcceptsAnything,
		),
		ProvenReason: "the effect cannot precede its cause",
	},
}

// --- Bodies -------------------------------------------------------------------

// refusesAnUnlandedCause builds the dangling edge, because the derived entry
// leaves the cause list at its zero and a log admitting only settled causes
// accepts it.
//
// It ends by landing the cause and retrying: a log that refused everything
// would satisfy the first half and record nothing at all.
func refusesAnUnlandedCause(
	tb testing.TB, s causalchain.Log, _ causalchaintest.LogFixture,
) {
	tb.Helper()
	dangling := causalchain.Entry{ID: effectID, DependsOn: []string{causeID}}
	testkit.ErrorIs(tb, s.Append(tb.Context(), dangling),
		causalchaintest.ErrUnmetDependency, "the effect cannot precede its cause")

	testkit.NoError(tb, s.Append(tb.Context(), causalchain.Entry{ID: causeID}),
		"the cause lands first")
	testkit.NoError(tb, s.Append(tb.Context(), dangling),
		"and then the effect is admitted")
}

// --- Planted defects ----------------------------------------------------------

// acceptsAnything records every entry without reading its dependencies, which
// is a log with the graph declared and never walked. Every generated check
// passes against it — each appends one entry whose cause list is the zero, and
// an entry depending on nothing is admissible.
type acceptsAnything struct{ entries []causalchain.Entry }

func newAcceptsAnything() *acceptsAnything { return &acceptsAnything{} }

func (a *acceptsAnything) Append(_ context.Context, e causalchain.Entry) error {
	a.entries = append(a.entries, e)
	return nil
}

func (a *acceptsAnything) Replay(context.Context) ([]causalchain.Entry, error) {
	return slices.Clone(a.entries), nil
}
