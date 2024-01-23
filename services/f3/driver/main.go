// Copyright Earl Warren <contact@earl-warren.org>
// Copyright Loïc Dachary <loic@dachary.org>
// SPDX-License-Identifier: MIT

package driver

import (
	"lab.forgefriends.org/friendlyforgeformat/gof3/options"
	f3_tree "lab.forgefriends.org/friendlyforgeformat/gof3/tree/f3"
)

func init() {
	f3_tree.RegisterForgeFactory(Name, newTreeDriver)
	options.RegisterFactory(Name, newOptions)
}
