// Copyright 2017 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package models

import (
	"time"
)

// IssueDependency is connection request for receiving issue notification.
type IssueDependency struct {
	ID          int64     `xorm:"pk autoincr"`
	UserID      int64     `xorm:"UNIQUE(watch) NOT NULL"`
	IssueID     int64     `xorm:"UNIQUE(watch) NOT NULL"`
	DependencyID int64    `xorm:"UNIQUE(watch) NOT NULL"`
	Created     time.Time `xorm:"-"`
	CreatedUnix int64     `xorm:"NOT NULL"`
	Updated     time.Time `xorm:"-"`
	UpdatedUnix int64     `xorm:"NOT NULL"`
}

// BeforeInsert is invoked from XORM before inserting an object of this type.
func (iw *IssueDependency) BeforeInsert() {
	var (
		t = time.Now()
		u = t.Unix()
	)
	iw.Created = t
	iw.CreatedUnix = u
	iw.Updated = t
	iw.UpdatedUnix = u
}

// BeforeUpdate is invoked from XORM before updating an object of this type.
func (iw *IssueDependency) BeforeUpdate() {
	var (
		t = time.Now()
		u = t.Unix()
	)
	iw.Updated = t
	iw.UpdatedUnix = u
}

// CreateIssueDependency creates a new dependency for an issue
func CreateIssueDependency(userID, issueID int64, depID int64) (err error, exists bool, depExists bool) {
	err = x.Sync(new(IssueDependency))
	if err != nil {
		return err, exists, false
	}

	// Check if it aleready exists
	exists, err = issueDepExists(x, issueID, depID)
	if err != nil {
		return err, exists, false
	}

	// If it not exists, create it, otherwise show an error message
	if !exists {
		// Check if the other issue exists
		var issue = Issue{}
		issueExists, err := x.Id(depID).Get(&issue)
		if issueExists {
			newId := new(IssueDependency)
			newId.UserID = userID
			newId.IssueID = issueID
			newId.DependencyID = depID

			if _, err := x.Insert(newId); err != nil {
				return err, exists, false
			}

			// Add comment referencing the new dependency
			comment := &Comment{
				IssueID:  issueID,
				PosterID: userID,
				Type:     CommentTypeAddedDependency,
			}

			if _, err := x.Insert(comment); err != nil {
				return err, exists, false
			}
		}
		return err, exists, true
	}
	return nil, exists, false
}

// Check if the dependency already exists
func issueDepExists(e Engine, issueID int64, depID int64) (exists bool, err error) {
	var Dependencies = IssueDependency{IssueID: issueID, DependencyID: depID}

	exists, err = e.Get(&Dependencies)

	// Check for dependencies the other way around
	// Otherwise two issues could block each other which would result in none of them could be closed.
	if !exists {
		Dependencies.IssueID = depID
		Dependencies.DependencyID = issueID
		exists, err = e.Get(&Dependencies)
	}

	return
}
