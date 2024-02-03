// Copyright Earl Warren <contact@earl-warren.org>
// Copyright Loïc Dachary <loic@dachary.org>
// SPDX-License-Identifier: MIT

package driver

import (
	"context"
	"fmt"

	forgejo_options "code.gitea.io/gitea/services/f3/driver/options"

	f3_tree "lab.forgefriends.org/friendlyforgeformat/gof3/tree/f3"
	"lab.forgefriends.org/friendlyforgeformat/gof3/tree/generic"
)

type treeDriver struct {
	generic.NullTreeDriver

	options *forgejo_options.Options
}

func (o *treeDriver) Init() {
	o.NullTreeDriver.Init()
}

func (o *treeDriver) Factory(ctx context.Context, kind generic.Kind) generic.NodeDriverInterface {
	switch kind {
	case f3_tree.KindUsers:
		return newUsers()
	case f3_tree.KindOrganizations:
		return newOrganizations()
	case f3_tree.KindTopics:
		return newTopics()
	case f3_tree.KindForge:
		return newForge()
	case generic.KindRoot:
		return newRoot(o.GetTree().(f3_tree.TreeInterface).NewFormat(kind))
	default:
		panic(fmt.Errorf("unexpected kind %s", kind))
	}
}

func newTreeDriver(tree generic.TreeInterface, anyOptions any) generic.TreeDriverInterface {
	driver := &treeDriver{
		options: anyOptions.(*forgejo_options.Options),
	}
	driver.Init()
	return driver
}
