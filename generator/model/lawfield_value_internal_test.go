// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"testing"

	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/core/lawid"
	"go.thesmos.sh/testkit/generator/core/tiers"
	"go.thesmos.sh/testkit/generator/internal/subject"
)

func TestValueOpField(t *testing.T) {
	t.Parallel()

	errRet := res(namedRef("error"))
	keysPool := func() *Bindings {
		return &Bindings{
			Subject: subject.Subject{IfaceName: "Mixed"},
			Keys:    Pool{Field: fieldKey, Q: qStr},
			Actions: []*Action{{Pool: poolKeys}},
		}
	}

	t.Run("a single-value write binds and a chatty one refuses", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: subject.Subject{IfaceName: "Mixed"}}
		field, reason := bindField(b, lawid.CommutativeWrite, "Write",
			projected("Apply", []golang.Param{arg("ctx", ctxRef()), arg("d", namedRef("Delta"))},
				[]golang.Return{errRet}))
		testkit.True(t, reason == "" && field.In != nil, "one value in, one error out: "+reason)

		_, reason = bindField(b, lawid.CommutativeWrite, "Write",
			projected("Apply", []golang.Param{arg("ctx", ctxRef()), arg("d", namedRef("Delta"))},
				[]golang.Return{res(namedRef("int")), errRet}))
		testkit.Assert(t, reason).Contains("more than an error", "a write answers its error alone")
	})

	t.Run("a composite write anchors on the fixture key", func(t *testing.T) {
		t.Parallel()
		b := keysPool()
		field, reason := bindField(b, lawid.IdempotentWrite, "Write",
			stamp(projected("Put", []golang.Param{
				arg("ctx", ctxRef()), arg("k", namedRef(qStr)), arg("v", namedRef(qStr)),
			}, []golang.Return{errRet}), "", qStr, qStr))
		testkit.True(t, reason == "", "the pair anchors on the fixture key: "+reason)
		testkit.Equal(t, string(field.Kind()), "model.lawfield.WriteFixedKey", "through the anchored spelling")
		testkit.True(t, b.LawsUseFixture, "and the property now owes the fixture")

		_, reason = bindField(keysPool(), lawid.IdempotentWrite, "Write",
			stamp(projected("Put", []golang.Param{
				arg("ctx", ctxRef()), arg("k", namedRef("int")), arg("v", namedRef(qStr)),
			}, []golang.Return{errRet}), "", "int", qStr))
		testkit.Assert(t, reason).Contains("beside a pool of", "the anchor and the pool must agree")

		_, reason = bindField(keysPool(), lawid.IdempotentWrite, "Write",
			stamp(projected("Put", []golang.Param{
				arg("ctx", ctxRef()), arg("k", namedRef(qStr)), arg("v", namedRef(qStr)),
			}, []golang.Return{res(namedRef(qStr)), errRet}), "", qStr, qStr))
		testkit.Assert(t, reason).Contains("more than an error", "an anchored write answers its error alone")

		_, reason = bindField(&Bindings{Subject: subject.Subject{IfaceName: "Mixed"}}, lawid.IdempotentWrite, "Write",
			projected("Set", []golang.Param{
				arg("ctx", ctxRef()), arg("a", namedRef(qStr)),
				arg("b", namedRef(qStr)), arg("c", namedRef(qStr)),
			}, []golang.Return{errRet}))
		testkit.Assert(t, reason).Contains("several inputs", "three inputs compose nothing")
	})
}

