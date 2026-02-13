// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"forgejo.org/models/db"
	issues_model "forgejo.org/models/issues"
	"forgejo.org/models/perm"
	project_model "forgejo.org/models/project"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unit"
	"forgejo.org/modules/base"
	"forgejo.org/modules/json"
	"forgejo.org/modules/markup"
	"forgejo.org/modules/markup/markdown"
	"forgejo.org/modules/optional"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/util"
	"forgejo.org/modules/web"
	"forgejo.org/services/context"
	"forgejo.org/services/forms"
	project_service "forgejo.org/services/project"
)

const (
	tplProjects     base.TplName = "repo/projects/list"
	tplProjectsNew  base.TplName = "repo/projects/new"
	tplProjectsView base.TplName = "repo/projects/view"
)

// MustEnableProjects check if projects are enabled in settings
func MustEnableProjects(ctx *context.Context) {
	if unit.TypeProjects.UnitGlobalDisabled() {
		ctx.NotFound("EnableRepoProjects", nil)
		return
	}

	if ctx.Repo.Repository != nil {
		if !ctx.Repo.CanRead(unit.TypeProjects) {
			ctx.NotFound("MustEnableProjects", nil)
			return
		}
	}
}

// Projects renders the home page of projects
func Projects(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("repo.projects")

	sortType := ctx.FormTrim("sort")

	isShowClosed := strings.ToLower(ctx.FormTrim("state")) == "closed"
	keyword := ctx.FormTrim("q")
	repo := ctx.Repo.Repository
	page := ctx.FormInt("page")
	if page <= 1 {
		page = 1
	}

	ctx.Data["OpenCount"] = repo.NumOpenProjects
	ctx.Data["ClosedCount"] = repo.NumClosedProjects

	var total int
	if !isShowClosed {
		total = repo.NumOpenProjects
	} else {
		total = repo.NumClosedProjects
	}

	projects, count, err := db.FindAndCount[project_model.Project](ctx, project_model.SearchOptions{
		ListOptions: db.ListOptions{
			PageSize: setting.UI.IssuePagingNum,
			Page:     page,
		},
		RepoID:   repo.ID,
		IsClosed: optional.Some(isShowClosed),
		OrderBy:  project_model.GetSearchOrderByBySortType(sortType),
		Type:     project_model.TypeRepository,
		Title:    keyword,
	})
	if err != nil {
		ctx.ServerError("GetProjects", err)
		return
	}

	for i := range projects {
		projects[i].RenderedContent, err = markdown.RenderString(&markup.RenderContext{
			Links: markup.Links{
				Base: ctx.Repo.RepoLink,
			},
			Metas:   ctx.Repo.Repository.ComposeMetas(ctx),
			GitRepo: ctx.Repo.GitRepo,
			Ctx:     ctx,
		}, projects[i].Description)
		if err != nil {
			ctx.ServerError("RenderString", err)
			return
		}
	}

	ctx.Data["Projects"] = projects

	if isShowClosed {
		ctx.Data["State"] = "closed"
	} else {
		ctx.Data["State"] = "open"
	}

	numPages := 0
	if count > 0 {
		numPages = (int(count) - 1/setting.UI.IssuePagingNum)
	}

	pager := context.NewPagination(total, setting.UI.IssuePagingNum, page, numPages)
	pager.AddParam(ctx, "state", "State")
	ctx.Data["Page"] = pager

	ctx.Data["CanWriteProjects"] = ctx.Repo.CanWrite(unit.TypeProjects)
	ctx.Data["IsShowClosed"] = isShowClosed
	ctx.Data["IsProjectsPage"] = true
	ctx.Data["SortType"] = sortType

	numOpenIssues, err := issues_model.NumIssuesInProjects(ctx, projects, ctx.Doer, ctx.Org.Organization, optional.Some(false))
	if err != nil {
		ctx.ServerError("NumIssuesInProjects", err)
		return
	}
	numClosedIssues, err := issues_model.NumIssuesInProjects(ctx, projects, ctx.Doer, ctx.Org.Organization, optional.Some(true))
	if err != nil {
		ctx.ServerError("NumIssuesInProjects", err)
		return
	}
	ctx.Data["NumOpenIssuesInProject"] = numOpenIssues
	ctx.Data["NumClosedIssuesInProject"] = numClosedIssues

	ctx.HTML(http.StatusOK, tplProjects)
}

