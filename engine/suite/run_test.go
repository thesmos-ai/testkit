// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"flag"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"

	"go.thesmos.sh/testkit/clock"
)

// sut is the smallest thing a check can be handed.
type sut struct{ name string }

// legsOf drives one subject through a suite and returns the report's legs,
// keyed by check ID.
//
// Internal because the report is written from a Cleanup — parallel legs
// outlive Run's body, so there is nothing to return — and runSubject already
// takes the report to fill. Driving it here reads the same value the consumer
// eventually sees, without inventing an accessor for the tests' benefit.
func legsOf(t *testing.T, s Suite[sut], sub Subject[sut], parallel bool) map[string]Leg {
	t.Helper()

	rep := &Report{Format: ReportFormat, Suite: s.Name, Subjects: 1, Checks: len(s.Checks)}
	var mu sync.Mutex
	t.Run("drive", func(it *testing.T) {
		runSubject(it, s, sub, parallel, rep, &mu)
	})
	rep.finish()

	out := make(map[string]Leg, len(rep.Legs))
	for _, l := range rep.Legs {
		out[l.Check] = l
	}
	return out
}

func aSubject() Subject[sut] {
	return Subject[sut]{Name: "only", New: func(testing.TB) sut { return sut{name: "only"} }}
}

// TestNoteMarksTheLegDidNotRun is the point of the channel: a check that finds
// its preconditions never engaged says so, and the leg stops being counted
// among the passes.
func TestNoteMarksTheLegDidNotRun(t *testing.T) {
	t.Parallel()

	legs := legsOf(t, Suite[sut]{
		Name: "notes",
		Checks: []Check[sut]{
			{
				ID: "Method/engaged", Class: "model/laws", Claim: "engaged",
				RunWith: func(testing.TB, Subject[sut]) {},
			},
			{
				ID: "Method/vacuous", Class: "model/laws", Claim: "did not engage",
				RunWith: func(_ testing.TB, sub Subject[sut]) { sub.Note(ReasonVacuous) },
			},
		},
	}, aSubject(), false)

	if got := legs["Method/engaged"].Outcome; got != Passed {
		t.Errorf("an unnoted leg passed, got %q", got)
	}
	if got := legs["Method/vacuous"].Outcome; got != DidNotRun {
		t.Errorf("a noted leg did not run, got %q", got)
	}
	if got := legs["Method/vacuous"].Reason; got != ReasonVacuous {
		t.Errorf("the reason survives to the report, got %q", got)
	}
}

// TestTierComesOnlyFromTheCheck pins the honesty rule: the runner reports the
// tier a check declared and never derives one.
//
// Deriving is what it used to do — model class plus an oracle meant
// "differential", otherwise "twin" — and both halves lied. It could not see a
// derived reference at all, because that is built inside the check and looks
// like any other constructor from outside; and it labelled checks comparing
// against no reference whatsoever, like a clocked expiry claim, as riding the
// twin floor.
func TestTierComesOnlyFromTheCheck(t *testing.T) {
	t.Parallel()

	legs := legsOf(t, Suite[sut]{
		Name: "tiers",
		Checks: []Check[sut]{
			{
				ID: "Method/derived", Class: "model/laws", Claim: "built a derived reference",
				RunWith: func(_ testing.TB, sub Subject[sut]) { sub.NoteTier("derived") },
			},
			{
				ID: "Method/silent", Class: "model/clocked", Claim: "built no reference",
				RunWith: func(testing.TB, Subject[sut]) {},
			},
		},
	}, aSubject(), false)

	if got := legs["Method/derived"].Tier; got != "derived" {
		t.Errorf("the declared tier reaches the report, got %q", got)
	}
	if got := legs["Method/silent"].Tier; got != "" {
		t.Errorf("a check that built no reference has no tier, got %q", got)
	}
}

// TestAnOracleAloneDoesNotNameATier is the other half of the same rule.
//
// A subject carrying the oracle's constructor used to be enough to stamp every
// model-class leg "differential", including legs that never touched it.
func TestAnOracleAloneDoesNotNameATier(t *testing.T) {
	t.Parallel()

	sub := aSubject()
	sub.reference = func(testing.TB) sut { return sut{name: "oracle"} }

	legs := legsOf(t, Suite[sut]{
		Name: "oracle",
		Checks: []Check[sut]{{
			ID: "Method/quiet", Class: "model/laws", Claim: "never asked for the reference",
			RunWith: func(testing.TB, Subject[sut]) {},
		}},
	}, sub, false)

	if got := legs["Method/quiet"].Tier; got != "" {
		t.Errorf("an available oracle is not a tier a check used, got %q", got)
	}
}

