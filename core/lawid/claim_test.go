// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package lawid_test

import (
	"strings"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/core/lawid"
)

// claimCase is one corpus spelling filled end to end; a drift here
// rewrites every lock in the fleet.
type claimCase struct {
	name  string
	id    string
	fills []string
	want  string
}

func (c claimCase) Name() string { return c.name }

// The claim wordings are the corpus's spelling, filled end to end.
//
// The case body declines a helper mark: TableTest calls it from inside
// its own t.Run, so t.Helper() would report each failure at the
// runner's call site rather than at the assertion line that failed.
//
//nolint:thelper // the case body is the test, not a helper; see above
func TestClaimFillingsMatchTheCorpus(t *testing.T) {
	t.Parallel()

	testkit.TableTest(t, []claimCase{
		{
			"a static claim fills from nothing",
			lawid.TTLExpiry, nil,
			"an entry stops being readable once its lifetime has run out",
		},
		{
			"the after-close claim speaks the declared teardown",
			lawid.LifecycleAfterClose,
			[]string{lawid.PlaceClose, "Close"},
			"once Close has run, every method reports the closed sentinel",
		},
		{
			"the poison claim speaks the subject's token",
			lawid.PoisonConsistent,
			[]string{lawid.PlaceSubject, "store"},
			"once the store reports it is closed, it keeps reporting it",
		},
		{
			"the cursor close claim speaks partner and handle",
			lawid.CursorCloseIdempotent,
			[]string{lawid.PlaceClose, "Close", lawid.PlaceProduced, "cursor"},
			"a second Close on a cursor changes nothing",
		},
		{
			"the cursor read claim speaks next and handle",
			lawid.CursorNextAfterClose,
			[]string{lawid.PlaceNext, "Next", lawid.PlaceProduced, "cursor"},
			"once a cursor is closed, Next reports the closed sentinel",
		},
		{
			"over-supplying is free",
			lawid.LeaseDoubleAcquireBlocks,
			[]string{lawid.PlaceClose, "Release", lawid.PlaceSubject, "lease"},
			"a second acquire of a held key reports the held sentinel",
		},
	}, func(t *testing.T, tc claimCase) {
		c, worded := lawid.ClaimOf(tc.id)
		testkit.True(t, worded, tc.id+" is worded")
		got, err := c.Fill(tc.fills...)
		testkit.NoError(t, err, "every placeholder the claim speaks has a fill")
		testkit.Equal(t, got, tc.want, "the filled claim is the corpus manifests' spelling")
	})
}

func TestFillRefusesTheHalfDone(t *testing.T) {
	t.Parallel()

	t.Run("an unfilled placeholder is an error, not prose", func(t *testing.T) {
		t.Parallel()
		c, _ := lawid.ClaimOf(lawid.LifecycleAfterClose)
		_, err := c.Fill()
		testkit.Error(t, err, "a bracket in a manifest row would diff forever after")
		testkit.Contains(t, err.Error(), lawid.PlaceClose, "the error names the missing placeholder")
	})

	t.Run("an odd pair list is an error", func(t *testing.T) {
		t.Parallel()
		c, _ := lawid.ClaimOf(lawid.TTLExpiry)
		_, err := c.Fill(lawid.PlaceClose)
		testkit.Error(t, err, "half a pair fills nothing")
	})

	t.Run("an unworded identifier answers false", func(t *testing.T) {
		t.Parallel()
		_, worded := lawid.ClaimOf("not-a-law")
		testkit.False(t, worded, "the consumer refuses by name rather than inventing a sentence")
	})
}

func TestClaimCensus(t *testing.T) {
	t.Parallel()

	t.Run("every worded identifier is a registered law", func(t *testing.T) {
		t.Parallel()
		// Orphan detection is the half that bites: a claim keyed on a
		// misspelled identifier words nothing and reads as coverage,
		// so the count of live identifiers that answer worded must
		// equal the table — a key outside the registry lowers it.
		worded := 0
		for _, id := range lawid.All() {
			if _, ok := lawid.ClaimOf(id); ok {
				worded++
			}
		}
		testkit.Equal(t, worded, 10, "the corpus-pinned wordings, each keyed by a live identifier")
	})

	t.Run("every claim speaks only the placeholder vocabulary", func(t *testing.T) {
		t.Parallel()
		for _, id := range lawid.All() {
			c, worded := lawid.ClaimOf(id)
			if !worded {
				continue
			}
			stripped := string(c)
			for _, p := range lawid.Placeholders() {
				stripped = strings.ReplaceAll(stripped, p, "")
			}
			testkit.False(t, strings.ContainsAny(stripped, "{}"),
				id+" interpolates only the closed vocabulary")
		}
	})
}
