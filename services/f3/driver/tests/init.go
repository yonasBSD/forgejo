// Copyright Earl Warren <contact@earl-warren.org>
// Copyright Loïc Dachary <loic@dachary.org>
// SPDX-License-Identifier: MIT

package tests

import (
	driver_options "code.gitea.io/gitea/services/f3/driver/options"

	tests_forge "lab.forgefriends.org/friendlyforgeformat/gof3/tree/tests/f3/forge"
)

func init() {
	tests_forge.RegisterFactory(driver_options.Name, newForgeTest)
}