// TestNotesDoNotLeakBetweenParallelLegs guards the per-leg copy in runSubject.
//
// The subject is shared by every check on it, and the closures each leg
// registers capture that one variable. Installing the note channel on it left
// all eight legs writing to and reading from whichever channel the last
// iteration created.
//
// This guard only bites under -race, and the measurement says so: with the
// copy removed the detector reports eight races here, and without the detector
// the same code passes. Each leg writes its tier and reads it straight back,
// so nothing observes the corruption unless two legs actually interleave
// between those two lines. Run it with -race or it is asserting very little —
// which is the honest description of a test for a data race, and the reason
// this file's guard is not presented as a check on the reported tier.
func TestNotesDoNotLeakBetweenParallelLegs(t *testing.T) {
	t.Parallel()

	names := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
	want := func(name string) string {
		if name > "d" {
			return "differential"
		}
		return "derived"
	}

	checks := make([]Check[sut], 0, len(names))
	for _, name := range names {
		tier := Tier(want(name))
		checks = append(checks, Check[sut]{
			ID: ID("Method/" + name), Class: "model/laws", Claim: "declares its own tier",
			RunWith: func(_ testing.TB, sub Subject[sut]) { sub.NoteTier(tier) },
		})
	}

	legs := legsOf(t, Suite[sut]{Name: "parallel", Checks: checks}, aSubject(), true)
	for _, name := range names {
		id := "Method/" + name
		if got := legs[id].Tier; got != want(name) {
			t.Errorf("%s reported tier %q, want %q — a leg read another leg's note",
				id, got, want(name))
		}
	}
}

// TestNoteOnADetachedSubjectIsANoOp holds the nil guard: a consumer holding a
// Subject outside a run can call these without a panic, and nothing records.
func TestNoteOnADetachedSubjectIsANoOp(t *testing.T) {
	t.Parallel()

	sub := aSubject()
	sub.Note("vacuous")
	sub.NoteTier("derived")
	if sub.note != nil {
		t.Fatal("a subject outside a run has no note channel to write to")
	}
	if strings.TrimSpace(sub.Name) == "" {
		t.Fatal("the subject is otherwise untouched")
	}
}

// TestSkipIsDidNotRun pins the recorder's t.Skipped() branch: a check that
// skips itself never reached a verdict, and recording it Passed would count
// an unexercised claim among the passes — the silent-green class.
func TestSkipIsDidNotRun(t *testing.T) {
	t.Parallel()

	legs := legsOf(t, Suite[sut]{
		Name: "skips",
		Checks: []Check[sut]{{
			ID: "Method/skips", Class: "signature/zero-on-error", Claim: "skips itself",
			Run: func(tb testing.TB, _ sut) { tb.Skip("no error to inspect") },
		}},
	}, aSubject(), false)

	if got := legs["Method/skips"].Outcome; got != DidNotRun {
		t.Errorf("a skipped check did not run, got %q", got)
	}
	if got := legs["Method/skips"].Reason; got != ReasonVacuous {
		t.Errorf("a self-skip is a precondition that never engaged, got %q", got)
	}
}

// TestLegVerdict is the recorder's truth table, pure so a red path can be
// pinned without staging a real subtest failure.
func TestLegVerdict(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name                   string
		unmet, failed, skipped bool
		noted                  string
		wantOut                Disposition
		wantReason             string
	}{
		{name: "clean pass", wantOut: Passed},
		{
			name: "unmet wins over everything", unmet: true, failed: true, noted: "x",
			wantOut: DidNotRun, wantReason: ReasonUnmet,
		},
		{
			name: "a failure beats a noted reason", failed: true, noted: ReasonVacuous,
			wantOut: Failed,
		},
		{
			name: "a self-skip did not run", skipped: true,
			wantOut: DidNotRun, wantReason: ReasonVacuous,
		},
		{
			name: "a skip keeps the check's own reason", skipped: true, noted: "custom",
			wantOut: DidNotRun, wantReason: "custom",
		},
		{
			name: "a noted reason alone did not run", noted: ReasonVacuous,
			wantOut: DidNotRun, wantReason: ReasonVacuous,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			out, reason := legVerdict(c.unmet, c.failed, c.skipped, c.noted)
			if out != c.wantOut || reason != c.wantReason {
				t.Errorf("legVerdict(%v,%v,%v,%q) = (%q,%q), want (%q,%q)",
					c.unmet, c.failed, c.skipped, c.noted, out, reason, c.wantOut, c.wantReason)
			}
		})
	}
}

