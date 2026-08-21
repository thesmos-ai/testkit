// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// `pagination` is the model tier's: no-duplicates and resumable are claims
// about a whole walk. The row below is the walk itself against a subject the
// row seeds — nothing seeds one now but its own constructor, and this one
// starts empty.
package paginationtest_test

import (
	"context"
	"fmt"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/pagination"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/pagination/paginationtest"
)

// TestContractContract runs the generated checks and this package's own.
func TestContractContract(t *testing.T) {
	t.Parallel()

	paginationtest.RunContract(t, inMemory("in-memory"), contractChecks)
}

// TestContractContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestContractContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	paginationtest.RunContract(t,
		inMemory("in-memory"),
		paginationtest.ContractSuite.Without(paginationtest.ContractSuite.Checks.Put.Smoke()),
	)
}

// TestContractChecksCanFail drives the row against its planted defect.
func TestContractChecksCanFail(t *testing.T) {
	t.Parallel()

	paginationtest.ProveContract(t, contractChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) paginationtest.ContractHarness[*paginationtest.InMemory] {
	return paginationtest.ContractHarness[*paginationtest.InMemory]{
		Name: name, New: paginationtest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var contractChecks = paginationtest.ContractChecks{
	{
		Method: "Page", Name: "walks-in-key-order",
		Claim: "Page walks every entry once, in key order",
		Run:   walksInKeyOrder,
		ProvenBy: paginationtest.BrokenContract(
			"a paginator that answers every page from the start", newRestartsEveryPage,
		),
		ProvenReason: "keys arrive strictly ascending",
	},
}

// --- Bodies -------------------------------------------------------------------

// walksInKeyOrder seeds its own entries and walks them.
//
// The loop is bounded rather than open: a paginator handing back a cursor that
// never terminates would otherwise hang until the test binary's own timeout,
// which reports as a panic in whatever test happened to be running.
func walksInKeyOrder(
	tb testing.TB, s pagination.Contract, _ paginationtest.ContractFixture,
) {
	tb.Helper()
	for i := range entries {
		testkit.NoError(tb, s.Put(tb.Context(), pagination.Value{
			Key: entryKey(i), Body: "seeded",
		}), "an entry is stored")
	}

	seen := map[string]bool{}
	last, cur := "", pagination.Cursor("")
	for range maxPages {
		items, next, more, err := s.Page(tb.Context(), cur)
		testkit.NoError(tb, err, "a page is readable")
		for _, v := range items {
			testkit.Equal(tb, v.Key > last, true, "keys arrive strictly ascending")
			last = v.Key
			seen[v.Key] = true
		}
		if !more {
			break
		}
		cur = next
	}
	for i := range entries {
		testkit.Equal(tb, seen[entryKey(i)], true, "the stored entry paged out")
	}
}

// entryKey names the seeded entries so they sort in the order they were
// written, which is what makes "strictly ascending" observable at all.
func entryKey(i int) string { return fmt.Sprintf("b6-%02d", i) }

// --- Planted defects ----------------------------------------------------------

const (
	// entries is how many the row seeds: more than one page's worth, so the
	// walk takes several and the cursor is actually used.
	entries = 5

	// maxPages bounds the walk. A paginator that never terminates fails
	// this bound rather than hanging the run.
	maxPages = 100
)

// restartsEveryPage ignores the cursor and answers from the beginning, which is
// the resumption bug in its plainest form: the walk still terminates, still
// reports every entry, and hands the same key back on the second page — so only
// the ascending comparison catches it.
type restartsEveryPage struct{ held []pagination.Value }

func newRestartsEveryPage() *restartsEveryPage { return &restartsEveryPage{} }

func (r *restartsEveryPage) Put(_ context.Context, v pagination.Value) error {
	r.held = append(r.held, v)
	return nil
}

func (r *restartsEveryPage) Page(
	_ context.Context, _ pagination.Cursor,
) ([]pagination.Value, pagination.Cursor, bool, error) {
	if len(r.held) == 0 {
		return nil, "", false, nil
	}
	// One entry per page and always the first, so the walk advances by the
	// "more" flag alone and every page repeats what the last one served.
	return r.held[:1], pagination.Cursor(r.held[0].Key), len(r.held) > 1, nil
}
