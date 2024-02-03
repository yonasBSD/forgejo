// Copyright Earl Warren <contact@earl-warren.org>
// Copyright Loïc Dachary <loic@dachary.org>
// SPDX-License-Identifier: MIT

package driver

import (
	driver_options "code.gitea.io/gitea/services/f3/driver/options"

	"lab.forgefriends.org/friendlyforgeformat/gof3/options"
	f3_tree "lab.forgefriends.org/friendlyforgeformat/gof3/tree/f3"
)

func init() {
	f3_tree.RegisterForgeFactory(driver_options.Name, newTreeDriver)
	options.RegisterFactory(driver_options.Name, newOptions)
}
