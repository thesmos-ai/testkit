// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// `sticky` is the model tier's: that a session keeps landing on the same
// replica is a claim about a sequence of routed calls. The row below is what
// one write and one read settle.
package stickytest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/sticky"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/sticky/stickytest"
)

// TestMixedContract runs the generated checks and this package's own.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	stickytest.RunMixed(t, inMemory("in-memory"), mixedChecks)
}

// TestMixedContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	stickytest.RunMixed(t,
		inMemory("in-memory"),
		stickytest.MixedSuite.Without(stickytest.MixedSuite.Checks.Store.Smoke()),
	)
}

// TestMixedChecksCanFail drives the row against its planted defect.
func TestMixedChecksCanFail(t *testing.T) {
	t.Parallel()

	stickytest.ProveMixed(t, inMemory("in-memory"), mixedChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) stickytest.MixedHarness[*stickytest.InMemory] {
	return stickytest.MixedHarness[*stickytest.InMemory]{
		Name: name, New: stickytest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var mixedChecks = stickytest.MixedChecks{
	{
		Method: "Get", Name: "reads-back-what-store-wrote",
		Claim: "Get returns what Store wrote",
		Run:   readsBackWhatStoreWrote,
		ProvenBy: stickytest.BrokenMixed(
			"a store whose reads land somewhere the write did not", newRoutesElsewhere,
		),
		ProvenReason: "the written key is present",
	},
}

// --- Bodies -------------------------------------------------------------------

func readsBackWhatStoreWrote(
	tb testing.TB, s sticky.Mixed, fx stickytest.MixedFixture,
) {
	tb.Helper()
	written := fx.Value()
	testkit.NoError(tb, s.Store(tb.Context(), written), "the value is stored")

	got, err := s.Get(tb.Context(), written.Key)
	testkit.NoError(tb, err, "the written key is present")
	testkit.Equal(tb, got.Key, written.Key,
		"and Get answers under the key it was stored with")
}

// --- Planted defects ----------------------------------------------------------

// routesElsewhere keeps writes in one place and answers reads from another,
// which is the failure stickiness exists to prevent — and one that looks
// identical to an empty store from the read side alone.
type routesElsewhere struct {
	written map[string]sticky.Value
	read    map[string]sticky.Value
}

func newRoutesElsewhere() *routesElsewhere {
	return &routesElsewhere{
		written: map[string]sticky.Value{},
		read:    map[string]sticky.Value{},
	}
}

func (r *routesElsewhere) Store(_ context.Context, v sticky.Value) error {
	r.written[v.Key] = v
	return nil
}

func (r *routesElsewhere) Get(_ context.Context, key string) (sticky.Value, error) {
	v, held := r.read[key]
	if !held {
		return sticky.Value{}, stickytest.ErrNotFound
	}
	return v, nil
}
