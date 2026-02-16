// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package structs

// Project represents a project
// swagger:model
type Project struct {
	ID     int64     `json:"id"`
	Title  string    `json:"title"`
	State  StateType `json:"state"`
	Column string    `json:"column"`
}
