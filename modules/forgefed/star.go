// Copyright 2023 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	ap "github.com/go-ap/activitypub"
)

type (
	SourceType string
)

type SourceTypes []SourceType

const (
	StarType ap.ActivityVocabularyType = "Star"
)

const (
	ForgejoSourceType SourceType = "frogejo"
)

var KnownSourceTypes = SourceTypes{
	ForgejoSourceType,
}

// Star activity for adding a star to an repository
// swagger:model
type Star struct {
	// swagger:ignore
	ap.Activity
	// Source identifies the system generated this Activity. Exact one value has to be specified.
	Source SourceType `jsonld:"source,omitempty"`
}

// RepositoryNew initializes a Repository type actor
func StarNew(id ap.ID, ob ap.ID) *Star {
	a := ap.ActivityNew(id, StarType, ob)
	// TODO: is this not handeld by ActivityNew??
	a.Type = StarType
	o := Star{Activity: *a}
	o.Source = ForgejoSourceType
	return &o
}
