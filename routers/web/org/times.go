// Copyright 2014 The Gogs Authors. All rights reserved.
// Copyright 2022 The Gitea Authors.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package org

import (
	"net/http"
	"time"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/setting"

	"xorm.io/builder"
)

const (
	tplByRepos      base.TplName = "org/times/times_by_repos"
	tplByMembers    base.TplName = "org/times/times_by_members"
	tplByMilestones base.TplName = "org/times/times_by_milestones"
)

// parseTimes contains functionality that is required in all these functions,
// like parsing the date from the request, setting default dates, etc.
func parseTimes(ctx *context.Context) (unixfrom, unixto int64, err error) {
	// View variables
	ctx.Data["PageIsOrgTimes"] = true
	ctx.Data["AppSubURL"] = setting.AppSubURL

	// Time range from request, if any
	from := ctx.FormString("from")
	to := ctx.FormString("to")
	// Defaults for "from" and "to" dates, if not in request
	if from == "" {
		// DEFAULT of "from": start of current month
		from = time.Now().Format("2006-01") + "-01"
	}
	if to == "" {
		// DEFAULT of "to": today
		to = time.Now().Format("2006-01-02")
	}

	// Prepare Form values
	ctx.Data["RangeFrom"] = from
	ctx.Data["RangeTo"] = to

	// Prepare unix time values for SQL
	gfrom, err := time.Parse("2006-01-02", from)
	if err != nil {
		ctx.ServerError("TimesError", err)
	}
	unixfrom = gfrom.Unix()
	gto, err := time.Parse("2006-01-02", to)
	if err != nil {
		ctx.ServerError("TimesError", err)
	}
	// Humans expect that we include the ending day too
	unixto = gto.Add(1440 * time.Minute).Unix()
	return
}

// TimesByRepos renders worktime by repositories.
func TimesByRepos(ctx *context.Context) {
	// Run common functionality
	unixfrom, unixto, err := parseTimes(ctx)
	if err != nil {
		return
	}

	// Set submenu tab
	ctx.Data["TabIsByRepos"] = true

	// Get the data from the DB
	type result struct {
		Name    string `xorm:"name"`
		SumTime int64  `xorm:"sumtime"`
	}
	sql, args, err := builder.Select("repository.name", "SUM(tracked_time.time) AS sumtime").
		From("tracked_time").
		InnerJoin("issue", "tracked_time.issue_id = issue.id").
		InnerJoin("repository", "issue.repo_id = repository.id").
		Where(builder.Eq{"repository.owner_id": ctx.Org.Organization.ID}).
		And(builder.Eq{"tracked_time.deleted": 0}).
		And(builder.Gte{"tracked_time.created_unix": unixfrom}).
		And(builder.Lt{"tracked_time.created_unix": unixto}).
		GroupBy("repository.id").
		OrderBy("repository.name").
		ToSQL()
	if err != nil {
		ctx.ServerError("TimesError", err)
		return
	}
	var results []result
	err = db.GetEngine(ctx).SQL(sql, args...).Find(&results)
	if err != nil {
		ctx.ServerError("TimesError", err)
		return
	}
	ctx.Data["results"] = results

	// Reply with view
	ctx.HTML(http.StatusOK, tplByRepos)
}

// TimesByMilestones renders work time by milestones.
func TimesByMilestones(ctx *context.Context) {
	// Run common functionality
	unixfrom, unixto, err := parseTimes(ctx)
	if err != nil {
		return
	}

	// Set submenu tab
	ctx.Data["TabIsByMilestones"] = true

	// Get the data from the DB
	type result struct {
		RepoName     string `xorm:"reponame"`
		Name         string `xorm:"name"`
		ID           string `xorm:"id"`
		SumTime      int64  `xorm:"sumtime"`
		HideRepoName bool   `xorm:"-"`
	}
	sql, args, err := builder.Select("repository.name AS reponame", "milestone.name", "milestone.id", "SUM(tracked_time.time) AS sumtime").
		From("tracked_time").
		InnerJoin("issue", "tracked_time.issue_id = issue.id").
		InnerJoin("repository", "issue.repo_id = repository.id").
		LeftJoin("milestone", "issue.milestone_id = milestone.id").
		Where(builder.Eq{"repository.owner_id": ctx.Org.Organization.ID}).
		And(builder.Eq{"tracked_time.deleted": 0}).
		And(builder.Gte{"tracked_time.created_unix": unixfrom}).
		And(builder.Lt{"tracked_time.created_unix": unixto}).
		GroupBy("repository.id, milestone.id").
		OrderBy("repository.name, milestone.deadline_unix, milestone.id").
		ToSQL()
	if err != nil {
		ctx.ServerError("TimesError", err)
		return
	}
	var results []result
	err = db.GetEngine(ctx).SQL(sql, args...).Find(&results)
	if err != nil {
		ctx.ServerError("TimesError", err)
		return
	}

	// Show only the first RepoName, for nicer output.
	prevreponame := ""
	for i := 0; i < len(results); i++ {
		res := &results[i]
		if prevreponame == res.RepoName {
			res.HideRepoName = true
		}
		prevreponame = res.RepoName
	}

	// Send results to view
	ctx.Data["results"] = results

	// Reply with view
	ctx.HTML(http.StatusOK, tplByMilestones)
}

// TimesByMembers renders worktime by project member persons.
func TimesByMembers(ctx *context.Context) {
	// Run common functionality
	unixfrom, unixto, err := parseTimes(ctx)
	if err != nil {
		return
	}

	// Set submenu tab
	ctx.Data["TabIsByMembers"] = true

	// Get the data from the DB
	type result struct {
		Name    string `xorm:"name"`
		SumTime int64  `xorm:"sumtime"`
	}
	sql, args, err := builder.Select("user.name", "SUM(tracked_time.time) AS sumtime").
		From("tracked_time").
		InnerJoin("issue", "tracked_time.issue_id = issue.id").
		InnerJoin("repository", "issue.repo_id = repository.id").
		InnerJoin("user", "tracked_time.user_id = user.id").
		Where(builder.Eq{"repository.owner_id": ctx.Org.Organization.ID}).
		And(builder.Eq{"tracked_time.deleted": 0}).
		And(builder.Gte{"tracked_time.created_unix": unixfrom}).
		And(builder.Lt{"tracked_time.created_unix": unixto}).
		GroupBy("user.id").
		OrderBy("sumtime DESC").
		ToSQL()
	if err != nil {
		ctx.ServerError("TimesError", err)
		return
	}
	var results []result
	err = db.GetEngine(ctx).SQL(sql, args...).Find(&results)
	if err != nil {
		ctx.ServerError("TimesError", err)
		return
	}
	ctx.Data["results"] = results

	ctx.HTML(http.StatusOK, tplByMembers)
}