// TestPanicBecomesAFailedLegAndTheReportStillPrints re-runs this binary
// against a deliberately panicking suite. Three things must hold: the run
// fails rather than aborting the process mid-flight, the panic is a named
// failure on its own leg, and the report still prints — the exact three
// things a raw panic used to destroy at once.
func TestPanicBecomesAFailedLegAndTheReportStillPrints(t *testing.T) {
	t.Parallel()
	if os.Getenv("GEN_SUITE_PANIC_HELPER") == "1" {
		Run(t, Suite[sut]{
			Name: "panics",
			Checks: []Check[sut]{
				{
					ID: "Method/panics", Class: "signature/smoke", Claim: "panics",
					Run: func(testing.TB, sut) { panic("nil map, say") },
				},
				{
					ID: "Method/fine", Class: "signature/smoke", Claim: "passes",
					Run: func(testing.TB, sut) {},
				},
			},
		}, aSubject())
		return
	}

	// Re-run this binary with the helper armed; only the branch above
	// executes there, so the child is one suite and nothing else.
	//nolint:gosec // G204: the test re-executes its own binary with constant flags.
	cmd := exec.CommandContext(t.Context(), os.Args[0],
		"-test.run", "TestPanicBecomesAFailedLegAndTheReportStillPrints",
		"-test.v", "-test.count=1",
	)
	cmd.Env = append(os.Environ(), "GEN_SUITE_PANIC_HELPER=1")
	out, err := cmd.CombinedOutput()

	if err == nil {
		t.Fatalf("a suite with a panicking check must fail; output:\n%s", out)
	}
	text := string(out)
	if !strings.Contains(text, "panicked") {
		t.Errorf("the panic must be a named failure; output:\n%s", text)
	}
	if !strings.Contains(text, "1 passed, 1 failed") {
		t.Errorf("the report must still print, with the panic as a failed leg; output:\n%s", text)
	}
}

// TestZeroFalsifiabilityEncodesUnproven pins the normalization: the zero
// value and the named constant are one state, spelled one way, in a
// versioned encoding consumers switch on.
func TestZeroFalsifiabilityEncodesUnproven(t *testing.T) {
	t.Parallel()

	legs := legsOf(t, Suite[sut]{
		Name: "zero",
		Checks: []Check[sut]{{
			ID: "Method/hand", Class: "own/hand-written", Claim: "no proof claimed",
			Run: func(testing.TB, sut) {},
		}},
	}, aSubject(), false)

	if got := legs["Method/hand"].Falsifiable; got != string(FalsifiableUnproven) {
		t.Errorf("the zero Falsifiability encodes as %q, want %q", got, FalsifiableUnproven)
	}
}

// TestInducerResolvesWrappedSentinels pins errors.Is resolution: a consumer
// who registered a wrapped sentinel followed the failure message's advice,
// and the gate must find it.
func TestInducerResolvesWrappedSentinels(t *testing.T) {
	t.Parallel()

	base := sentinelError("closed")
	sub := aSubject()
	sub.Induces = map[error]func(testing.TB, sut){
		wrapError{base}: func(testing.TB, sut) {},
	}

	if _, ok := sub.Inducer(base); !ok {
		t.Error("a wrapped registration must satisfy the bare sentinel")
	}

	sub.Induces = map[error]func(testing.TB, sut){base: nil}
	if _, ok := sub.Inducer(base); ok {
		t.Error("a nil trigger is not armed; passing the gate would panic at the call site")
	}
}

type sentinelError string

func (e sentinelError) Error() string { return string(e) }

type wrapError struct{ inner error }

