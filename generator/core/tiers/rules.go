// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package tiers

import (
	"slices"

	"go.thesmos.sh/testkit/core/lawid"
)

// Rule selects one law from the classifications a method carries.
//
// A rule rather than a table entry keyed on the classification, because three
// cases the law catalogue already contains cannot be keyed that way. `cursor`
// selects two laws from one stamp — "Next after Close reports the sentinel" and
// "a second Close is a no-op" — and a single-valued entry can only name one.
// `idempotent` selects a different law beside `lifecycle` than beside `writer`,
// so the selector is the whole classification set rather than any one member.
// And `publisher` selects between five laws on the strength of a parameter's
// value, which is not a classification at all.
type Rule struct {
	// Law is the identifier the bound law reports under.
	Law string

	// Needs is the classification set a method must carry in full — its
	// detector shape, its mixins, and the roles it fills in any contract, all
	// in one namespace because a method carries them in one namespace.
	//
	// Every rule whose Needs are satisfied binds. There is no precedence and no
	// supersession: two laws selected by one stamp are two properties, not two
	// candidates. Where two rules would otherwise contest a method, the more
	// specific one names the extra classification that distinguishes it, which
	// is what keeps `{writer, idempotent}` and `{lifecycle, idempotent}` apart
	// with no tiebreak to write down.
	Needs []string

	// When are conditions on the classification's own parameters, all of which
	// must hold. Empty for a rule the classifications alone decide.
	When []Condition

	// Fields is the manifest — one entry per exported field of the law struct,
	// naming what fills it.
	//
	// Exhaustive by gate rather than by convention: the conformance gate walks
	// the law type by reflection and fails on any exported field this does not
	// name, and on any name that is not a field. A law that grows a field
	// breaks the gate loudly instead of binding with it zero-filled, which is
	// the failure mode this whole structure exists to prevent — a nil closure
	// in a law is a check that silently asserts nothing.
	Fields []Field
}

// Condition is one requirement on a classification's parameter.
type Condition struct {
	// Param is the stamp key, in the annotator's own spelling —
	// `shape.contract.<name>.param.<key>` or `shape.mixin.<name>.<param>`.
	//
	// Composed by eidos rather than spelled by hand anywhere it is read;
	// [TestEveryConditionReadsADeclaredParameter] holds every literal here
	// to the key the registry composes, because a key off by one segment
	// matches nothing and reports nothing.
	Param string

	// Equals is the value the stamp must carry. Empty means "stamped at all,
	// with any value".
	Equals string

	// NotEquals holds where the stamp carries anything else, including
	// nothing at all.
	//
	// Not the same as Absent, and the difference is a real law: `codec`
	// defaults to exact fidelity, so the exact roundtrip is claimed both by
	// `fidelity=exact` and by saying nothing. Absent alone would unbind the
	// law for anyone who wrote the default out.
	NotEquals string

	// Absent inverts the test: the stamp must not be present. Used where a law
	// is the unrefined form of another — a publisher with no declared delivery
	// mode claims delivery and nothing about duplicates.
	Absent bool
}

// Field is one exported field of a law struct and the source that fills it.
type Field struct {
	// Name is the field's identifier on the law struct.
	Name string

	// Kind says which derivation fills it.
	Kind FieldKind

	// From qualifies Kind, in a vocabulary each kind defines. See the FieldKind
	// constants.
	From string

	// PerValue admits the second way a quantity is declared: carried on
	// the drawn value rather than fixed by the stamp From names.
	//
	// `ttl duration=1m` sets one lifetime for every write. A store whose
	// writes each say how long they last stamps no duration and puts the
	// number on the value instead — SETEX, a message with its own
	// expiry — and it is the same claim worded the same way, so declining
	// the law there would leave the commoner of the two shapes unchecked.
	//
	// Read only where the stamp is absent. A declaration carrying both has
	// said the quantity twice, and the stamp is the one somebody wrote on
	// purpose.
	PerValue bool

	// Optional marks a field a correct binding may leave at its zero value.
	//
	// Without it the gate cannot tell a deliberate nil from a field the binding
	// forgot, and the manifest would have to choose between rejecting sound
	// bindings and admitting broken ones.
	Optional bool

	// SUTOnly marks a field the law calls on the subject alone, whose effect
	// the subject itself undoes before Check returns.
	//
	// A law field normally may not reach a method the derived oracle answers
	// inertly: the subject would move and the reference would not, and every
	// comparison after it would report a divergence the subject never caused.
	// That reasoning assumes the call leaves something behind. Where the law's
	// whole point is a scope the subject discards — an errored transaction, an
	// acquire it releases — nothing is left to fall behind, and refusing the
	// binding costs the claim for a desync that cannot happen.
	//
	// The safety argument lives with the conduct register, which records the
	// same laws as self-cleaning. Setting this on a field of a law that is not
	// self-cleaning is how a shared pair silently desynchronizes, so the two
	// must be read together.
	SUTOnly bool
}

// FieldKind names how a law's field is filled.
type FieldKind string

