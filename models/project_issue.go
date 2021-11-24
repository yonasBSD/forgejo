// Copyright 2020 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package models

import (
	"context"
	"fmt"

	"code.gitea.io/gitea/models/db"
	user_model "code.gitea.io/gitea/models/user"
)

// ProjectIssue saves relation from issue to a project
type ProjectIssue struct {
	ID        int64 `xorm:"pk autoincr"`
	IssueID   int64 `xorm:"INDEX"`
	ProjectID int64 `xorm:"INDEX"`

	// If 0, then it has not been added to a specific board in the project
	ProjectBoardID int64 `xorm:"INDEX"`
}

func init() {
	db.RegisterModel(new(ProjectIssue))
}

func deleteProjectIssuesByProjectID(e db.Engine, projectID int64) error {
	_, err := e.Where("project_id=?", projectID).Delete(&ProjectIssue{})
	return err
}

//  ___
// |_ _|___ ___ _   _  ___
//  | |/ __/ __| | | |/ _ \
//  | |\__ \__ \ |_| |  __/
// |___|___/___/\__,_|\___|

// LoadProject load the project the issue was assigned to
func (i *Issue) LoadProject() (err error) {
	return i.loadProject(db.GetEngine(db.DefaultContext))
}

func (i *Issue) loadProject(e db.Engine) (err error) {
	if i.Project == nil {
		var p Project
		if _, err = e.Table("project").
			Join("INNER", "project_issue", "project.id=project_issue.project_id").
			Where("project_issue.issue_id = ?", i.ID).
			Get(&p); err != nil {
			return err
		}
		i.Project = &p
	}
	return
}

// ProjectID return project id if issue was assigned to one
func (i *Issue) ProjectID() int64 {
	return i.projectID(db.GetEngine(db.DefaultContext))
}

func (i *Issue) projectID(e db.Engine) int64 {
	var ip ProjectIssue
	has, err := e.Where("issue_id=?", i.ID).Get(&ip)
	if err != nil || !has {
		return 0
	}
	return ip.ProjectID
}

// ProjectBoardID return project board id if issue was assigned to one
func (i *Issue) ProjectBoardID() int64 {
	return i.projectBoardID(db.GetEngine(db.DefaultContext))
}

func (i *Issue) projectBoardID(e db.Engine) int64 {
	var ip ProjectIssue
	has, err := e.Where("issue_id=?", i.ID).Get(&ip)
	if err != nil || !has {
		return 0
	}
	return ip.ProjectBoardID
}

//  ____            _           _
// |  _ \ _ __ ___ (_) ___  ___| |_
// | |_) | '__/ _ \| |/ _ \/ __| __|
// |  __/| | | (_) | |  __/ (__| |_
// |_|   |_|  \___// |\___|\___|\__|
//               |__/

// NumIssues return counter of all issues assigned to a project
func (p *Project) NumIssues() int {
	c, err := db.GetEngine(db.DefaultContext).Table("project_issue").
		Where("project_id=?", p.ID).
		GroupBy("issue_id").
		Cols("issue_id").
		Count()
	if err != nil {
		return 0
	}
	return int(c)
}

// NumClosedPrivateOwnIssues return counter of closed private issues of the user, assigned to a project
func (p *Project) NumClosedPrivateOwnIssues(userID int64) int {
	c, err := db.GetEngine(db.DefaultContext).Table("project_issue").
		Join("INNER", "issue", "project_issue.issue_id=issue.id").
		Where("project_issue.project_id=? AND issue.is_closed=? AND issue.is_private=? AND issue.poster_id=?", p.ID, true, true, userID).
		Cols("issue_id").
		Count()
	if err != nil {
		return 0
	}
	return int(c)
}

// NumClosedPrivateIssues return counter of closed private issues assigned to a project
func (p *Project) NumClosedPrivateIssues() int {
	c, err := db.GetEngine(db.DefaultContext).Table("project_issue").
		Join("INNER", "issue", "project_issue.issue_id=issue.id").
		Where("project_issue.project_id=? AND issue.is_closed=? AND issue.is_private=?", p.ID, true, true).
		Cols("issue_id").
		Count()
	if err != nil {
		return 0
	}
	return int(c)
}

// NumClosedIssues return counter of closed issues assigned to a project
func (p *Project) NumClosedIssues() int {
	c, err := db.GetEngine(db.DefaultContext).Table("project_issue").
		Join("INNER", "issue", "project_issue.issue_id=issue.id").
		Where("project_issue.project_id=? AND issue.is_closed=? AND issue.is_private=?", p.ID, true, false).
		Cols("issue_id").
		Count()
	if err != nil {
		return 0
	}
	return int(c)
}

