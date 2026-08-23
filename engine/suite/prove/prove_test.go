// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package prove

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/clock"
	"go.thesmos.sh/testkit/engine/suite"
)

type fake struct{}

func alwaysRed() suite.Check[fake] {
	return suite.Check[fake]{
		ID: "X/always-red", Class: "own/test", Claim: "fails on anything",
		Falsifiable: suite.Proven(),
		Run:         func(tb testing.TB, _ fake) { tb.Errorf("red") },
	}
}

func neverRed() suite.Check[fake] {
	return suite.Check[fake]{
		ID: "X/never-red", Class: "own/test", Claim: "passes on anything",
		Falsifiable: suite.Proven(),
		Run:         func(testing.TB, fake) {},
	}
}

func defect() Defect[fake] {
	return Defect[fake]{Subject: suite.Subject[fake]{
		Name: "any defect", New: func(testing.TB) fake { return fake{} },
	}}
}

// TestCaught pins the harness core: a check that fails against the defect
// is caught, one that stays green is not, and a panicking check counts as
// caught because the real runner reports a panic as a failed leg.
func TestCaught(t *testing.T) {
	t.Parallel()

	if ok, _ := caught(alwaysRed(), defect().Subject); !ok {
		t.Error("a check that fails against the defect must count as caught")
	}
	if ok, _ := caught(neverRed(), defect().Subject); ok {
		t.Error("a check that stays green against the defect must not count as caught")
	}

	panics := suite.Check[fake]{
		ID: "X/panics", Claim: "panics",
		Run: func(testing.TB, fake) { panic("planted") },
	}
	if ok, _ := caught(panics, defect().Subject); !ok {
		t.Error("a panicking check must count as caught, as the runner's panic guard does")
	}

	fatals := suite.Check[fake]{
		ID: "X/fatals", Claim: "fatals and must not run past it",
		Run: func(tb testing.TB, _ fake) {
			tb.Fatalf("red")
			panic("unreachable: Fatalf must not return on the captive TB")
		},
	}
	if ok, _ := caught(fatals, defect().Subject); !ok {
		t.Error("a fatal check must count as caught")
	}
}

// TestParity pins the claim/evidence truth table.
func TestParity(t *testing.T) {
	t.Parallel()

	unproven := neverRed()
	unproven.ID = "X/unproven"
	unproven.Falsifiable = suite.Falsifiability{}

	argued := neverRed()
	argued.ID = "X/argued"
	argued.Falsifiable = suite.Argued("cannot be staged")

	checks := []suite.Check[fake]{alwaysRed(), neverRed(), unproven, argued}

	msgs := parity(checks, Defects[fake]{
		"X/always-red": defect(), // proven with evidence: silent
		// "X/never-red" proven without evidence: reported
		"X/unproven": defect(), // evidence without the claim: reported
		// "X/argued" with neither: silent
	})

	if len(msgs) != 2 {
		t.Fatalf("parity reported %d mismatches, want 2:\n%s", len(msgs), strings.Join(msgs, "\n"))
	}
	if !strings.Contains(msgs[0], "X/never-red") || !strings.Contains(msgs[0], "downgrade the claim") {
		t.Errorf("the unevidenced claim must be reported with its fix, got %q", msgs[0])
	}
	if !strings.Contains(msgs[1], "X/unproven") || !strings.Contains(msgs[1], "claim is owed") {
		t.Errorf("the unclaimed evidence must be reported with its fix, got %q", msgs[1])
	}
}

// TestAllRunsEveryDefect drives the public surface end to end on the happy
// path; the failing paths are the two pure pieces above.
func TestAllRunsEveryDefect(t *testing.T) {
	t.Parallel()
	All(t, []suite.Check[fake]{alwaysRed()}, Defects[fake]{"X/always-red": defect()})
}

// TestRedDemandsTheStatedReason pins the specificity contract: a red that
// does not mention the defect's reason is not evidence.
func TestRedDemandsTheStatedReason(t *testing.T) {
	t.Parallel()

	reasoned := alwaysRed() // fails saying "red"
	d := defect()
	d.Reason = "red"
	Red(t, []suite.Check[fake]{reasoned}, reasoned.ID, d) // right reason: green

	// caught's message must carry what the check said, so the wrong-reason
	// arm has something to compare.
	_, msg := caught(reasoned, d.Subject)
	if !strings.Contains(msg, "red") {
		t.Errorf("the captive failure text must surface to Red, got %q", msg)
	}
}

// TestGreenToleratesTheNearMiss pins the specificity primitive.
func TestGreenToleratesTheNearMiss(t *testing.T) {
	t.Parallel()
	Green(t, []suite.Check[fake]{neverRed()}, defect().Subject)
}

