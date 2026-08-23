// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gate

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io/fs"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// The rule identifiers a [Vacuity] reports under.
//
// Named rather than numbered because a red row is read by somebody deciding
// whether the assertion is wrong or the rule is, and "rule 3" tells them
// nothing about which question they are answering.
const (
	// RuleSelfComparison is an assertion whose two sides are the same
	// expression — `Equal(t, 0, 0)`, `NotEqual(t, got, got)`.
	RuleSelfComparison = "self-comparison"

	// RuleZeroExpectation is an assertion against a `var` that was declared
	// and never assigned, so the comparison is against the type's zero.
	RuleZeroExpectation = "zero-expectation"

	// RuleConfiguredNeedle is a substring assertion whose needle the same
	// function handed to the subject through a `With…` option.
	RuleConfiguredNeedle = "configured-needle"

	// RuleUncheckedRejection is a [testkit.Rejects] whose answer nothing
	// constrains, so the guard proves that something failed.
	RuleUncheckedRejection = "unchecked-rejection"

	// RuleUnfailableCheck is a law whose every return is a pass or a refused
	// precondition, so no input reaches a verdict.
	RuleUnfailableCheck = "unfailable-check"
)

// Vacuity is one assertion that cannot fail for the reason it names.
//
// The class this whole programme has been removing one instance at a time: a
// check that runs, reports success, and would report success against an
// implementation that does nothing. Every earlier fix was found by reading.
// This finds them by walking, which is the difference between a class that was
// cleared once and a class that stays cleared.
type Vacuity struct {
	// Rule is which of the five it is, File and Line where.
	Rule string
	File string
	Line int

	// Detail is the expression or identifier that made it vacuous, so a red
	// row can be answered without opening the file first.
	Detail string
}

// String renders one finding for a failing gate.
func (v Vacuity) String() string {
	return fmt.Sprintf("%s:%d [%s] %s", v.File, v.Line, v.Rule, v.Detail)
}

// comparisons are the assertion helpers whose two value arguments are being
// held against each other.
//
// Matched on the callee's own name rather than on the package, because the
// generated corpus reaches them through `testkit.` and the engine's own tests
// through a dot-import — and a rule that saw only one of those would report a
// clean half of the tree.
//
//nolint:gochecknoglobals // a lookup table, read-only after init.
var comparisons = map[string]bool{
	"Equal": true, "NotEqual": true, "Same": true, "NotSame": true,
	"ElementsMatch": true, "Contains": true,
}

// Vacuities walks every Go file under root and reports what cannot fail.
//
// Source rather than the emit graph, deliberately. The graph says what the
// generator meant to write; these rules are about what the written assertion
// can do, and a template that renders `Equal(t, x, x)` from two different
// projection fields looks correct at every level above the output.
//
// Paths come back relative to root, which is what makes a register row keyable:
// an absolute path differs per machine and a path relative to the caller's
// working directory differs per module the test runs from.
func Vacuities(root string) ([]Vacuity, error) {
	var out []Vacuity
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return err
		case d.IsDir(), !strings.HasSuffix(path, ".go"):
			return nil
		}
		// Trimmed rather than [filepath.Rel]-ed: WalkDir only ever hands back
		// paths under the root it was given, so the relative form is the
		// suffix and Rel's error arm is one nothing can reach — which is a
		// thing this file is in a poor position to leave lying around.
		rel := strings.TrimPrefix(path, root+string(filepath.Separator))
		found, parseErr := vacuitiesIn(path, filepath.ToSlash(rel))
		if parseErr != nil {
			return parseErr
		}
		out = append(out, found...)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("gate: walk %s for vacuities: %w", root, err)
	}
	slices.SortFunc(out, func(a, b Vacuity) int {
		if a.File != b.File {
			return strings.Compare(a.File, b.File)
		}
		return a.Line - b.Line
	})
	return out, nil
}

// vacuitiesIn runs every rule over one file.
func vacuitiesIn(path, rel string) ([]Vacuity, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
	if err != nil {
		return nil, fmt.Errorf("gate: parse %s: %w", path, err)
	}

	var out []Vacuity
	for _, decl := range file.Decls {
		fn, isFunc := decl.(*ast.FuncDecl)
		if !isFunc || fn.Body == nil {
			continue
		}
		s := &scan{fset: fset, path: rel, fn: fn}
		s.collect()
		out = append(out, s.selfComparisons()...)
		out = append(out, s.zeroExpectations()...)
		out = append(out, s.configuredNeedles()...)
		out = append(out, s.uncheckedRejections()...)
		out = append(out, s.unfailableCheck()...)
	}
	return out, nil
}

