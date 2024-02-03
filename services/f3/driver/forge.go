// Copyright Earl Warren <contact@earl-warren.org>
// Copyright Loïc Dachary <loic@dachary.org>
// SPDX-License-Identifier: MIT

package driver

import (
	"context"

	"lab.forgefriends.org/friendlyforgeformat/gof3/f3"
	"lab.forgefriends.org/friendlyforgeformat/gof3/tree/generic"
)

type forge struct {
	generic.NullDriver
}

func newForge() generic.NodeDriverInterface {
	return &forge{}
}

func (o *forge) Equals(context.Context, generic.NodeInterface) bool { return true }
func (o *forge) Get(context.Context) bool                           { return true }
func (o *forge) Put(context.Context) generic.NodeID                 { return generic.NodeID("forge") }
func (o *forge) Patch(context.Context)                              {}
func (o *forge) Delete(context.Context)                             {}
func (o *forge) NewFormat() f3.Interface                            { return &f3.Forge{} }
func (o *forge) FromFormat(f3.Interface)                            {}

func (o *forge) ToFormat() f3.Interface {
	return &f3.Forge{
		Common: f3.NewCommon("forge"),
		URL:    o.String(),
	}
}
