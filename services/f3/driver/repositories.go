// Copyright Earl Warren <contact@earl-warren.org>
// Copyright Loïc Dachary <loic@dachary.org>
// SPDX-License-Identifier: MIT

package driver

import (
	"context"

	"lab.forgefriends.org/friendlyforgeformat/gof3/f3"
	f3_tree "lab.forgefriends.org/friendlyforgeformat/gof3/tree/f3"
	"lab.forgefriends.org/friendlyforgeformat/gof3/tree/generic"
)

type repositories struct {
	container
}

func (o *repositories) ListPage(ctx context.Context, page int) generic.ChildrenSlice {
	children := generic.NewChildrenSlice(0)
	if page > 1 {
		return children
	}

	names := []string{f3.RepositoryNameDefault}
	project := f3_tree.GetProject(o.GetNode()).ToFormat().(*f3.Project)
	if project.HasWiki {
		names = append(names, f3.RepositoryNameWiki)
	}

	return f3_tree.ConvertListed(ctx, o.GetNode(), f3_tree.ConvertToAny(names...)...)
}

func newRepositories() generic.NodeDriverInterface {
	return &repositories{}
}
