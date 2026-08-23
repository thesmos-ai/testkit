// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"errors"
	"flag"
	"fmt"
	"maps"
	"os"
	"runtime/debug"
	"slices"
	"sort"
	"strings"
	"sync"
	"testing"
)

// Run executes every check against every subject.
//
// Subtests nest as t.Run(subject) then t.Run(checkID), so an ID is also a
// -run pattern: go test -run 'TestStore/pebble/model' reruns one family
// against one backend.
//
// Run does all its validation before anything executes, and fails on:
// zero subjects; two subjects with the same name; a subject that is both
// Serial and Oracle; a dropped ID that names no check; a malformed ID; a
// check with neither or both of Run and RunWith; and more than one
// oracle. Every message names what to fix.
//
// A check whose Caps the subject cannot meet fails, and never skips. The
// message says which field of the subject to fill in, or which ID to drop.
//
// The report is written from t.Cleanup, not from this function's body:
// parallel subtests outlive the body, so a log here would print an empty
// report.
func Run[S any](t *testing.T, s Suite[S], subjects ...Subject[S]) {
	t.Helper()

	rapidPolicyOnce.Do(func() { errRapidPolicy = applyRapidEnv() })
	if errRapidPolicy != nil {
		t.Fatalf("suite.Run: %v", errRapidPolicy)
	}
	if err := rapidBridgeAlive(s); err != nil {
		t.Fatalf("suite.Run: %v", err)
	}

	if err := validate(s, subjects); err != nil {
		t.Fatalf("suite.Run: %v", err)
	}

	oracle, hasOracle := findOracle(subjects)
	rep := &Report{
		Format:   ReportFormat,
		Suite:    s.Name,
		Subjects: len(subjects),
		// Checks that will actually run: the multiplication in the
		// report's headline has to hold, and dropped checks produce no
		// legs.
		Checks: len(s.Checks) - len(s.dropped),
	}
	var mu sync.Mutex

	for id := range s.dropped {
		rep.Dropped = append(rep.Dropped, string(id))
	}
	sort.Strings(rep.Dropped)

	// RunFailed reconciles the report with the test's own exit state: a
	// cleanup that fails, or an unmet capability's Fatal, reddens the run
	// without producing a Failed leg, and a report that said "0 failed"
	// about a red run would be the silent-green class in its own narrator.
	// The flag's zero means "rapid drew its own seed", and that internal
	// draw is not observable through any rapid API — its one home is the
	// replay line a failure prints. Recording "0" here would hand triage
	// bots a lying number; "randomized" is the truth this run can state.
	if f := flag.Lookup("rapid.seed"); f != nil {
		if v := f.Value.String(); v != "0" {
			rep.RapidSeed = v
		} else {
			rep.RapidSeed = "randomized"
		}
	}

	t.Cleanup(func() {
		rep.RunFailed = t.Failed()
		rep.finish()
		t.Log("\n" + rep.Text())
		if dir := os.Getenv(EnvReportDir); dir != "" {
			if err := rep.WriteArtifact(dir, t.Name()); err != nil {
				t.Errorf("%s is set and the report could not be written: %v", EnvReportDir, err)
			}
		}
	})

	if hasOracle {
		rep.Oracle = oracle.Name
	}

	// One loop for both kinds of subject. A serial subject runs inline,
	// while parallel subtests queue behind the parent's return — so serial
	// subjects execute FIRST, alone, and the parallel group runs after.
	// Serial means "never concurrent with anything", not "last".
	for _, sub := range subjects {
		if hasOracle && !sub.Oracle {
			sub.reference = oracle.New
		}
		t.Run(sub.Name, func(t *testing.T) {
			if sub.Setup != nil {
				sub.Setup(t)
			}
			if sub.Teardown != nil {
				t.Cleanup(func() { sub.Teardown(t) })
			}
			if !sub.Serial {
				t.Parallel()
			}
			runSubject(t, s, sub, !sub.Serial, rep, &mu)
		})
	}
}

// dropHint renders the drop a failure message teaches, through the
// suite's formatter when the generated package supplied one.
func (s Suite[S]) dropHint(id ID) string {
	if s.DropHint != nil {
		return s.DropHint(id)
	}
	return fmt.Sprintf("Without(%q)", id)
}

