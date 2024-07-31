// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package git_test

import (
	"code.gitea.io/gitea/modules/git"
)

const (
	testReposDir = "tests/repos/"
)

// openRepositoryWithDefaultContext opens the repository at the given path with DefaultContext.
func openRepositoryWithDefaultContext(repoPath string) (*git.Repository, error) {
	return git.OpenRepository(git.DefaultContext, repoPath)
}
