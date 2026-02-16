// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package structs

// ProjectTemplateConfig describes a project template type
type ProjectTemplateConfig struct {
	Type        uint8  `json:"type"`
	Key         string `json:"key"`
	Description string `json:"description"`
}
