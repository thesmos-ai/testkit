// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gate

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	backendgolang "go.thesmos.sh/eidos/backend/golang"
	"go.thesmos.sh/eidos/frontend/golang"
	"go.thesmos.sh/eidos/pipeline"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/sdk"
	"go.thesmos.sh/eidos/sink"

	"go.thesmos.sh/testkit/core/brand"
	vocab "go.thesmos.sh/testkit/engine/suite"
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/model"
	"go.thesmos.sh/testkit/generator/suite"
)

// Emitted is what the model tier actually asserted for one interface: the
// law identifiers bound into its generated registry, and the reference kind
// the sequences ran against.
type Emitted struct {
	// Fixture is the interface's package path — the corpus address a red
	// gate prints.
	Fixture string

	// Laws are the bound identifiers; Twin reports the reference floor.
	Laws []string

	// Linearizable reports that the concurrency family the shape selected
	// actually reached a row — the leg rendered rather than being refused
	// for something the declaration does not supply.
	Linearizable bool

	// ConcFamily is the concurrency model this interface's shape selects —
	// "kv", "lease", "cas", "append", "session" — empty where none does.
	//
	// Measured because the derivation runs and nothing renders it: a
	// fixture whose shape picks a family owes a linearizability leg, and
	// until one is emitted the selection is work the generator does and
	// throws away. A census that cannot see the family cannot count what
	// the corpus is owed.
	ConcFamily string
	Twin       bool

	// SimPair reports that this interface's shape gives the crash schedule
	// a write to acknowledge and a read to collect the debt, and Recovery
	// that the leg over them actually reached a row.
	//
	// Both, because the gap between them is the interesting number: a pair
	// that derives and no row is a claim the shape supports and the run
	// declines, and the header owes a reason for every one.
	SimPair  bool
	Recovery bool

	// Refusals are the header lines this tier owes a reader: one per claim
	// a rule reached and the run could not state, worded where the refusal
	// was decided.
	//
	// Measured because a refusal that reaches no header is the one absence
	// nothing in the output shows. The row is not planned, so no manifest
	// row goes missing and no check reports; the claim simply is not made,
	// and a reader has no way to tell that from a claim nobody thought of.
	Refusals map[string]string

	// PropSugars are the drawn-input row fields this interface offers a
	// consumer, one per method the tier can already draw an argument for.
	//
	// Measured rather than assumed: the un-sugared Prop is always offered
	// and the sugared fields are what make it worth using, so a fixture
	// whose sugars quietly stop being derived is a surface that got
	// smaller with nothing to show for it.
	PropSugars []string

	// Dir is the fixture's corpus-relative directory and IfaceName its bare
	// interface name — together what the unarmed-door census needs to find
	// the consumer tests and compose the option spellings they would call.
	Dir       string
	IfaceName string

	// Doors maps each guarded law to the config fields its registration
	// reads, and Clocked lists the laws armed only on the run's clock —
	// both invisible skips unless a consumer arms them or a register row
	// argues why not. Unarmed maps each law to the optional roles nothing
	// declared.
	Doors   map[string][]string
	Clocked []string
	Unarmed map[string][]string

	// SentinelStamped reports a declaration whose miss identity the derived
	// oracle routes; SentinelArmed that the sequences carry it. The census
	// holds the first to imply the second — a stamp that stops reaching the
	// sequences is a silent regression of the identity comparison.
	SentinelStamped bool
	SentinelArmed   bool
}

// Emission runs the real generators over the corpus in memory and reports
// what each armed interface bound — the assertion half of the gate.
//
// [Annotate] measures that a classification is stamped somewhere;
// [Coverage.Elsewhere] measures that a law for it exists in the catalogue.
// Neither measures that the stamp bought an assertion, which is the gap the
// generated-suite audit proved by deleting a fixture's whole claim and
// watching the corpus stay green. This is the measurement that closes it:
// the same pipeline the CLI runs, the same plugin set, the queued
// [model.Bindings] read back before any file is rendered.
//
// # Hazards
//
// The run executes every generator, not just the model tier: the model
// plugin reads the suite's projection from the emit queue, and a subset
// would measure a pipeline production never runs. Like [Annotate], the run
// is entirely in memory.
func Emission(ctx context.Context, root string, patterns ...string) ([]Emitted, error) {
	pipe, err := runCorpus(ctx, root, patterns, corpusGenerators())
	if err != nil {
		return nil, err
	}
	return emittedFrom(pipe), nil
}