func TestGeneratorFieldArms(t *testing.T) {
	t.Parallel()

	errRet := res(namedRef("error"))
	pooled := &Bindings{
		Subject: subject.Subject{IfaceName: "Mixed"},
		Keys:    Pool{Field: fieldKey, Q: qStr},
		Values:  Pool{Field: "Body", Q: "string"},
		Actions: []*Action{{Pool: poolKeys}, {Pool: poolValues}},
	}
	classify := projected("Classify",
		[]golang.Param{arg("ctx", ctxRef()), arg("in", namedRef(qStr))},
		[]golang.Return{res(namedRef(qStr)), errRet})

	genField := func(b *Bindings, law, from string, m *subject.Method, fields ...tiers.Field) (*LawField, string) {
		r := tiers.Rule{Law: law, Fields: append(fields, tiers.Field{
			Name: "Pool", Kind: tiers.KindGenerator, From: from,
		})}
		return lawFieldOf(b, nil, r, r.Fields[len(r.Fields)-1], m, nil)
	}

	t.Run("the shared pools bind where actions draw them", func(t *testing.T) {
		t.Parallel()
		field, reason := genField(pooled, lawid.Cacheable, "keys", nil)
		testkit.True(t, reason == "" && field.Pool == "keys", "the keys pool is shared: "+reason)
		field, reason = genField(pooled, lawid.Cacheable, "values", nil)
		testkit.True(t, reason == "" && field.Pool == "values", "the values pool is shared: "+reason)

		bare := &Bindings{Subject: subject.Subject{IfaceName: "Mixed"}}
		_, reason = genField(bare, lawid.Cacheable, "keys", nil)
		testkit.Assert(t, reason).Contains("no action here declares", "an undeclared pool refuses")
		_, reason = genField(bare, lawid.Cacheable, "values", nil)
		testkit.Assert(t, reason).Contains("no action here declares", "both spellings of it")
	})

	t.Run("the law pools declare themselves once, at one type", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: subject.Subject{IfaceName: "Mixed"}}
		roleField := tiers.Field{Name: "Call", Kind: tiers.KindRole, From: "self"}

		field, reason := genField(b, lawid.TotalOver, "inputs", classify, roleField)
		testkit.True(t, reason == "" && field.Pool == "inputs", "the inputs pool declares: "+reason)
		testkit.Equal(t, len(b.LawPools), 1, "once")

		_, reason = genField(b, lawid.TotalOver, "inputs", classify, roleField)
		testkit.True(t, reason == "", "a second law at the same type reuses it: "+reason)
		testkit.Equal(t, len(b.LawPools), 1, "still once")

		intCall := projected("Grade", []golang.Param{arg("ctx", ctxRef()), arg("in", namedRef("int"))},
			[]golang.Return{res(namedRef("int")), errRet})
		_, reason = genField(b, lawid.TotalOver, "inputs", intCall, roleField)
		testkit.Assert(t, reason).Contains("already draws", "one name, one element type")

		_, reason = genField(b, lawid.TotalOver, "inputs", nil,
			tiers.Field{Name: "Limit", Kind: tiers.KindDefault})
		testkit.Assert(t, reason).Contains("draws a domain no role here states",
			"an input pool needs a role to read its domain from")

		field, reason = genField(b, lawid.XSSSafe, "payloads", nil)
		testkit.True(t, reason == "" && field.Pool == "payloads", "the payloads pool declares: "+reason)

		_, reason = genField(b, lawid.PublisherDelivers, "messages", nil)
		testkit.Assert(t, reason).Contains("no action here declares",
			"the messages pool is the values pool, and a fixture whose sequences publish nothing has none")

		_, reason = genField(b, lawid.PublisherDelivers, "nonesuch", nil)
		testkit.Assert(t, reason).Contains("does not compose", "an unbuilt pool refuses by name")
	})
}

