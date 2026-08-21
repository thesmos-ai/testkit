// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Internal for the reason derive_stamps_test.go is: the fixtures
// populate the unexported stamp projections through the real keys on
// real bags, which only this package's own constructors reach.
package suite

import (
	"testing"

	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/lifecycle"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/writer"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/atomic"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/core/lawid"
	vocab "go.thesmos.sh/testkit/engine/suite"
	"go.thesmos.sh/testkit/generator/internal/projection"
	"go.thesmos.sh/testkit/generator/internal/subject"
)

// lawMethod builds one law-bearing method: mixins attached, their
// params stamped through the real keys on one bag that serves both
// readers — the projection map the wording fills from, and the source
// node the selection's When clauses read.
func lawMethod(name string, mixinNames []string, stamp func(*sdk.Bag)) subject.Method {
	bag := sdk.NewBag()
	if stamp != nil {
		stamp(bag)
	}
	src := &node.Method{Name: name}
	src.MetaBag = bag
	return subject.Method{
		Sig:         &golang.Sig{Name: name, Source: src},
		Mixins:      mixinNames,
		MixinParams: mixinParamsOf(bag, mixinNames),
	}
}

// afterClose stamps the kv-shaped after-close declaration.
func afterClose(bag *sdk.Bag) {
	shape.MixinParamKey(MixinAfterClose, MixinAfterCloseClose).Set(bag, "Close", "test")
	shape.MixinParamKey(MixinAfterClose, MixinAfterCloseSentinel).Set(bag, "kv.ErrClosed", "test")
}

// lawStore is the kv Store shape as the laws deriver reads it: ttl
// and after-close on the writer, after-close across the readers, a
// concurrency claim, and a suite-owned idempotent teardown.
func lawStore() Iface {
	put := lawMethod(
		"Put",
		[]string{MixinTTL, MixinAfterClose, MixinConcurrent},
		func(bag *sdk.Bag) {
			afterClose(bag)
			shape.MetaShape.Set(bag, writer.Name, "test")
		},
	)
	get := lawMethod("Get", []string{MixinAfterClose}, afterClose)
	length := lawMethod("Len", []string{MixinAfterClose}, afterClose)
	closeM := lawMethod("Close", []string{MixinIdempotent}, func(bag *sdk.Bag) {
		shape.MetaShape.Set(bag, lifecycle.Name, "test")
	})
	return Iface{
		Name:      "Store",
		Token:     "store",
		Qualifier: "store",
		Methods:   []subject.Method{put, get, length, closeM},
	}
}

func lawPlansByID(t *testing.T, f Iface) (map[vocab.ID]projection.CheckPlan, []Refusal) {
	t.Helper()
	plans, refusals := Laws{}.Derive(f)
	byID := map[vocab.ID]projection.CheckPlan{}
	for _, p := range plans {
		id, err := p.ID.Render()
		testkit.NoError(t, err, "the derived ID is well formed")
		byID[id] = p
	}
	return byID, refusals
}

func TestLawsDeriveTheStoreRows(t *testing.T) {
	t.Parallel()

	byID, refusals := lawPlansByID(t, lawStore())
	testkit.Len(t, refusals, 0, "every earned own-leg law is worded and fillable")

	t.Run("the after-close law carries its stamped probe set", func(t *testing.T) {
		t.Parallel()
		p, held := byID["model/store/AUTO-LIFECYCLE-AFTER-CLOSE"]
		testkit.True(t, held, "the after-close mixin earns its own leg")
		testkit.Equal(t, p.Class, vocab.ClassLifecycle, "under the lifecycle class")
		testkit.Equal(t, p.Claim, "once Close has run, every method reports the closed sentinel",
			"the claim fills the declared teardown's name")
		testkit.Len(t, p.Binds, 1, "one law, one bind")
		testkit.Equal(t, p.Binds[0].Render(), "AUTO-LIFECYCLE-AFTER-CLOSE(Put Get Len)",
			"the probe set is exactly the stamped carriers, in method order")
	})

	t.Run("the ttl law rides the clocked leg from the binding fact", func(t *testing.T) {
		t.Parallel()
		p, held := byID["model/store/AUTO-TTL-EXPIRY"]
		testkit.True(t, held, "the ttl mixin earns the clocked leg")
		testkit.Equal(t, p.Class, vocab.ClassClocked, "timeaware laws report as clocked")
		testkit.Equal(t, p.Claim, "an entry stops being readable once its lifetime has run out",
			"the static corpus wording")
	})

	t.Run("the stamped sentinel licenses the poison law", func(t *testing.T) {
		t.Parallel()
		p, held := byID["model/store/AUTO-POISON-CONSISTENT"]
		testkit.True(t, held, "the poisoned state is the closed state")
		testkit.Equal(t, p.Class, vocab.ClassPoison, "under the poison class")
		testkit.Equal(t, p.Claim, "once the store reports it is closed, it keeps reporting it",
			"the claim fills the subject's token")
	})

	t.Run("the concurrency claim earns the linearizable leg", func(t *testing.T) {
		t.Parallel()
		p, held := byID["model/store/AUTO-LINEARIZABLE"]
		testkit.True(t, held, "the concurrent mixin lowers to the linearize engine")
		testkit.Equal(t, p.Class, vocab.ClassConcurrent, "under the concurrent class")
		testkit.Equal(
			t,
			p.Claim,
			"concurrent operation histories are linearizable",
			"the leg wording",
		)
	})

	t.Run("a suite-tabled mixin's law has no model twin", func(t *testing.T) {
		t.Parallel()
		_, held := byID["model/store/"+lawid.IdempotentLifecycle]
		testkit.False(t, held, "Close/idempotent is the suite's row; one tier owns each claim")
	})
}