// emittedFrom reads the bindings back off a store the corpus already ran
// against.
//
// Split from [Emission] so [Measure] can read it off the same run as the
// evidence census. See that function for why the number of runs is worth
// caring about.
func emittedFrom(pipe *pipeline.Pipeline) []Emitted {
	out := make([]Emitted, 0, 128)
	for origin, b := range bindingsByOrigin(pipe) {
		e := Emitted{Fixture: b.IfaceName, IfaceName: b.IfaceName, Twin: b.Reference.Twin()}
		if iface, ok := origin.(*sdk.Interface); ok {
			e.Fixture = iface.Package + "." + iface.Name
			e.Dir = strings.TrimPrefix(iface.Package, "go.thesmos.sh/testkit/conformance/corpus/")
		}
		for _, l := range b.Laws {
			e.Laws = append(e.Laws, l.ID)
			if len(l.Supplied) > 0 {
				if e.Doors == nil {
					e.Doors = map[string][]string{}
				}
				e.Doors[l.ID] = append(e.Doors[l.ID], l.Supplied...)
			}
			if l.Clocked {
				e.Clocked = append(e.Clocked, l.ID)
			}
			if len(l.Unarmed) > 0 {
				if e.Unarmed == nil {
					e.Unarmed = map[string][]string{}
				}
				e.Unarmed[l.ID] = append(e.Unarmed[l.ID], l.Unarmed...)
			}
		}
		e.ConcFamily = b.ConcFamily
		e.SimPair = b.Sim()
		for _, u := range b.Unbound {
			if e.Refusals == nil {
				e.Refusals = map[string]string{}
			}
			e.Refusals[u.Method] = u.Reason
		}
		for _, r := range b.Rows {
			switch r.ID.Seg {
			case vocab.SegLinearizable:
				e.Linearizable = true
			case vocab.SegRecovery:
				e.Recovery = true
			}
		}
		for _, s := range model.PropSugarsOf(b) {
			e.PropSugars = append(e.PropSugars, s.Field)
		}
		sort.Strings(e.PropSugars)
		e.SentinelStamped = b.Reference.MissSym != nil && b.Reference.Derived()
		for _, a := range b.Actions {
			if a.Sentinel != nil {
				e.SentinelArmed = true
			}
		}
		sort.Strings(e.Laws)
		sort.Strings(e.Clocked)
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Fixture < out[j].Fixture })
	return out
}

// bindingsByOrigin reads each interface's model tier off the region it is
// contributed into.
//
// Not from the pending emits: the model tier stopped queueing its bindings
// when it became a contributor to the harness, because a queued emit value
// always renders into a file and this tier emits none. The bindings reach
// the store as one more declaration under the harness's decls region, and
// that is where a census has to look for them.
//
// This gate read the pending queue for a while after the move, and measured
// nothing: every fixture reported no laws, so every stamped classification
// read as bound nowhere and every reference read as not a twin. A census
// that measures nothing reports the whole corpus as covered by nobody,
// which is the failure mode [runCorpus] names and the one a gate can least
// afford.
func bindingsByOrigin(pipe *pipeline.Pipeline) map[sdk.Node]*model.Bindings {
	out := map[sdk.Node]*model.Bindings{}
	for origin, c := range sdk.PendingByOrigin[*suite.Contract](pipe.Store().Emit()) {
		for _, item := range c.Decls().Items {
			if b, ok := item.(*model.Bindings); ok {
				out[origin] = b
				break
			}
		}
	}
	return out
}

// runCorpus builds the pipeline the CLI runs, against a memory sink, and runs
// it over the corpus — handing back the store for a census to read.
//
// One runner for every census that reads the emit queue rather than the files,
// so they measure one production. A census assembling its own plugin set would
// measure a pipeline nothing runs, and would keep measuring it after the real
// set changed — the failure mode a gate can least afford.
//
// One function rather than a builder and a runner, because the two error paths
// are the whole of what a caller has to handle and splitting them multiplied
// those paths by the number of censuses without adding an answer either of
// them could give.
//
// The generator set is a parameter rather than read here, for two reasons that
// point the same way. A census cannot quietly assemble its own — the argument
// is what makes "the same plugins the CLI runs" a fact at the call site. And
// the malformed-set arm becomes reachable: a gate whose pipeline fails to
// build must say so, because the alternative is a census that measures nothing
// and reports every classification as covered by nobody, or as covered by
// everybody, depending on which direction it reads.
func runCorpus(
	ctx context.Context, root string, patterns []string, gens []plugin.Generator,
) (*pipeline.Pipeline, error) {
	builder := pipeline.New().
		WithBrand(brand.Name).
		WithDirectivePrefix(brand.DirectivePrefix).
		WithSourceRoot(root).
		WithFrontend(golang.New())
	for _, a := range generator.Annotators() {
		builder = builder.WithAnnotator(a)
	}
	for _, g := range gens {
		builder = builder.WithGenerator(g)
	}
	pipe, err := builder.
		WithBackend(backendgolang.New()).
		WithSink(sink.NewMemory()).
		Build()
	if err != nil {
		return nil, fmt.Errorf("gate: build corpus pipeline: %w", err)
	}

	scoped := make([]string, len(patterns))
	for i, p := range patterns {
		scoped[i] = filepath.Join(root, p)
	}
	if err := pipe.Run(ctx, scoped...); err != nil {
		return nil, fmt.Errorf("gate: run corpus pipeline: %w", err)
	}
	return pipe, nil
}

// corpusGenerators is the generator half of the plugin set the CLI registers.
//
// Every testkit generator implements the role; the registry's type is the
// plugin universe's, so the assertion narrows it back.
func corpusGenerators() []plugin.Generator {
	all := generator.Generators()
	out := make([]plugin.Generator, 0, len(all))
	for _, g := range all {
		if gen, ok := g.(plugin.Generator); ok {
			out = append(out, gen)
		}
	}
	return out
}