func TestConstFieldArms(t *testing.T) {
	t.Parallel()

	constField := func(law, name, from string, optional bool, m *subject.Method) (*LawField, string) {
		r := tiers.Rule{Law: law, Fields: []tiers.Field{
			{Name: name, Kind: tiers.KindConstant, From: from, Optional: optional},
		}}
		return lawFieldOf(&Bindings{Subject: subject.Subject{IfaceName: "Mixed"}}, nil, r, r.Fields[0], m, nil)
	}

	stamped := func(key, value string) *subject.Method {
		m := unstamped()
		sdk.EnsureKey(key, sdk.StringParser).Set(m.Source.EnsureMeta(), value, "test")
		return m
	}

	t.Run("an optional constant with no stamp is omitted", func(t *testing.T) {
		t.Parallel()
		field, reason := constField(lawid.AggregatorBounded, "Min", "shape.mixin.bounded.min",
			true, unstamped())
		testkit.True(t, field == nil && reason == "", "zero is the declared floor")
	})

	// The declared bound is emitted once as a constant and named
	// wherever it is used — the harness constructors take it, this law
	// enforces it, and a consumer's policy check needs the same number.
	// Writing it out here put two homes for one number in one file, four
	// lines apart: `const mixedCapacity = 5` and `Max: 5`.
	t.Run("the declared bound names its constant", func(t *testing.T) {
		t.Parallel()
		field, reason := constField(lawid.AggregatorBounded, "Max", tiers.ParamBoundedLimit,
			false, stamped(tiers.ParamBoundedLimit, "100"))
		testkit.True(t, reason == "", "the stamp parses: "+reason)
		testkit.Equal(t, field.Lit, "mixedCapacity",
			"the constant the harness already declares, not the number again")
	})

	// Every other numeric stamp is still its own literal: only the bound
	// has a constant to name, because only the bound reaches more than
	// one place.
	t.Run("another numeric stamp renders as its literal", func(t *testing.T) {
		t.Parallel()
		field, reason := constField(lawid.AggregatorBounded, "Min", "shape.mixin.bounded.min",
			false, stamped("shape.mixin.bounded.min", "3"))
		testkit.True(t, reason == "" && field.Lit == "3", "the stamp's own number")
	})

	t.Run("the workflow's transitions parse or refuse", func(t *testing.T) {
		t.Parallel()
		field, reason := constField(lawid.ValidTransition, "Allowed",
			"shape.contract.workflow.param.transitions", false,
			stamped("shape.contract.workflow.param.transitions", "Draft>Live, Live>Closed"))
		testkit.True(t, reason == "", "a from>to list parses: "+reason)
		testkit.Equal(t, len(field.Pairs), 2, "into its pairs")

		_, reason = constField(lawid.ValidTransition, "Allowed",
			"shape.contract.workflow.param.transitions", false,
			stamped("shape.contract.workflow.param.transitions", "Draft"))
		testkit.Assert(t, reason).Contains("not a from>to", "an edge needs both ends")
	})

	t.Run("a contract stamp is read off any carrier", func(t *testing.T) {
		t.Parallel()
		host := unstamped()
		host.Contracts = []string{"lease"}
		sdk.EnsureKey("shape.contract.lease.param.held", sdk.StringParser).
			Set(host.Source.EnsureMeta(), "example.com/lease.ErrHeld", "test")
		sibling := unstamped()
		sibling.Contracts = []string{"lease"}

		v, ok := stampValue(harnessOf(host, sibling), sibling, "shape.contract.lease.param.held")
		testkit.True(t, ok && v == "example.com/lease.ErrHeld",
			"the host's stamp speaks for every role method")

		_, ok = stampValue(harnessOf(host), sibling, "shape.mixin.bounded.limit")
		testkit.False(t, ok, "a mixin stamp stays the selecting method's own")
	})
}

