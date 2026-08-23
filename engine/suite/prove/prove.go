// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package prove is the falsifiability harness: it drives a check against a
// planted defect and fails unless the check goes red.
//
// "Proven able to fail" in the report is a claim, and a claim in prose rots
// the first time someone weakens an assertion without noticing. This package
// is how the claim stays a fact — for the generated checks, whose proofs the
// generator emits as a [Defects] map beside the suite, and for hand-written
// checks, whose proofs their author writes the same way. One harness, both
// tiers; a consumer proving their own check uses exactly the machinery the
// generated proofs use.
//
// A planted defect is the smallest implementation that breaks exactly the
// claim under proof — typically the generated stub delegating to a correct
// implementation everywhere except the one method the defect overrides.
//
// It is a separate package rather than part of suite because suite is the
// vocabulary a consumer's non-test code composes with, and this harness
// needs the captive TB from the testkit root. Consumers import it from
// test files; a generated package's run surface may link it so that
// consumers get ProveX without importing this package themselves.
//
// One operational note: a proof's red is EXPECTED, and a property that
// fails writes a rapid reproduction file. The captive TB is named after
// the check so those files land in per-check buckets — sharing one bucket
// let a stale file from one proof replay into another, where its draws no
// longer fit and the run failed reporting "fail file is no longer valid"
// instead of anything about the claim. A proof that reds as expected then
// removes its own bucket: the file is not a finding, and leaving it would
// stop "anything under testdata/rapid after a run" from meaning one. A
// red for the wrong reason keeps its files — that one needs debugging.
package prove

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/engine/suite"
)

// Defect is one planted defect. The embedded subject carries whatever the
// check needs — OnClock for a clocked check, Induces for a poison check —
// exactly as a real subject would.
//
// Reason, when set, is a substring the captive failure must contain. Any
// red used to count: a defect that died on an incidental guard — an unmet
// capability, a stray panic — kept its Proven stamp while proving nothing
// about the claim. A proof is evidence only when the check rejected the
// defect FOR THE DEFECT.
type Defect[S any] struct {
	suite.Subject[S]

	// Reason, when set, is a substring the red's failure message must
	// contain: where the package owns the failure prose, the proof
	// asserts it, so a red from an incidental guard stops counting as
	// evidence.
	Reason string
}

// Defects maps a check ID to the planted defect that proves the check can
// fail.
type Defects[S any] map[suite.ID]Defect[S]

// Answering returns a copy in which every defect supplies the given
// capability doors, leaving whatever a defect already answered for itself.
//
// A defect stands in for a real subject, and the doors in the open half of
// the registry are facts about the DECLARATION rather than about any one
// instance — see [suite.Doors]. So a planted defect has no separate answer
// to give, and without this it has none at all: it is built by a generated
// proofs map that never met the harness, so a check declaring a door could
// only ever red on wiring. [Red] refuses that red; this is what stops it
// arising.
func (d Defects[S]) Answering(doors map[suite.Capability]any) Defects[S] {
	if len(doors) == 0 {
		return d
	}
	out := make(Defects[S], len(d))
	for id, defect := range d {
		provides := make(map[suite.Capability]any, len(doors)+len(defect.Provides))
		maps.Copy(provides, doors)
		maps.Copy(provides, defect.Provides)
		defect.Provides = provides
		out[id] = defect
	}
	return out
}

// One names a planted defect by its constructor — the sugar every
// generated proofs file writes its map with. The tb reaches the
// constructor because a defect over a real medium registers cleanup.
func One[S any](name string, build func(tb testing.TB) S) Defect[S] {
	return Defect[S]{Subject: suite.Subject[S]{Name: name, New: build}}
}

// Reasoned pins the substring this defect's red must contain, so a red
// from an incidental guard stops counting as evidence.
func (d Defect[S]) Reasoned(reason string) Defect[S] {
	d.Reason = reason
	return d
}