func (w wrapError) Error() string { return "wrap: " + w.inner.Error() }
func (w wrapError) Unwrap() error { return w.inner }

// rapidTestFlags stands in for the flags rapid registers when linked; this
// test binary does not link rapid, so the policy's flag.Lookup would
// otherwise find nothing to map onto.
func rapidTestFlags() {
	if flag.Lookup("rapid.seed") == nil {
		flag.String("rapid.seed", "0", "test stand-in")
	}
	if flag.Lookup("rapid.checks") == nil {
		flag.Int("rapid.checks", 100, "test stand-in")
	}
}

// TestApplyRapidEnv pins the seed policy: the environment maps onto
// rapid's flags, an explicit command-line flag wins over it, and a value
// the flag rejects is an error naming the variable.
func TestApplyRapidEnv(t *testing.T) {
	rapidTestFlags()

	t.Setenv(EnvRapidSeed, "12345")
	t.Setenv(EnvRapidChecks, "7")
	if err := applyRapidEnv(); err != nil {
		t.Fatalf("applyRapidEnv: %v", err)
	}
	if got := flag.Lookup("rapid.seed").Value.String(); got != "12345" {
		t.Errorf("rapid.seed = %q, want the pinned 12345", got)
	}
	if got := flag.Lookup("rapid.checks").Value.String(); got != "7" {
		t.Errorf("rapid.checks = %q, want the budget 7", got)
	}

	t.Setenv(EnvRapidChecks, "not-a-number")
	err := applyRapidEnv()
	if err == nil || !strings.Contains(err.Error(), EnvRapidChecks) {
		t.Errorf("a rejected value must fail naming the variable, got %v", err)
	}
}

// TestDoors pins what a defect or a control may borrow from a run's
// subjects: the open-set answers, which are facts about the declaration,
// and not the three per-instance arms.
func TestDoors(t *testing.T) {
	t.Parallel()

	first := aSubject()
	first.Provides = map[Capability]any{"less": "first", CapClock: "not a constructor"}
	second := aSubject()
	second.Provides = map[Capability]any{"less": "second", "stats": 7}

	doors := Doors(first, second)

	if got := doors["less"]; got != "first" {
		t.Errorf("the first answer wins, got %v", got)
	}
	if got := doors["stats"]; got != 7 {
		t.Errorf("a door only the second subject answers still arrives, got %v", got)
	}
	if doors[CapClock] != nil {
		t.Error("a clock is built by a constructor, not lifted off another subject")
	}
	if Doors[sut]() != nil {
		t.Error("no subjects is no doors")
	}
}

// TestCanRun pins the exported gate across every capability arm: an armed
// subject passes, a bare one is refused naming the field that arms it, and
// the refusal never teaches a drop — outside a run a drop is not the fix.
func TestCanRun(t *testing.T) {
	t.Parallel()

	sentinel := sentinelError("closed")

	armed := aSubject()
	armed.OnClock = func(testing.TB, *clock.TestClock) sut { return sut{} }
	armed.Recover = func(_ testing.TB, s sut) sut { return s }
	armed.Induces = map[error]func(testing.TB, sut){sentinel: func(testing.TB, sut) {}}
	armed.Provides = map[Capability]any{"door": struct{}{}}

	cases := []struct {
		name    string
		needs   Caps
		wantWhy string
	}{
		{name: "clock", needs: NeedsClock(), wantWhy: "OnClock"},
		{name: "induce", needs: NeedsInduce(sentinel), wantWhy: "Induce"},
		{name: "recover", needs: NeedsRecover(), wantWhy: "Recover"},
		{name: "open door", needs: Needs(Capability("door"), nil), wantWhy: "Provides"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			chk := Check[sut]{ID: "Method/needy", Needs: c.needs}
			if ok, why := CanRun(chk, armed); !ok {
				t.Errorf("an armed subject must pass the gate, got %q", why)
			}
			ok, why := CanRun(chk, aSubject())
			if ok {
				t.Fatal("a bare subject must be refused")
			}
			if !strings.Contains(why, c.wantWhy) {
				t.Errorf("the refusal must name the arm, got %q", why)
			}
			if strings.Contains(why, "drop the check") {
				t.Errorf("CanRun must not teach a drop, got %q", why)
			}
		})
	}

	// A declaration mistake — a non-error induce value — is the check's
	// own gap, reported with no arm to offer.
	bad := Check[sut]{ID: "Method/bad-decl", Needs: Needs(CapInduce, 42)}
	if ok, why := CanRun(bad, armed); ok || !strings.Contains(why, "not a sentinel") {
		t.Errorf("a non-sentinel induce declaration must be refused, got (%v, %q)", ok, why)
	}
}

