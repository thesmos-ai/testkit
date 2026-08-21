// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package projection

import (
	"strings"

	"go.thesmos.sh/eidos/sdk"
)

// PoolPlan is one role's derived pool: the three members every drawn
// position of that role cycles through, each with a named origin —
// pool[0] the field's default stamp verbatim, pool[1] the
// distinctness transform, pool[2] the hostile member. Three because
// that is the least a pool can hold and still fund a miss (a value
// nothing wrote), an overwrite (a second value), and the hostile
// coverage real drivers die on.
type PoolPlan struct {
	// Role is the //testkit:role stamp the pool serves ("key",
	// "payload").
	Role string

	// Field is the emitted config field's identifier ("KeyPool"),
	// derived from the stamped field's own name through the naming
	// policy.
	Field string

	// Members are the three rendered member expressions, in origin
	// order.
	Members [3]Expr

	// Type is the member's own type, which the emitted config declares
	// a slice of. Through the backend rather than spelled here: a type
	// from the subject's package is an import the config has to
	// register, and only the backend registers one.
	Type sdk.Ref
}

// The pool member transforms — the textual policies pool[1] and
// pool[2] are derived by. Text policies rather than value policies,
// because a default stamp is Go source and stays Go source all the
// way to the emitted file; the transforms never parse what they can
// splice.

// distinctSwap is the textual payload convention the distinctness
// transform pivots on: the corpus's defaults spell "test-*", and the
// second member swaps the word so a miss has a key nothing wrote.
const (
	distinctFrom = "test"
	distinctTo   = "other"
)

// hostilePrefix opens the hostile string member: a NUL byte and an
// invalid UTF-8 sequence, spelled as escape sequences so the emitted
// literal carries them and the generator's own source does not.
const hostilePrefix = `"\x00hostile\xff`

// missTo is the textual payload a key outside the corpus carries.
const missTo = "unseeded"

// DistinctMember is pool[1]: the stamp with its textual payload
// swapped "test" → "other". False when the stamp carries no swap
// point — the pool cannot fund a distinct second member, and the
// caller refuses the derivation rather than emitting two equal
// members for [suite.DistinctPool] to reject at every consumer's run.
func DistinctMember(stamp Expr) (Expr, bool) {
	out := strings.Replace(string(stamp), distinctFrom, distinctTo, 1)
	if out == string(stamp) {
		return "", false
	}
	return Expr(out), true
}

// MissMember is the key deliberately outside a seeded corpus: the
// stamp with its textual payload swapped "test" → "unseeded".
//
// A third word rather than reusing pool[1]. That member is a second
// SEEDED key — the corpus zips every member of the key pool — so a miss
// body drawing it would hit, and the check would pass while asserting
// the opposite of its claim. The swap point is the same one
// [DistinctMember] uses, so a stamp that funds one funds the other.
func MissMember(stamp Expr) Expr {
	out := strings.Replace(string(stamp), distinctFrom, missTo, 1)
	if out == string(stamp) {
		// No swap point. The caller has already refused the pool for
		// exactly this reason, so this is unreachable through
		// [CorpusOf]; returning the stamp keeps it total rather than
		// making a second refusal path nothing exercises.
		return stamp
	}
	return Expr(out)
}

// HostileMember is pool[2]: for a quoted-string stamp, the
// NUL/invalid-UTF-8 member suffixed with the role word; for a
// composite stamp, its first string literal emptied — an empty
// payload is the hostile a struct can always hold. False where
// neither form applies, or where the literal carries escapes this
// splice would mangle.
func HostileMember(stamp Expr, role string) (Expr, bool) {
	s := string(stamp)
	if strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`) && len(s) >= 2 {
		return Expr(hostilePrefix + role + `"`), true
	}
	open := strings.Index(s, `"`)
	if open < 0 {
		return "", false
	}
	closing := strings.Index(s[open+1:], `"`)
	if closing < 0 {
		return "", false
	}
	if strings.Contains(s[open+1:open+1+closing], `\`) {
		// An escaped literal cannot be spliced by index without
		// parsing it; refusing keeps the transform honest.
		return "", false
	}
	return Expr(s[:open+1] + s[open+1+closing:]), true
}

// RoleKey and RolePayload are the two roles a seeded corpus is zipped
// from. Named here rather than matched as literals at each reader,
// because the vocabulary the role directive stamps is validated by its
// readers and this is one of them.
const (
	RoleKey     = "key"
	RolePayload = "payload"
)

// CorpusPlan is the seeded corpus a reader-only interface is populated
// through: one entry per key, values cycled.
//
// Derivable only where BOTH roles are stamped. An interface nothing can
// write to and whose inputs carry no roles cannot be seeded at all — the
// suite would have to invent both what a key is and what a value is —
// and that is a refusal rather than an empty map, because an empty
// corpus makes every read miss and every hit check vacuous.
type CorpusPlan struct {
	// Key and Value are the pools the corpus is zipped from.
	Key, Value PoolPlan

	// MissKey is the key deliberately outside it, which the miss body
	// draws instead of the fixture's alternate — an alternate is a
	// second SEEDED key, and a miss needs one nothing wrote.
	MissKey Expr
}

// CorpusOf pairs the key and payload pools, false where either is absent.
func CorpusOf(pools []PoolPlan) (CorpusPlan, bool) {
	var key, value PoolPlan
	var haveKey, haveValue bool
	for _, p := range pools {
		switch p.Role {
		case RoleKey:
			key, haveKey = p, true
		case RolePayload:
			value, haveValue = p, true
		}
	}
	if !haveKey || !haveValue {
		return CorpusPlan{}, false
	}
	return CorpusPlan{Key: key, Value: value, MissKey: MissMember(key.Members[0])}, true
}
