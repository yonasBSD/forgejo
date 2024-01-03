// Copyright 2023 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	"code.gitea.io/gitea/modules/validation"
	ap "github.com/go-ap/activitypub"
)

// ForgeLike activity data type
// swagger:model
type ForgeLike struct {
	// swagger:ignore
	ap.Activity
}

func (s ForgeLike) MarshalJSON() ([]byte, error) {
	return s.Activity.MarshalJSON()
}

func (s *ForgeLike) UnmarshalJSON(data []byte) error {
	return s.Activity.UnmarshalJSON(data)
}

func (s ForgeLike) Validate() []string {
	var result []string
	result = append(result, validation.ValidateNotEmpty(string(s.Type), "type")...)
	result = append(result, validation.ValidateNotEmpty(s.Actor.GetID().String(), "actor")...)
	result = append(result, validation.ValidateNotEmpty(s.Object.GetID().String(), "object")...)
	result = append(result, validation.ValidateNotEmpty(s.StartTime.String(), "startTime")...)

	return result
}
