// Copyright 2014 The Gogs Authors. All rights reserved.
// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package git

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ObjectType git object type
type ObjectType string

const (
	// ObjectCommit commit object type
	ObjectCommit ObjectType = "commit"
	// ObjectTree tree object type
	ObjectTree ObjectType = "tree"
	// ObjectBlob blob object type
	ObjectBlob ObjectType = "blob"
	// ObjectTag tag object type
	ObjectTag ObjectType = "tag"
	// ObjectBranch branch object type
	ObjectBranch ObjectType = "branch"
)

// Bytes returns the byte array for the Object Type
func (o ObjectType) Bytes() []byte {
	return []byte(o)
}

type EmptyReader struct{}

func (EmptyReader) Read(p []byte) (int, error) {
	return 0, io.EOF
}

func (repo *Repository) GetObjectFormat() (ObjectFormat, error) {
	if repo != nil && repo.objectFormat != nil {
		return repo.objectFormat, nil
	}

	str, err := repo.hashObject(EmptyReader{}, false)
	if err != nil {
		return nil, err
	}
	hash, err := NewIDFromString(str)
	if err != nil {
		return nil, err
	}

	repo.objectFormat = hash.Type()

	return repo.objectFormat, nil
}

// HashObject takes a reader and returns hash for that reader
func (repo *Repository) HashObject(reader io.Reader) (ObjectID, error) {
	idStr, err := repo.hashObject(reader, true)
	if err != nil {
		return nil, err
	}
	return NewIDFromString(idStr)
}

func (repo *Repository) hashObject(reader io.Reader, save bool) (string, error) {
	var cmd *Command
	if save {
		cmd = NewCommand(repo.Ctx, "hash-object", "-w", "--stdin")
	} else {
		cmd = NewCommand(repo.Ctx, "hash-object", "--stdin")
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	err := cmd.Run(&RunOpts{
		Dir:    repo.Path,
		Stdin:  reader,
		Stdout: stdout,
		Stderr: stderr,
	})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(stdout.String()), nil
}

// GetRefType gets the type of the ref based on the string
func (repo *Repository) GetRefType(ref string) ObjectType {
	if repo.IsTagExist(ref) {
		return ObjectTag
	} else if repo.IsBranchExist(ref) {
		return ObjectBranch
	} else if repo.IsCommitExist(ref) {
		return ObjectCommit
	} else if _, err := repo.GetBlob(ref); err == nil {
		return ObjectBlob
	}
	return ObjectType("invalid")
}

// Always returns the absolute path to the alternate, if one is set
func (repo *Repository) GetAlternatePaths() ([]string, error) {
	data, err := os.ReadFile(filepath.Join(repo.Path, "objects", "info", "alternates"))
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	lines := bytes.Split(data, []byte{'\n'})
	res := make([]string, 0, len(lines))

	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		path := strings.TrimSpace(string(line))
		if !filepath.IsAbs(path) {
			path = filepath.Join(repo.Path, "objects", path)
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			return nil, err
		}
		res = append(res, abs)
	}

	return res, nil
}

// Input paths need to be absolute
func (repo *Repository) SetAlternatePaths(paths []string, makeRelative bool) error {
	objectsDir := filepath.Join(repo.Path, "objects")
	lines := make([]string, 0, len(paths))

	for _, alt := range paths {
		if makeRelative {
			rel, err := filepath.Rel(objectsDir, alt)
			if err != nil {
				return err
			}
			lines = append(lines, rel)
		} else {
			abs, err := filepath.Abs(alt)
			if err != nil {
				return err
			}
			lines = append(lines, abs)
		}
	}

	data := []byte(strings.Join(lines, "\n") + "\n")
	altFile := filepath.Join(objectsDir, "info", "alternates")

	err := os.MkdirAll(filepath.Dir(altFile), 0o755)
	if err != nil {
		return err
	}

	return os.WriteFile(altFile, data, 0o644)
}