// RenderNewProject render creating a project page
func RenderNewProject(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("repo.projects.new")
	ctx.Data["TemplateConfigs"] = project_model.GetTemplateConfigs()
	ctx.Data["CardTypes"] = project_model.GetCardConfig()
	ctx.Data["CanWriteProjects"] = ctx.Repo.CanWrite(unit.TypeProjects)
	ctx.Data["CancelLink"] = ctx.Repo.Repository.Link() + "/projects"
	ctx.HTML(http.StatusOK, tplProjectsNew)
}

// NewProjectPost creates a new project
func NewProjectPost(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.CreateProjectForm)
	ctx.Data["Title"] = ctx.Tr("repo.projects.new")

	if ctx.HasError() {
		RenderNewProject(ctx)
		return
	}

	opts := project_service.CreateProjectOptions{
		Title:        form.Title,
		Description:  form.Content,
		TemplateType: form.TemplateType,
		CardType:     form.CardType,
		CanWrite:     true, // Web UI users always have write access
	}

	_, err := project_service.CreateProject(ctx, ctx.Repo.Repository, ctx.Doer, opts)
	if err != nil {
		ctx.ServerError("CreateProject", err)
		return
	}

	ctx.Flash.Success(ctx.Tr("repo.projects.create_success", form.Title))
	ctx.Redirect(ctx.Repo.RepoLink + "/projects")
}

// ChangeProjectStatus updates the status of a project between "open" and "close"
func ChangeProjectStatus(ctx *context.Context) {
	var toClose bool
	switch ctx.Params(":action") {
	case "open":
		toClose = false
	case "close":
		toClose = true
	default:
		ctx.JSONRedirect(ctx.Repo.RepoLink + "/projects")
		return
	}
	id := ctx.ParamsInt64(":id")

	if err := project_model.ChangeProjectStatusByRepoIDAndID(ctx, ctx.Repo.Repository.ID, id, toClose); err != nil {
		ctx.NotFoundOrServerError("ChangeProjectStatusByRepoIDAndID", project_model.IsErrProjectNotExist, err)
		return
	}
	ctx.JSONRedirect(project_model.ProjectLinkForRepo(ctx.Repo.Repository, id))
}

// DeleteProject delete a project
func DeleteProject(ctx *context.Context) {
	p, err := project_model.GetProjectByID(ctx, ctx.ParamsInt64(":id"))
	if err != nil {
		if project_model.IsErrProjectNotExist(err) {
			ctx.NotFound("", nil)
		} else {
			ctx.ServerError("GetProjectByID", err)
		}
		return
	}
	if p.RepoID != ctx.Repo.Repository.ID {
		ctx.NotFound("", nil)
		return
	}

	if err := project_model.DeleteProjectByID(ctx, p.ID); err != nil {
		ctx.Flash.Error("DeleteProjectByID: " + err.Error())
	} else {
		ctx.Flash.Success(ctx.Tr("repo.projects.deletion_success"))
	}

	ctx.JSONRedirect(ctx.Repo.RepoLink + "/projects")
}

// RenderEditProject allows a project to be edited
func RenderEditProject(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("repo.projects.edit")
	ctx.Data["PageIsEditProjects"] = true
	ctx.Data["CanWriteProjects"] = ctx.Repo.CanWrite(unit.TypeProjects)
	ctx.Data["CardTypes"] = project_model.GetCardConfig()

	p, err := project_model.GetProjectByID(ctx, ctx.ParamsInt64(":id"))
	if err != nil {
		if project_model.IsErrProjectNotExist(err) {
			ctx.NotFound("", nil)
		} else {
			ctx.ServerError("GetProjectByID", err)
		}
		return
	}
	if p.RepoID != ctx.Repo.Repository.ID {
		ctx.NotFound("", nil)
		return
	}

	ctx.Data["projectID"] = p.ID
	ctx.Data["title"] = p.Title
	ctx.Data["content"] = p.Description
	ctx.Data["card_type"] = p.CardType
	ctx.Data["redirect"] = ctx.FormString("redirect")
	ctx.Data["CancelLink"] = project_model.ProjectLinkForRepo(ctx.Repo.Repository, p.ID)

	ctx.HTML(http.StatusOK, tplProjectsNew)
}

