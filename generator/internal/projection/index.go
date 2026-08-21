// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package projection

import (
	"fmt"
	"sort"

	"go.thesmos.sh/eidos/lang/golang"

	"go.thesmos.sh/testkit/core/lawid"
	"go.thesmos.sh/testkit/engine/suite"
)

// IndexPlan is the generated check index for one interface: the typed
// tree a consumer writes `Without` against, and the one surface that
// names every check this package emits.
//
// A tree of empty structs rather than a map of identifier constants,
// because the index is the drop surface: `ix.Get.ReportsAMiss()` is
// checked by the compiler and a mistyped string is not, and a check
// that is renamed or retired breaks the consumers that named it —
// which is the whole point of dropping by identity rather than by
// prose.
type IndexPlan struct {
	// Groups are the index's members in emission order: one per
	// method that carries checks, then one for the family-scoped
	// tiers.
	Groups []IndexGroup
}

// HasFamily reports whether any group is family-scoped, which is what
// decides whether the qualifier constant is spelled at all. An unused
// constant does not compile, so the question has to be answered before
// the constant block renders rather than inside it.
func (p IndexPlan) HasFamily() bool {
	for _, g := range p.Groups {
		if g.Family != "" {
			return true
		}
	}
	return false
}

// IndexGroup is one member of the index — a method's checks, or a
// family's.
type IndexGroup struct {
	// Field is the member's name in the index struct: the exported
	// method name ("Append"), or the family's word ("Model").
	Field string

	// Method is the source method this group indexes, empty on a
	// family group. The emitted accessors read the method-name
	// constant rather than a literal, so the name has one home.
	Method string

	// Family is the check family this group indexes, empty on a
	// method group. Exactly one of Method and Family is set, which
	// mirrors [IDPlan]'s own scoping rule.
	Family string

	// FamilyConst is the engine identifier the family is declared
	// under, empty on a method group. Composed rather than tabulated
	// because the vocabulary declares each family as Family<Word> and
	// the field above is already that word.
	FamilyConst string

	Accessors []IndexAccessor
}

// IndexAccessor is one check's entry point in the index.
type IndexAccessor struct {
	// Name is the accessor's identifier ("Smoke", "CloseIdempotent").
	Name string

	// Seg is the identity's last segment, which the emitted body
	// spells through the runtime vocabulary.
	Seg string

	// LawConst is the lawid identifier the segment is declared under,
	// empty where the segment is the runtime's own. The emitted body
	// names the law through it rather than repeating the AUTO- text.
	LawConst string

	// SegConst is the engine identifier the segment is declared under,
	// empty on a law. Exactly one of the two is set, and together they
	// are what lets the emitted accessor compose its ID out of the
	// grammar's own words rather than out of a string.
	SegConst string
}

// IndexOf projects the index from an inventory.
//
// Grouping is by ID scope rather than by deriver, because the index is
// the consumer's map of the suite and a consumer does not know which
// deriver produced what: they know they called Get, or that the model
// tier is running. Two derivers landing checks on one method land in
// one group.
//
// Order is emission order for methods — the interface's own
// declaration order, which the inventory already carries — with the
// family groups last and sorted. A generated file whose members
// reshuffle between runs is a diff nobody can review.
func IndexOf(inv Inventory) (IndexPlan, error) {
	var out IndexPlan
	byMethod := map[string]int{}
	byFamily := map[string]int{}

	for _, c := range inv.Checks {
		acc, err := accessorOf(c.ID)
		if err != nil {
			return IndexPlan{}, err
		}
		switch {
		case c.ID.Method != "":
			i, ok := byMethod[c.ID.Method]
			if !ok {
				i = len(out.Groups)
				byMethod[c.ID.Method] = i
				out.Groups = append(out.Groups, IndexGroup{
					Field:  golang.ExportedName(c.ID.Method),
					Method: c.ID.Method,
				})
			}
			out.Groups[i].Accessors = append(out.Groups[i].Accessors, acc)
		default:
			i, ok := byFamily[c.ID.Family]
			if !ok {
				i = len(out.Groups)
				byFamily[c.ID.Family] = i
				field := golang.ExportedName(c.ID.Family)
				out.Groups = append(out.Groups, IndexGroup{
					Field:       field,
					Family:      c.ID.Family,
					FamilyConst: familyConstPrefix + field,
				})
			}
			out.Groups[i].Accessors = append(out.Groups[i].Accessors, acc)
		}
	}

	sort.SliceStable(out.Groups, func(i, j int) bool {
		a, b := out.Groups[i], out.Groups[j]
		if (a.Method != "") != (b.Method != "") {
			return a.Method != ""
		}
		if a.Method != "" {
			return false
		}
		return a.Field < b.Field
	})
	return out, nil
}