func runSubject[S any](
	t *testing.T, s Suite[S], sub Subject[S], parallel bool,
	rep *Report, mu *sync.Mutex,
) {
	t.Helper()
	for _, c := range s.Checks {
		if s.dropped[c.ID] {
			continue
		}
		if sub.Excused[c.ID] {
			// Excused, not silent: the leg is in the report under its own
			// reason, so a subject's exemptions are always visible.
			mu.Lock()
			rep.add(sub.Name, string(c.ID), string(c.Class),
				legOf{
					outcome: DidNotRun, reason: ReasonExcused,
					falsifiable: c.Falsifiable, strength: c.Strength,
				})
			mu.Unlock()
			continue
		}
		t.Run(string(c.ID), func(t *testing.T) {
			if parallel {
				t.Parallel()
			}

			// unmet is the runner's own verdict, tracked apart from the
			// check's: a capability the subject cannot meet means the
			// claim was never judged, even though the Fatal below reddens
			// the test.
			unmet := false

			// A per-leg copy of the subject, because the note travels back
			// through it. The parameter is shared by every check on this
			// subject, and these legs run in parallel: installing the note
			// on the shared value would have each leg overwrite the last
			// one's channel before any of them ran — a wrong answer and a
			// data race by the same line.
			sub := sub
			sub.note = &legNote{}

			// The recorder. Registered first so it runs last, after the
			// panic guard below has had its say. Precedence, most to
			// least specific:
			//
			//	unmet          -> DidNotRun (the claim was never judged)
			//	t.Failed()     -> Failed    (the subject broke the claim;
			//	                             a noted reason does not undo a
			//	                             real failure)
			//	t.Skipped()    -> DidNotRun (a self-skip is a precondition
			//	                             that never engaged)
			//	noted reason   -> DidNotRun
			//	otherwise      -> Passed
			defer func() {
				leg := legOf{
					falsifiable: c.Falsifiable, strength: c.Strength,
					tier: tierOf(sub), unengaged: sub.note.unengaged,
				}
				leg.outcome, leg.reason = legVerdict(unmet, t.Failed(), t.Skipped(), sub.note.reason)
				mu.Lock()
				rep.add(sub.Name, string(c.ID), string(c.Class), leg)
				mu.Unlock()
			}()

			// The panic guard. A panicking check is a failed check, not a
			// dead process: without this, the panic unwinds past the
			// recorder while t.Failed() is still false — committing a
			// Passed leg — and then kills the run before the report
			// prints.
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("check %s panicked: %v\n%s", c.ID, r, debug.Stack())
				}
			}()

			if missing := unmetCaps(s, c, sub); missing != "" {
				unmet = true
				t.Fatal(missing)
			}
			switch {
			case c.RunWith != nil:
				c.RunWith(t, sub)
			default:
				c.Run(t, sub.New(t))
			}
		})
	}
}

// legVerdict is the recorder's whole decision, pure so its truth table is
// testable without staging real subtest failures. Precedence, most to
// least specific:
//
//	unmet   -> DidNotRun (the claim was never judged)
//	failed  -> Failed    (a noted reason does not undo a real failure)
//	skipped -> DidNotRun (a self-skip is a precondition that never engaged)
//	noted   -> DidNotRun
//	else    -> Passed
func legVerdict(unmet, failed, skipped bool, noted string) (Disposition, string) {
	switch {
	case unmet:
		return DidNotRun, ReasonUnmet
	case failed:
		return Failed, ""
	case skipped:
		return DidNotRun, reasonOr(noted, ReasonVacuous)
	case noted != "":
		return DidNotRun, noted
	default:
		return Passed, ""
	}
}

// reasonOr prefers the check's own account when it gave one.
func reasonOr(noted, fallback string) string {
	if noted != "" {
		return noted
	}
	return fallback
}

// unmetCaps renders the runner's failure for every capability the subject
// does not meet, drop hint included, or the empty string when the subject
// can run the check. It collects EVERY unmet capability, not the first:
// these messages are the tutorial, and a serial tutorial makes a consumer
// arm one field, rerun, and meet the next. Sorted, so the list is stable.
func unmetCaps[S any](s Suite[S], c Check[S], sub Subject[S]) string {
	gaps := capGaps(c, sub)
	missing := make([]string, 0, len(gaps))
	for _, g := range gaps {
		msg := g.need
		if g.arm != "" {
			msg += ".\n  " + g.arm + ",\n  or drop the check with " + s.dropHint(c.ID)
		}
		missing = append(missing, msg)
	}
	return strings.Join(missing, "\n")
}

