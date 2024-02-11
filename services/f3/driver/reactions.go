// Copyright Earl Warren <contact@earl-warren.org>
// Copyright Loïc Dachary <loic@dachary.org>
// SPDX-License-Identifier: MIT

package driver

import (
	"lab.forgefriends.org/friendlyforgeformat/gof3/tree/generic"
)

type reactions struct {
	container
}

func newReactions() generic.NodeDriverInterface {
	return &reactions{}
}
