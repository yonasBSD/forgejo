// Copyright Earl Warren <contact@earl-warren.org>
// Copyright Loïc Dachary <loic@dachary.org>
// SPDX-License-Identifier: MIT

package tests

import (
	"testing"

	driver_options "code.gitea.io/gitea/services/f3/driver/options"

	"lab.forgefriends.org/friendlyforgeformat/gof3/options"
	f3_tree "lab.forgefriends.org/friendlyforgeformat/gof3/tree/f3"
	"lab.forgefriends.org/friendlyforgeformat/gof3/tree/generic"
	forge_test "lab.forgefriends.org/friendlyforgeformat/gof3/tree/tests/f3/forge"
)

type forgeTest struct {
	forge_test.Base
}

func (o *forgeTest) NewOptions(t *testing.T) options.Interface {
	return newTestOptions(t)
}

func (o *forgeTest) GetExceptions() []generic.Kind {
	return []generic.Kind{
		f3_tree.KindAssets,
		f3_tree.KindPullRequests,
		f3_tree.KindReactions,
		f3_tree.KindReviews,
		f3_tree.KindReviewComments,
		f3_tree.KindTopics,
	}
}

func (o *forgeTest) GetNonTestUsers() []string {
	return []string{
		"user1",
	}
}

func newForgeTest() forge_test.Interface {
	t := &forgeTest{}
	t.SetName(driver_options.Name)
	return t
}
