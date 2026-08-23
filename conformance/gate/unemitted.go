// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gate

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Shipped is one exported constructor the engine offers a generator, and
// whether the corpus emits a call to it.
type Shipped struct {
	// Pkg is the engine package's local name as generated code spells it —
	// `action`, `ref` — and Name the constructor.
	Pkg, Name string

	// Emitted reports that some generated file in the corpus calls it.
	Emitted bool
}

// Key is the register's key for this constructor.
func (s Shipped) Key() string { return s.Pkg + "." + s.Name }

// ShippedConstructors reads every exported constructor out of the engine
// packages a conformance package reaches, and reports which the corpus
// actually calls.
//
// Source rather than reflection, because these are generic functions: they
// have no runtime value to enumerate, and a hand-kept list is a second home
// for a fact the source already states.
//
// The emission side is a text scan of the corpus, which is exactly what it
// should be here. These calls are spelled one way — the package's local
// name, a dot, the constructor — and a scan that agreed with the emit
// graph but not with the bytes on disk would be measuring the wrong thing:
// what ships is what a consumer compiles.
//
// Where a caller counts as one differs by package, and the difference is
// what each package is for. action and ref exist to be EMITTED, so the
// corpus's generated output is the whole question. suite and legs are
// written for a person as much as for a generator — suite is the
// vocabulary a consumer states checks in, legs the bridge both sides use
// — so anything in the tree calling one is a caller, generated or not.
//
// Except the package's own directory. A function reached only by its own
// test is what this gate exists to find, and a scan that let a test vouch
// for it would report every one of them as alive.
func ShippedConstructors(engineRoot, corpusRoot string) ([]Shipped, error) {
	repoRoot := filepath.Dir(engineRoot)
	pkgs := []struct{ local, dir, callers string }{
		{"action", filepath.Join(engineRoot, "model", "action"), corpusRoot},
		{"ref", filepath.Join(engineRoot, "model", "ref"), corpusRoot},
		{"suite", filepath.Join(engineRoot, "suite"), repoRoot},
		{"legs", filepath.Join(engineRoot, "legs"), repoRoot},
	}

	var out []Shipped
	for _, p := range pkgs {
		names, err := exportedCtors(p.dir)
		if err != nil {
			return nil, err
		}
		called, err := calledNames(p.callers, p.local, p.dir)
		if err != nil {
			return nil, err
		}
		for _, n := range names {
			out = append(out, Shipped{Pkg: p.local, Name: n, Emitted: called[p.local+"."+n]})
		}
	}
	slices.SortFunc(out, func(a, b Shipped) int { return strings.Compare(a.Key(), b.Key()) })
	return out, nil
}

// exportedCtors is every exported function in a package that returns
// something — the constructors a generator can call.
//
// Test files are skipped, and so is anything returning nothing: a helper
// that only asserts is not a constructor, and holding one to this register
// would be asking why the corpus does not call an assertion.
//
// One file at a time rather than through go/parser's directory form, which
// is deprecated for ignoring build tags. Ignoring them is what this wants:
// a constructor behind a tag is still shipped, and a census that skipped it
// would let one hide.
func exportedCtors(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("gate: read %s for constructors: %w", dir, err)
	}

	fset := token.NewFileSet()
	var out []string
	for _, entry := range entries {
		name := entry.Name()
		switch {
		case entry.IsDir(), !strings.HasSuffix(name, ".go"):
			continue
		case strings.HasSuffix(name, "_test.go"):
			continue
		}
		path := filepath.Join(dir, name)
		file, parseErr := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if parseErr != nil {
			return nil, fmt.Errorf("gate: parse %s for constructors: %w", path, parseErr)
		}
		for _, decl := range file.Decls {
			fn, isFunc := decl.(*ast.FuncDecl)
			switch {
			case !isFunc, fn.Recv != nil, !fn.Name.IsExported():
				continue
			case fn.Type.Results == nil, len(fn.Type.Results.List) == 0:
				continue
			}
			out = append(out, fn.Name.Name)
		}
	}
	slices.Sort(out)
	return slices.Compact(out), nil
}

