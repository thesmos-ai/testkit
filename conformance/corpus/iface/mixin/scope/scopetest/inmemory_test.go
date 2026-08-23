// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// `scope` is the model tier's: that one scope's writes stay out of another's
// needs two scopes and a comparison. What one settles is the row below — that a
// write is readable back under the scope it was made in.
package scopetest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/scope"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/scope/scopetest"
)

// TestMixedContract runs the generated checks and this package's own.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	scopetest.RunMixed(t, inMemory("in-memory"), mixedChecks)
}

// TestMixedContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	scopetest.RunMixed(t,
		inMemory("in-memory"),
		scopetest.MixedSuite.Without(scopetest.MixedSuite.Checks.Set.Smoke()),
	)
}

// TestMixedChecksCanFail drives the row against its planted defect.
func TestMixedChecksCanFail(t *testing.T) {
	t.Parallel()

	scopetest.ProveMixed(t, inMemory("in-memory"), mixedChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) scopetest.MixedHarness[*scopetest.InMemory] {
	return scopetest.MixedHarness[*scopetest.InMemory]{
		Name: name, New: scopetest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var mixedChecks = scopetest.MixedChecks{
	{
		Method: "Get", Name: "reads-within-its-scope",
		Claim: "Get reads within its scope",
		Run:   readsWithinItsScope,
		ProvenBy: scopetest.BrokenMixed(
			"a store that ignores the scope it was given", newIgnoresTheScope,
		),
		ProvenReason: "carrying what was written",
	},
}

// --- Bodies -------------------------------------------------------------------

func readsWithinItsScope(tb testing.TB, s scope.Mixed, fx scopetest.MixedFixture) {
	tb.Helper()
	testkit.NoError(tb, s.Set(tb.Context(), fx.Scope(), fx.Key(), fx.Value()),
		"writing within a scope succeeds")

	got, err := s.Get(tb.Context(), fx.Scope(), fx.Key())
	testkit.NoError(tb, err, "and reading it back succeeds")
	testkit.Equal(tb, got, fx.Value(), "carrying what was written")
}

// --- Planted defects ----------------------------------------------------------

// ignoresTheScope files everything under one namespace and answers the zero
// where it should answer the value, which is the shape a store that took the
// scope parameter and never threaded it has.
type ignoresTheScope struct{ held map[string]string }

func newIgnoresTheScope() *ignoresTheScope {
	return &ignoresTheScope{held: map[string]string{}}
}

func (i *ignoresTheScope) Set(_ context.Context, _, key, value string) error {
	i.held[key] = value
	return nil
}

func (i *ignoresTheScope) Get(_ context.Context, _, key string) (string, error) {
	if _, held := i.held[key]; !held {
		return "", scopetest.ErrNotFound
	}
	return "", nil
}