// AccessorOf spells one check's index accessor, for the row that names
// it through the index tree.
//
// Exported so the row and the index cannot disagree: a row calling
// `ix.Get.Miss()` where the index declares `ReportsAMiss` compiles
// nowhere, and two derivations of one accessor is how that happens.
func AccessorOf(id IDPlan) (IndexAccessor, error) { return accessorOf(id) }

// accessorOf spells one check's index accessor.
//
// Method-scoped checks are named for their segment and family-scoped
// ones for their law, because that is what distinguishes them: a
// method's checks differ by what is asked of it, and a family's by
// which law is bound. The two vocabularies live in [suite] and
// [lawid] respectively, and neither is restated here.
func accessorOf(id IDPlan) (IndexAccessor, error) {
	if id.Method == "" {
		if name, ok := lawid.AccessorOf(id.Seg); ok {
			constant, _ := lawid.ConstOf(id.Seg)
			return IndexAccessor{Name: name, Seg: id.Seg, LawConst: constant}, nil
		}
		if lawid.IsLaw(id.Seg) {
			return IndexAccessor{}, fmt.Errorf(
				"projection: law %s has no index accessor; word it in lawid", id.Seg,
			)
		}
	}
	name, ok := segAccessor(id.Seg)
	if !ok {
		return IndexAccessor{}, fmt.Errorf(
			"projection: segment %q has no index accessor; word it in segAccessors", id.Seg,
		)
	}
	constant, ok := suite.SegConst(id.Seg)
	if !ok {
		return IndexAccessor{}, fmt.Errorf(
			"projection: segment %q is not declared in the engine vocabulary, "+
				"so an emitted accessor could only spell it as a literal", id.Seg,
		)
	}
	return IndexAccessor{Name: name, Seg: id.Seg, SegConst: constant}, nil
}

// segAccessor spells a segment as an index accessor.
//
// The platform's exported name is the rule — its word splitter already
// treats "-" as a separator, so "zero-on-error" reaches ZeroOnError
// without this file knowing that segments are kebab. Deriving rather
// than tabulating is deliberate: the segment vocabulary grows
// upstream, and a table would send a new segment to a refusal here
// instead of to the obvious identifier.
func segAccessor(seg string) (string, bool) {
	if seg == "" {
		return "", false
	}
	if name, ok := segAccessors()[seg]; ok {
		return name, true
	}
	return golang.ExportedName(seg), true
}

// segAccessors are the segments whose accessor is not their name.
//
// Four, each for its own reason and none of them a style preference.
// Cancel and differential read as claims about the subject where the
// segment is a noun. Nilcontext is one run-together word that no
// casing rule can split, and linearizable is spelled as an identity
// rather than a word. Every other segment reaches its accessor
// mechanically, and a fifth entry here should have to argue for
// itself.
// familyConstPrefix is how the engine vocabulary declares a family.
const familyConstPrefix = "Family"

func segAccessors() map[string]string {
	return map[string]string{
		suite.SegCancel:       "Cancels",
		suite.SegNilContext:   "NilContext",
		suite.SegDifferential: "Agrees",
		suite.SegLinearizable: "Linearizable",
	}
}
