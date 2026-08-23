// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"fmt"

	"go.thesmos.sh/eidos/emit/builder"
	"go.thesmos.sh/eidos/sdk"
)

// preambleFor puts the guidance every suite in a package shares onto the
// package doc, once.
//
// It used to sit on each interface's own doc comment, which is where Go
// puts guidance and is right for one suite in a file. For three it is the
// same explanation three times with every name changed — a reader who has
// read it once is reading it again to find the two lines that differ.
//
// What stays on the interface is the part that IS about that interface:
// the entry points, spelled with its own names. What moves here is the
// part that would read identically whatever the interface was called.
//
// Once per package rather than per interface, because that is the scope
// the backend gives a package doc: it renders above `package <name>` in
// every file of the package, so contributing it twice would say it twice.
// The package NAME rather than its path, because that is what the
// backend matches a package doc against — and it is the one part of the
// routing readable before Layout runs, off the `pkg=` key the consumer
// wrote. An interface routed without one is left alone: the doc would
// have nowhere to attach, and guessing the name is how a comment comes
// to sit above the wrong file.
func outPackage(iface *sdk.Interface) string {
	dir := iface.Directive(sdk.OutDirective)
	if dir == nil {
		return ""
	}
	return dir.Value("pkg")
}

func preambleFor(ctx *sdk.GeneratorContext, pkg string) error {
	if pkg == "" {
		return nil
	}
	if _, already := ctx.Store.Emit().Packages().ByQName(pkg); already {
		return nil
	}
	node, err := builder.For(Name).
		Package(pkg, pkg).
		Docs(preambleLines...).
		Build()
	if err != nil {
		return fmt.Errorf("%s: build the package doc for %q: %w", Name, pkg, err)
	}
	if err := ctx.Store.Emit().Packages().Add(pkg, node); err != nil {
		return fmt.Errorf("%s: register the package doc for %q: %w", Name, pkg, err)
	}
	return nil
}

// preambleLines is the shared guidance, without `//`.
//
// Written once here rather than in a template, because it names no
// generated identifier: every sentence in it is true of any suite this
// generator emits, which is precisely why it belongs above the package
// clause rather than on one declaration in it.
//
//nolint:gochecknoglobals // prose, read-only after init.
var preambleLines = []string{
	"Conformance checks worked out from the interfaces this package doubles.",
	"",
	"One call runs every check for an interface against one implementation.",
	"Describe the implementation in a literal and hand it over — each",
	"interface's own Run function is documented beside it, with the names",
	"to use.",
	"",
	"Nothing else is required to start. The rest is there when you need it:",
	"a harness field to add only when a check fails asking for it, checks of",
	"your own that run beside the generated ones, a Prove entry that drives",
	"each of yours against the broken implementation it names, and a typed",
	"index for dropping a check by identity rather than by string.",
	"",
	"Nothing here is written by hand. Regenerate rather than edit: an edit",
	"survives until the next run and no longer.",
}
