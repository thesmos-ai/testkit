// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model_test

import (
	"strings"
	"testing"

	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/eidostest/plugintest"
	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/sdk"
	"go.thesmos.sh/eidos/store"

	"go.thesmos.sh/testkit/generator/internal/gentest"
	"go.thesmos.sh/testkit/generator/model"
	"go.thesmos.sh/testkit/generator/suite"
)

// sessionStore is a session fixture in store form: a reader carrying a
// session mixin whose version= names the given member of a value struct
// declared with field Rev and zero-arg method Stamp — the two spellings the
// refusal must tell apart.
func sessionStore(t *testing.T, member string) *sdk.Store {
	t.Helper()
	valueRef := func() *sdk.TypeRef { return storefixture.PkgNamed("example.com/sess", "Value") }
	s := storefixture.New().
		Package("sess", "example.com/sess").
		Struct("Value", func(b *storefixture.StructBuilder) {
			b.Pos(gentest.AtFile("sess/iface.go"))
			b.Field("Key", storefixture.Named("string"), nil)
			b.Field("Rev", storefixture.Named("int64"), nil)
			b.Method("Stamp", func(m *storefixture.MethodBuilder) {
				m.Return(storefixture.Named("int64"))
			})
		}).
		Interface("Mixed", func(i *storefixture.InterfaceBuilder) {
			i.Pos(gentest.AtFile("sess/iface.go"))
			i.Directive(storefixture.Directive("suite"))
			i.Directive(storefixture.Directive("model"))
			i.Method("Store", func(m *storefixture.MethodBuilder) {
				gentest.Ctx(m)
				m.Param("v", valueRef())
				gentest.Err(m)
			})
			i.Method("Get", func(m *storefixture.MethodBuilder) {
				gentest.Ctx(m)
				m.Param("key", storefixture.Named("string"))
				m.Return(valueRef())
				gentest.Err(m)
			})
		}).
		Build()
	stampShape(s, "Store", "writer", "", "example.com/sess.Value")
	stampShape(s, "Get", "reader", "string", "example.com/sess.Value")
	for _, iface := range s.Nodes().Interfaces().Items() {
		for _, m := range iface.Methods {
			if m.Name != "Get" {
				continue
			}
			shape.MetaMixins.Set(m.EnsureMeta(), []string{"monotonicreads"}, "test")
			shape.MixinParamKey("monotonicreads", "version").Set(m.EnsureMeta(), member, "test")
		}
	}
	return s
}

// casStore is the cas-contract fixture in store form: a writer carrying the
// contract whose version= names the given member of the cell's value struct,
// declared with field Version and zero-arg method Next.
func casStore(t *testing.T, member string) *sdk.Store {
	t.Helper()
	valueRef := func() *sdk.TypeRef { return storefixture.PkgNamed("example.com/cc", "Value") }
	s := storefixture.New().
		Package("cc", "example.com/cc").
		Struct("Value", func(b *storefixture.StructBuilder) {
			b.Pos(gentest.AtFile("cc/iface.go"))
			b.Field("Body", storefixture.Named("string"), nil)
			b.Field("Version", storefixture.Named("int64"), nil)
			b.Method("Next", func(m *storefixture.MethodBuilder) {
				m.Return(storefixture.Named("int64"))
			})
		}).
		Interface("Cell", func(i *storefixture.InterfaceBuilder) {
			i.Pos(gentest.AtFile("cc/iface.go"))
			i.Directive(storefixture.Directive("suite"))
			i.Directive(storefixture.Directive("model"))
			i.Method("Put", func(m *storefixture.MethodBuilder) {
				gentest.Ctx(m)
				m.Param("v", valueRef())
				gentest.Err(m)
			})
			i.Method("Get", func(m *storefixture.MethodBuilder) {
				gentest.Ctx(m)
				m.Return(valueRef())
				gentest.Err(m)
			})
		}).
		Build()
	stampShape(s, "Put", "writer", "", "example.com/cc.Value")
	stampShape(s, "Get", "aggregator", "", "example.com/cc.Value")
	for _, iface := range s.Nodes().Interfaces().Items() {
		for _, m := range iface.Methods {
			if m.Name != "Put" {
				continue
			}
			bag := m.EnsureMeta()
			shape.MetaContracts.Set(bag, []string{"cas"}, "test")
			shape.ContractRoleKey("cas").Set(bag, "writer", "test")
			shape.ContractParamKey("cas", "version").Set(bag, member, "test")
			shape.ContractParamKey("cas", "mismatch").
				Set(bag, "example.com/cc.ErrMismatch", "test")
		}
	}
	return s
}

