// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package projection_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/internal/projection"
)

// memberCase is one stamp and the member a transform derives from it;
// ok=false pins a refusal.
type memberCase struct {
	name   string
	got    projection.Expr
	ok     bool
	want   projection.Expr
	refuse bool
}

func (c memberCase) Name() string { return c.name }

//nolint:thelper // the case body is the test, not a helper; see core/lawid
func TestPoolMemberTransforms(t *testing.T) {
	t.Parallel()

	distinct := func(stamp string) (projection.Expr, bool) {
		return projection.DistinctMember(projection.Expr(stamp))
	}
	hostile := func(stamp, role string) (projection.Expr, bool) {
		return projection.HostileMember(projection.Expr(stamp), role)
	}

	cases := []memberCase{}
	add := func(name string, got projection.Expr, ok bool, want projection.Expr, refuse bool) {
		cases = append(cases, memberCase{name, got, ok, want, refuse})
	}

	// The corpus pins: kv's derived pools, member for member.
	e, ok := distinct(`"test-key"`)
	add("distinct swaps the textual payload", e, ok, `"other-key"`, false)
	e, ok = distinct(`Value{Body: "test-body"}`)
	add("distinct reaches inside a composite", e, ok, `Value{Body: "other-body"}`, false)
	e, ok = distinct(`"localhost"`)
	add("no swap point refuses", e, ok, "", true)

	e, ok = hostile(`"test-key"`, "key")
	add("hostile string carries NUL and invalid UTF-8, role-suffixed", e, ok, `"\x00hostile\xffkey"`, false)
	e, ok = hostile(`Value{Body: "test-body"}`, "payload")
	add("hostile composite empties its payload", e, ok, `Value{Body: ""}`, false)
	e, ok = hostile(`42`, "key")
	add("no textual form refuses", e, ok, "", true)
	e, ok = hostile(`Value{Body: "with \" escape"}`, "payload")
	add("an escaped literal refuses rather than mangling", e, ok, "", true)

	testkit.TableTest(t, cases, func(t *testing.T, tc memberCase) {
		if tc.refuse {
			testkit.False(t, tc.ok, "the transform refuses what it cannot honestly derive")
			return
		}
		testkit.True(t, tc.ok, "the corpus member derives")
		testkit.Equal(t, tc.got, tc.want, "the member is the corpus manifests' spelling")
	})
}

func TestPoolFieldNamePolicy(t *testing.T) {
	t.Parallel()

	testkit.Equal(t, projection.PoolFieldName("Key"), "KeyPool", "the stamped field opens its pool")
	testkit.Equal(t, projection.PoolFieldName("value"), "ValuePool", "exported through the shared convention")
}