// TestOneAndReasoned pins the proofs-file sugar: One builds the subject
// from its constructor, and Reasoned pins the substring on a copy.
func TestOneAndReasoned(t *testing.T) {
	t.Parallel()

	d := One("a broken fake", func(testing.TB) fake { return fake{} })
	if d.Name != "a broken fake" || d.New == nil {
		t.Errorf("One must carry the name and constructor, got %+v", d)
	}
	r := d.Reasoned("the claim")
	if r.Reason != "the claim" || d.Reason != "" {
		t.Errorf("Reasoned must pin the copy and leave the original, got %q / %q", r.Reason, d.Reason)
	}
}

// TestRedProblem pins the catch's truth table: an uncaught defect and a
// wrong-reason red are each named, a right-reason red is clean, and with
// no stated reason any red counts.
func TestRedProblem(t *testing.T) {
	t.Parallel()

	d := defect()
	if p := redProblem[fake]("X/id", d, false, ""); !strings.Contains(p, "passed it") {
		t.Errorf("an uncaught defect must be reported, got %q", p)
	}
	if p := redProblem[fake]("X/id", d, true, "anything at all"); p != "" {
		t.Errorf("with no stated reason any red is evidence, got %q", p)
	}

	d.Reason = "the claim"
	if p := redProblem[fake]("X/id", d, true, "red about something else"); !strings.Contains(p, "wrong reason") {
		t.Errorf("a red missing the stated reason must be reported, got %q", p)
	}
	if p := redProblem[fake]("X/id", d, true, "broke the claim exactly"); p != "" {
		t.Errorf("a red carrying the stated reason is clean, got %q", p)
	}
}