// keyedStore is the keyed-put fixture: a reader, a composite writer, and a
// delete-stamped plain writer, with the delete law's stamps carried.
func keyedStore(t *testing.T, sentinel string) *sdk.Store {
	t.Helper()
	return keyedStoreWith(t, sentinel, nil)
}

// keyedStoreWith is [keyedStore] plus extra methods.
func keyedStoreWith(t *testing.T, sentinel string, extra func(i *storefixture.InterfaceBuilder)) *sdk.Store {
	t.Helper()
	s := storefixture.New().
		Package("kv", "example.com/kv").
		Interface("Mixed", func(i *storefixture.InterfaceBuilder) {
			i.Pos(gentest.At())
			i.Directive(storefixture.Directive("suite"))
			i.Directive(storefixture.Directive("model"))
			i.Method("Get", func(m *storefixture.MethodBuilder) {
				gentest.Ctx(m)
				m.Param("key", storefixture.Named("string"))
				m.Return(storefixture.Named("string"))
				gentest.Err(m)
			})
			i.Method("Put", func(m *storefixture.MethodBuilder) {
				gentest.Ctx(m)
				m.Param("key", storefixture.Named("string"))
				m.Param("value", storefixture.Named("string"))
				gentest.Err(m)
			})
			i.Method("Del", func(m *storefixture.MethodBuilder) {
				gentest.Ctx(m)
				m.Param("key", storefixture.Named("string"))
				gentest.Err(m)
			})
			if extra != nil {
				extra(i)
			}
		}).
		Build()
	stampShape(s, "Get", "reader", "string", "string")
	stampShape(s, "Put", "compositewriter", "string", "string")
	stampShape(s, "Del", "writer", "string", "")
	for _, iface := range s.Nodes().Interfaces().Items() {
		for _, m := range iface.Methods {
			if m.Name != "Del" {
				continue
			}
			bag := m.EnsureMeta()
			shape.MetaMixins.Set(bag, []string{"deleteremoves"}, "test")
			shape.MixinParamKey("deleteremoves", "read").
				Set(bag, "example.com/kv.Get", "test")
			shape.MixinParamKey("deleteremoves", "sentinel").
				Set(bag, sentinel, "test")
		}
	}
	return s
}

// kvStore declares a reader over readV and a writer taking writeV, plus the
// Doc struct where either names it — the smallest interface the reference
// derivation walks all the way.
func kvStore(t *testing.T, readV, writeV string, fields ...field) *sdk.Store {
	t.Helper()
	return kvStoreWith(t, readV, writeV, nil, fields...)
}

// kvStoreWith is [kvStore] with directive options, for the supplied-reference
// arms.
func kvStoreWith(
	t *testing.T,
	readV, writeV string,
	opts []storefixture.DirectiveOption,
	fields ...field,
) *sdk.Store {
	t.Helper()
	f := storefixture.New().Package("kv", "example.com/kv")
	if len(fields) > 0 {
		f = f.Struct("Doc", func(b *storefixture.StructBuilder) {
			b.Pos(gentest.At())
			for _, fl := range fields {
				b.Field(fl.name, typeOf(fl.typ), nil)
			}
		})
	}
	s := f.Interface("Store", func(i *storefixture.InterfaceBuilder) {
		i.Pos(gentest.At())
		i.Directive(storefixture.Directive("suite"))
		i.Directive(storefixture.Directive("model", opts...))
		i.Method("Get", func(m *storefixture.MethodBuilder) {
			gentest.Ctx(m)
			m.Param("key", storefixture.Named("string"))
			m.Return(typeOf(readV))
			gentest.Err(m)
		})
		i.Method("Put", func(m *storefixture.MethodBuilder) {
			gentest.Ctx(m)
			m.Param("v", typeOf(writeV))
			gentest.Err(m)
		})
	}).Build()
	stampShape(s, "Get", "reader", "string", readV)
	stampShape(s, "Put", "writer", "", writeV)
	return s
}