// calledNames is every `<local>.<Name>` REFERRED TO under root, skipping
// the package's own directory so it cannot vouch for itself.
//
// Parsed, not grepped, and this file is why. A text scan counts the
// selector wherever it appears — including a comment arguing that nothing
// calls it — so the register vouched for the very symbol it registered
// the moment its prose named one. Selectors off the syntax tree cannot do
// that.
//
// Any reference, not only a call: a generic instantiation, a type alias,
// a value handed somewhere else. All of them are somebody depending on
// the name, which is what this asks.
func calledNames(root, local, own string) (map[string]bool, error) {
	ownAbs, err := filepath.Abs(own)
	if err != nil {
		return nil, fmt.Errorf("gate: resolve %s: %w", own, err)
	}
	fset := token.NewFileSet()
	out := map[string]bool{}
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if notACaller[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		// The package's own directory and no further. Its SUBpackages are
		// callers like any other — suite/prove states rows in the same
		// vocabulary a consumer does — and skipping the subtree reported
		// seventeen live helpers as dead.
		if abs, absErr := filepath.Abs(filepath.Dir(path)); absErr == nil && abs == ownAbs {
			return nil
		}
		file, parseErr := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if parseErr != nil {
			return fmt.Errorf("gate: parse %s for %s callers: %w", path, local, parseErr)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			sel, isSel := n.(*ast.SelectorExpr)
			if !isSel {
				return true
			}
			if pkg, isIdent := sel.X.(*ast.Ident); isIdent && pkg.Name == local {
				out[local+"."+sel.Sel.Name] = true
			}
			return true
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("gate: walk %s for %s callers: %w", root, local, err)
	}
	return out, nil
}

// notACaller are directories whose contents do not count as calling
// anything.
//
// gen holds example output from whichever generator run last touched it,
// and it is not regenerated by the drift check — so a helper the current
// generator stopped emitting is still spelled there, and a scan counting
// it would report the helper as alive on the strength of an artefact.
// That is exactly the loophole this gate exists to close, and it closed
// on suite.LowerInductions the first time this walk ran.
//
//nolint:gochecknoglobals // a lookup table, read-only after init.
var notACaller = map[string]bool{".git": true, "gen": true, "docs": true}

// The verdicts a whole family shares, spelled once. A family whose rows
// differ only in which constructor they name is one decision, and repeating
// its prose nine times invites nine half-edits of it.
const (
	reasonMethodExpr = "the method-expression form, for a consumer writing actions by hand; " +
		"the generator emits closures because it reorders and drops arguments a " +
		"method expression cannot"

	reasonQueueOracle = "OPEN: a queue-shaped delivery oracle, superseded for the generator by " +
		"FanOut, which is written at the shapes a Go publisher declares; whether " +
		"these three should stay is unsettled"
)

// UnemittedConstructors registers every engine constructor no generated
// file calls, with the verdict that keeps its absence honest.
//
// Four kinds of row, and the difference matters to a reader deciding
// whether to pick one up:
//
//   - the method-expression forms, which exist for a hand-written consumer
//     and are used by one; the generator emits closures because it has to
//     handle shapes a method expression cannot express
//   - an oracle below the derivation bar [ContractStore] sets out: the
//     family needs something no stamp supplies
//   - a role driven through some other constructor, where the reason is in
//     [tiers.ContractActionFor] or the shape table
//   - OPEN: an absence nobody has settled. Those say so. A row here is a
//     lead, not a verdict, and closing one means either emitting the
//     constructor or replacing the row with a reason.
//
//nolint:gochecknoglobals // a debt register, read-only, test-facing.
var UnemittedConstructors = map[string]string{
	// The method-expression family. Each takes `Iface.Method` where the
	// generator's form takes a closure, and each has a hand-written user in
	// gen/example — grep one to see it. The generator cannot use them: its
	// closures reorder arguments, drop a context the method does not take,
	// and anchor a composite write on the fixture key, none of which a
	// method expression can say.
	"action.AggregatorOf":      reasonMethodExpr,
	"action.CompositeWriterOf": reasonMethodExpr,
	"action.DeleterOf":         reasonMethodExpr,
	"action.EvictingReaderOf":  reasonMethodExpr,
	"action.LifecycleOf":       reasonMethodExpr,
	"action.PoolOf":            reasonMethodExpr,
	"action.ReaderOf":          reasonMethodExpr,
	"action.StreamOf":          reasonMethodExpr,
	"action.WriterOf":          reasonMethodExpr,

	// Driven through another constructor, with the reason already written
	// where the routing is decided.
	"action.AcquireLease": "the lease's acquire draws its key, so it is driven as a keyed writer rather than through this — see tiers.ContractActionFor, which argues the absence",
	"action.Cursor":       "this drains to exhaustion and closes, which reddens the second invocation on a correct cursor-shaped subject — see tiers.ContractActionFor",
	"action.Persister":    "this compares save-answered IDs, and the persister fixture's writer answers none — see tiers.ContractActionFor",
	"action.ChainAppend":  "the chain's append is driven by the recording form beside it, ChainAppendRecording, because the history a chain law reads is filled by the action rather than beside it",
	"action.Deleter":      "a delete is driven as a writer whose deleteremoves stamp assigns it the oracle's Delete — one path rather than two, see tiers.KeyedStoreMixinOp",
	"action.Subscriber":   "a subscription is driven by the delivery set now, which opens handles on both sides and keeps them; this opens one and compares the open path alone",
	"action.Stress":       "the concurrent leg supplies its own stress actions through model.ConcurrentConfig rather than through the sequential action list",

	// The clock is a law field rather than an action: a clocked law advances
	// the run's clock itself, through the Advance closure its binding fills,
	// so an action that advanced it beside them would move time under a law
	// mid-check.
	"action.AdvanceClock":       "advancing is a law field here, filled by the clocked binding's own Advance closure; an action moving time beside it would advance the clock under a law mid-check",
	"action.AdvanceClockSynced": "advancing is a law field here, filled by the clocked binding's own Advance closure; an action moving time beside it would advance the clock under a law mid-check",

	// Below the bar ContractStore sets out: derivable whole from stamps, or
	// not at all. Each names what it would need.
	"ref.NewBalancedPool":          "a pool oracle needs the resource constructor to hand out, which no stamp supplies — see ContractStore, whose docblock sets the bar",
	"ref.NewCompensatingSaga":      "a saga oracle needs its step list, which no stamp supplies — see ContractStore, whose docblock sets the bar",
	"ref.NewCoalescer":             "a single-flight oracle needs the function it coalesces, which no stamp supplies — see ContractStore, whose docblock sets the bar",
	"ref.NewAtomicCell":            "the general cell takes its version beside the value; the generated adapter speaks a value carrying its own version, which is what VersionedCell is for and what the cas row derives",
	"ref.NewSetCollection":         "the deduplicating collection is chosen by CollectionDedupes, and no corpus fixture pairs a value writer with a collector under noduplicates — the mixin sits on drains that already dedupe",
	"ref.NewPartitionedAppendOnly": "the per-partition chain, for a Replay keyed by partition; the corpus's chain replays whole, and a partitioned fixture would be the thing that reaches this",

	// OPEN. Investigated far enough to say the shape exists in the corpus
	// and the constructor is not reached, and no further. Each is a lead.
	"action.Appender":        "OPEN: the appender fixture's Run answers an offset and is classified as a keyed reader, so the offsets are driven but not through this; whether the classifier or this constructor is wrong is unsettled",
	"action.Watcher":         "OPEN: the watcher fixture's watch answers a channel and is declined as a live handle, where this exercises the open path exactly as Subscriber did before the delivery set replaced it",
	"action.Paginator":       "OPEN: the paginated-reader fixture drives Page as a multireader, so the walk is exercised and the cursor threading this constructor owns is not",
	"action.Saga":            "OPEN: the saga fixture drives its steps as writers, so the compensation ordering this constructor owns is checked by the law alone",
	"action.StreamConsumer":  "OPEN: the streamconsumer fixture's Next is classified as a multiaggregator, so the drain is driven and the consumer shape this serves is not presented anywhere",
	"action.ChainReplay":     "OPEN: the chain fixture's replay is driven through the contract role table rather than this, and whether the two should be one path is unsettled",
	"action.ChainVerify":     "OPEN: the chain fixture's verify is driven through the contract role table rather than this, and whether the two should be one path is unsettled",
	"action.GetOrCompute":    "OPEN: the singleflight fixture rides the twin floor for want of a coalescer oracle, and this constructor is what would drive it once one exists",
	"action.Publisher":       "OPEN: superseded in practice by the delivery set, which drives publish itself; whether this should be deleted or kept for a consumer is unsettled",
	"action.TransactionFunc": "OPEN: the tx fixture drives Begin as a composite that consumes commit and rollback, so the callable-taking form this serves is not presented",
	"action.PredicateVar":    "OPEN: the var form beside Predicate, which is emitted; no corpus fixture presents a predicate over a package-level var",
	"action.Unknown":         "OPEN: the fallback for a shape nothing classified, which the generator never reaches because it declines an unclassified method earlier and says so in the header",

	"ref.NewBootOnlyRegistry":  "OPEN: a register-once map, for an interface whose write refuses a second attempt; the corpus's if-absent fixture is that shape and rides the twin floor",
	"ref.NewBoundedCursor":     "OPEN: the cursor oracle, unreachable while tiers.ContractActionFor declines the cursor action it would be compared through",
	"ref.NewCursorTable":       "OPEN: the pagination oracle, unreachable while the paginated walk is driven as a multireader",
	"ref.NewFoldMachine":       "OPEN: a fold over an event stream; no corpus fixture declares one, and which stamp would select it is unsettled",
	"ref.NewGuardedStates":     "OPEN: the workflow oracle; the workflow fixture stamps its transitions and the law reads them, but no contract row selects this store",
	"ref.NewMonotonicLog":      "OPEN: the appender oracle, unreachable for the same reason action.Appender is",
	"ref.NewPureScheduler":     "OPEN: a topological scheduler; no corpus fixture declares one, and which stamp would select it is unsettled",
	"ref.NewRollingCounter":    "OPEN: the windowed oracle; the windowed fixture binds its law and rides the twin floor, and no contract row selects this store",
	"ref.NewSnapshotIsolation": "OPEN: the transaction oracle; the isolation claims defeat store modelling per tiers.DefeatsOracles, but this store models exactly them and the two have never been reconciled",
	// engine/suite. Every one of these has references, and every one of
	// them is in gen/ — output from a generator run older than the
	// current templates. That is what makes them a group: the suite tier
	// used to emit them and now emits something else, and nothing chose
	// to retire them when it changed.
	"suite.LowerInductions": "superseded: the generated harness lowers its own Induce map through legs.AsBuilt, which is the sanctioned downcast; this took the trigger with the sentinel beside it and the emitted form does not",
	"suite.LowerRecover":    "superseded: the generated harness lowers Recover through legs.AsBuilt beside the induction map, for the one reason both lowerings exist — the harness speaks the constructor's type and the subject speaks the interface",
	"suite.RowID":           "superseded by suite.Row and suite.BindRow, which take the method and the name as fields and seal the identity themselves rather than asking a consumer to compose one",
	"suite.HandRowID":       "superseded by suite.Row and suite.BindRow, for the rows carrying no method — the same seal, with the method left empty",
	"suite.Falsify":         "superseded by suite.Row: the row carries Proven and Argued as fields, and the binding settles the falsifiability from them rather than from a call a consumer makes",
	"suite.OneBody":         "superseded by suite.Row: exactly one body is a shape the struct can state, and a row with two of them no longer type-checks",

	"suite.Needs":        "OPEN: the generated rows spell the capability as a suite.Caps literal — `suite.Caps{suite.CapRecover: nil}` — where this and its three shorthands exist to keep one spelling; emitting the constructor would close it",
	"suite.NeedsInduce":  "OPEN: the shorthand for an induced sentinel, unreached for the reason suite.Needs is — the generator writes the map literal instead",
	"suite.NeedsRecover": "OPEN: the shorthand for the recovery door, unreached for the reason suite.Needs is; the generator writes the map literal instead",

	"suite.ClassConst":  "OPEN: the Go spelling of a class constant, for a generator rendering one — this generator renders classes through its own vocab package, and which of the two owns the spelling is unsettled",
	"suite.RedConst":    "OPEN: the Go spelling of a red segment, unreached for the reason suite.ClassConst is",
	"suite.FamilyNames": "OPEN: the family vocabulary as a sorted list, for a consumer enumerating them; nothing in the corpus enumerates families, and whether anything should is unsettled",
	"suite.IsFamily":    "OPEN: the family membership test, unreached for the reason suite.FamilyNames is",
	"suite.DiffLock":    "OPEN: the lock-file diff a stale checks.lock reports; the corpus's Invariants test renders its own diff, and the two have never been reconciled",
	"suite.RenderLock":  "OPEN: the lock-file renderer beside suite.DiffLock, unreached for the same reason and settled by the same decision",

	"ref.NewAtLeastOnce": reasonQueueOracle,
	"ref.NewAtMostOnce":  reasonQueueOracle,
	"ref.NewExactlyOnce": reasonQueueOracle,
}
