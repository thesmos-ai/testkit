// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"testing"

	"go.thesmos.sh/eidos/lang/golang"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/core/lawid"
	"go.thesmos.sh/testkit/generator/internal/projection"
	"go.thesmos.sh/testkit/generator/internal/subject"
)

// defectBindings is the smallest derivation a rule reads: an interface
// name to compose the double's option from, and the methods to pick a
// target out of.
func defectBindings(methods ...subject.Method) *Bindings {
	b := &Bindings{Methods: methods}
	b.IfaceName = "Mixed"
	return b
}

// stamped is a method carrying one mixin parameter.
func stamped(name, param, value string) subject.Method {
	m := subject.Method{Sig: &golang.Sig{Name: name}}
	m.Name = name
	m.Mixins = []string{mixinAfterClose}
	m.MixinParams = map[string]string{mixinAfterClose + "." + param: value}
	return m
}

// The rule table answers for the laws it words and declines the rest.
//
// A law without a row is the honest residue — the domain composites no
// mechanical rule reaches — and the row it belongs to ships Argued
// saying so. Answering for one would be a proof nobody derived.
func TestOnlyRuledLawsCarryADefect(t *testing.T) {
	t.Parallel()

	b := defectBindings(stamped("Close", mixinAfterCloseSentinel, "kv.ErrClosed"))

	_, _, ruled, _ := defectFor(b, &LawBinding{ID: lawid.PoisonConsistent})
	testkit.True(t, ruled, "a law the table words plants its defect")

	_, _, unruled, _ := defectFor(b, &LawBinding{ID: lawid.CursorCloseIdempotent})
	testkit.False(t, unruled,
		"and one it does not stays Argued rather than wearing a proof nobody wrote")
}

// A rule whose target is not on this interface declines.
//
// The table says which laws have a rule; whether the rule REACHES is a
// fact about the declaration. A defect over a method that is not there
// does not compile, so the two questions are asked separately.
func TestARuleWithNoTargetDeclines(t *testing.T) {
	t.Parallel()

	bare := defectBindings(subject.Method{Sig: &golang.Sig{Name: "Read"}})

	_, _, planted, _ := defectFor(bare, &LawBinding{ID: lawid.PoisonConsistent})
	testkit.False(t, planted, "no stamped sentinel, so no sentinel to heal from")

	_, _, drops := differentialDefect(bare)
	testkit.False(t, drops, "and nothing writes, so there is no dropped write to plant")
}

// The poison defect names the sentinel the declaration stamped.
//
// The same one that licensed the law. A defect reporting a different
// error breaks a claim nobody made, and passes the one under proof.
func TestThePoisonDefectNamesTheStampedSentinel(t *testing.T) {
	t.Parallel()

	b := defectBindings(stamped("Close", mixinAfterCloseSentinel, "kv.ErrClosed"))

	defect, over, planted, _ := defectFor(b, &LawBinding{ID: lawid.PoisonConsistent})
	testkit.True(t, planted, "the stamp is there, so the rule reaches")
	heals, is := defect.(projection.SentinelOnce)
	testkit.True(t, is, "the un-sticky poison the law forbids")
	testkit.Equal(t, string(heals.Sentinel), "kv.ErrClosed", "at the stamped identity")
	testkit.Equal(t, over.Name, "Close", "planted through the method that stamped it")
}

// The appender defect goes through the law's own carrier, not through a
// driven writer.
//
// That law appends through a closure of its own, and the corpus's
// appender fixture drives nothing but a reader. A defect over a method
// the law never calls is one it cannot notice.
func TestTheAppenderDefectFollowsTheLawsCarrier(t *testing.T) {
	t.Parallel()

	appendM := subject.Method{Sig: &golang.Sig{Name: "Append"}}
	appendM.Name = "Append"
	b := defectBindings(appendM)

	defect, over, planted, _ := defectFor(b, &LawBinding{
		ID: lawid.AppenderMonotonicOffsets, carrier: appendM,
	})
	testkit.True(t, planted, "the carrier is on the interface, so the rule reaches")
	testkit.Equal(t, over.Name, "Append", "planted through the method the law drives")
	_, frozen := defect.(projection.FreezeReturn)
	testkit.True(t, frozen, "the position stops moving while the writes keep landing")
}

// A row that goes Argued says which of the two gaps it met.
//
// One sentence used to serve both and was false for half of them: it
// told a reader no rule exists, on rows where one exists and the
// declaration did not supply what it needs. The two are fixed in
// different places — the rule table, or your own stamp — so a reader
// sent to the wrong one loses the time twice.
func TestArguedSaysWhichGapItMet(t *testing.T) {
	t.Parallel()

	t.Run("no rule reaches the claim", func(t *testing.T) {
		t.Parallel()
		_, _, planted, why := defectFor(defectBindings(), &LawBinding{ID: "AUTO-NOBODY-WROTE-THIS"})
		testkit.False(t, planted, "nothing in the table reaches it")
		testkit.Equal(t, why, NoRule, "so the reader is sent to the rule table")
	})

	t.Run("a rule reaches it and the declaration falls short", func(t *testing.T) {
		t.Parallel()
		// The lifecycle rule exists and reads a sentinel stamp. A
		// declaration carrying none gives it nothing to plant.
		_, _, planted, why := defectFor(defectBindings(),
			&LawBinding{ID: lawid.LifecycleAfterClose})
		testkit.False(t, planted, "the rule found no sentinel to report once")
		testkit.Equal(t, why, RuleDeclined,
			"so the reader is sent to their own declaration instead")
	})
}