// typeOf spells a fixture type from its stamp form.
func typeOf(q string) *sdk.TypeRef {
	if idx := strings.LastIndexByte(q, '.'); idx >= 0 {
		return storefixture.PkgNamed(q[:idx], q[idx+1:])
	}
	return storefixture.Named(q)
}

// bindingsOf runs the harness generator over the store, then derives one
// interface's model tier from what it queued.
//
// Called rather than read out of the emit graph, because this tier
// queues nothing: it contributes into the harness generator's output and
// emits no file of its own, and a queued emit value would render into
// one. So the derivation is reached the way the plugin reaches it.
func bindingsOf(t *testing.T, s *sdk.Store) *model.Bindings {
	t.Helper()
	plugintest.Generate(t, suite.New(), s)

	ctx := &sdk.GeneratorContext{Store: s, Reader: store.NewReader(s), Diag: diag.New()}
	c := sdk.NewProvenance(model.Name)
	harnesses := sdk.PendingByOrigin[*suite.Contract](s.Emit())

	for _, iface := range ctx.Reader.Interfaces().Slice() {
		if !iface.HasPositiveDirective(model.DirectiveName) {
			continue
		}
		harness, hosted := harnesses[sdk.Node(iface)]
		if !hosted {
			continue
		}
		if b, ok := model.BindingsFor(ctx, c, iface, harness); ok {
			return b
		}
	}
	t.Fatal("no interface in the store derived a model tier")
	return nil
}

// ifaceIn returns the store's first model-annotated interface, for a test
// that needs the declaration as well as the derivation.
func ifaceIn(t *testing.T, s *sdk.Store) *sdk.Interface {
	t.Helper()
	for _, iface := range s.Nodes().Interfaces().Items() {
		if iface.HasPositiveDirective(model.DirectiveName) {
			return iface
		}
	}
	t.Fatal("no interface in the store carries the model directive")
	return nil
}

// generateBoth runs the two generators in bucket order, the way the pipeline
// does: model reads the projection suite queues.
func generateBoth(t *testing.T, s *sdk.Store) *diag.Sink {
	t.Helper()
	plugintest.Generate(t, suite.New(), s)
	return plugintest.Generate(t, model.New(), s)
}

// mixed is the corpus fixture in store form, stamped the way the annotator
// stamps it: a writer carrying the validates mixin, the validator it names,
// and a reader.
// genericFixture is a one-parameter generic store, its model directive
// carrying whatever the case supplies — the witness key or nothing.
func genericFixture(t *testing.T, opts ...storefixture.DirectiveOption) *sdk.Store {
	t.Helper()
	s := storefixture.New().
		Package("gen", "example.com/gen").
		Interface("Store", func(i *storefixture.InterfaceBuilder) {
			i.Pos(gentest.AtFile("gen/iface.go"))
			i.Directive(storefixture.Directive("suite"))
			i.Directive(storefixture.Directive("model", opts...))
			i.TypeParam("V", nil)
			i.Method("Get", func(m *storefixture.MethodBuilder) {
				gentest.Ctx(m)
				m.Param("key", storefixture.Named("string"))
				m.Return(storefixture.TypeParamRef("V"))
				gentest.Err(m)
			})
			i.Method("Put", func(m *storefixture.MethodBuilder) {
				gentest.Ctx(m)
				m.Param("key", storefixture.Named("string"))
				m.Param("v", storefixture.TypeParamRef("V"))
				gentest.Err(m)
			})
		}).
		Build()

	// Stamped the way the annotator stamps a generic pair: the parameter's
	// bare name is the value spelling, which is exactly what the witness
	// substitution rewrites.
	stampShape(s, "Get", "reader", "string", "V")
	stampShape(s, "Put", "compositewriter", "string", "V")
	return s
}

