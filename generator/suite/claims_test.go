// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite_test

import (
	"testing"

	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/lang/golang"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/internal/subject"
	"go.thesmos.sh/testkit/generator/suite"
)

// claimCase is one corpus spelling; a drifted claim here rewrites
// every lock in the fleet.
type claimCase struct {
	name string
	got  string
	want string
}

func (c claimCase) Name() string { return c.name }

// claimMethod builds the signature a claim is worded from: the draws
// as (name, type) pairs, then the result slots. The context parameter
// is omitted — wording never speaks it.
func claimMethod(name string, params []golang.Param, returns ...golang.Return) subject.Method {
	return subject.Method{Sig: &golang.Sig{Name: name, Params: params, Returns: returns}}
}

func keyParam(ident string) golang.Param {
	return golang.Param{Name: ident, Source: storefixture.Named("Key")}
}

func TestClaimWordingsMatchTheCorpus(t *testing.T) {
	t.Parallel()

	errReturn := golang.Return{Error: true}
	get := claimMethod("Get", []golang.Param{keyParam("key")},
		golang.Return{Source: storefixture.Named("Value")}, errReturn)
	incr := claimMethod("Incr", []golang.Param{keyParam("key")}, errReturn)
	total := claimMethod("Total", []golang.Param{keyParam("key")},
		golang.Return{Source: storefixture.Named("int")}, errReturn)
	// Lookup draws `id Key`: the corpus words the TYPE ("a seeded
	// key"), which is what pins drawNoun to the declared type's word
	// rather than the parameter's identifier.
	lookup := claimMethod("Lookup", []golang.Param{keyParam("id")},
		golang.Return{Source: storefixture.Named("Value")}, errReturn)
	size := claimMethod("Size", nil, golang.Return{Source: storefixture.Named("int")}, errReturn)
	length := claimMethod("Len", nil, golang.Return{Source: storefixture.Named("int")}, errReturn)
	subscribe := claimMethod("Subscribe",
		[]golang.Param{{Name: "topic", Source: storefixture.Named("Topic")}},
		golang.Return{Source: storefixture.Chan(storefixture.Named("Event"))}, errReturn)
	put := claimMethod("Put", []golang.Param{
		keyParam("key"), {Name: "value", Source: storefixture.Named("Value")},
	}, errReturn)
	closeM := claimMethod("Close", nil, errReturn)
	// Append draws `e Entry`: the reconciled noun form the design doc
	// records — the type's word, not the parameter's identifier.
	appendM := claimMethod("Append",
		[]golang.Param{{Name: "e", Source: storefixture.Named("Entry")}}, errReturn)
	scan := claimMethod("Scan", nil,
		golang.Return{Source: storefixture.Named("Cursor")}, errReturn)

	testkit.TableTest(t, []claimCase{
		{"smoke with one draw names its noun", suite.SmokeClaim(get, false), "Get survives a call with a derived key"},
		{"smoke with no draws is bare", suite.SmokeClaim(length, false), "Len survives a call"},
		{
			"smoke with several draws says derived inputs",
			suite.SmokeClaim(put, false),
			"Put survives a call with derived inputs",
		},
		{
			"smoke nouns the declared type",
			suite.SmokeClaim(appendM, false),
			"Append survives a call with a derived entry",
		},
		{
			"seeded smoke speaks the seed seam",
			suite.SmokeClaim(lookup, true),
			"Lookup survives a call with a seeded key",
		},
		{
			"opener smoke closes what it opens",
			suite.OpenerSmokeClaim(scan, "cursor"),
			"Scan survives a call and the cursor it opens closes",
		},
		{
			"borrow smoke returns what it borrowed",
			suite.BorrowSmokeClaim(put),
			"Put survives returning a borrowed resource",
		},
		{"cancel", suite.CancelClaim(get), "Get reports a cancelled context as cancelled"},
		{"deadline", suite.DeadlineClaim(get), "Get reports an expired deadline as exceeded"},
		{"nilcontext", suite.NilCtxClaim(get), "Get returns an error rather than panicking on a nil context"},
		{"zero of a named type", suite.ZeroOnErrorClaim(get), "Get returns the zero Value alongside any error"},
		{"zero of a scalar", suite.ZeroOnErrorClaim(length), "Len returns zero alongside any error"},
		{"zero of a channel", suite.ZeroOnErrorClaim(subscribe), "Subscribe returns a nil channel alongside any error"},
		{"idempotent", suite.IdempotentClaim(closeM), "a second Close after a clean one is accepted"},
		{
			"accumulates is idempotent's mirror over the same two calls",
			suite.AccumulatesClaim(incr),
			"a second Incr is accepted rather than refused as a repeat",
		},
		{
			"sentinel miss speaks the bare name",
			suite.MissClaim(get, "kv.ErrNotFound", "wrote"),
			"Get reports ErrNotFound for a key nothing wrote",
		},
		{"zero miss", suite.MissClaim(total, "", "counted"), "Total reports zero for a key nothing has counted"},
		{
			"seeded miss",
			suite.MissClaim(lookup, "ErrNotFound", "seeded"),
			"Lookup reports ErrNotFound for a key nothing seeded",
		},
		{"seeded hit", suite.HitClaim(lookup), "Lookup returns the seeded value for every seeded key"},
		{"seeded count", suite.CountClaim(size), "Size equals the number of seeded entries"},
	}, func(t *testing.T, tc claimCase) {
		testkit.Equal(t, tc.got, tc.want, "the claim wording is the corpus manifests' spelling")
	})
}