// EditProjectPost response for editing a project
func EditProjectPost(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.CreateProjectForm)
	projectID := ctx.ParamsInt64(":id")

	ctx.Data["Title"] = ctx.Tr("repo.projects.edit")
	ctx.Data["PageIsEditProjects"] = true
	ctx.Data["CanWriteProjects"] = ctx.Repo.CanWrite(unit.TypeProjects)
	ctx.Data["CardTypes"] = project_model.GetCardConfig()
	ctx.Data["CancelLink"] = project_model.ProjectLinkForRepo(ctx.Repo.Repository, projectID)

	if ctx.HasError() {
		ctx.HTML(http.StatusOK, tplProjectsNew)
		return
	}

	p, err := project_model.GetProjectByID(ctx, projectID)
	if err != nil {
		if project_model.IsErrProjectNotExist(err) {
			ctx.NotFound("", nil)
		} else {
			ctx.ServerError("GetProjectByID", err)
		}
		return
	}
	if p.RepoID != ctx.Repo.Repository.ID {
		ctx.NotFound("", nil)
		return
	}

	opts := project_service.UpdateProjectOptions{
		Title:       &form.Title,
		Description: &form.Content,
		CardType:    &form.CardType,
	}

	if err = project_service.UpdateProject(ctx, p, opts); err != nil {
		ctx.ServerError("UpdateProject", err)
		return
	}

	ctx.Flash.Success(ctx.Tr("repo.projects.edit_success", p.Title))
	if ctx.FormString("redirect") == "project" {
		ctx.Redirect(p.Link(ctx))
	} else {
		ctx.Redirect(ctx.Repo.RepoLink + "/projects")
	}
}

// ViewProject renders the project with board view
func ViewProject(ctx *context.Context) {
	project, err := project_model.GetProjectByID(ctx, ctx.ParamsInt64(":id"))
	if err != nil {
		if project_model.IsErrProjectNotExist(err) {
			ctx.NotFound("", nil)
		} else {
			ctx.ServerError("GetProjectByID", err)
		}
		return
	}
	if project.RepoID != ctx.Repo.Repository.ID {
		ctx.NotFound("", nil)
		return
	}

	columns, err := project.GetColumns(ctx)
	if err != nil {
		ctx.ServerError("GetProjectColumns", err)
		return
	}

	issuesMap, err := issues_model.LoadIssuesFromColumnList(ctx, columns, ctx.Doer, nil, optional.None[bool]())
	if err != nil {
		ctx.ServerError("LoadIssuesOfColumns", err)
		return
	}

	if project.CardType != project_model.CardTypeTextOnly {
		issuesAttachmentMap := make(map[int64][]*repo_model.Attachment)
		for _, issuesList := range issuesMap {
			for _, issue := range issuesList {
				if issueAttachment, err := repo_model.GetAttachmentsByIssueIDImagesLatest(ctx, issue.ID); err == nil {
					issuesAttachmentMap[issue.ID] = issueAttachment
				}
			}
		}
		ctx.Data["issuesAttachmentMap"] = issuesAttachmentMap
	}

	linkedPrsMap := make(map[int64][]*issues_model.Issue)
	for _, issuesList := range issuesMap {
		for _, issue := range issuesList {
			var referencedIDs []int64
			for _, comment := range issue.Comments {
				if comment.RefIssueID != 0 && comment.RefIsPull {
					referencedIDs = append(referencedIDs, comment.RefIssueID)
				}
			}

			if len(referencedIDs) > 0 {
				if linkedPrs, err := issues_model.Issues(ctx, &issues_model.IssuesOptions{
					IssueIDs: referencedIDs,
					IsPull:   optional.Some(true),
				}); err == nil {
					linkedPrsMap[issue.ID] = linkedPrs
				}
			}
		}
	}
	ctx.Data["LinkedPRs"] = linkedPrsMap

	project.RenderedContent, err = markdown.RenderString(&markup.RenderContext{
		Links: markup.Links{
			Base: ctx.Repo.RepoLink,
		},
		Metas:   ctx.Repo.Repository.ComposeMetas(ctx),
		GitRepo: ctx.Repo.GitRepo,
		Ctx:     ctx,
	}, project.Description)
	if err != nil {
		ctx.ServerError("RenderString", err)
		return
	}

	ctx.Data["Title"] = project.Title
	ctx.Data["IsProjectsPage"] = true
	ctx.Data["CanWriteProjects"] = ctx.Repo.CanWrite(unit.TypeProjects)
	ctx.Data["Project"] = project
	ctx.Data["IssuesMap"] = issuesMap
	ctx.Data["Columns"] = columns

	ctx.HTML(http.StatusOK, tplProjectsView)
}