// TestUnmetCapsTeachesTheDrop pins the runner's rendering: the same gate
// CanRun exposes, with the drop hint appended — the one difference
// between the two surfaces.
func TestUnmetCapsTeachesTheDrop(t *testing.T) {
	t.Parallel()

	c := Check[sut]{ID: "Method/needs-clock", Needs: NeedsClock()}
	why := unmetCaps(Suite[sut]{}, c, aSubject())
	wantDrop := `drop the check with Without("Method/needs-clock")`
	if !strings.Contains(why, "OnClock") || !strings.Contains(why, wantDrop) {
		t.Errorf("the runner's message must carry the arm and the drop, got %q", why)
	}
}

// TestSetupAndTeardownBracketTheSubject pins the heavy-fixture seam:
// Setup once before any leg, Teardown once after the last, New still
// per leg.
func TestSetupAndTeardownBracketTheSubject(t *testing.T) {
	var order []string
	var mu sync.Mutex
	note := func(s string) { mu.Lock(); order = append(order, s); mu.Unlock() }

	t.Run("run", func(t *testing.T) {
		Run(t, Suite[sut]{
			Name: "s",
			Checks: []Check[sut]{
				{ID: "A/one", Class: ClassSmoke, Claim: "one", Run: func(testing.TB, sut) { note("leg") }},
				{ID: "B/two", Class: ClassSmoke, Claim: "two", Run: func(testing.TB, sut) { note("leg") }},
			},
		}, Subject[sut]{
			Name:     "x",
			New:      func(testing.TB) sut { return sut{} },
			Setup:    func(testing.TB) { note("setup") },
			Teardown: func(testing.TB) { note("teardown") },
		})
	})

	want := []string{"setup", "leg", "leg", "teardown"}
	if len(order) != 4 || order[0] != "setup" || order[3] != "teardown" {
		t.Errorf("setup and teardown must bracket every leg exactly once: got %v want %v", order, want)
	}
}

// TestNoteUnengagedReachesTheLeg pins the partial-vacuity plumbing: a
// PASSING leg that bound laws it never engaged carries their names into
// the report.
func TestNoteUnengagedReachesTheLeg(t *testing.T) {
	legs := legsOf(t, Suite[sut]{
		Name: "s",
		Checks: []Check[sut]{{
			ID:    "model/laws",
			Class: ClassLaws,
			Claim: "laws hold",
			RunWith: func(_ testing.TB, sub Subject[sut]) {
				sub.NoteUnengaged([]string{"AUTO-QUIET-LAW"})
			},
		}},
	}, Subject[sut]{Name: "x", New: func(testing.TB) sut { return sut{} }}, false)

	leg := legs["model/laws"]
	if leg.Outcome != Passed || len(leg.Unengaged) != 1 || leg.Unengaged[0] != "AUTO-QUIET-LAW" {
		t.Errorf("a passing leg must carry the laws it never engaged: %+v", leg)
	}
}

// TestValidateNamesTheUnknowns pins the wiring refusals that render the
// known-check list: a drop or excuse naming no check is a fix-naming
// error carrying every real ID.
func TestValidateNamesTheUnknowns(t *testing.T) {
	t.Parallel()

	s := Suite[sut]{Name: "s", Checks: []Check[sut]{
		{ID: "A/one", Class: ClassSmoke, Claim: "one", Run: func(testing.TB, sut) {}},
	}}

	ghost := s.Without("Z/ghost")
	err := validate(ghost, []Subject[sut]{{Name: "x", New: func(testing.TB) sut { return sut{} }}})
	if err == nil || !strings.Contains(err.Error(), `"Z/ghost"`) || !strings.Contains(err.Error(), "A/one") {
		t.Errorf("an unknown drop must name itself and the known checks, got %v", err)
	}

	err = validate(s, []Subject[sut]{{
		Name:    "x",
		New:     func(testing.TB) sut { return sut{} },
		Excused: map[ID]bool{"Z/ghost": true},
	}})
	if err == nil || !strings.Contains(err.Error(), "excused") || !strings.Contains(err.Error(), "A/one") {
		t.Errorf("an unknown excuse must name the subject and the known checks, got %v", err)
	}

	for class, why := range map[Class]string{
		"noslash":            "a class without a family segment",
		"alien/smoke":        "a family the report does not bucket",
		"signature/Bad_Slug": "a leaf that is not a slug",
	} {
		err = validate(Suite[sut]{Name: "s", Checks: []Check[sut]{
			{ID: "A/one", Class: class, Claim: "one", Run: func(testing.TB, sut) {}},
		}}, []Subject[sut]{{Name: "x", New: func(testing.TB) sut { return sut{} }}})
		if err == nil || !strings.Contains(err.Error(), "class") {
			t.Errorf("%s must be refused, got %v", why, err)
		}
	}
}

