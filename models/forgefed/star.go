// Copyright 2023 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	"code.gitea.io/gitea/modules/validation"
	ap "github.com/go-ap/activitypub"
)

// Star activity data type
// swagger:model
type Star struct {
	// swagger:ignore
	ap.Activity
}

func (a Star) MarshalJSON() ([]byte, error) {
	return a.Activity.MarshalJSON()
}

func (s *Star) UnmarshalJSON(data []byte) error {
	return s.UnmarshalJSON(data)
}

func (s Star) Validate() []string {
	var result []string
	result = append(result, validation.ValidateNotEmpty(string(s.Type), "type")...)

	return result
}