// scan holds one function body and what a pass over it collected.
type scan struct {
	fset *token.FileSet
	path string
	fn   *ast.FuncDecl

	// configured are the string literals this body handed to a `With…` call,
	// which is how a test tells a stand-in what to answer.
	configured []string

	// assigned names every identifier the body writes to after declaring it,
	// including through a pointer — the difference between a `var` that holds
	// the zero and one that holds a result.
	assigned map[string]bool
}

// collect makes the one pass the rules read from.
func (s *scan) collect() {
	s.assigned = map[string]bool{}
	ast.Inspect(s.fn.Body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.CallExpr:
			if name := calleeName(node.Fun); strings.HasPrefix(name, "With") && len(name) > len("With") {
				for _, a := range node.Args {
					if lit, ok := stringLiteral(a); ok {
						s.configured = append(s.configured, lit)
					}
				}
			}
		case *ast.AssignStmt:
			for _, lhs := range node.Lhs {
				if id, ok := lhs.(*ast.Ident); ok {
					s.assigned[id.Name] = true
				}
			}
		case *ast.UnaryExpr:
			// `&x` hands the body's writer somewhere this pass cannot follow,
			// so the variable counts as assigned. Conservative on purpose: a
			// missed vacuity is a gap, a false one is a gate nobody trusts.
			if node.Op == token.AND {
				if id, ok := node.X.(*ast.Ident); ok {
					s.assigned[id.Name] = true
				}
			}
		case *ast.IncDecStmt:
			if id, ok := node.X.(*ast.Ident); ok {
				s.assigned[id.Name] = true
			}
		}
		return true
	})
}

// selfComparisons reports an assertion holding an expression against itself.
//
// Restricted to operand-free expressions — identifiers, selectors, literals.
// `Equal(t, subject.Get(k), subject.Get(k))` renders identically on both sides
// and is a real idempotence claim, so a rule comparing rendered text without
// that restriction would fail the checks most worth keeping.
func (s *scan) selfComparisons() []Vacuity {
	var out []Vacuity
	ast.Inspect(s.fn.Body, func(n ast.Node) bool {
		call, isCall := n.(*ast.CallExpr)
		if !isCall || !comparisons[calleeName(call.Fun)] || len(call.Args) < 3 {
			return true
		}
		left, right := call.Args[1], call.Args[2]
		if !pure(left) || !pure(right) {
			return true
		}
		if a, b := s.render(left), s.render(right); a == b {
			out = append(out, s.at(call.Pos(), RuleSelfComparison,
				calleeName(call.Fun)+" holds "+a+" against itself"))
		}
		return true
	})
	return out
}

// zeroExpectations reports a `var` the body handed to the subject and then
// held the subject's answer against.
//
// Three conditions, and dropping any one of them fails correct code. Declared
// without a value and never written to, so it holds the type's zero. Passed
// into a call, which is what makes it the subject's input rather than a
// standing expectation. And used as an assertion operand — the round trip that
// gives the subject the zero and asks whether it came back.
//
// The middle condition is what a first version left out, and 1060 findings
// said so: `var zero T` compared against a result is the ordinary way to write
// "this answered nothing", and the generated suppressions 1.5 shipped are
// exactly that. A rule flagging those would have been deleted within a week.
func (s *scan) zeroExpectations() []Vacuity {
	bare := map[string]bool{}
	ast.Inspect(s.fn.Body, func(n ast.Node) bool {
		spec, isValue := n.(*ast.ValueSpec)
		if !isValue || len(spec.Values) > 0 {
			return true
		}
		for _, name := range spec.Names {
			bare[name.Name] = true
		}
		return true
	})
	if len(bare) == 0 {
		return nil
	}

	supplied, asserted := map[string]bool{}, map[string]token.Pos{}
	ast.Inspect(s.fn.Body, func(n ast.Node) bool {
		call, isCall := n.(*ast.CallExpr)
		if !isCall {
			return true
		}
		if comparisons[calleeName(call.Fun)] && len(call.Args) >= 3 {
			for _, arg := range call.Args[1:3] {
				if id, isIdent := arg.(*ast.Ident); isIdent && bare[id.Name] {
					asserted[id.Name] = call.Pos()
				}
			}
			return true
		}
		for _, arg := range call.Args {
			if id, isIdent := arg.(*ast.Ident); isIdent && bare[id.Name] {
				supplied[id.Name] = true
			}
		}
		return true
	})

	var out []Vacuity
	for name, pos := range asserted {
		if s.assigned[name] || !supplied[name] {
			continue
		}
		out = append(out, s.at(pos, RuleZeroExpectation,
			name+" was declared, never written to, handed to the subject and then compared "+
				"against what came back — both sides are the type's zero"))
	}
	slices.SortFunc(out, func(a, b Vacuity) int { return a.Line - b.Line })
	return out
}