// TestClockConstAndTypeArms pins the remaining B1 arms: the duration stamp
// rendered as nanoseconds, the triple-returning result instantiation, and
// the strict value check a value-instantiating row turns on.
func TestClockConstAndTypeArms(t *testing.T) {
	t.Parallel()

	errRet := res(namedRef("error"))

	// The count and the unit the stamp spelled, not the nanoseconds they
	// multiply out to. Both compile to the same value; only one can be
	// read against the declaration. `timeout=100ms` and
	// `Timeout: 100000000` agree and nothing shows that they do.
	ttlField := func(t *testing.T, stamp string) (*LawField, string) {
		t.Helper()
		m := unstamped()
		sdk.EnsureKey("shape.mixin.ttl.ttl", sdk.StringParser).Set(m.Source.EnsureMeta(), stamp, "test")
		r := tiers.Rule{Law: lawid.TTLExpiry, Fields: []tiers.Field{
			{Name: "TTL", Kind: tiers.KindConstant, From: "shape.mixin.ttl.ttl"},
		}}
		return lawFieldOf(&Bindings{Subject: subject.Subject{IfaceName: "Mixed"}},
			nil, r, r.Fields[0], m, nil)
	}

	t.Run("a duration stamp keeps the unit it was written in", func(t *testing.T) {
		t.Parallel()
		field, reason := ttlField(t, "5s")
		testkit.True(t, reason == "", "a duration stamp binds: "+reason)
		testkit.Equal(t, field.Lit, "5", "the count the declaration spelled")
		testkit.Equal(t, string(field.KindName), LawFieldKindPrefix+"ConstDur",
			"rendered as count times unit, so the two can be read against each other")
	})

	// A compound has no single symbol to multiply, so it falls back to
	// the nanosecond literal rather than being spelled wrong.
	t.Run("a compound duration falls back to nanoseconds", func(t *testing.T) {
		t.Parallel()
		field, reason := ttlField(t, "1h30m")
		testkit.True(t, reason == "", "it still binds: "+reason)
		testkit.Equal(t, field.Lit, "5400000000000", "as nanoseconds, which is exact if unreadable")
	})

	t.Run("a triple-returning role instantiates at its first result", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: subject.Subject{IfaceName: "Mixed"}}
		r := roleRule(lawid.CursorNextAfterClose, "Next")
		next := projected("Next", []golang.Param{arg("ctx", ctxRef())},
			[]golang.Return{res(namedRef(qStr)), res(namedRef("bool")), errRet})
		ref, reason := resolveArg(b, nil, r, tiers.ResultOf("Next"), next, nil)
		testkit.True(t, reason == "" && ref != nil,
			"NextOp carries the triple whole, so the type is the first result: "+reason)

		void := projected("Next", []golang.Param{arg("ctx", ctxRef())}, []golang.Return{errRet})
		_, reason = resolveArg(b, nil, r, tiers.ResultOf("Next"), void, nil)
		testkit.Assert(t, reason).Contains("nothing to observe",
			"a next answering only an error observes nothing")
	})

	t.Run("a value-instantiating row holds the reader to the pool's value", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{
			Subject: subject.Subject{IfaceName: "Mixed"},
			Keys:    Pool{Q: qStr},
			Values:  Pool{Q: qStr, Pin: fieldKey},
		}
		mismatched := stamp(projected("Get",
			[]golang.Param{arg("ctx", ctxRef()), arg("k", namedRef(qStr))},
			[]golang.Return{res(namedRef("int")), res(namedRef("error"))}), "reader", qStr, "int")
		_, reason := bindField(b, lawid.TTLExpiry, "Read", mismatched)
		testkit.Assert(t, reason).Contains("beside pools of",
			"TTLExpiry draws the value pool, so the reader must answer it")

		// Windowed's count reads (string → int) beside string pools and binds:
		// its row draws no value, so only the key is held to the pool.
		counted, reason := bindField(b, lawid.Windowed, "Count", mismatched)
		testkit.True(t, reason == "" && counted != nil,
			"a keyless row leaves the reader's value its own: "+reason)
	})

	t.Run("the template accessors answer their imports", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{}
		testkit.True(t, b.ClockPkg() != "", "the clock package is spelled for the clocked property")
		held := b.LeaseHeld()
		testkit.True(t, held.Sym == nil && held.Name == "", "no ctor error means no held sentinel")
		b.Reference.CtorErrs = []CtorErr{{Name: "ErrHeld"}}
		testkit.Equal(t, b.LeaseHeld().Name, "ErrHeld", "the first named ctor error is the held sentinel")
	})
}

// TestPublisherModeConstant pins the mode spelling map: the three directive
// spellings land on the engine enum, and anything else refuses by name.
func TestPublisherModeConstant(t *testing.T) {
	t.Parallel()

	modeField := func(law, value string) (*LawField, string) {
		r := tiers.Rule{Law: law, Fields: []tiers.Field{
			{Name: "Mode", Kind: tiers.KindConstant, From: "shape.contract.publisher.param.mode"},
		}}
		m := unstamped()
		sdk.EnsureKey("shape.contract.publisher.param.mode", sdk.StringParser).
			Set(m.Source.EnsureMeta(), value, "test")
		return lawFieldOf(&Bindings{Subject: subject.Subject{IfaceName: "Contract"}}, nil, r, r.Fields[0], m, nil)
	}

	field, reason := modeField(lawid.PublisherAtLeastOnce, "at-least-once")
	testkit.True(t, reason == "" && field.Const != nil, "the at-least bound spells its enum: "+reason)
	_, reason = modeField(lawid.PublisherAtMostOnce, "at-most-once")
	testkit.True(t, reason == "", "the at-most bound spells its enum: "+reason)
	_, reason = modeField(lawid.PublisherExactlyOnce, "exactly-once")
	testkit.True(t, reason == "", "the exactly bound spells its enum: "+reason)
	_, reason = modeField(lawid.PublisherExactlyOnce, "sometimes")
	testkit.Assert(t, reason).Contains("not a delivery mode", "an unknown spelling refuses by name")
}