// TestRedScrubsTheExpectedRedsBucket pins the cleanup contract: an
// expected red removes the reproduction files rapid wrote — and ONLY
// those. A consumer may keep hand-curated corpus files in the same
// bucket, and an expected red is no license to delete what it did not
// write.
//
//nolint:paralleltest // t.Chdir forbids Parallel; the wd change must not race other tests.
func TestRedScrubsTheExpectedRedsBucket(t *testing.T) {
	t.Chdir(t.TempDir())

	dir := filepath.Join("testdata", "rapid", "X_always-red")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatalf("stage the bucket: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "stale.fail"), []byte("#"), 0o600); err != nil {
		t.Fatalf("stage the stale file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "corpus-seed"), []byte("#"), 0o600); err != nil {
		t.Fatalf("stage the consumer file: %v", err)
	}

	Red(t, []suite.Check[fake]{alwaysRed()}, "X/always-red", defect())

	if _, err := os.Stat(filepath.Join(dir, "stale.fail")); !os.IsNotExist(err) {
		t.Errorf("an expected red must remove rapid's reproduction files; stat: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "corpus-seed")); err != nil {
		t.Errorf("a file the scrub did not write must survive it: %v", err)
	}
}

// TestRapidFailFileConvention is the canary under scrubBucket's one
// external assumption: rapid writes its reproduction files under
// testdata/rapid/<TestName>. If rapid ever moves its layout, the scrub
// becomes a silent no-op and the litter returns — this test reds the day
// that happens, instead of leaving the discovery to whoever wonders why
// the tree fills with .fail files. The child process runs a genuinely
// failing property in a temp working directory; the parent asserts the
// file landed where the scrub looks.
func TestRapidFailFileConvention(t *testing.T) {
	if dir := os.Getenv("PROVE_RAPID_CANARY_DIR"); dir != "" {
		t.Chdir(dir)
		rapid.Check(t, func(rt *rapid.T) { rt.Fatalf("canary: expected red") })
		return
	}

	work := t.TempDir()
	//nolint:gosec // G204: the test re-executes its own binary with constant flags.
	cmd := exec.CommandContext(t.Context(), os.Args[0],
		"-test.run", "TestRapidFailFileConvention", "-test.count=1",
	)
	cmd.Env = append(os.Environ(), "PROVE_RAPID_CANARY_DIR="+work)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("the canary property must fail; output:\n%s", out)
	}
	matches, globErr := filepath.Glob(filepath.Join(
		work, "testdata", "rapid", "TestRapidFailFileConvention", "*.fail",
	))
	if globErr != nil || len(matches) == 0 {
		t.Errorf("rapid did not write its reproduction file under testdata/rapid/<TestName>, "+
			"where scrubBucket looks — the scrub is a silent no-op now (glob %v, err %v)\noutput:\n%s",
			matches, globErr, out)
	}
}

// TestGreenSkipsExcusedChecks pins the excuse seam: alwaysRed reds against
// anything, so only the named skip keeps this test green.
func TestGreenSkipsExcusedChecks(t *testing.T) {
	t.Parallel()

	sub := defect().Subject
	sub.Excused = map[suite.ID]bool{"X/always-red": true}
	Green(t, []suite.Check[fake]{alwaysRed()}, sub)
}

// TestAnswering pins the seam that lets a generated defect take a check
// declaring a door: the doors come from the run's harness, and a defect
// keeps whatever it answered for itself.
func TestAnswering(t *testing.T) {
	t.Parallel()

	owned := defect()
	owned.Provides = map[suite.Capability]any{"less": "the defect's own"}
	in := Defects[fake]{"X/plain": defect(), "X/owned": owned}

	out := in.Answering(map[suite.Capability]any{"less": "the harness's", "stats": 1})

	if got := out["X/plain"].Provides["less"]; got != "the harness's" {
		t.Errorf("a defect answering nothing must take the harness's door, got %v", got)
	}
	if got := out["X/owned"].Provides["less"]; got != "the defect's own" {
		t.Errorf("a defect's own answer must survive, got %v", got)
	}
	if got := out["X/owned"].Provides["stats"]; got != 1 {
		t.Errorf("and the doors it did not answer must still arrive, got %v", got)
	}
	if in["X/plain"].Provides != nil {
		t.Error("the original map must be left alone")
	}
	if same := in.Answering(nil); len(same) != len(in) {
		t.Errorf("no doors is the same map, got %d entries", len(same))
	}
}

// TestRedGate pins the proof's capability gate from the other side: a
// defect that cannot take the check is refused naming the arm, because
// the wiring red it would otherwise produce is not evidence.
//
// alwaysRed reds against anything, so only the gate firing first keeps
// the unarmed arm's assertion meaningful.
func TestRedGate(t *testing.T) {
	t.Parallel()

	needy := alwaysRed()
	needy.ID = "X/needs-clock"
	needy.Needs = suite.NeedsClock()

	if msg := redGate(needy, defect().Subject); !strings.Contains(msg, "OnClock") {
		t.Errorf("an unarmed defect must be refused naming the arm, got %q", msg)
	}

	armed := defect()
	armed.OnClock = func(testing.TB, *clock.TestClock) fake { return fake{} }
	if msg := redGate(needy, armed.Subject); msg != "" {
		t.Errorf("an armed defect must pass the gate, got %q", msg)
	}
	Red(t, []suite.Check[fake]{needy}, needy.ID, armed)
}

// TestGreenGate pins the control's capability gate: an unarmed control is
// refused naming the arm — a wiring red would poison the specificity
// measurement — and an armed one runs the check.
func TestGreenGate(t *testing.T) {
	t.Parallel()

	needy := neverRed()
	needy.ID = "X/needs-clock"
	needy.Needs = suite.NeedsClock()

	if msg := greenGate(needy, defect().Subject); !strings.Contains(msg, "OnClock") {
		t.Errorf("an unarmed control must be refused naming the arm, got %q", msg)
	}

	armed := defect().Subject
	armed.OnClock = func(testing.TB, *clock.TestClock) fake { return fake{} }
	if msg := greenGate(needy, armed); msg != "" {
		t.Errorf("an armed control must pass the gate, got %q", msg)
	}
	Green(t, []suite.Check[fake]{needy}, armed)
}

// TestScrubHonorsTheTestdataOverride pins the hermetic-runner seam:
// with TESTKIT_TESTDATA_DIR set, the expected red's scrub reaches the
// named root instead of the working directory.
//
//nolint:paralleltest // t.Setenv and t.Chdir both forbid Parallel.
func TestScrubHonorsTheTestdataOverride(t *testing.T) {
	t.Chdir(t.TempDir())
	root := t.TempDir()
	t.Setenv("TESTKIT_TESTDATA_DIR", root)

	dir := filepath.Join(root, "rapid", "X_always-red")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "stale.fail"), []byte("#"), 0o600); err != nil {
		t.Fatal(err)
	}

	Red(t, []suite.Check[fake]{alwaysRed()}, "X/always-red", defect())

	if _, err := os.Stat(filepath.Join(dir, "stale.fail")); !os.IsNotExist(err) {
		t.Errorf("the scrub must follow the override root; stat: %v", err)
	}
	if entries, _ := os.ReadDir("."); len(entries) != 0 {
		t.Errorf("nothing may be scrubbed or created under the working directory: %v", entries)
	}
}