// Red drives one check against one planted defect and fails when the check
// stays green.
//
// The check runs on a captive TB with Goexit semantics, in its own
// goroutine, because that is the contract checks are written against:
// Fatalf must not return, and rapid relies on exactly that. A panic that
// escapes the check counts as caught, which is what the real runner's panic
// guard does — a defect that panics a check is a defect the run reports.
func Red[S any](t *testing.T, checks []suite.Check[S], id suite.ID, defect Defect[S]) {
	t.Helper()

	i := slices.IndexFunc(checks, func(c suite.Check[S]) bool { return c.ID == id })
	if i < 0 {
		t.Fatalf("this proof names %q, which the check set does not hold; "+
			"a proof for a deleted check is dead weight — delete it too", id)
	}
	if msg := redGate(checks[i], defect.Subject); msg != "" {
		t.Error(msg)
		return
	}
	failed, msg := caught(checks[i], defect.Subject)
	if problem := redProblem(id, defect, failed, msg); problem != "" {
		t.Error(problem)
		return
	}
	scrubBucket(t, id)
}

// redGate returns the refusal for a defect that cannot take the check,
// empty when it can.
//
// The counterpart of [greenGate], and needed for the same reason from the
// other side: an unarmed defect reds on wiring, and a wiring red is not
// evidence that the check can catch anything. Left ungated, a check
// needing a capability earns its Proven stamp from a subject that never
// reached the claim — which is the exact failure the Reason substring
// exists to catch, in the one case no reason is written for.
//
// It rides [suite.CanRun], the runner's own gate, so the proof judges a
// defect exactly as the run judges a subject.
func redGate[S any](c suite.Check[S], sub suite.Subject[S]) string {
	ok, why := suite.CanRun(c, sub)
	if ok {
		return ""
	}
	return fmt.Sprintf("the defect %q cannot take %s:\n%s\n"+
		"  arm the defect's subject — it stands in for a real one, and a\n"+
		"  defect that reds on wiring proves nothing about the claim", sub.Name, c.ID, why)
}

// redProblem judges the catch: empty when the check rejected the defect
// FOR THE DEFECT, else what to report. Pure, so its truth table is
// testable without staging a captive run.
func redProblem[S any](id suite.ID, defect Defect[S], failed bool, msg string) string {
	switch {
	case !failed:
		return fmt.Sprintf("%s claims to be provable, but %q passed it.\n"+
			"  either the check has been weakened until this defect slips through,\n"+
			"  or the defect no longer breaks the claim; fix whichever is true", id, defect.Name)
	case defect.Reason != "" && !strings.Contains(msg, defect.Reason):
		return fmt.Sprintf("%s went red against %q, but for the wrong reason.\n"+
			"  the failure must mention %q and said:\n  %s\n"+
			"  a red from an incidental guard proves nothing about the claim", id, defect.Name, defect.Reason, msg)
	default:
		return ""
	}
}

// scrubBucket removes the reproduction files an EXPECTED red left behind.
// A genuine failure's file is rapid's replay pin and stays; an expected
// red proved something worked, and its file would be noise plus a
// stale-replay hazard on the next run. After a clean pass, anything left
// under testdata/rapid IS a finding — absence is the signal, the same
// contract the report artifact keeps.
func scrubBucket(t *testing.T, id suite.ID) {
	t.Helper()
	root := filepath.Join("testdata", "rapid")
	// Hermetic runners (Bazel, Buck) place the package's files away
	// from the test's working directory; the override names where.
	if env := os.Getenv("TESTKIT_TESTDATA_DIR"); env != "" {
		root = filepath.Join(env, "rapid")
	}
	dir := filepath.Join(root, bucket(id))
	// Only rapid's own reproduction files are scrubbed — a consumer may
	// keep hand-curated corpus files under the same bucket, and an
	// expected red is no license to delete what it did not write.
	matches, err := filepath.Glob(filepath.Join(dir, "*.fail"))
	if err != nil {
		t.Logf("could not scan %s for the expected red's reproduction files: %v", dir, err)
		return
	}
	for _, m := range matches {
		if err := os.Remove(m); err != nil { //nolint:gosec // paths come from the glob above
			t.Logf("could not remove %s, the expected red's reproduction file: %v", m, err)
		}
	}
}

// All runs every defect in its own subtest, named by check ID so
// `-run 'TestStoreProofs/Put'` reruns one method's proofs, and enforces
// parity between the claims and the evidence:
//
//   - a check claiming Proven with no defect here fails: plant one, or
//     downgrade the claim to Argued
//   - a defect for a check that does not claim Proven fails: the evidence
//     exists, so the claim is owed — add Proven to the check
//   - a defect naming no check at all fails inside its own subtest
func All[S any](t *testing.T, checks []suite.Check[S], defects Defects[S]) {
	t.Helper()
	for _, msg := range parity(checks, defects) {
		t.Error(msg)
	}
	for id, defect := range defects {
		t.Run(string(id), func(t *testing.T) {
			t.Parallel()
			Red(t, checks, id, defect)
		})
	}
}

