// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package misc

import (
	"net/http"
	"strings"

	project_model "forgejo.org/models/project"
	api "forgejo.org/modules/structs"
	"forgejo.org/services/context"
)

// ListProjectTemplates returns available project template types
func ListProjectTemplates(ctx *context.APIContext) {
	// swagger:operation GET /project/templates miscellaneous listProjectTemplates
	// ---
	// summary: Returns a list of all project template types
	// produces:
	// - application/json
	// responses:
	//   "200":
	//     "$ref": "#/responses/ProjectTemplateList"
	configs := project_model.GetTemplateConfigs()
	result := make([]api.ProjectTemplateConfig, len(configs))
	for i, cfg := range configs {
		// Derive a stable key from the translation key (last segment)
		parts := strings.Split(cfg.Translation, ".")
		key := parts[len(parts)-1]

		result[i] = api.ProjectTemplateConfig{
			Type:        uint8(cfg.TemplateType),
			Key:         key,
			Description: ctx.Locale.TrString(cfg.Translation),
		}
	}
	ctx.JSON(http.StatusOK, result)
}