// CanRun reports whether the subject can take the check, and when it
// cannot, why. It is the same gate the runner applies before a leg —
// exported because prove.Green must judge a control with the runner's own
// gate: a second implementation of "can this subject take this check"
// would drift from the first, and the two verdicts must never disagree.
// The runner's rendering appends the drop hint; this one does not,
// because outside a run a drop is not the fix.
func CanRun[S any](c Check[S], sub Subject[S]) (ok bool, why string) {
	gaps := capGaps(c, sub)
	if len(gaps) == 0 {
		return true, ""
	}
	parts := make([]string, len(gaps))
	for i, g := range gaps {
		parts[i] = g.need
		if g.arm != "" {
			parts[i] += "; " + g.arm
		}
	}
	return false, strings.Join(parts, "\n")
}

// capGap is one unmet capability: the need the check declared and the
// harness change that arms it. Two halves because two renderers compose
// them — the runner appends its drop hint, CanRun does not — and the
// facts must not be restated per renderer.
type capGap struct {
	need string // "<id> needs a clock: subject "bare" has no OnClock"
	arm  string // "set OnClock on its harness — ..."; empty when the gap is the check's own declaration
}

// capGaps collects every capability the subject does not meet, sorted so
// the rendered list is stable.
func capGaps[S any](c Check[S], sub Subject[S]) []capGap {
	var gaps []capGap
	for _, need := range slices.Sorted(maps.Keys(c.Needs)) {
		if g, unmet := capGapOf(c, sub, need, c.Needs[need]); unmet {
			gaps = append(gaps, g)
		}
	}
	return gaps
}

// capGapOf answers for one capability.
//
// A capability the registry does not know is reported rather than ignored: a
// check declaring a need nothing can answer is a check whose need nobody
// reads, which is the shape of defect the open set exists to make impossible
// rather than to reintroduce.
func capGapOf[S any](c Check[S], sub Subject[S], need Capability, value any) (capGap, bool) {
	switch need {
	case CapClock:
		if sub.OnClock != nil {
			return capGap{}, false
		}
		return capGap{
			need: fmt.Sprintf("%s needs a clock: subject %q has no OnClock", c.ID, sub.Name),
			arm:  "set OnClock on its harness — a constructor reading the run's clock",
		}, true
	case CapInduce:
		sentinel, isErr := value.(error)
		if !isErr {
			return capGap{
				need: fmt.Sprintf("%s declares %s with %T, which is not a sentinel", c.ID, need, value),
			}, true
		}
		if _, ok := sub.Inducer(sentinel); ok {
			return capGap{}, false
		}
		return capGap{
			need: fmt.Sprintf("%s needs to induce %v: subject %q has no trigger for it", c.ID, sentinel, sub.Name),
			arm:  "add one to its harness's Induce map",
		}, true
	case CapRecover:
		if sub.Recover != nil {
			return capGap{}, false
		}
		return capGap{
			need: fmt.Sprintf("%s needs to recover over durable state: subject %q has no Recover", c.ID, sub.Name),
			arm:  "set Recover on its harness — a constructor over the prior instance's medium",
		}, true
	default:
		if v, ok := sub.Provides[need]; ok {
			// A check may gate on the VALUE, not just the key: when the
			// need's payload is a validator, a provided value it rejects
			// is a wiring mistake reported here, beside every other one,
			// instead of a type-assertion panic inside the check body.
			if validate, isValidator := c.Needs[need].(func(any) error); isValidator {
				if err := validate(v); err != nil {
					return capGap{
						need: fmt.Sprintf("%s provides capability %q, but the value is unusable: %v",
							sub.Name, need, err),
						arm: "fix the value on the harness's Provide field",
					}, true
				}
			}
			return capGap{}, false
		}
		return capGap{
			need: fmt.Sprintf("%s needs capability %q: subject %q does not provide it", c.ID, need, sub.Name),
			arm:  "set it on the harness's Provide field (lowered onto Subject.Provides)",
		}, true
	}
}

// tierOf reads the tier a check declared through NoteTier, and answers only
// when the check said so itself.
//
// It used to derive this: any model-class check got "differential" when an
// oracle existed and "twin" otherwise. Both halves were wrong. It cannot see
// the middle tier at all — a reference derived from the interface's shape is
// built inside the check, and from out here it is just another constructor —
// and it labelled checks that compare against no reference whatsoever, like a
// clocked expiry claim, as riding the twin floor. A tier nobody reported is
// now absent rather than guessed, and absent is the truth for a check that
// never built a reference.
func tierOf[S any](sub Subject[S]) string {
	if sub.note == nil {
		return ""
	}
	return sub.note.tier
}