func TestTextNamesPartialVacuityAndTiers(t *testing.T) {
	t.Parallel()

	rep := &Report{Format: ReportFormat, Suite: "S", Subjects: 2, Checks: 1}
	rep.add("a", "model/laws", string(ClassLaws), legOf{
		outcome: Passed, falsifiable: Proven(), tier: string(TierDerived),
		unengaged: []string{"AUTO-QUIET"},
	})
	rep.add("b", "model/laws", string(ClassLaws), legOf{
		outcome: Passed, falsifiable: Proven(), tier: string(TierTwin),
	})
	rep.finish()

	text := rep.Text()
	if !strings.Contains(text, "passed with unengaged laws: model/laws [AUTO-QUIET]") {
		t.Errorf("a green leg that asserted less than it bound must say so:\n%s", text)
	}
	if !strings.Contains(text, "derived references and twins") {
		t.Errorf("the fallback tier line must name the mix:\n%s", text)
	}
}

// A run that drives properties needs the flags the seed bridge maps
// onto, and says so when they are gone.
//
// applyRapidEnv skips a flag it cannot find on purpose — a CI variable
// is global and most packages link no rapid — and that silence is right
// everywhere except here. A package with model or sim checks links the
// engine and rapid with it, so a missing flag means rapid renamed one
// and a pinned seed has been doing nothing, with the run still green and
// no longer reproducible.
//
// The lookup is handed in because a binary that links rapid always finds
// the flags: without the seam this guard could not fail where it runs,
// and a test of it would assert nothing.
func TestTheSeedBridgeIsHeldAliveForPropertyRuns(t *testing.T) {
	t.Parallel()

	absent := func(string) *flag.Flag { return nil }
	property := Suite[sut]{Checks: []Check[sut]{{ID: "model/store/differential"}}}
	crash := Suite[sut]{Checks: []Check[sut]{{ID: "sim/store/recovery"}}}
	fixed := Suite[sut]{Checks: []Check[sut]{{ID: "Get/smoke"}, {ID: "mixin/store/reader"}}}

	t.Run("a model check with no flag behind it is refused", func(t *testing.T) {
		t.Parallel()
		err := bridgeAlive(property, absent)
		if err == nil {
			t.Fatal("a dead bridge passed; a pinned seed would silently do nothing")
		}
		for _, want := range []string{"rapid.seed", EnvRapidSeed, "silently doing nothing"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("message does not name %q: %v", want, err)
			}
		}
	})

	t.Run("the crash schedule counts too", func(t *testing.T) {
		t.Parallel()
		if bridgeAlive(crash, absent) == nil {
			t.Fatal("the sim family draws and shrinks, so it needs the bridge as much")
		}
	})

	t.Run("a run of fixed call sequences is left alone", func(t *testing.T) {
		t.Parallel()
		if err := bridgeAlive(fixed, absent); err != nil {
			t.Fatalf("no check here draws, so no bridge is owed: %v", err)
		}
	})

	// No subtest here for the live lookup. This package links testing
	// and clock and nothing else — the doctrine legs exists to keep — so
	// rapid's flags are absent, and TestRapidEnvAppliesToFlags registers
	// stand-ins for them into the global set. Asserting either way about
	// flag.Lookup from here would be asserting about which test ran
	// first. The live path is exercised where rapid is genuinely linked,
	// which is every generated package in the corpus.
}
