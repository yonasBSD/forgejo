// Copyright 2023 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	"code.gitea.io/gitea/modules/context"
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

// Star activity data type
// swagger:model
type Star struct {
	// swagger:ignore
	ap.Activity
	// Source identifies the system which generated this activity. Exactly one value has to be specified.
	Source SourceType `jsonld:"source,omitempty"`
}

// Infos needed to star a repo
type StarRepo struct {
	StargazerID int `json:"Stargazer"`
	RepoID      int `json:"RepoToStar"`
}

// StarNew initializes a Star type activity
// Guess: no return value needed, we may need to add the star to the context
func StarNew(id ap.ID, ob ap.ID) *Star {
	a := ap.ActivityNew(id, StarType, ob)
	o := Star{Activity: *a, Source: ForgejoSourceType}
	return &o
}

func AddStar(ctx *context.APIContext) {

}

func (a Star) MarshalJSON() ([]byte, error) {
	b := make([]byte, 0)
	ap.JSONWrite(&b, '{')

	ap.JSONWriteStringProp(&b, "source", string(a.Source))
	if !ap.JSONWriteActivityValue(&b, a.Activity) {
		return nil, nil
	}
	ap.JSONWrite(&b, '}')
	return b, nil
}