func TestLawsBundleTheObservational(t *testing.T) {
	t.Parallel()

	iface := Iface{Name: "Store", Token: "store", Qualifier: "store", Methods: []subject.Method{
		lawMethod("Set", []string{atomic.Name}, nil),
	}}
	byID, refusals := lawPlansByID(t, iface)
	testkit.Len(t, refusals, 0, "a bundled law needs no wording of its own")

	p, held := byID["model/store/laws"]
	testkit.True(t, held, "an observational law rides the shared sequences")
	testkit.Equal(t, p.Class, vocab.ClassLaws, "under the laws class")
	testkit.Equal(
		t,
		p.Claim,
		"every bound law holds over random operation sequences",
		"the bundle wording",
	)
	testkit.Len(t, p.Binds, 1, "the bundle's binds itemize its laws")
	testkit.Equal(t, p.Binds[0].Law, lawid.AtomicWrite, "the atomic mixin's law, by identifier")
}

func TestLawsDeriveTheCursorRows(t *testing.T) {
	t.Parallel()

	bag := sdk.NewBag()
	shape.MetaContracts.Set(bag, []string{ContractCursor}, "test")
	shape.ContractRoleKey(ContractCursor).Set(bag, ContractCursorOpen, "test")
	shape.ContractPartnerKey(ContractCursor, ContractCursorClose).Set(bag, "Close", "test")
	shape.ContractPartnerKey(ContractCursor, ContractCursorNext).Set(bag, "Next", "test")
	roles, partners, params := contractDataOf(bag)
	src := &node.Method{Name: "Scan"}
	src.MetaBag = bag
	opener := subject.Method{
		Sig:              &golang.Sig{Name: "Scan", Source: src},
		Contracts:        shape.Contracts(bag),
		ContractRoles:    roles,
		ContractPartners: partners,
		ContractParams:   params,
	}

	byID, refusals := lawPlansByID(
		t,
		Iface{Name: "Log", Token: "log", Qualifier: "log", Methods: []subject.Method{opener}},
	)
	testkit.Len(t, refusals, 0, "the cursor partners word both laws")

	t.Run("a second close changes nothing", func(t *testing.T) {
		t.Parallel()
		p, held := byID["model/log/AUTO-CURSOR-CLOSE-IDEMPOTENT"]
		testkit.True(t, held, "the open role earns the produced handle's lifecycle laws")
		testkit.Equal(t, p.Class, vocab.ClassLifecycle, "under the lifecycle class")
		testkit.Equal(t, p.Claim, "a second Close on a cursor changes nothing",
			"the claim fills the close partner and the handle's word")
	})

	t.Run("a closed cursor's read speaks the sentinel", func(t *testing.T) {
		t.Parallel()
		p, held := byID["model/log/AUTO-CURSOR-NEXT-AFTER-CLOSE"]
		testkit.True(t, held, "the second cursor law from the same stamp")
		testkit.Equal(t, p.Claim, "once a cursor is closed, Next reports the closed sentinel",
			"the claim fills the next partner and the handle's word")
	})
}

func TestLawsRefuseWhatTheyCannotWord(t *testing.T) {
	t.Parallel()

	// After-close with no close name: the claim template needs {close}
	// and no stamp supplies it.
	iface := Iface{Name: "Store", Token: "store", Qualifier: "store", Methods: []subject.Method{
		lawMethod("Put", []string{MixinAfterClose}, func(bag *sdk.Bag) {
			shape.MixinParamKey(MixinAfterClose, MixinAfterCloseSentinel).
				Set(bag, "kv.ErrClosed", "test")
		}),
	}}
	byID, refusals := lawPlansByID(t, iface)
	_, held := byID["model/store/AUTO-LIFECYCLE-AFTER-CLOSE"]
	testkit.False(t, held, "a half-filled claim must not render into a manifest")
	testkit.True(t, len(refusals) > 0, "the gap is named, never silent")
	found := false
	for _, r := range refusals {
		if r.What == lawid.LifecycleAfterClose+" for Store" {
			found = true
			testkit.Contains(
				t,
				r.Why,
				lawid.PlaceClose,
				"the refusal names the missing placeholder",
			)
		}
	}
	testkit.True(t, found, "the refusal is attributed to the law and the interface")
}