// UpdateIssueProject change an issue's project
func UpdateIssueProject(ctx *context.Context) {
	issues := getActionIssues(ctx)
	if ctx.Written() {
		return
	}

	if err := issues.LoadProjects(ctx); err != nil {
		ctx.ServerError("LoadProjects", err)
		return
	}
	if _, err := issues.LoadRepositories(ctx); err != nil {
		ctx.ServerError("LoadProjects", err)
		return
	}

	projectID := ctx.FormInt64("id")
	for _, issue := range issues {
		if issue.Project != nil && issue.Project.ID == projectID {
			continue
		}
		if err := issues_model.IssueAssignOrRemoveProject(ctx, issue, ctx.Doer, projectID, 0); err != nil {
			if errors.Is(err, util.ErrPermissionDenied) {
				continue
			}
			ctx.ServerError("IssueAssignOrRemoveProject", err)
			return
		}
	}

	if ctx.FormBool("htmx") {
		prepareHiddenCommentType(ctx)
		if ctx.Written() {
			return
		}

		issue := issues[0]
		// Force reload project
		issue.Project = nil
		if err := issue.LoadProject(ctx); err != nil {
			ctx.ServerError("LoadProject", err)
			return
		}

		if issue.Project != nil {
			columns, err := issue.Project.GetColumns(ctx)
			if err != nil {
				ctx.ServerError("GetProjectColumns", err)
				return
			}
			ctx.Data["ProjectColumns"] = columns

			currentColumn, err := issue.LoadProjectColumn(ctx)
			if err != nil {
				ctx.ServerError("LoadProjectColumn", err)
				return
			}
			ctx.Data["IssueProjectColumn"] = currentColumn
		}

		retrieveProjects(ctx, ctx.Repo.Repository)
		if ctx.Written() {
			return
		}

		comment := &issues_model.Comment{}
		has, err := db.GetEngine(ctx).Where("issue_id = ? AND type = ?", issue.ID, issues_model.CommentTypeProject).OrderBy("id DESC").Limit(1).Get(comment)
		if err != nil {
			ctx.ServerError("GetLatestProjectComment", err)
			return
		}
		if has {
			if err := comment.LoadProject(ctx); err != nil {
				ctx.ServerError("LoadProject", err)
				return
			}
			if err := comment.LoadPoster(ctx); err != nil {
				ctx.ServerError("LoadPoster", err)
				return
			}
			issue.Comments = issues_model.CommentList{comment}
		} else {
			issue.Comments = issues_model.CommentList{}
		}

		ctx.Data["Issue"] = issue
		ctx.Data["HasIssuesOrPullsWritePermission"] = ctx.Repo.CanWriteIssuesOrPulls(issue.IsPull)
		ctx.HTML(http.StatusOK, "htmx/project_sidebar")
	} else {
		ctx.JSONOK()
	}
}

// UpdateIssueProjectColumn changes the column an issue is in within its current project
func UpdateIssueProjectColumn(ctx *context.Context) {
	issueID := ctx.FormInt64("issue_id")
	columnID := ctx.FormInt64("column_id")

	issue, err := issues_model.GetIssueByID(ctx, issueID)
	if err != nil {
		ctx.NotFoundOrServerError("GetIssueByID", issues_model.IsErrIssueNotExist, err)
		return
	}

	if issue.RepoID != ctx.Repo.Repository.ID {
		ctx.NotFound("", nil)
		return
	}

	if err := issue.LoadProject(ctx); err != nil {
		ctx.ServerError("LoadProject", err)
		return
	}
	if issue.Project == nil {
		ctx.NotFound("issue has no project", nil)
		return
	}

	card, err := project_model.GetProjectCard(ctx, issue.Project.ID, issue.ID)
	if err != nil {
		ctx.ServerError("GetProjectCard", err)
		return
	}

	if err := project_model.MoveCardToColumn(ctx, card.ID, columnID, 0); err != nil {
		if project_model.IsErrProjectColumnNotExist(err) {
			ctx.NotFound("column not found or does not belong to project", nil)
			return
		}
		ctx.ServerError("MoveCardToColumn", err)
		return
	}

	if ctx.FormBool("htmx") {
		columns, err := issue.Project.GetColumns(ctx)
		if err != nil {
			ctx.ServerError("GetProjectColumns", err)
			return
		}
		ctx.Data["ProjectColumns"] = columns

		currentColumn, err := issue.LoadProjectColumn(ctx)
		if err != nil {
			ctx.ServerError("LoadProjectColumn", err)
			return
		}
		ctx.Data["IssueProjectColumn"] = currentColumn

		ctx.Data["Issue"] = issue
		ctx.Data["HasIssuesOrPullsWritePermission"] = ctx.Repo.CanWriteIssuesOrPulls(issue.IsPull)
		ctx.HTML(http.StatusOK, "htmx/project_column_sidebar")
	} else {
		ctx.JSONOK()
	}
}

