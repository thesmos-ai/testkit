// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// paginated-reader stacks the reader detector with the pagination contract, and
// this fixture exists because the contract changes what the detector generates
// rather than adding to it: a bare reader asserts one call and one result, and
// a paginated one has to be driven as a loop.
//
// `pagination` is the model tier's under ADR-0028 — `AUTO-PAGINATOR-NO-DUPLICATES`
// and `AUTO-PAGINATOR-RESUMABLE` state it — so the generated family is the
// signature-derived one. The loop the contract implies is stated here, through
// the extension point rather than as a package test: every claim below needs
// only the interface, so every one runs against each subject a consumer
// declares and again through the double.
//
// The derived cursors are integers the reader never issued, which is what makes
// its "an error carries the zero value" check able to fail at all.
package paginatedreadertest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	paginatedreader "go.thesmos.sh/testkit/conformance/corpus/iface/composite/paginated-reader"
	"go.thesmos.sh/testkit/conformance/corpus/iface/composite/paginated-reader/paginatedreadertest"
)

// TestPaginatedReaderContract runs the generated checks and this package's own,
// against a subject holding the corpus and one holding nothing.
func TestPaginatedReaderContract(t *testing.T) {
	t.Parallel()

	paginatedreadertest.RunPaginatedReader(t,
		holding("in-memory"),
		empty("in-memory, empty"),
		paginatedReaderChecks,
	)
}

// TestPaginatedReaderContractWithoutSmoke drops a check through the typed index
// rather than a string, so a check that is renamed or stops being emitted
// breaks this compile instead of silently declining nothing.
func TestPaginatedReaderContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	paginatedreadertest.RunPaginatedReader(t,
		holding("in-memory"),
		paginatedreadertest.PaginatedReaderSuite.Without(
			paginatedreadertest.PaginatedReaderSuite.Checks.Page.Smoke(),
		),
	)
}

// TestPaginatedReaderChecksCanFail drives every row against its planted defect.
func TestPaginatedReaderChecksCanFail(t *testing.T) {
	t.Parallel()

	paginatedreadertest.ProvePaginatedReader(t, holding("in-memory"), paginatedReaderChecks)
}

// --- Harnesses ---------------------------------------------------------------

// corpus is what every reader in this package pages over.
//
// Five values against a page size of two, so the walk takes three pages and the
// last one is short — which is where an off-by-one either drops the tail or
// serves it twice.
var corpus = []paginatedreader.Value{
	{Key: "a", Body: "one"},
	{Key: "b", Body: "two"},
	{Key: "c", Body: "three"},
	{Key: "d", Body: "four"},
	{Key: "e", Body: "five"},
}

func holding(name string) paginatedreadertest.PaginatedReaderHarness[*paginatedreadertest.InMemory] {
	return paginatedreadertest.PaginatedReaderHarness[*paginatedreadertest.InMemory]{
		Name: name,
		New:  func() *paginatedreadertest.InMemory { return paginatedreadertest.NewInMemory(corpus...) },
	}
}

