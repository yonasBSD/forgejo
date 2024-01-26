// Copyright Earl Warren <contact@earl-warren.org>
// Copyright Loïc Dachary <loic@dachary.org>
// SPDX-License-Identifier: MIT

package driver

import (
	forgejo_options "code.gitea.io/gitea/services/f3/driver/options"

	"lab.forgefriends.org/friendlyforgeformat/gof3/tree/generic"
)

type treeDriver struct {
	generic.NullTreeDriver

	options *forgejo_options.Options
}

func (o *treeDriver) Init() {
	o.NullTreeDriver.Init()
}

func newTreeDriver(tree generic.TreeInterface, anyOptions any) generic.TreeDriverInterface {
	driver := &treeDriver{
		options: anyOptions.(*forgejo_options.Options),
	}
	driver.Init()
	return driver
}
