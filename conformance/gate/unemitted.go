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
// packages a generator emits calls to, and reports which the corpus
// actually reaches.
//
// Source rather than reflection, because these are generic functions: they
// have no runtime value to enumerate, and a hand-kept list is a second home
// for a fact the source already states.
//
// The emission side is a text scan of the generated files, which is exactly
// what it should be here. Generated code spells these calls one way — the
// package's local name, a dot, the constructor — and a scan that agreed
// with the emit graph but not with the bytes on disk would be measuring the
// wrong thing: what ships is what a consumer compiles.
func ShippedConstructors(engineRoot, corpusRoot string) ([]Shipped, error) {
	pkgs := map[string]string{
		"action": filepath.Join(engineRoot, "model", "action"),
		"ref":    filepath.Join(engineRoot, "model", "ref"),
	}

	var out []Shipped
	for local, dir := range pkgs {
		names, err := exportedCtors(dir)
		if err != nil {
			return nil, err
		}
		for _, n := range names {
			out = append(out, Shipped{Pkg: local, Name: n})
		}
	}

	emitted, err := emittedNames(corpusRoot)
	if err != nil {
		return nil, err
	}
	for i := range out {
		out[i].Emitted = emitted[out[i].Key()]
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

// emittedNames is every `<pkg>.<Name>` a generated file in the corpus
// spells, for the packages this register covers.
//
// Matched on the identifier's boundary rather than on a following paren: a
// generic call carries its type arguments first — `action.NewDelivery[T,
// M](` — and a scan looking for the paren would report a constructor as
// unemitted on the one axis where the generator had to instantiate it.
func emittedNames(corpusRoot string) (map[string]bool, error) {
	out := map[string]bool{}
	err := filepath.WalkDir(corpusRoot, func(path string, d fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return err
		case d.IsDir(), !strings.HasSuffix(path, ".gen.go"):
			return nil
		}
		body, readErr := readFile(path)
		if readErr != nil {
			return readErr
		}
		for _, local := range []string{"action", "ref"} {
			scanQualified(body, local, out)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("gate: walk %s for emitted constructors: %w", corpusRoot, err)
	}
	return out, nil
}

// scanQualified records every `<local>.<Name>` in body, where Name starts
// with an upper-case letter.
func scanQualified(body, local string, into map[string]bool) {
	prefix := local + "."
	for rest := body; ; {
		i := strings.Index(rest, prefix)
		if i < 0 {
			return
		}
		// A preceding identifier rune means this is some other selector
		// ending in the same letters — `myaction.Foo` is not `action.Foo`.
		if i > 0 && isIdentRune(rest[i-1]) {
			rest = rest[i+len(prefix):]
			continue
		}
		rest = rest[i+len(prefix):]
		end := 0
		for end < len(rest) && isIdentRune(rest[end]) {
			end++
		}
		if end > 0 && rest[0] >= 'A' && rest[0] <= 'Z' {
			into[prefix+rest[:end]] = true
		}
	}
}

// readFile is os.ReadFile as a string, named so the walk above reads as a
// sequence of questions rather than a sequence of conversions.
func readFile(path string) (string, error) {
	b, err := os.ReadFile(path) //nolint:gosec // paths come from the walk
	if err != nil {
		return "", fmt.Errorf("gate: read %s: %w", path, err)
	}
	return string(b), nil
}

// isIdentRune reports the bytes a Go identifier is made of, which is all
// this scan needs: the names it matches are ASCII by convention.
func isIdentRune(c byte) bool {
	return c == '_' ||
		(c >= '0' && c <= '9') ||
		(c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z')
}

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
	"ref.NewAtLeastOnce":       reasonQueueOracle,
	"ref.NewAtMostOnce":        reasonQueueOracle,
	"ref.NewExactlyOnce":       reasonQueueOracle,
}