// empty is the subject with nothing to page over, so every claim below has to
// say what it means when there is no second page.
func empty(name string) paginatedreadertest.PaginatedReaderHarness[*paginatedreadertest.InMemory] {
	return paginatedreadertest.PaginatedReaderHarness[*paginatedreadertest.InMemory]{
		Name: name,
		New:  func() *paginatedreadertest.InMemory { return paginatedreadertest.NewInMemory() },
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var paginatedReaderChecks = paginatedreadertest.PaginatedReaderChecks{
	{
		Method: "Page", Name: "refuses-an-unissued-cursor",
		Claim: "Page refuses a cursor it did not issue",
		Run:   refusesAnUnissuedCursor,
		ProvenBy: paginatedreadertest.BrokenPaginatedReader(
			"a reader that resumes from any integer it is handed",
			planted(acceptsAnyCursor),
		),
		ProvenReason: "a cursor from nowhere is refused",
	},

	{
		Method: "Page", Name: "walks-every-value-once",
		Claim: "Page walks every value once and terminates",
		Run:   walksEveryValueOnce,
		ProvenBy: paginatedreadertest.BrokenPaginatedReader(
			"a reader whose pages overlap by one", planted(overlapsItsPages),
		),
		ProvenReason: "no value is served on two pages",
	},

	{
		Method: "Page", Name: "resumes-where-a-cursor-was-issued",
		Claim: "Page resumes where a cursor was issued",
		Run:   resumesWhereACursorWasIssued,
		ProvenBy: paginatedreadertest.BrokenPaginatedReader(
			"a reader that keeps its own position and ignores the cursor",
			planted(advancesOnEveryCall),
		),
		ProvenReason: "the cursor decides where a read resumes",
	},
}

// --- Bodies -------------------------------------------------------------------

// refusesAnUnissuedCursor is what separates an opaque token from an offset.
//
// An offset accepts any integer, so a reader that takes invented cursors
// resumes somewhere nobody asked for.
func refusesAnUnissuedCursor(
	tb testing.TB, s paginatedreader.PaginatedReader,
	fx paginatedreadertest.PaginatedReaderFixture,
) {
	tb.Helper()
	items, next, err := s.Page(tb.Context(), fx.CursorOther())
	testkit.ErrorIs(tb, err, paginatedreadertest.ErrUnknownCursor,
		"a cursor from nowhere is refused")
	testkit.Len(tb, items, 0, "with no page beside the error")
	testkit.Equal(tb, next, paginatedreadertest.End, "and no cursor to continue from")
}

func walksEveryValueOnce(
	tb testing.TB, s paginatedreader.PaginatedReader,
	_ paginatedreadertest.PaginatedReaderFixture,
) {
	tb.Helper()
	seen := walk(tb, s, paginatedreadertest.Start)
	testkit.Equal(tb, len(dedupe(seen)), len(seen), "no value is served on two pages")
}

// resumesWhereACursorWasIssued is `AUTO-PAGINATOR-RESUMABLE`'s own shape: an
// issued cursor, and not the reader's memory, is what decides where a read
// picks up.
//
// It reads the same cursor twice rather than comparing a resumed walk against
// the full one. That comparison cannot fail: walk(Start) IS the first page
// followed by walk(second), so asserting the second equals the first minus a
// page re-derives the walk's own definition and holds for every reader,
// correct or not. Reading one cursor twice asks something a reader can get
// wrong — a paginator keeping its position in a field rather than in the token
// answers the two reads differently.
func resumesWhereACursorWasIssued(
	tb testing.TB, s paginatedreader.PaginatedReader,
	_ paginatedreadertest.PaginatedReaderFixture,
) {
	tb.Helper()
	_, second, err := s.Page(tb.Context(), paginatedreadertest.Start)
	testkit.NoError(tb, err, "the first page is readable")
	if second == paginatedreadertest.End {
		// Nothing to resume from, which is the empty subject. The claim is
		// vacuous rather than untested — there is no second page.
		return
	}

	once, _, err := s.Page(tb.Context(), second)
	testkit.NoError(tb, err, "the issued cursor is readable")
	twice, _, err := s.Page(tb.Context(), second)
	testkit.NoError(tb, err, "and readable again")
	testkit.Equal(tb, twice, once,
		"the cursor decides where a read resumes, not the reader")
}

// walk reads from a cursor to the end, failing on anything the reader refuses.
//
// A bounded loop rather than an open one: a reader handing back a cursor that
// never terminates would otherwise hang until the test binary's own timeout,
// which reports as a panic in whatever test happened to be running.
func walk(
	tb testing.TB, subject paginatedreader.PaginatedReader, from int,
) []paginatedreader.Value {
	tb.Helper()

	const maxPages = 16

	var seen []paginatedreader.Value
	cursor := from
	for range maxPages {
		items, next, err := subject.Page(tb.Context(), cursor)
		testkit.NoError(tb, err, "an issued cursor is readable")
		seen = append(seen, items...)
		if next == paginatedreadertest.End {
			return seen
		}
		cursor = next
	}
	tb.Fatalf("the walk did not terminate within %d pages", maxPages)
	return nil
}

// dedupe returns the distinct values, in first-seen order.
func dedupe(in []paginatedreader.Value) []paginatedreader.Value {
	seen := map[paginatedreader.Value]bool{}
	out := make([]paginatedreader.Value, 0, len(in))
	for _, v := range in {
		if seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}

// --- Planted defects ----------------------------------------------------------

// plantedPage is what a planted reader serves per page. The same as the real
// subject's, so an off-by-one lands in the same place: a defect paging
// differently would be testing the arithmetic rather than the claim.
const plantedPage = 2

// fault names what one planted reader gets wrong.
//
// Every one of them terminates. A reader that loops forever fails the walk's
// own bound instead of the claim, which is a red for the wrong reason and
// evidence for nothing.
type fault int

const (
	// acceptsAnyCursor resumes from whatever integer it is handed, which is
	// an offset pretending to be a token.
	acceptsAnyCursor fault = iota

	// overlapsItsPages advances by one less than it serves, so the value at
	// each boundary is served on two pages.
	overlapsItsPages

	// advancesOnEveryCall keeps its position in a field and reads the cursor
	// only to decide whether to stop, so two reads of one cursor answer
	// different pages.
	advancesOnEveryCall
)

// planted builds the constructor for one broken reader, holding what the
// harness's own subject holds so a walk has the same thing to walk.
func planted(wrong fault) func() *plantedReader {
	return func() *plantedReader { return &plantedReader{wrong: wrong, values: corpus} }
}

type plantedReader struct {
	wrong fault

	values []paginatedreader.Value

	// at is advancesOnEveryCall's own position, which is the whole of that
	// defect: a correct reader has nowhere to keep one.
	at int
}

func (r *plantedReader) Page(
	_ context.Context, cursor int,
) ([]paginatedreader.Value, int, error) {
	if cursor < 0 || cursor > len(r.values) {
		if r.wrong == acceptsAnyCursor {
			// Anything out of range reads as the end, which is what an
			// offset-shaped reader does with a number nobody issued.
			return nil, paginatedreadertest.End, nil
		}
		return nil, paginatedreadertest.End, paginatedreadertest.ErrUnknownCursor
	}

	from := cursor
	if r.wrong == advancesOnEveryCall {
		from = r.at
		r.at = min(r.at+plantedPage, len(r.values))
	}
	to := min(from+plantedPage, len(r.values))

	next := to
	if r.wrong == overlapsItsPages && to < len(r.values) {
		next = to - 1
	}
	if next >= len(r.values) {
		next = paginatedreadertest.End
	}
	return r.values[from:to], next, nil
}