func findOracle[S any](subjects []Subject[S]) (Subject[S], bool) {
	for _, sub := range subjects {
		if sub.Oracle {
			return sub, true
		}
	}
	var zero Subject[S]
	return zero, false
}

func validate[S any](s Suite[S], subjects []Subject[S]) error {
	if len(s.Checks) == 0 {
		return errors.New("the suite has no checks")
	}
	if len(subjects) == 0 {
		return errors.New("no subjects; pass a harness")
	}

	seenSubject := map[string]bool{}
	oracles := []string{}
	for _, sub := range subjects {
		if sub.Name == "" {
			return errors.New("a subject has no name")
		}
		if seenSubject[sub.Name] {
			return fmt.Errorf("two subjects are both named %q", sub.Name)
		}
		seenSubject[sub.Name] = true
		if sub.New == nil {
			return fmt.Errorf("subject %q has no constructor", sub.Name)
		}
		if sub.Oracle {
			oracles = append(oracles, sub.Name)
		}
		if sub.Oracle && sub.Serial {
			return fmt.Errorf(
				"subject %q is both Serial and Oracle. The oracle's constructor is "+
					"called concurrently by every parallel subject's model legs, so a "+
					"subject that cannot take concurrent construction cannot be the "+
					"reference; pick one", sub.Name,
			)
		}
	}
	if len(oracles) > 1 {
		return fmt.Errorf(
			"%d subjects are marked Oracle (%s); at most one subject is the reference",
			len(oracles), strings.Join(oracles, ", "),
		)
	}

	known := map[ID]bool{}
	for _, c := range s.Checks {
		if err := ValidateID(c.ID); err != nil {
			return err
		}
		if known[c.ID] {
			return fmt.Errorf("two checks share the ID %q", c.ID)
		}
		known[c.ID] = true

		switch {
		case c.Run == nil && c.RunWith == nil:
			return fmt.Errorf("check %q sets neither Run nor RunWith", c.ID)
		case c.Run != nil && c.RunWith != nil:
			return fmt.Errorf("check %q sets both Run and RunWith; set exactly one", c.ID)
		}
		if c.Claim == "" {
			return fmt.Errorf("check %q has no claim; the claim is written to checks.lock", c.ID)
		}
		if err := validateClass(c.ID, c.Class); err != nil {
			return err
		}
	}

	for id := range s.dropped {
		if !known[id] {
			return fmt.Errorf(
				"the drop of %q names no check in this suite.\n  known checks:\n%s",
				id, indentedIDs(s.Checks),
			)
		}
	}
	for _, sub := range subjects {
		for id := range sub.Excused {
			if !known[id] {
				return fmt.Errorf(
					"subject %q is excused from %q, which names no check in this suite.\n  known checks:\n%s",
					sub.Name, id, indentedIDs(s.Checks),
				)
			}
		}
	}
	return nil
}

func indentedIDs[S any](checks []Check[S]) string {
	ids := make([]string, 0, len(checks))
	for _, c := range checks {
		ids = append(ids, "    "+string(c.ID))
	}
	sort.Strings(ids)
	return strings.Join(ids, "\n")
}

// The environment contract a CI system drives the runner through. File
// artifacts and seed policy are configuration of the run, not of any one
// package, which is why these are read here rather than generated per
// package: every consumer's test binary gets the same contract for free.
const (
	// EnvReportDir, when set, makes every run write its versioned JSON
	// report into the directory, named after the package, test and suite.
	//
	// The artifact is written from Cleanup, so a killed run writes
	// nothing: ABSENCE IS THE FAILURE SIGNAL. A consumer of the directory
	// must treat a missing artifact for a scheduled run exactly as it
	// treats RunFailed — no start-marker is written, by contract rather
	// than omission.
	EnvReportDir = "TESTKIT_REPORT_DIR"

	// EnvRapidSeed pins rapid's seed — deterministic presubmit. Leave it
	// unset on scheduled runs to keep exploring; a failure prints the seed
	// to replay it with.
	EnvRapidSeed = "TESTKIT_RAPID_SEED"

	// EnvRapidChecks caps rapid's per-property iteration count — the
	// presubmit time budget, owned by CI config rather than by each
	// package's flags.
	EnvRapidChecks = "TESTKIT_RAPID_CHECKS"
)

var (
	rapidPolicyOnce sync.Once
	errRapidPolicy  error
)

