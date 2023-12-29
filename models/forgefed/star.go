// Copyright 2023 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	ap "github.com/go-ap/activitypub"
	"github.com/valyala/fastjson"
)

const (
	StarType ap.ActivityVocabularyType = "Star"
)

// Star activity data type
// swagger:model
type Star struct {
	// swagger:ignore
	ap.Activity
	// Source identifies the system which generated this activity. Exactly one value has to be specified.
	Source SourceType `jsonld:"source,omitempty"`
}

// StarNew initializes a Star type activity
// ToDo: May be used later in creating signed activities
func StarNew(id, ob ap.ID) *Star {
	a := ap.ActivityNew(id, StarType, ob)
	o := Star{Activity: *a, Source: ForgejoSourceType}
	return &o
}

func (s Star) MarshalJSON() ([]byte, error) {
	b := make([]byte, 0)
	ap.JSONWrite(&b, '{')

	ap.JSONWriteStringProp(&b, "source", string(s.Source))
	if !ap.JSONWriteActivityValue(&b, s.Activity) {
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
