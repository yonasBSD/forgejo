// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package convert

import (
	"context"

	issues_model "forgejo.org/models/issues"
	project_model "forgejo.org/models/project"
	"forgejo.org/modules/log"
	api "forgejo.org/modules/structs"
)

// ToAPIProject converts a project to its API representation for embedding in issue/PR responses
func ToAPIProject(ctx context.Context, issue *issues_model.Issue, p *project_model.Project) *api.Project {
	result := &api.Project{
		ID:    p.ID,
		Title: p.Title,
		State: p.State(),
	}

	if columnID := issue.ProjectColumnID(ctx); columnID > 0 {
		if column, err := project_model.GetColumn(ctx, columnID); err == nil {
			result.Column = column.Title
		} else {
			log.Error("GetColumn[%d]: %v", columnID, err)
		}
	}

	return result
}
