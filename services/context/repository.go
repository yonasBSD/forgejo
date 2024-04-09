// Copyright 2023 The forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package context

import (
	"net/http"

	repo_model "code.gitea.io/gitea/models/repo"
)

// RepositoryIDAssignmentAPI returns a middleware to handle context-repo assignment for api routes
func RepositoryIDAssignmentAPI() func(ctx *APIContext) {
	return func(ctx *APIContext) {
		// TODO: enough validation for security?
		repositoryID := ctx.ParamsInt64(":repository-id")

		var err error
		repository := new(Repository)
		repository.Repository, err = repo_model.GetRepositoryByID(ctx, repositoryID)
		if err != nil {
			ctx.Error(http.StatusInternalServerError, "GetRepositoryByID", err)
		}
		ctx.Repo = repository
	}
}