// configuredNeedles reports a substring assertion whose needle the same body
// configured.
//
// A stand-in told to answer "alpha", asked for its answer, and held to
// containing "alpha" asserts that the stand-in works. That is a true statement
// and it is not the one the check is named for.
func (s *scan) configuredNeedles() []Vacuity {
	if len(s.configured) == 0 {
		return nil
	}
	var out []Vacuity
	ast.Inspect(s.fn.Body, func(n ast.Node) bool {
		call, isCall := n.(*ast.CallExpr)
		if !isCall || calleeName(call.Fun) != "Contains" || len(call.Args) < 2 {
			return true
		}
		needle, isLit := stringLiteral(call.Args[len(call.Args)-2])
		if !isLit || needle == "" {
			return true
		}
		for _, supplied := range s.configured {
			if strings.Contains(supplied, needle) {
				out = append(out, s.at(call.Pos(), RuleConfiguredNeedle,
					strconv.Quote(needle)+" is part of "+strconv.Quote(supplied)+
						", which this function configured"))
				return true
			}
		}
		return true
	})
	return out
}

// uncheckedRejections reports a rejection nothing constrains.
//
// [testkit.Rejects] answers with the message the check reported, and a guard
// that discards it has asserted that *something* failed — which every guard
// proves by construction, including one whose stand-in panicked before the
// check's own assertion ran.
func (s *scan) uncheckedRejections() []Vacuity {
	var out []Vacuity
	ast.Inspect(s.fn.Body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.ExprStmt:
			if call, isCall := node.X.(*ast.CallExpr); isCall && calleeName(call.Fun) == "Rejects" {
				out = append(out, s.at(call.Pos(), RuleUncheckedRejection,
					"the rejection message is discarded, so this asserts only that something failed"))
			}
		case *ast.AssignStmt:
			for i, rhs := range node.Rhs {
				call, isCall := rhs.(*ast.CallExpr)
				if !isCall || calleeName(call.Fun) != "Rejects" || i >= len(node.Lhs) {
					continue
				}
				if id, blank := node.Lhs[i].(*ast.Ident); blank && id.Name == "_" {
					out = append(out, s.at(call.Pos(), RuleUncheckedRejection,
						"the rejection message is assigned to _, so this asserts only that something failed"))
				}
			}
		}
		return true
	})
	return out
}

// unfailableCheck reports a law whose every return is a pass.
//
// The static twin of the saturation prover, and the honest form of the gate
// test 1.3 proposed: that one grepped law files for refusal-shaped comments,
// which tests the comment. This reads the returns. A `Check` that can answer
// only `nil` or `law.Vacuous` reports success on every input, and the run that
// carries it counts a law that was never able to disagree.
func (s *scan) unfailableCheck() []Vacuity {
	// Shipped laws only. A test file declares laws that cannot fail on
	// purpose — a probe the runner is measured against, a stand-in proving the
	// all-vacuous warning fires — and flagging those would make the rule's
	// first four findings the four deliberate ones.
	if s.fn.Name.Name != "Check" || s.fn.Recv == nil || strings.HasSuffix(s.path, "_test.go") {
		return nil
	}
	verdict := false
	ast.Inspect(s.fn.Body, func(n ast.Node) bool {
		ret, isReturn := n.(*ast.ReturnStmt)
		if !isReturn {
			return true
		}
		for _, r := range ret.Results {
			if !passing(r) {
				verdict = true
			}
		}
		return true
	})
	if verdict {
		return nil
	}
	return []Vacuity{s.at(s.fn.Pos(), RuleUnfailableCheck,
		s.receiver()+".Check answers nil or Vacuous on every path, so no input reaches a verdict")}
}

