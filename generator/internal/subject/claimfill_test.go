// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package subject_test

import (
	"errors"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/core/lawid"
	"go.thesmos.sh/testkit/generator/internal/subject"
)

// A law's claim is the catalogue's sentence, filled from the stamps that
// selected it.
func TestClaimOfWordsFromTheCatalogue(t *testing.T) {
	t.Parallel()

	claim, err := subject.ClaimOf(lawid.TTLExpiry, "store", nil)
	testkit.NoError(t, err, "a worded law with no placeholders needs no carrier")
	testkit.Equal(t, claim, "an entry stops being readable once its lifetime has run out",
		"the catalogue's own wording, verbatim")
}

// Every claim may name the subject, and no carrier can supply it: it is
// a fact about the interface, not about the method that stamped the law.
func TestClaimOfFillsTheSubjectFromTheToken(t *testing.T) {
	t.Parallel()

	claim, err := subject.ClaimOf(lawid.PoisonConsistent, "store", nil)
	testkit.NoError(t, err, "the subject placeholder is filled without a carrier")
	testkit.Contains(t, claim, "the store", "and filled with the interface's own word")
}

// The two refusals are distinguishable, because they are fixed in
// different files: an unworded law needs a sentence in the catalogue, and
// an unfilled one needs a name on the stamp that selected it.
func TestClaimOfTellsItsTwoRefusalsApart(t *testing.T) {
	t.Parallel()

	t.Run("a law the catalogue does not word", func(t *testing.T) {
		t.Parallel()
		// An identifier no catalogue will ever hold, rather than a real
		// law that happens to be unworded today: wording one is a normal
		// change, and a test pinned to a specific gap fails the day
		// somebody closes it.
		_, err := subject.ClaimOf("AUTO-NOT-A-LAW", "store", nil)
		testkit.True(t, errors.Is(err, subject.ErrUnworded),
			"the gap is in the catalogue, and the error says so")
	})

	t.Run("a wording naming something no stamp supplies", func(t *testing.T) {
		t.Parallel()
		_, err := subject.ClaimOf(lawid.LifecycleAfterClose, "store", nil)
		testkit.True(t, err != nil, "an unfilled claim is refused")
		testkit.False(t, errors.Is(err, subject.ErrUnworded),
			"and not as an unworded one — the fix is on the stamp, not in lawid")
		testkit.Contains(t, err.Error(), lawid.PlaceClose,
			"the error names the placeholder that went unfilled")
	})
}

// A stamped reference is spoken as a reader would say it.
//
// The annotator resolves `close=Close` through the package and the type
// so the generator can find the method. A claim is a sentence somebody
// reads in a report, and the resolved path is not one.
func TestClaimFillsSpeakBareNames(t *testing.T) {
	t.Parallel()

	// As the annotator leaves it: the stamp said `close=Close`, and the
	// value carries the package and type it was resolved through.
	stamped := subject.Method{MixinParams: map[string]string{
		"lifecycleafterclose.close": "example.com/kv.Store.Close",
	}}

	claim, err := subject.ClaimOf(lawid.LifecycleAfterClose, "store",
		[]subject.Method{stamped})
	testkit.NoError(t, err, "the close placeholder is filled from the stamp")
	testkit.Contains(t, claim, "once Close has run", "by the name the stamp wrote")
	testkit.NotContains(t, claim, "example.com",
		"and not by the path it was resolved through, which is not a sentence")
}
