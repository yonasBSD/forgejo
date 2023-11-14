// Copyright 2023 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	ap "github.com/go-ap/activitypub"
	"github.com/valyala/fastjson"
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
	Activity ap.Activity
	// Source identifies the system which generated this activity. Exactly one value has to be specified.
	Source SourceType `jsonld:"source,omitempty"`
}

// StarNew initializes a Star type activity
func StarNew(id ap.ID, ob ap.ID) *Star {
	a := ap.ActivityNew(id, StarType, ob)
	o := Star{Activity: *a, Source: ForgejoSourceType}
	return &o
}

// ToDo: should Star be *Star?
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

func JSONLoadStar(val *fastjson.Value, s *Star) error {
	if err := ap.OnActivity(&s.Activity, func(a *ap.Activity) error {
		return ap.JSONLoadActivity(val, a)
	}); err != nil {
		return err
	}

	s.Source = SourceType(ap.JSONGetString(val, "source"))
	return nil
}

func (s *Star) UnmarshalJSON(data []byte) error {
	p := fastjson.Parser{}
	val, err := p.ParseBytes(data)
	if err != nil {
		return err
	}
	return JSONLoadStar(val, s)
}