// passing reports whether a return expression is a pass or a refused
// precondition rather than a verdict.
func passing(e ast.Expr) bool {
	switch x := e.(type) {
	case *ast.Ident:
		return x.Name == "nil" || x.Name == "Vacuous"
	case *ast.SelectorExpr:
		return x.Sel.Name == "Vacuous"
	default:
		return false
	}
}

// receiver names the type a method is on, for a diagnostic that can be found.
//
// Unguarded: [scan.unfailableCheck] is the only caller and has already refused
// a declaration with no receiver, so a nil check here would be an arm nothing
// reaches. A method always carries exactly one receiver field.
func (s *scan) receiver() string { return s.render(s.fn.Recv.List[0].Type) }

// at builds one finding at a position.
func (s *scan) at(pos token.Pos, rule, detail string) Vacuity {
	return Vacuity{
		Rule:   rule,
		File:   s.path,
		Line:   s.fset.Position(pos).Line,
		Detail: detail,
	}
}

// render prints an expression back to source, which is how two of them are
// compared and how a finding names what it found.
func (s *scan) render(e ast.Expr) string {
	var b strings.Builder
	// The error is a malformed node, which a file the parser accepted does not
	// have. Handling it would add an arm nothing can reach and nothing checks,
	// which is the defect these rules are for.
	_ = printer.Fprint(&b, s.fset, e)
	return b.String()
}

// pure reports whether an expression names something rather than doing
// something — an identifier, a selector chain, a literal, or an index into
// one.
//
// The restriction that keeps [scan.selfComparisons] from failing an
// idempotence claim, which is two identical calls compared on purpose.
func pure(e ast.Expr) bool {
	switch x := e.(type) {
	case *ast.Ident, *ast.BasicLit:
		return true
	case *ast.SelectorExpr:
		return pure(x.X)
	case *ast.IndexExpr:
		return pure(x.X) && pure(x.Index)
	default:
		return false
	}
}

// calleeName returns the bare name a call invokes, empty for a call through
// something with no name of its own.
func calleeName(fun ast.Expr) string {
	switch x := fun.(type) {
	case *ast.Ident:
		return x.Name
	case *ast.SelectorExpr:
		return x.Sel.Name
	case *ast.IndexExpr:
		return calleeName(x.X)
	case *ast.IndexListExpr:
		return calleeName(x.X)
	default:
		return ""
	}
}

// stringLiteral unquotes a string-literal expression, false for anything else.
func stringLiteral(e ast.Expr) (string, bool) {
	lit, isLit := e.(*ast.BasicLit)
	if !isLit || lit.Kind != token.STRING {
		return "", false
	}
	// The parser accepted this literal, so it unquotes. Reported through the
	// bool rather than branched on, for the reason above: an arm for the
	// impossible is an arm nothing checks.
	v, err := strconv.Unquote(lit.Value)
	return v, err == nil
}

// VacuityRow is one registered class of assertion that cannot fail, with the
// count it may not exceed.
type VacuityRow struct {
	// Ceiling is how many findings the class currently has. Held exactly
	// rather than as a maximum: a class that shrank and left the number behind
	// is a ratchet that stopped ratcheting, and the next regression hides
	// under the slack.
	Ceiling int

	// Why says what the class is and what would close it — or, for one that
	// nothing closes, what makes it correct.
	Why string
}