// DeleteProjectColumn allows for the deletion of a project column
func DeleteProjectColumn(ctx *context.Context) {
	if ctx.Doer == nil {
		ctx.JSON(http.StatusForbidden, map[string]string{
			"message": "Only signed in users are allowed to perform this action.",
		})
		return
	}

	if !ctx.Repo.IsOwner() && !ctx.Repo.IsAdmin() && !ctx.Repo.CanAccess(perm.AccessModeWrite, unit.TypeProjects) {
		ctx.JSON(http.StatusForbidden, map[string]string{
			"message": "Only authorized users are allowed to perform this action.",
		})
		return
	}

	project, err := project_model.GetProjectByID(ctx, ctx.ParamsInt64(":id"))
	if err != nil {
		if project_model.IsErrProjectNotExist(err) {
			ctx.NotFound("", nil)
		} else {
			ctx.ServerError("GetProjectByID", err)
		}
		return
	}

	pb, err := project_model.GetColumn(ctx, ctx.ParamsInt64(":columnID"))
	if err != nil {
		ctx.ServerError("GetProjectColumn", err)
		return
	}
	if pb.ProjectID != ctx.ParamsInt64(":id") {
		ctx.JSON(http.StatusUnprocessableEntity, map[string]string{
			"message": fmt.Sprintf("ProjectColumn[%d] is not in Project[%d] as expected", pb.ID, project.ID),
		})
		return
	}

	if project.RepoID != ctx.Repo.Repository.ID {
		ctx.JSON(http.StatusUnprocessableEntity, map[string]string{
			"message": fmt.Sprintf("ProjectColumn[%d] is not in Repository[%d] as expected", pb.ID, ctx.Repo.Repository.ID),
		})
		return
	}

	if err := project_model.DeleteColumnByID(ctx, ctx.ParamsInt64(":columnID")); err != nil {
		ctx.ServerError("DeleteProjectColumnByID", err)
		return
	}

	ctx.JSONOK()
}

// AddColumnToProjectPost allows a new column to be added to a project.
func AddColumnToProjectPost(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.EditProjectColumnForm)
	if !ctx.Repo.IsOwner() && !ctx.Repo.IsAdmin() && !ctx.Repo.CanAccess(perm.AccessModeWrite, unit.TypeProjects) {
		ctx.JSON(http.StatusForbidden, map[string]string{
			"message": "Only authorized users are allowed to perform this action.",
		})
		return
	}

	project, err := project_model.GetProjectForRepoByID(ctx, ctx.Repo.Repository.ID, ctx.ParamsInt64(":id"))
	if err != nil {
		if project_model.IsErrProjectNotExist(err) {
			ctx.NotFound("", nil)
		} else {
			ctx.ServerError("GetProjectByID", err)
		}
		return
	}

	if err := project_model.NewColumn(ctx, &project_model.Column{
		ProjectID: project.ID,
		Title:     form.Title,
		Color:     form.Color,
		CreatorID: ctx.Doer.ID,
	}); err != nil {
		ctx.ServerError("NewProjectColumn", err)
		return
	}

	ctx.JSONOK()
}

