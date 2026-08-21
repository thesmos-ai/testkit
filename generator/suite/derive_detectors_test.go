// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Internal for the reason derive_stamps_test.go is: the miss rule reads
// its sentinel through the unexported mixin-param projection.
package suite

import (
	"testing"

	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/aggregator"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/reader"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/writer"

	"go.thesmos.sh/testkit"
	vocab "go.thesmos.sh/testkit/engine/suite"
	"go.thesmos.sh/testkit/generator/internal/projection"
	"go.thesmos.sh/testkit/generator/internal/subject"
)

// Where a detector rule declines, and whether it says so.
//
// Two ways to decline and they are not the same event. A rule that
// REFUSES names a gap the generated header prints, because the shape
// was owed a claim and something specific stopped it. A rule that
// simply licenses nothing was never owed one — an aggregator on an
// interface nothing seeds has no number to equal, and printing a gap
// for it would report the absence of a check nobody expected.
//
// The corpus exercises both paths incidentally; nothing pinned which
// was which, so a rule that started refusing where it used to be silent
// would have added noise to 129 generated headers without failing
// anything.

// detectorCase is one shape a detector rule meets, and what it owes.
type detectorCase struct {
	name string
	rule stampRule
	m    subject.Method

	// seeded says the run zips a corpus from its pools, which is what
	// puts something there for a hit to find.
	seeded bool

	// want is the segments the rule licenses; refusals the gaps it
	// names rather than emitting.
	want     []string
	refusals int

	// why is the substring a named gap must carry, empty where the
	// case expects none.
	why string
}

func (c detectorCase) Name() string { return c.name }

func TestDetectorRulesDeclineTheUnstateable(t *testing.T) {
	t.Parallel()

	testkit.TableTest(t, []detectorCase{
		{
			name: "a reader with nothing to supply it refuses, and says what would",
			rule: missRule,
			m:    stampMethod("Encode", reader.Name),
			// A transform is reader-shaped and stores nothing, so no
			// draw is one nothing supplied.
			refusals: 1,
			why:      "no method here stores anything",
		},
		{
			name: "a reader taking no input is silent rather than refused",
			rule: missRule,
			m:    bareMethod("All", reader.Name),
			// Nothing was owed: a miss is about an input, and this
			// method has none for a caller to get wrong.
		},
		{
			name:   "a seeded reader owes both the miss and the hit",
			rule:   missRule,
			m:      stampMethod("Lookup", reader.Name),
			seeded: true,
			want:   []string{vocab.SegMiss, vocab.SegHit},
		},
		{
			name: "an aggregator on an unseeded run is silent",
			rule: countRule,
			m:    bareMethod("Len", aggregator.Name),
			// No number to equal, and the miss beside it already names
			// the gap — a second refusal would say it twice.
		},
		{
			name:   "a seeded aggregator owes its count",
			rule:   countRule,
			m:      bareMethod("Size", aggregator.Name),
			seeded: true,
			want:   []string{vocab.SegCount},
		},
		{
			name:     "an answering writer that answers nothing refuses",
			rule:     answerRule,
			m:        erroring(stampMethod("Put", "")),
			refusals: 1,
			why:      "it answers no value",
		},
		{
			name: "an answering writer that answers owes the non-zero claim",
			rule: answerRule,
			m:    erroring(answering(stampMethod("Put", ""), "Value")),
			want: []string{vocab.SegAnswer},
		},
	}, func(t *testing.T, tc detectorCase) {
		f := stampIface(tc.m)
		if tc.seeded {
			f = seededIface(tc.m)
		}
		plans, refusals := tc.rule(f, tc.m, callOf(tc.m))

		testkit.Len(t, refusals, tc.refusals, "the rule names exactly these gaps")
		if tc.why != "" {
			testkit.Contains(t, refusals[0].Why, tc.why,
				"the reason sends a reader to the thing they have to change")
		}

		got := make([]string, 0, len(plans))
		for _, p := range plans {
			got = append(got, p.ID.Seg)
		}
		testkit.Equal(t, got, tc.want, "the rule licenses exactly these checks")
	})
}

// The miss body and its double are chosen by the same question, and a
// run that answered them differently would plant evidence for a claim
// it did not make.
func TestMissBodyAndDefectAgreeOnTheSentinel(t *testing.T) {
	t.Parallel()

	t.Run("a declared sentinel takes the errors.Is arm on both sides", func(t *testing.T) {
		t.Parallel()
		m := sentinelReader()
		f := stampIface(m, stampMethod("Put", writer.Name))

		body := missBody(f, m, "kv.ErrNotFound")
		testkit.Equal(t, body.BodyKind(), projection.KindReportsSentinel,
			"with a sentinel the claim is that error and nothing else")

		defect := missDefect(f, m, "kv.ErrNotFound")
		testkit.Equal(t, defect.DefectKind(), projection.KindAnswersAnyway,
			"and a double that merely answers has already broken it")
	})

	t.Run("no sentinel takes the zero arm on both sides", func(t *testing.T) {
		t.Parallel()
		m := stampMethod("Lookup", reader.Name)
		f := seededIface(m)

		body := missBody(f, m, "")
		testkit.Equal(t, body.BodyKind(), projection.KindAnswersZero,
			"without one the claim is about the value slot")

		defect := missDefect(f, m, "")
		testkit.Equal(t, defect.DefectKind(), projection.KindAnswersWithValue,
			"so the double has to plant a live value; the zero would state the claim")
	})
}