func mixed(t *testing.T, opts ...storefixture.DirectiveOption) *sdk.Store {
	t.Helper()
	return mixedWith(t, nil, opts...)
}

// mixedWith is [mixed] plus extra methods, for the fixtures that probe what an
// unmappable method does to an otherwise ordinary interface.
func mixedWith(
	t *testing.T,
	extra func(i *storefixture.InterfaceBuilder),
	opts ...storefixture.DirectiveOption,
) *sdk.Store {
	t.Helper()
	s := storefixture.New().
		Package("validates", "example.com/validates").
		Struct("Payload", func(b *storefixture.StructBuilder) {
			b.Pos(gentest.AtFile("validates/iface.go"))
			b.Field("Key", storefixture.Named("string"), nil)
			b.Field("Body", storefixture.Named("string"), nil)
		}).
		Interface("Mixed", func(i *storefixture.InterfaceBuilder) {
			i.Pos(gentest.AtFile("validates/iface.go"))
			i.Directive(storefixture.Directive("suite"))
			i.Directive(storefixture.Directive("model", opts...))
			i.Method("Store", func(m *storefixture.MethodBuilder) {
				gentest.Ctx(m)
				m.Param("v", storefixture.PkgNamed("example.com/validates", "Payload"))
				gentest.Err(m)
			})
			i.Method("Validate", func(m *storefixture.MethodBuilder) {
				m.Param("v", storefixture.PkgNamed("example.com/validates", "Payload"))
				gentest.Err(m)
			})
			i.Method("Read", func(m *storefixture.MethodBuilder) {
				gentest.Ctx(m)
				m.Param("key", storefixture.Named("string"))
				m.Return(storefixture.PkgNamed("example.com/validates", "Payload"))
				gentest.Err(m)
			})
			if extra != nil {
				extra(i)
			}
		}).
		Build()

	stampShape(s, "Store", "writer", "", "example.com/validates.Payload")
	stampShape(s, "Read", "reader", "string", "example.com/validates.Payload")
	for _, iface := range s.Nodes().Interfaces().Items() {
		for _, m := range iface.Methods {
			if m.Name != "Store" {
				continue
			}
			bag := m.EnsureMeta()
			shape.MetaMixins.Set(bag, []string{"validates"}, "test")
			shape.MixinParamKey("validates", "fn").
				Set(bag, "example.com/validates.Validate", "test")
		}
	}
	return s
}

// readerOnly declares one stamped reader — nothing for a map to model.
func readerOnly(t *testing.T) *sdk.Store {
	t.Helper()
	s := storefixture.New().
		Package("ro", "example.com/ro").
		Interface("Fetcher", func(i *storefixture.InterfaceBuilder) {
			i.Pos(gentest.AtFile("ro/iface.go"))
			i.Directive(storefixture.Directive("suite"))
			i.Directive(storefixture.Directive("model"))
			i.Method("Fetch", func(m *storefixture.MethodBuilder) {
				gentest.Ctx(m)
				m.Param("key", storefixture.Named("string"))
				m.Return(storefixture.Named("string"))
				gentest.Err(m)
			})
		}).
		Build()
	stampShape(s, "Fetch", "reader", "string", "string")
	return s
}

// stampShape sets what the annotator would have written for one method.
func stampShape(s *sdk.Store, method, shapeName, keyType, valueType string) {
	for _, iface := range s.Nodes().Interfaces().Items() {
		for _, m := range iface.Methods {
			if m.Name != method {
				continue
			}
			bag := m.EnsureMeta()
			shape.MetaShape.Set(bag, shapeName, "test")
			if keyType != "" {
				shape.MetaKeyType.Set(bag, keyType, "test")
			}
			if valueType != "" {
				shape.MetaValueType.Set(bag, valueType, "test")
			}
		}
	}
}

