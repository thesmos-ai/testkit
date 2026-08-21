// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package projection_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/core/lawid"
	"go.thesmos.sh/testkit/generator/internal/projection"
)

// bindCase is one lock-column rendering.
type bindCase struct {
	name string
	bind projection.Bind
	want string
}

func (c bindCase) Name() string { return c.name }

//nolint:thelper // the case body is the test, not a helper; see core/lawid
func TestBindRendersTheLockColumnForm(t *testing.T) {
	t.Parallel()

	testkit.TableTest(t, []bindCase{
		{
			"a single-probe bind is the bare law ID",
			projection.Bind{Law: lawid.PoisonConsistent},
			lawid.PoisonConsistent,
		},
		{
			"a probe set renders space-separated in parens",
			projection.Bind{Law: lawid.LifecycleAfterClose, Probes: []string{"Put", "Get", "Len"}},
			lawid.LifecycleAfterClose + "(Put Get Len)",
		},
	}, func(t *testing.T, tc bindCase) {
		testkit.Equal(t, tc.bind.Render(), tc.want, "the lock-column format has this one home")
	})
}

func TestRenderBinds(t *testing.T) {
	t.Parallel()

	t.Run("no binds project to nil", func(t *testing.T) {
		t.Parallel()
		testkit.Assert(t, projection.RenderBinds(nil)).IsNil("an unbound check carries no allocation")
	})

	t.Run("binds project in declaration order", func(t *testing.T) {
		t.Parallel()
		got := projection.RenderBinds([]projection.Bind{
			{Law: lawid.LifecycleAfterClose, Probes: []string{"Put"}},
			{Law: lawid.PoisonConsistent},
		})
		testkit.Equal(t, got, []string{lawid.LifecycleAfterClose + "(Put)", lawid.PoisonConsistent},
			"the lock column preserves the plan's order")
	})
}