func checkProjectColumnChangePermissions(ctx *context.Context) (*project_model.Project, *project_model.Column) {
	if ctx.Doer == nil {
		ctx.JSON(http.StatusForbidden, map[string]string{
			"message": "Only signed in users are allowed to perform this action.",
		})
		return nil, nil
	}

	if !ctx.Repo.IsOwner() && !ctx.Repo.IsAdmin() && !ctx.Repo.CanAccess(perm.AccessModeWrite, unit.TypeProjects) {
		ctx.JSON(http.StatusForbidden, map[string]string{
			"message": "Only authorized users are allowed to perform this action.",
		})
		return nil, nil
	}

	project, err := project_model.GetProjectByID(ctx, ctx.ParamsInt64(":id"))
	if err != nil {
		if project_model.IsErrProjectNotExist(err) {
			ctx.NotFound("", nil)
		} else {
			ctx.ServerError("GetProjectByID", err)
		}
		return nil, nil
	}

	column, err := project_model.GetColumn(ctx, ctx.ParamsInt64(":columnID"))
	if err != nil {
		ctx.ServerError("GetProjectColumn", err)
		return nil, nil
	}
	if column.ProjectID != ctx.ParamsInt64(":id") {
		ctx.JSON(http.StatusUnprocessableEntity, map[string]string{
			"message": fmt.Sprintf("ProjectColumn[%d] is not in Project[%d] as expected", column.ID, project.ID),
		})
		return nil, nil
	}

	if project.RepoID != ctx.Repo.Repository.ID {
		ctx.JSON(http.StatusUnprocessableEntity, map[string]string{
			"message": fmt.Sprintf("ProjectColumn[%d] is not in Repository[%d] as expected", column.ID, ctx.Repo.Repository.ID),
		})
		return nil, nil
	}
	return project, column
}

// EditProjectColumn allows a project column's to be updated
func EditProjectColumn(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.EditProjectColumnForm)
	_, column := checkProjectColumnChangePermissions(ctx)
	if ctx.Written() {
		return
	}

	if form.Title != "" {
		column.Title = form.Title
	}
	column.Color = form.Color
	if form.Sorting != 0 {
		column.Sorting = form.Sorting
	}

	if err := project_model.UpdateColumn(ctx, column); err != nil {
		ctx.ServerError("UpdateProjectColumn", err)
		return
	}

	ctx.JSONOK()
}

// SetDefaultProjectColumn set default column for uncategorized issues/pulls
func SetDefaultProjectColumn(ctx *context.Context) {
	project, column := checkProjectColumnChangePermissions(ctx)
	if ctx.Written() {
		return
	}

	if err := project_model.SetDefaultColumn(ctx, project.ID, column.ID); err != nil {
		ctx.ServerError("SetDefaultColumn", err)
		return
	}

	ctx.JSONOK()
}

// MoveIssues moves or keeps issues in a column and sorts them inside that column
func MoveIssues(ctx *context.Context) {
	if ctx.Doer == nil {
		ctx.JSON(http.StatusForbidden, map[string]string{
			"message": "Only signed in users are allowed to perform this action.",
		})
		return
	}

	if !ctx.Repo.IsOwner() && !ctx.Repo.IsAdmin() && !ctx.Repo.CanAccess(perm.AccessModeWrite, unit.TypeProjects) {
		ctx.JSON(http.StatusForbidden, map[string]string{
			"message": "Only authorized users are allowed to perform this action.",
		})
		return
	}

	project, err := project_model.GetProjectByID(ctx, ctx.ParamsInt64(":id"))
	if err != nil {
		if project_model.IsErrProjectNotExist(err) {
			ctx.NotFound("ProjectNotExist", nil)
		} else {
			ctx.ServerError("GetProjectByID", err)
		}
		return
	}
	if project.RepoID != ctx.Repo.Repository.ID {
		ctx.NotFound("InvalidRepoID", nil)
		return
	}

	column, err := project_model.GetColumn(ctx, ctx.ParamsInt64(":columnID"))
	if err != nil {
		if project_model.IsErrProjectColumnNotExist(err) {
			ctx.NotFound("ProjectColumnNotExist", nil)
		} else {
			ctx.ServerError("GetProjectColumn", err)
		}
		return
	}

	if column.ProjectID != project.ID {
		ctx.NotFound("ColumnNotInProject", nil)
		return
	}

	type movedIssuesForm struct {
		Issues []struct {
			IssueID int64 `json:"issueID"`
			Sorting int64 `json:"sorting"`
		} `json:"issues"`
	}

	form := &movedIssuesForm{}
	if err = json.NewDecoder(ctx.Req.Body).Decode(&form); err != nil {
		ctx.ServerError("DecodeMovedIssuesForm", err)
		return
	}

	// Convert to service format
	cardPositions := make([]project_service.CardPosition, len(form.Issues))
	for i, issue := range form.Issues {
		cardPositions[i] = project_service.CardPosition{
			IssueID: issue.IssueID,
			Sorting: issue.Sorting,
		}
	}

	// Call shared service
	if err = project_service.ReorderCardsInColumn(ctx, column, cardPositions); err != nil {
		ctx.ServerError("ReorderCardsInColumn", err)
		return
	}

	ctx.JSONOK()
}
