// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package repo

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"forgejo.org/models/db"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/timeutil"
	"forgejo.org/modules/util"
)

// Alternate represents an alternate entry in the database
// TableName: alternates
// Columns: id (auto-increment), name (string)
type Alternate struct {
	ID   int64  `xorm:"pk autoincr"`
	Name string `xorm:"NOT NULL UNIQUE"`

	CreatedUnix timeutil.TimeStamp `xorm:"INDEX created"`
}

func init() {
	db.RegisterModel(new(Alternate))
}

func (alternate *Alternate) LogString() string {
	if alternate == nil {
		return "<Alternate nil>"
	}
	return fmt.Sprintf("<Alternate %d:%s>", alternate.ID, alternate.Name)
}

// Creates a new Alternate aimed for usage for the given repo
func CreateAlternateForRepo(ctx context.Context, repo *Repository) (*Alternate, error) {
	if repo.AlternateID != 0 {
		return nil, fmt.Errorf("repo %s already has an alternate assigned", repo.FullName())
	}

	alt := &Alternate{
		// This name safeguards against deletion and recreation of a new repo using the same repo name
		Name: strings.ToLower(fmt.Sprintf("%s_%s_%d", repo.OwnerName, repo.Name, repo.ID)),
	}

	err := db.Insert(ctx, alt)
	if err != nil {
		return nil, fmt.Errorf("failed to insert new Alternate entry: %w", err)
	}

	return alt, nil
}

// Gets the path where the alternate is stored on disk.
func (alternate *Alternate) GetPath() string {
	return filepath.Join(setting.RepoRootPath, "@alternates", fmt.Sprintf("%s.git", alternate.Name))
}

// Checks if any repositories are using this alternate.
func (alternate *Alternate) HasRepositories(ctx context.Context) (bool, error) {
	return db.GetEngine(ctx).Table("repository").Where("alternate_id = ?", alternate.ID).Exist(new(Repository))
}

// GetAlternateByID returns the alternate by given id if exists.
func GetAlternateByID(ctx context.Context, id int64) (*Alternate, error) {
	alternate := new(Alternate)
	has, err := db.GetEngine(ctx).ID(id).Get(alternate)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, ErrAlternateNotExist{id}
	}
	return alternate, nil
}

// ErrAlternateNotExist represents a "AlternateNotExist" kind of error.
type ErrAlternateNotExist struct {
	ID int64
}

// IsErrAlternateNotExist checks if an error is a ErrAlternateNotExist.
func IsErrAlternateNotExist(err error) bool {
	_, ok := err.(ErrAlternateNotExist)
	return ok
}

func (err ErrAlternateNotExist) Error() string {
	return fmt.Sprintf("alternate does not exist [id: %d]", err.ID)
}

// Unwrap unwraps this error as a ErrNotExist error
func (err ErrAlternateNotExist) Unwrap() error {
	return util.ErrNotExist
}
