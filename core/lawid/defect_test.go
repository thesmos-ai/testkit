// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package lawid_test

import (
	"slices"
	"strings"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/core/lawid"
)

// Every law declares what it can see.
//
// Totality, and the reason the table is worth having at all: the saturation
// prover trusts a law's class to decide whether a kill counts, so a law with no
// class is one every wear proves — which is the criterion this axis replaced.
func TestEveryLawDeclaresADefectClass(t *testing.T) {
	t.Parallel()

	for _, id := range lawid.All() {
		testkit.True(t, len(lawid.ClassOf(id)) > 0, id+" declares the defect class its name promises")
	}
}

// Every class is claimed by some law.
//
// The other end of totality. A class nothing claims is a word in a vocabulary
// with no speaker, and the wardrobe would be tagged against it forever — which
// is how a taxonomy grows a tail nobody can delete because nobody can tell
// whether it is load-bearing.
func TestEveryClassIsClaimed(t *testing.T) {
	t.Parallel()

	claimed := map[lawid.DefectClass]bool{}
	for _, id := range lawid.All() {
		for _, c := range lawid.ClassOf(id) {
			claimed[c] = true
		}
	}
	for _, c := range lawid.Classes() {
		testkit.True(t, claimed[c], string(c)+" is a class some law claims to catch")
	}
}

// A declared class is nothing but a declared class.
//
// The table is hand-written and the prover trusts it, so a value outside the
// vocabulary would be a class matching no wear — a law that reads as
// unsaturatable for a reason nobody wrote down.
func TestEveryDeclaredClassIsInTheVocabulary(t *testing.T) {
	t.Parallel()

	for _, id := range lawid.All() {
		for _, c := range lawid.ClassOf(id) {
			testkit.True(t, slices.Contains(lawid.Classes(), c),
				id+" declares "+string(c)+", which is in the vocabulary")
		}
	}
}

// Where the identifier says what the defect is, the declaration agrees.
//
// The mitigation the derived design would have made unnecessary and this one
// has to buy. Reading the class out of the identifier's own words classifies
// 54 of the 83 and cannot reach the rest, so the words are used as a *check*
// rather than as the source: a law whose name says "no duplicates" and whose
// row does not say duplication is a row somebody mistyped, and a rename that
// leaves its class behind fails here.
//
// The 29 the words cannot reach are listed rather than skipped silently, which
// is what tells a reviewer which rows a machine checked and which rows they
// have to.
func TestClassesAgreeWithTheName(t *testing.T) {
	t.Parallel()

	var unchecked []string
	for _, id := range lawid.All() {
		want, says := lawid.ClassFromName(id)
		if !says {
			unchecked = append(unchecked, id)
			continue
		}
		got := lawid.ClassOf(id)
		for _, c := range want {
			testkit.True(t, slices.Contains(got, c),
				id+" names "+string(c)+" and declares it")
		}
	}

	// A ceiling, so the unreadable share can only shrink: a law added with a
	// name that says nothing about its defect is a name worth reconsidering
	// before the row is written.
	testkit.Equal(t, len(unchecked), 30,
		"29 identifiers name a relation rather than a defect, so their rows are "+
			"declared against nothing a machine reads: "+strings.Join(unchecked, ", "))
}