// VacuityDebt registers every class of unfailable assertion this tree carries.
//
// Keyed by `<rule> <path prefix>`, matched longest-prefix-first, because the
// decision a reader makes is about a class rather than about a line. Eight
// hundred generated sites share one cause and one fix; eight hundred rows
// would be a register nobody reads and a diff nobody can review.
//
// # Two kinds of row, and the difference matters
//
// Three of these are correct and always will be: the assertion library's own
// tests hold `Equal` to agreeing with itself and drive `Rejects` for its
// fatal, which is what testing an assertion helper looks like. Removing them
// would mean removing the tests.
//
// The other two are what remains of a class the detector found on its first
// run and this commit mostly closed. Nothing in the programme's first eleven
// items named it: the stub's generated tests handed the double a zero per
// argument and then asserted the recording carried it, and asserted a pinned
// zero came back from a slot no literal could be written for. Both sides the
// zero, 832 times. 1.5 pinned what `Returns` answers and stopped there.
//
// Pinning the arguments and asserting only the pinnable slots took it to 322.
// What is left is the arguments no literal exists for — a context first among
// them, which every method takes — and closing those needs a value from the
// consumer rather than from the generator.
//
//nolint:gochecknoglobals // a debt register, read-only, test-facing.
var VacuityDebt = map[string]VacuityRow{
	"self-comparison assert_test.go": {
		Ceiling: 4,
		Why: "the assertion library's own tests: Equal agreeing with itself and NotEqual " +
			"disagreeing with itself are the positive and negative cases of the helpers " +
			"under test, and nothing closes this because nothing should",
	},
	"self-comparison rejects_test.go": {
		Ceiling: 1,
		Why: "the same, for the surrogate TB Rejects drives — the test asserts the recorded " +
			"call count against itself to prove the recorder counted, which is the helper " +
			"under test rather than a subject",
	},
	"unchecked-rejection rejects_test.go": {
		Ceiling: 4,
		Why: "Rejects is what these tests are about, so they call it for its fatal rather " +
			"than for its answer; a guard elsewhere that discarded the message would be the " +
			"defect this rule is for, and here the discard is the subject",
	},
	"zero-expectation conformance/corpus": {
		// 312 -> 314: the cursor producer arm's Open and the pool stats
		// role's Stats each take only a context, which lands at its zero
		// exactly as this row's reason describes.
		//
		// 314 -> 319: the roledtypes fixture, added so the pool
		// derivation is exercised by this gate rather than by the
		// validated packs alone. Its Put and Get each take a context,
		// and the five findings are that context at its zero — the same
		// class, attributed by removing the fixture and watching the
		// count return to 314.
		//
		// 319 -> 322: the seededreader fixture, added so the seed seam is
		// exercised by this gate — a harness receiving its corpus because
		// nothing on the interface writes. Its Lookup and Len each take a
		// context, attributed the same way.
		//
		// 322 -> 324: the accumulates fixture, added because eidos
		// registered the mixin and this gate asks for a fixture per
		// classification. Its Add and Total each take a context, which
		// lands at its zero exactly as this row's reason describes —
		// attributed by removing the fixture and watching the count
		// return to 322.
		//
		// 324 -> 325: the batch-writer fixture gained the reader role
		// eidos added for it, so `mode=atomic` has something to be
		// observed through. Its Get takes a context, which lands at its
		// zero exactly as this row's reason describes.
		//
		// 325 -> 330: the restrictedpool fixture, added because pool
		// provenance had no fixture at all — the hostile member every
		// derived pool carries reached no draw, and no corpus package had
		// both a config a run can replace and a tier that draws from one.
		// Its Put and Get each take a context, attributed by removing the
		// fixture and watching the count return to 325.
		//
		// 330 -> 333: the ttlperwrite fixture, added because the corpus
		// carried a lifetime only as a directive constant — no fixture
		// declared one as a field on the value, which is the shape a
		// defect can reach for. Its Put and Read take a context, and its
		// Entry carries one more.
		Ceiling: 333,
		Why: "what is left of the generated stub's zero arguments after pinning every one a " +
			"literal can be written for: a context, an interface, a variadic tail, a type " +
			"from a package the run never read. Those are handed in at their zero and the " +
			"recorded-call check compares the recording against the same zero, which passes " +
			"for a recorder that stored nothing. Closing it needs a value the generator " +
			"cannot derive — a consumer-supplied one, which is the fixture escape hatch the " +
			"suite tier already has and the stub tier does not",
	},
	"zero-expectation generator/stub/testdata": {
		Ceiling: 1,
		Why: "the golden copy of the same residue, which moves when the template that writes " +
			"it does and closes in the same change; one site rather than the corpus's many " +
			"because the golden fixture declares one context parameter",
	},
}

// VacuityCounts sorts findings into their registered classes and reports what
// matched nothing.
//
// Longest prefix wins, so a narrow row can carve an exception out of a broad
// one without either being written to know about the other.
func VacuityCounts(all []Vacuity) (map[string]int, []Vacuity) {
	counts := make(map[string]int, len(VacuityDebt))
	var unregistered []Vacuity
	for _, v := range all {
		best, bestLen := "", -1
		for key := range VacuityDebt {
			rule, prefix, split := strings.Cut(key, " ")
			if !split || rule != v.Rule || !strings.HasPrefix(v.File, prefix) {
				continue
			}
			if len(prefix) > bestLen {
				best, bestLen = key, len(prefix)
			}
		}
		if best == "" {
			unregistered = append(unregistered, v)
			continue
		}
		counts[best]++
	}
	return counts, unregistered
}