// The kinds, and the vocabulary each gives [Field.From].
const (
	// KindRole is a closure that calls a method. From names which:
	//
	//   - `self`                  the method carrying the stamp that selected the law
	//   - `<contract>.<role>`     the method filling that role of that contract
	//   - `<mixin>.<param>`       the method that mixin's parameter names
	//   - `family.<detector>`     a method of that shape in the same state cluster
	//
	// The last is the one RFC-0003's four-kind sketch had no way to say.
	// AUTO-WRITE-OBSERVABLE needs the writer *and* the reader that observes it,
	// and only one of the two carries the stamp that selected the law.
	KindRole FieldKind = "role"

	// KindGenerator is one of the run's shared rapid generators, named by
	// From — `keys`, `values`, `inputs`, `messages`, `entries`, `cursors`.
	//
	// Shared, never rebuilt per law: keys drawn from a private pool never
	// revisit what another action wrote, and a comparison law over a key space
	// nothing collides in passes over a history with nothing interesting in it.
	KindGenerator FieldKind = "generator"

	// KindConstant is a value read from a classification's parameter stamp.
	// From is the stamp key, spelled as [Condition.Param] spells it.
	KindConstant FieldKind = "constant"

	// KindTrace is left at its zero value. The runner calls BindTrace on any
	// law implementing law.TraceBinder, so a binding that filled it would be
	// overwritten. From is empty.
	KindTrace FieldKind = "trace"

	// KindHandle is a runtime object the generated file constructs once and
	// shares — From names the derivation, e.g. `key-projection` for the field
	// extractor a value type yields, `history` for the per-iteration chain
	// trace, `subject-factory` for a fresh subject.
	KindHandle FieldKind = "handle"

	// KindDefault is a field the law itself defaults, correctly, and the
	// binding must leave at its zero value. From is empty.
	//
	// Distinct from KindSupplied, and the distinction is the one RFC-0003
	// draws for bench budgets one tier down: a repetition count is a knob
	// about how hard the search looks, and the law owns it; a bound is a
	// claim about the subject, and only a declaration can state it. Filling a
	// knob from the generator would be inventing a number, and marking it
	// supplied would make every law report a skip for a field that was never
	// missing.
	KindDefault FieldKind = "default"

	// KindSupplied is a field no stamp names and no signature derives — an
	// equality, a total order, a merge. From names the option that fills it, so
	// a skipped law can say what would arm it.
	//
	// With Optional, the zero value is a sound default and the law binds
	// regardless; without it the law does not bind until a consumer supplies
	// one, and reports as a skip naming the option rather than as silence.
	KindSupplied FieldKind = "supplied"
)

// Select returns every law the given classifications and parameters earn.
//
// classifications is the method's full set — detector shape, mixins, contract
// roles. params maps a stamp key to its value, in [Condition.Param]'s spelling;
// a key absent from the map is an unstamped parameter.
//
// Order is the rule list's, which is grouped by axis and stable, so a generated
// binding set and the header that describes it agree without either sorting.
func Select(classifications []string, params map[string]string) []Rule {
	var out []Rule
	for _, r := range rules {
		if r.matches(classifications, params) {
			out = append(out, r)
		}
	}
	return out
}

// matches reports whether one rule's requirements all hold.
func (r Rule) matches(classifications []string, params map[string]string) bool {
	for _, need := range r.Needs {
		if !slices.Contains(classifications, need) {
			return false
		}
	}
	for _, c := range r.When {
		value, stamped := params[c.Param]
		switch {
		case c.Absent:
			if stamped {
				return false
			}
		case c.NotEquals != "":
			// An unstamped parameter satisfies this: it carries something
			// other than the excluded value by virtue of carrying nothing.
			if stamped && value == c.NotEquals {
				return false
			}
		case !stamped:
			return false
		case c.Equals != "" && value != c.Equals:
			return false
		}
	}
	return true
}

// LawsFor returns every law identifier some rule could select on the strength
// of this classification, sorted and deduplicated.
//
// Weaker than [Select], and deliberately. This answers "could this
// classification ever earn a law", which is what a header and a census need;
// [Select] answers "what does this method earn", which is what a binding needs.
// A rule needing two classifications is reported under both, because from
// either one's point of view the law is reachable.
func LawsFor(classification string) []string {
	var out []string
	for _, r := range rules {
		if slices.Contains(r.Needs, classification) && !slices.Contains(out, r.Law) {
			out = append(out, r.Law)
		}
	}
	slices.Sort(out)
	return out
}

// Rules returns the whole catalogue, in declaration order.
//
// Exported for the conformance gate, which walks it two ways: every law it
// names against the shipped catalogue, and every field manifest against the law
// struct's own reflection. Neither check can live in this module, which must
// not depend on `engine` (docs/adr/0005).
func Rules() []Rule { return slices.Clone(rules) }

// LawNegation is one claim-law conflict: a mixin whose semantics negate a law
// another classification would earn on the same interface.
type LawNegation struct {
	Law, Mixin, Reason string
}

// LawNegated reports whether the named mixin's claim negates the law, with
// the reason the generated header prints.
//
// Selection composes per method, but a claim holds over the interface: the
// sticky stamp rides the reader, and it is the writer-earned observability
// law it negates. The corpus proved the row — the first pool wide enough to
// draw a same-key overwrite failed one of the two laws whichever way the
// subject behaved, because on a sticky store they contradict each other.
func LawNegated(law, mixin string) (string, bool) {
	for _, n := range lawNegations {
		if n.Law == law && n.Mixin == mixin {
			return n.Reason, true
		}
	}
	return "", false
}

// LawNegations returns every conflict row, for the censuses.
func LawNegations() []LawNegation { return slices.Clone(lawNegations) }

//nolint:gochecknoglobals // a lookup table, read-only after init.
var lawNegations = []LawNegation{
	{
		Law:    lawid.WriteObservable,
		Mixin:  mixinSticky,
		Reason: "the sticky claim negates it — a write to a resolved key must not change what a read answers",
	},
}
