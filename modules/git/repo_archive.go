// Copyright 2015 The Gogs Authors. All rights reserved.
// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package git

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"code.gitea.io/gitea/modules/setting"
)

// ArchiveType archive types
type ArchiveType int

const (
	// ZIP zip archive type
	ZIP ArchiveType = iota + 1
	// TARGZ tar gz archive type
	TARGZ
	// BUNDLE bundle archive type
	BUNDLE
)

// String converts an ArchiveType to string
func (a ArchiveType) String() string {
	switch a {
	case ZIP:
		return "zip"
	case TARGZ:
		return "tar.gz"
	case BUNDLE:
		return "bundle"
	}
	return "unknown"
}

func ToArchiveType(s string) ArchiveType {
	switch s {
	case "zip":
		return ZIP
	case "tar.gz":
		return TARGZ
	case "bundle":
		return BUNDLE
	}
	return 0
}

// CreateArchive create archive content to the target path
func (repo *Repository) CreateArchive(ctx context.Context, format ArchiveType, target io.Writer, usePrefix bool, commitID, lfsURL string) error {
	if format.String() == "unknown" {
		return fmt.Errorf("unknown format: %v", format)
	}

	cmd := NewCommand(ctx)

	// Add arguments to make git-archive LFS aware and set the LFS server URL.
	if setting.LFS.StartServer {
		cmd.AddArguments(ToTrustedCmdArgs([]string{
			"-c", "filter.lfs.process=git-lfs filter-process",
			"-c", "filter.lfs.required=true",
			"-c", "filter.lfs.smudge=git-lfs smudge -- %f",
			"-c", "filter.lfs.clean=git-lfs clean -- %f",
			"-c", "remote.origin.lfsurl=" + lfsURL,
			"-c", "http." + lfsURL + ".extraHeader=Authorization: Bearer " + setting.InternalToken,
		})...,
		)
	}
	cmd.AddArguments("archive")
	if usePrefix {
		cmd.AddOptionFormat("--prefix=%s", filepath.Base(strings.TrimSuffix(repo.Path, ".git"))+"/")
	}
	cmd.AddOptionFormat("--format=%s", format.String())
	cmd.AddDynamicArguments(commitID)

	// Avoid LFS hooks getting installed because of /etc/gitconfig, which can break pull requests.
	env := append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1")

	var stderr strings.Builder
	err := cmd.Run(&RunOpts{
		Dir:    repo.Path,
		Stdout: target,
		Stderr: &stderr,
		Env:    env,
	})
	if err != nil {
		return ConcatenateError(err, stderr.String())
	}
	return nil
}
