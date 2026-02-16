// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"testing"

	project_model "forgejo.org/models/project"
	api "forgejo.org/modules/structs"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

func TestAPIListProjectTemplates(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	req := NewRequest(t, "GET", "/api/v1/project/templates")
	resp := MakeRequest(t, req, http.StatusOK)

	var templates []api.ProjectTemplateConfig
	DecodeJSON(t, resp, &templates)

	configs := project_model.GetTemplateConfigs()
	assert.Len(t, templates, len(configs))

	// Verify each template has expected type and key
	assert.EqualValues(t, 0, templates[0].Type)
	assert.Equal(t, "none", templates[0].Key)

	assert.EqualValues(t, 1, templates[1].Type)
	assert.Equal(t, "basic_kanban", templates[1].Key)

	assert.EqualValues(t, 2, templates[2].Type)
	assert.Equal(t, "bug_triage", templates[2].Key)

	// Descriptions should be non-empty (translated strings)
	for _, tmpl := range templates {
		assert.NotEmpty(t, tmpl.Description)
	}
}