// Green asserts every check tolerates the subject — the negative
// control's primitive. Red measures sensitivity: planted defects must be
// caught. Green measures specificity: a NEAR-miss the claims do not
// forbid must pass, or the suite is red on correct code and nobody has
// measured which. A corpus without a negative control only ever proves
// its checks can fire, never that they fire selectively.
//
// The control passes the runner's own capability gate first: a control
// that cannot take a check reds on wiring, and a wiring red recorded as
// "the suite rejected correct code" poisons the measurement. Arm the
// control's harness, or mark the check in the subject's Excused map — an
// excused check is skipped BY NAME, because it contributes no specificity
// evidence either way.
func Green[S any](t *testing.T, checks []suite.Check[S], subject suite.Subject[S]) {
	t.Helper()
	for _, c := range checks {
		t.Run(string(c.ID), func(t *testing.T) {
			t.Parallel()
			if subject.Excused[c.ID] {
				t.Skipf("excused: %s — no specificity evidence collectable from this control", c.ID)
			}
			if msg := greenGate(c, subject); msg != "" {
				t.Fatal(msg)
			}
			if failed, msg := caught(c, subject); failed {
				t.Errorf("%s rejected %q, which the claims do not forbid:\n  %s",
					c.ID, subject.Name, msg)
			}
		})
	}
}

// greenGate returns the refusal for a control that cannot take the check,
// empty when it can. It rides [suite.CanRun] — the runner's own gate — so
// the two verdicts cannot drift, and it is pure so the refusal is
// testable without a failing subtest.
func greenGate[S any](c suite.Check[S], sub suite.Subject[S]) string {
	ok, why := suite.CanRun(c, sub)
	if ok {
		return ""
	}
	return fmt.Sprintf("the control cannot take %s:\n%s\n"+
		"  arm the control's harness or excuse the check — an unarmed control\n"+
		"  reds on wiring, and a wiring red poisons the specificity measurement", c.ID, why)
}

// caught reports whether the check went red against the defect.
//
// It deliberately does not delegate to [testkit.Rejects], which drives the
// same captive-TB shape: Rejects lets a panic cross the goroutine and kill
// the process, because in ITS domain a panicking check is a defect in the
// check. Here the real runner records a panicking check as a failed leg —
// a defect that panics a check is a defect the run catches — and the proof
// harness must judge checks exactly as the runner does, or a proof could
// disagree with the run it vouches for. It also runs the captive TB's
// cleanups, which Rejects does not own.
func caught[S any](c suite.Check[S], defect suite.Subject[S]) (bool, string) {
	f := testkit.NewFailableTB().WithGoexit().WithName(bucket(c.ID))
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer func() {
			if r := recover(); r != nil {
				f.Errorf("check %s panicked: %v", c.ID, r)
			}
		}()
		if c.RunWith != nil {
			c.RunWith(f, defect)
		} else {
			c.Run(f, defect.New(f))
		}
	}()
	<-done
	f.RunCleanups()
	return f.Failed(), strings.Join(append(f.Logs(), f.Msg()), "\n")
}

// bucket renders a check ID as a rapid reproduction-file bucket: one per
// check, so an expected red's replay file cannot be replayed into a
// different proof. Separators become underscores because the name is a
// path segment.
func bucket(id suite.ID) string {
	return strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r == ' ' {
			return '_'
		}
		return r
	}, string(id))
}

// parity returns one message per claim/evidence mismatch, in check order so
// the output is stable. Pure, so its truth table is testable without
// staging real failures.
func parity[S any](checks []suite.Check[S], defects Defects[S]) []string {
	var msgs []string
	for _, c := range checks {
		_, hasDefect := defects[c.ID]
		proven := c.Falsifiable.State == suite.FalsifiableProven
		switch {
		case proven && !hasDefect:
			msgs = append(msgs, fmt.Sprintf(
				"%s claims Proven and no defect is planted for it; "+
					"plant one, or downgrade the claim to Argued", c.ID,
			))
		case hasDefect && !proven:
			msgs = append(msgs, fmt.Sprintf(
				"%s has a planted defect but does not claim Proven; "+
					"the evidence exists, so the claim is owed — add Proven to the check", c.ID,
			))
		}
	}
	return msgs
}