// drainStore is the writer-plus-collector fixture: Add and Items, the
// noduplicates claim on the collector, and a value type with or without the
// conventional identity field.
func drainStore(t *testing.T, keyedValue bool) *sdk.Store {
	t.Helper()
	// The decoy is what makes the struct search walk past a non-match: a
	// store with exactly one struct never exercises the mismatch arm.
	f := storefixture.New().Package("bag", "example.com/bag").
		Struct("Decoy", func(b *storefixture.StructBuilder) {
			b.Pos(gentest.AtFile("bag/iface.go"))
			b.Field("N", storefixture.Named("int"), nil)
		})
	valueRef := storefixture.Named("string")
	valueQ := "string"
	if keyedValue {
		// The two Decoy fields make the wide-draw walk recurse into a nested
		// struct and revisit it — the diamond that exercises the seen set.
		f = f.Struct("Value", func(b *storefixture.StructBuilder) {
			b.Pos(gentest.AtFile("bag/iface.go"))
			b.Field("Key", storefixture.Named("string"), nil)
			b.Field("Body", storefixture.Named("string"), nil)
			b.Field("Meta", storefixture.PkgNamed("example.com/bag", "Decoy"), nil)
			b.Field("More", storefixture.PkgNamed("example.com/bag", "Decoy"), nil)
		})
		valueRef = storefixture.PkgNamed("example.com/bag", "Value")
		valueQ = "example.com/bag.Value"
	}
	s := f.Interface("Mixed", func(i *storefixture.InterfaceBuilder) {
		i.Pos(gentest.AtFile("bag/iface.go"))
		i.Directive(storefixture.Directive("suite"))
		i.Directive(storefixture.Directive("model"))
		i.Method("Add", func(m *storefixture.MethodBuilder) {
			gentest.Ctx(m)
			m.Param("v", valueRef)
			gentest.Err(m)
		})
		i.Method("Items", func(m *storefixture.MethodBuilder) {
			gentest.Ctx(m)
			m.Return(storefixture.Slice(valueRef))
			gentest.Err(m)
		})
	}).Build()
	stampShape(s, "Add", "writer", "", valueQ)
	stampShape(s, "Items", "aggregator", "", "")
	// The claim rides on both halves: the drain carries the law, and the
	// writer exercises the second of the two mixin scans the dedupe
	// refinement makes.
	for _, iface := range s.Nodes().Interfaces().Items() {
		for _, m := range iface.Methods {
			if m.Name == "Add" || m.Name == "Items" {
				shape.MetaMixins.Set(m.EnsureMeta(), []string{"noduplicates"}, "test")
			}
		}
	}
	return s
}

// drawersOnly builds one writer, one answering writer and one mutator at
// three different value types, with no reader.
//
// The absence of a reader is deliberate twice over: it drops the reference to
// the twin, which is the only configuration where an answering writer or a
// mutator survives to the pool guard rather than being held inert by an
// oracle that does not model its shape; and it leaves feederOf on its
// fallback, so the first declared writer types the pool.
func drawersOnly(t *testing.T) *sdk.Store {
	t.Helper()
	s := storefixture.New().
		Package("drawers", "example.com/drawers").
		Struct("Payload", func(b *storefixture.StructBuilder) {
			b.Pos(gentest.AtFile("drawers/iface.go"))
			b.Field("Key", storefixture.Named("string"), nil)
			b.Field("Body", storefixture.Named("string"), nil)
		}).
		Interface("Mixed", func(i *storefixture.InterfaceBuilder) {
			i.Pos(gentest.AtFile("drawers/iface.go"))
			i.Directive(storefixture.Directive("suite"))
			i.Directive(storefixture.Directive("model"))
			i.Method("Store", func(m *storefixture.MethodBuilder) {
				gentest.Ctx(m)
				m.Param("v", storefixture.PkgNamed("example.com/drawers", "Payload"))
				gentest.Err(m)
			})
			i.Method("Note", func(m *storefixture.MethodBuilder) {
				gentest.Ctx(m)
				m.Param("n", storefixture.Named("string"))
				m.Return(storefixture.Named("string"))
				gentest.Err(m)
			})
			i.Method("Bump", func(m *storefixture.MethodBuilder) {
				gentest.Ctx(m)
				m.Param("n", storefixture.Named("int"))
			})
		}).
		Build()
	stampShape(s, "Store", "writer", "", "example.com/drawers.Payload")
	stampShape(s, "Note", "answeringwriter", "", "string")
	stampShape(s, "Bump", "mutator", "", "int")
	return s
}

