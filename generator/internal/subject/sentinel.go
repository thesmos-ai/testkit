// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package subject

import (
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/notfound"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/ttl"
)

// MissSentinel is the error a method reports for an input nothing
// wrote, and whether it declares one.
//
// Two declarations, in precedence order. A `notfound sentinel=` is the
// read's own answer and the ordinary case. A `ttl notfound=` overrides
// it, because expiry and absence are separate conditions that usually
// coincide: a store whose lapsed reads report differently from its
// missing ones says so there, and one where they agree declares only
// the first and this falls through to it.
//
// A named list rather than a scan, and deliberately short. A sentinel
// scoped to somebody else's condition is not this one: `deleteremoves
// sentinel=` names what a read reports AFTER A DELETE, which coincides
// with a miss on a store that keeps no tombstone and is a different
// answer on one that does. The version that scanned every mixin for a
// `sentinel=` took the first it met, so an interface stamping ttl beside
// lifecycleafterclose was handed a post-close sentinel as its miss.
//
// The convention is documented on ttl's parameter upstream; this is its
// one implementation, so no caller has to remember the order. Both tiers
// read it, which is why it lives here rather than in either.
func MissSentinel(m Method) (string, bool) {
	if v, declared := stampedParam(m, ttl.Name, ttl.ParamNotFound); declared {
		return v, true
	}
	return stampedParam(m, notfound.Name, notfound.ParamSentinel)
}

// stampedParam reads one mixin parameter off the DECLARATION rather
// than off the projected map.
//
// The map exists because a template renders long after the node left
// scope, and it is right for that. This is asked during derivation,
// where the node is still in hand — and [Method.Shape] already reads
// the same way for the same reason. Reading the declaration also means
// a caller holding a hand-built projection, which every deriver test
// does, gets the answer the pipeline would give rather than an empty
// map's silence.
func stampedParam(m Method, mixin, param string) (string, bool) {
	if m.Source == nil {
		return "", false
	}
	v, found := shape.MixinParamKey(mixin, param).Get(m.Source.Meta())
	return v, found && v != ""
}