// NumOpenIssues return counter of open issues assigned to a project
func (p *Project) NumOpenIssues() int {
	c, err := db.GetEngine(db.DefaultContext).Table("project_issue").
		Join("INNER", "issue", "project_issue.issue_id=issue.id").
		Where("project_issue.project_id=? AND issue.is_closed=? AND issue.is_private=?", p.ID, false, false).
		Cols("issue_id").
		Count()
	if err != nil {
		return 0
	}
	return int(c)
}

// NumOpenPrivateIssues return counter of open private issues assigned to a project
func (p *Project) NumOpenPrivateIssues() int {
	c, err := db.GetEngine(db.DefaultContext).Table("project_issue").
		Join("INNER", "issue", "project_issue.issue_id=issue.id").
		Where("project_issue.project_id=? AND issue.is_closed=? AND issue.is_private=?", p.ID, false, true).
		Cols("issue_id").
		Count()
	if err != nil {
		return 0
	}
	return int(c)
}

// NumOpenPrivateOwnIssues return counter of open private issues of the user, assigned to a project
func (p *Project) NumOpenPrivateOwnIssues(userID int64) int {
	c, err := db.GetEngine(db.DefaultContext).Table("project_issue").
		Join("INNER", "issue", "project_issue.issue_id=issue.id").
		Where("project_issue.project_id=? AND issue.is_closed=? AND issue.is_private=? AND issue.poster_id=?", p.ID, false, true, userID).
		Cols("issue_id").
		Count()
	if err != nil {
		return 0
	}
	return int(c)
}

// ChangeProjectAssign changes the project associated with an issue
func ChangeProjectAssign(issue *Issue, doer *user_model.User, newProjectID int64) error {
	ctx, committer, err := db.TxContext()
	if err != nil {
		return err
	}
	defer committer.Close()

	if err := addUpdateIssueProject(ctx, issue, doer, newProjectID); err != nil {
		return err
	}

	return committer.Commit()
}

func addUpdateIssueProject(ctx context.Context, issue *Issue, doer *user_model.User, newProjectID int64) error {
	e := db.GetEngine(ctx)
	oldProjectID := issue.projectID(e)

	if _, err := e.Where("project_issue.issue_id=?", issue.ID).Delete(&ProjectIssue{}); err != nil {
		return err
	}

	if err := issue.loadRepo(e); err != nil {
		return err
	}

	if oldProjectID > 0 || newProjectID > 0 {
		if _, err := createComment(ctx, &CreateCommentOptions{
			Type:         CommentTypeProject,
			Doer:         doer,
			Repo:         issue.Repo,
			Issue:        issue,
			OldProjectID: oldProjectID,
			ProjectID:    newProjectID,
		}); err != nil {
			return err
		}
	}

	_, err := e.Insert(&ProjectIssue{
		IssueID:   issue.ID,
		ProjectID: newProjectID,
	})
	return err
}

//  ____            _           _   ____                      _
// |  _ \ _ __ ___ (_) ___  ___| |_| __ )  ___   __ _ _ __ __| |
// | |_) | '__/ _ \| |/ _ \/ __| __|  _ \ / _ \ / _` | '__/ _` |
// |  __/| | | (_) | |  __/ (__| |_| |_) | (_) | (_| | | | (_| |
// |_|   |_|  \___// |\___|\___|\__|____/ \___/ \__,_|_|  \__,_|
//               |__/

// MoveIssueAcrossProjectBoards move a card from one board to another
func MoveIssueAcrossProjectBoards(issue *Issue, board *ProjectBoard) error {
	ctx, committer, err := db.TxContext()
	if err != nil {
		return err
	}
	defer committer.Close()
	sess := db.GetEngine(ctx)

	var pis ProjectIssue
	has, err := sess.Where("issue_id=?", issue.ID).Get(&pis)
	if err != nil {
		return err
	}

	if !has {
		return fmt.Errorf("issue has to be added to a project first")
	}

	pis.ProjectBoardID = board.ID
	if _, err := sess.ID(pis.ID).Cols("project_board_id").Update(&pis); err != nil {
		return err
	}

	return committer.Commit()
}

func (pb *ProjectBoard) removeIssues(e db.Engine) error {
	_, err := e.Exec("UPDATE `project_issue` SET project_board_id = 0 WHERE project_board_id = ? ", pb.ID)
	return err
}
