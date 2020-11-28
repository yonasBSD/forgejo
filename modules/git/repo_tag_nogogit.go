// Copyright 2015 The Gogs Authors. All rights reserved.
// Copyright 2019 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// +build nogogit

package git

// IsTagExist returns true if given tag exists in the repository.
func (repo *Repository) IsTagExist(name string) bool {
	return IsReferenceExist(repo.Path, TagPrefix+name)
}

// GetTags returns all tags of the repository.
func (repo *Repository) GetTags() ([]string, error) {
	return callShowRef(repo.Path, TagPrefix, "--tags")
}
