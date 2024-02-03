// Copyright Earl Warren <contact@earl-warren.org>
// Copyright Loïc Dachary <loic@dachary.org>
// SPDX-License-Identifier: MIT

package driver

import (
	"context"

	"lab.forgefriends.org/friendlyforgeformat/gof3/tree/generic"
)

type common struct {
	generic.NullDriver
}

func (o *common) GetHelper() any {
	panic("not implemented")
}

func (o *common) ListPage(ctx context.Context, page int) generic.ChildrenSlice {
	return generic.NewChildrenSlice(0)
}

func (o *common) getKind() generic.Kind {
	return o.GetNode().GetKind()
}

func (o *common) IsNull() bool { return false }