// applyRapidEnv maps the seed and budget environment variables onto
// rapid's flags, once per binary.
//
// A flag passed explicitly on the command line wins over the environment:
// the developer replaying one failure with -rapid.seed must not have CI's
// pinned seed silently override the one they typed. A variable that is set
// while this binary links no rapid is ignored — a CI environment is global
// and most packages have no property checks.
func applyRapidEnv() error {
	explicit := map[string]bool{}
	flag.Visit(func(f *flag.Flag) { explicit[f.Name] = true })

	for env, flagName := range map[string]string{
		EnvRapidSeed:   "rapid.seed",
		EnvRapidChecks: "rapid.checks",
	} {
		v := os.Getenv(env)
		if v == "" || explicit[flagName] {
			continue
		}
		f := flag.Lookup(flagName)
		if f == nil {
			continue
		}
		if err := f.Value.Set(v); err != nil {
			return fmt.Errorf("%s=%q is not a valid -%s value: %w", env, v, flagName, err)
		}
	}
	return nil
}

// rapidBridgeAlive refuses a run whose property checks have lost the
// seed bridge.
//
// [applyRapidEnv] skips a flag it cannot find, on purpose: a CI
// environment is global and most packages link no rapid, so a set
// variable there means nothing. That silence is wrong for exactly one
// case — a run that DOES drive properties. Those packages link the
// engine and rapid with it, so the flags this maps onto have to exist;
// if one does not, rapid has renamed it and CI's pinned seed has been
// doing nothing, with every run still green and no longer reproducible.
//
// Asked here rather than in each generated package. It is a fact about
// the run, one binary links one rapid, and eighty copies of the same
// lookup would be eighty places to keep in step.
func rapidBridgeAlive[S any](s Suite[S]) error {
	return bridgeAlive(s, flag.Lookup)
}

// bridgeAlive is [rapidBridgeAlive] with the lookup handed in.
//
// Injected for one reason: a binary that links rapid always finds the
// flags, so the guard cannot fail where it runs and a test of it would
// assert nothing. The seam is what lets the absent case be driven and
// the message read.
func bridgeAlive[S any](s Suite[S], lookup func(string) *flag.Flag) error {
	if !drivesProperties(s) {
		return nil
	}
	for _, name := range []string{"rapid.seed", "rapid.checks"} {
		if lookup(name) == nil {
			return fmt.Errorf("this run drives property checks, so rapid is linked, and "+
				"flag -%s is absent: the %s bridge is dead and a pinned seed is "+
				"silently doing nothing", name, EnvRapidSeed)
		}
	}
	return nil
}

// drivesProperties reports whether any check in the suite is driven by
// the property engine rather than by a fixed call sequence.
//
// Read off the ID's family, which is the one place that fact is already
// written down: the model tier's legs and the crash schedule both draw
// and shrink, and every other family settles its claim with calls it
// spells itself.
func drivesProperties[S any](s Suite[S]) bool {
	for _, id := range s.IDs() {
		switch family, _, found := strings.Cut(string(id), "/"); {
		case !found:
			continue
		case family == FamilyModel, family == FamilySim:
			return true
		}
	}
	return false
}

// classFamilies is the closed set of class-family prefixes. The leaf stays
// open — classes bucket the report and new buckets are additive — but a
// family typo would mint a phantom bucket in the lock and the histogram,
// which is the same silent-vocabulary drift the unknown-capability arm
// refuses.
var classFamilies = map[string]bool{
	ClassFamilySignature: true,
	ClassFamilyMixin:     true,
	ClassFamilyModel:     true,
	ClassFamilySim:       true,
	ClassFamilyHand:      true,
}

// classFamilyList renders the known families for a failure message,
// derived from the one map so the prose cannot drift from the check.
func classFamilyList() string {
	names := slices.Sorted(maps.Keys(classFamilies))
	return strings.Join(names, ", ")
}

// validateClass holds a class to <family>/<slug> with a known family.
func validateClass(id ID, class Class) error {
	family, leaf, ok := strings.Cut(string(class), "/")
	if !ok || leaf == "" {
		return fmt.Errorf("check %q: class %q is not <family>/<slug>", id, class)
	}
	if !classFamilies[family] {
		return fmt.Errorf(
			"check %q: class family %q is not one this report buckets (%s)",
			id, class, classFamilyList(),
		)
	}
	for _, r := range leaf {
		if !isLower(r) && !isDigit(r) && r != '-' {
			return fmt.Errorf("check %q: class %q's leaf is not a slug — a-z, 0-9 and '-' only", id, class)
		}
	}
	return nil
}
