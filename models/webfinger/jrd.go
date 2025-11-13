// Copyright 2025 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package webfinger

// JRD represents a response to a WebFinger request.
//
// https://datatracker.ietf.org/doc/html/draft-ietf-appsawg-webfinger-14#section-4.4
// swagger:model
type JRD struct {
	Subject    string           `json:"subject,omitempty"`
	Aliases    []string         `json:"aliases,omitempty"`
	Properties map[string]any   `json:"properties,omitempty"`
	Links      []*Link          `json:"links,omitempty"`
}

// Link represents a JRD link in a WebFinger response.
//
// https://datatracker.ietf.org/doc/html/draft-ietf-appsawg-webfinger-14#section-4.4
// swagger:model
type Link struct {
	Rel        string            `json:"rel,omitempty"`
	Type       string            `json:"type,omitempty"`
	Href       string            `json:"href,omitempty"`
	Titles     map[string]string `json:"titles,omitempty"`
	Properties map[string]any    `json:"properties,omitempty"`
}
