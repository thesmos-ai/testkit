// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package subject

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/cursor"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/lifecycleafterclose"

	"go.thesmos.sh/testkit/core/lawid"
)

// The stamps a law's claim can name, re-exported from the classification
// plugins that own them.
//
// Here rather than read from either generator's own copy, because the
// filling below is the one derivation both tiers share: a claim worded in
// [lawid] and filled twice would say two things about one law, and only a
// manifest diff would ever show it.
const (
	mixinAfterClose      = lifecycleafterclose.Name
	mixinAfterCloseClose = lifecycleafterclose.ParamClose

	// contractCursor is the produced-handle protocol, and its two role
	// params are the methods a cursor claim speaks about.
	contractCursor      = cursor.Name
	contractCursorClose = cursor.ParamClose
	contractCursorNext  = cursor.ParamNext
)

// ClaimFills resolves the placeholder vocabulary a law's claim may
// interpolate, from the methods whose stamps selected it.
//
// Generic by construction: no law names its own fills, the stamps do, and
// over-supplying is free because an absent placeholder ignores its pair.
// First-stamped wins, so a law two methods carry is worded from the one
// the declaration lists first rather than from whichever the map happened
// to yield.
//
// token is the subject's own word. Every claim may name it, and no
// carrier can supply it — it is a fact about the interface, not about the
// method that stamped the law.
func ClaimFills(token string, carriers []Method) []string {
	pairs := []string{lawid.PlaceSubject, token}
	seen := map[string]bool{lawid.PlaceSubject: true}
	set := func(place, v string) {
		if v == "" || seen[place] {
			return
		}
		seen[place] = true
		pairs = append(pairs, place, bareName(v))
	}
	for _, m := range carriers {
		if v, ok := m.MixinParam(mixinAfterClose, mixinAfterCloseClose); ok {
			set(lawid.PlaceClose, v)
		}
		if slices.Contains(m.Contracts, contractCursor) {
			set(lawid.PlaceClose, roleOf(m, contractCursorClose))
			set(lawid.PlaceNext, roleOf(m, contractCursorNext))
			set(lawid.PlaceProduced, contractCursor)
		}
	}
	return pairs
}

// roleOf is the method that fills one role of the cursor contract: the
// partner where this method named one, and this method itself where it
// IS the role.
//
// Both, because a contract's roles are stamped from whichever member
// hosts the directive and that member does not name itself. The cursor
// fixture stamps `role=next close=Close` on Next, so the close partner
// resolves and the next one has nothing to resolve against — which left
// {next} unfilled and declined a law the declaration had fully
// described.
func roleOf(m Method, role string) string {
	if partner := m.ContractPartner(contractCursor, role); partner != "" {
		return partner
	}
	if m.ContractRoles[contractCursor] == role {
		return m.Name
	}
	return ""
}

// bareName is a stamped reference as a reader would say it: the last
// segment, dropping whatever package and type the annotator resolved it
// through.
//
// A claim is a sentence somebody reads in a report, and
// "once corpus/iface/mixin/lifecycleafterclose.Mixed.Close has run" is
// not one. The stamp said `close=Close` and that is the name to speak;
// the qualification exists so the generator can find the method, not so
// a manifest can quote it.
func bareName(v string) string {
	if i := strings.LastIndexByte(v, '.'); i >= 0 {
		return v[i+1:]
	}
	return v
}

// ErrUnworded is the refusal for a law [lawid] does not word yet, as
// distinct from one it words in terms no stamp here supplies.
//
// Two errors rather than one, because they are fixed in different files:
// an unworded law needs a sentence in the catalogue, and an unfilled one
// needs a name on the stamp that selected it. A caller that could not
// tell them apart would have to offer both remedies for either gap.
var ErrUnworded = errors.New("subject: the law catalogue does not word this law")

// ClaimOf words one law for a row that reports under it: the sentence
// [lawid] holds, filled from the carriers' own stamps.
//
// An error rather than a sentence of this package's invention. A law the
// catalogue does not word is a row to refuse by name, and a wording
// naming something no stamp supplies is a bracket that would reach a
// manifest and diff there forever. Both are gaps somewhere specific, and
// inventing prose would hide which — so the error names the placeholder
// that went unfilled.
func ClaimOf(law, token string, carriers []Method) (string, error) {
	template, worded := lawid.ClaimOf(law)
	if !worded {
		return "", fmt.Errorf("%w: %s", ErrUnworded, law)
	}
	return template.Fill(ClaimFills(token, carriers)...)
}