// compositePoolWith builds the shape that separates the two candidate
// baselines: a composite writer, no reader, and two plain writers at
// different value types.
//
// No reader is the point. feederOf picks the writer matching the reader's
// value type where one exists, so a reader would make the plain writer agree
// with the composite by construction and the two baselines would coincide —
// the bug would be invisible. Without one it falls to the first writer, which
// is Stash, and the two candidates finally disagree.
func compositePoolWith(t *testing.T) *sdk.Store {
	t.Helper()
	s := storefixture.New().
		Package("two", "example.com/two").
		Struct("Body", func(b *storefixture.StructBuilder) {
			b.Pos(gentest.AtFile("two/iface.go"))
			b.Field("Key", storefixture.Named("string"), nil)
			b.Field("Text", storefixture.Named("string"), nil)
		}).
		Struct("Tag", func(b *storefixture.StructBuilder) {
			b.Pos(gentest.AtFile("two/iface.go"))
			b.Field("Name", storefixture.Named("string"), nil)
		}).
		Interface("Mixed", func(i *storefixture.InterfaceBuilder) {
			i.Pos(gentest.AtFile("two/iface.go"))
			i.Directive(storefixture.Directive("suite"))
			i.Directive(storefixture.Directive("model"))
			// Declared first, so feederOf's fallback lands on it.
			i.Method("Stash", func(m *storefixture.MethodBuilder) {
				gentest.Ctx(m)
				m.Param("t", storefixture.PkgNamed("example.com/two", "Tag"))
				gentest.Err(m)
			})
			i.Method("Put", func(m *storefixture.MethodBuilder) {
				gentest.Ctx(m)
				m.Param("key", storefixture.Named("string"))
				m.Param("v", storefixture.PkgNamed("example.com/two", "Body"))
				gentest.Err(m)
			})
			i.Method("Save", func(m *storefixture.MethodBuilder) {
				gentest.Ctx(m)
				m.Param("v", storefixture.PkgNamed("example.com/two", "Body"))
				gentest.Err(m)
			})
		}).
		Build()
	stampShape(s, "Stash", "writer", "", "example.com/two.Tag")
	stampShape(s, "Put", "compositewriter", "string", "example.com/two.Body")
	stampShape(s, "Save", "writer", "", "example.com/two.Body")
	return s
}

// leasedWith builds the lease-contract fixture: a stamped carrier, its
// release partner, and whatever extra methods a probe needs.
func leasedWith(
	t *testing.T,
	idempotent bool,
	extra func(i *storefixture.InterfaceBuilder),
) *sdk.Store {
	t.Helper()
	s := storefixture.New().
		Package("ls", "example.com/ls").
		Interface("Locker", func(i *storefixture.InterfaceBuilder) {
			i.Pos(gentest.AtFile("ls/iface.go"))
			i.Directive(storefixture.Directive("suite"))
			i.Directive(storefixture.Directive("model"))
			for _, name := range []string{"Acquire", "Release"} {
				i.Method(name, func(m *storefixture.MethodBuilder) {
					gentest.Ctx(m)
					m.Param("key", storefixture.Named("string"))
					gentest.Err(m)
				})
			}
			if extra != nil {
				extra(i)
			}
		}).
		Build()
	stampShape(s, "Acquire", "writer", "", "string")
	stampShape(s, "Release", "writer", "", "string")
	for _, iface := range s.Nodes().Interfaces().Items() {
		for _, m := range iface.Methods {
			if m.Name != "Acquire" {
				continue
			}
			bag := m.EnsureMeta()
			shape.MetaContracts.Set(bag, []string{"lease"}, "test")
			shape.ContractRoleKey("lease").Set(bag, "acquire", "test")
			shape.ContractPartnerKey("lease", "release").
				Set(bag, "example.com/ls.Release", "test")
			if idempotent {
				shape.MetaMixins.Set(bag, []string{"idempotent"}, "test")
			}
		}
	}
	return s
}
