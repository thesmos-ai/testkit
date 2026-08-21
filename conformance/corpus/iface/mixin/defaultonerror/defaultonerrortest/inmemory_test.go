// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// `defaultonerror` is the model tier's: that a failed read answers the declared
// default rather than a zero needs a failure to induce, which is the model
// tier's to arrange. What one clean write and read settle is the row below.
package defaultonerrortest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/defaultonerror"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/defaultonerror/defaultonerrortest"
)

// TestMixedContract runs the generated checks and this package's own.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	defaultonerrortest.RunMixed(t, inMemory("in-memory"), mixedChecks)
}

// TestMixedContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	defaultonerrortest.RunMixed(t,
		inMemory("in-memory"),
		defaultonerrortest.MixedSuite.Without(defaultonerrortest.MixedSuite.Checks.Store.Smoke()),
	)
}

// TestMixedChecksCanFail drives the row against its planted defect.
func TestMixedChecksCanFail(t *testing.T) {
	t.Parallel()

	defaultonerrortest.ProveMixed(t, mixedChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) defaultonerrortest.MixedHarness[*defaultonerrortest.InMemory] {
	return defaultonerrortest.MixedHarness[*defaultonerrortest.InMemory]{
		Name: name, New: defaultonerrortest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var mixedChecks = defaultonerrortest.MixedChecks{
	{
		Method: "Get", Name: "reads-back-what-store-wrote",
		Claim: "Get returns what Store wrote",
		Run:   readsBackWhatStoreWrote,
		ProvenBy: defaultonerrortest.BrokenMixed(
			"a store that answers its default on the happy path", newAlwaysDefaults,
		),
		ProvenReason: "answers with what was stored",
	},
}

// --- Bodies -------------------------------------------------------------------

func readsBackWhatStoreWrote(
	tb testing.TB, s defaultonerror.Mixed, fx defaultonerrortest.MixedFixture,
) {
	tb.Helper()
	written := fx.Value()
	testkit.NoError(tb, s.Store(tb.Context(), written), "the value is stored")

	got, err := s.Get(tb.Context(), written.Key)
	testkit.NoError(tb, err, "the written key is present")
	testkit.Equal(tb, got, written, "and answers with what was stored")
}

// --- Planted defects ----------------------------------------------------------

// alwaysDefaults answers the fallback whether or not anything went wrong, which
// is the mixin's own hazard: a default that covers the failure path is correct,
// and one that covers the happy path hides every read.
type alwaysDefaults struct {
	held map[string]defaultonerror.Value
}

func newAlwaysDefaults() *alwaysDefaults {
	return &alwaysDefaults{held: map[string]defaultonerror.Value{}}
}

func (a *alwaysDefaults) Store(_ context.Context, v defaultonerror.Value) error {
	a.held[v.Key] = v
	return nil
}

func (a *alwaysDefaults) Get(
	_ context.Context, key string,
) (defaultonerror.Value, error) {
	if _, held := a.held[key]; !held {
		return defaultonerror.Value{}, defaultonerrortest.ErrNotFound
	}
	return defaultonerror.Value{}, nil
}
