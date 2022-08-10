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

// resultTimesByRepos is a struct for DB query results
type resultTimesByRepos struct {
	Name    string
	SumTime int64
}

// resultTimesByMilestones is a struct for DB query results
type resultTimesByMilestones struct {
	RepoName     string
	Name         string
	ID           string
	SumTime      int64
	HideRepoName bool
}

// resultTimesByMembers is a struct for DB query results
type resultTimesByMembers struct {
	Name    string
	SumTime int64
}

// parseOrgTimes contains functionality that is required in all these functions,
// like parsing the date from the request, setting default dates, etc.
func parseOrgTimes(ctx *context.Context) (unixfrom, unixto int64, err error) {
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
	from2, err := time.Parse("2006-01-02", from)
	if err != nil {
		ctx.ServerError("time.Parse", err)
	}
	unixfrom = from2.Unix()
	to2, err := time.Parse("2006-01-02", to)
	if err != nil {
		ctx.ServerError("time.Parse", err)
	}
	// Humans expect that we include the ending day too
	unixto = to2.Add(1440*time.Minute - 1*time.Second).Unix()
	return unixfrom, unixto, err
}

// TimesByRepos renders worktime by repositories.
func TimesByRepos(ctx *context.Context) {
	// Run common functionality
	unixfrom, unixto, err := parseOrgTimes(ctx)
	if err != nil {
		return
	}

	// View variables
	ctx.Data["PageIsOrgTimes"] = true
	ctx.Data["AppSubURL"] = setting.AppSubURL

	// Set submenu tab
	ctx.Data["TabIsByRepos"] = true

	results, err := getTimesByRepos(unixfrom, unixto, ctx.Org.Organization.ID)
	if err != nil {
		ctx.ServerError("getTimesByRepos", err)
		return
	}
	ctx.Data["results"] = results

	// Reply with view
	ctx.HTML(http.StatusOK, tplByRepos)
}

// getTimesByRepos fetches data from DB to serve TimesByRepos.
func getTimesByRepos(unixfrom, unixto, orgid int64) (results []resultTimesByRepos, err error) {
	// Get the data from the DB
	err = db.GetEngine(db.DefaultContext).
		Select("repository.name, SUM(tracked_time.time) AS sum_time").
		Table("tracked_time").
		Join("INNER", "issue", "tracked_time.issue_id = issue.id").
		Join("INNER", "repository", "issue.repo_id = repository.id").
		Where(builder.Eq{"repository.owner_id": orgid}).
		And(builder.Eq{"tracked_time.deleted": false}).
		And(builder.Gte{"tracked_time.created_unix": unixfrom}).
		And(builder.Lte{"tracked_time.created_unix": unixto}).
		GroupBy("repository.id").
		OrderBy("repository.name").
		Find(&results)
	return results, err
}

// TimesByMilestones renders work time by milestones.
func TimesByMilestones(ctx *context.Context) {
	// Run common functionality
	unixfrom, unixto, err := parseOrgTimes(ctx)
	if err != nil {
		return
	}

	// View variables
	ctx.Data["PageIsOrgTimes"] = true
	ctx.Data["AppSubURL"] = setting.AppSubURL

	// Set submenu tab
	ctx.Data["TabIsByMilestones"] = true

	// Get the data from the DB
	results, err := getTimesByMilestones(unixfrom, unixto, ctx.Org.Organization.ID)
	if err != nil {
		ctx.ServerError("getTimesByMilestones", err)
		return
	}

	// Send results to view
	ctx.Data["results"] = results

	// Reply with view
	ctx.HTML(http.StatusOK, tplByMilestones)
}

// getTimesByMilestones gets the actual data from the DB to serve TimesByMilestones.
func getTimesByMilestones(unixfrom, unixto, orgid int64) (results []resultTimesByMilestones, err error) {
	err = db.GetEngine(db.DefaultContext).
		Select("repository.name AS repo_name, milestone.name, milestone.id, SUM(tracked_time.time) AS sum_time").
		Table("tracked_time").
		Join("INNER", "issue", "tracked_time.issue_id = issue.id").
		Join("INNER", "repository", "issue.repo_id = repository.id").
		Join("LEFT", "milestone", "issue.milestone_id = milestone.id").
		Where(builder.Eq{"repository.owner_id": orgid}).
		And(builder.Eq{"tracked_time.deleted": false}).
		And(builder.Gte{"tracked_time.created_unix": unixfrom}).
		And(builder.Lte{"tracked_time.created_unix": unixto}).
		GroupBy("repository.id, milestone.id").
		OrderBy("repository.name, milestone.deadline_unix, milestone.id").
		Find(&results)

	// Show only the first RepoName, for nicer output.
	prevreponame := ""
	for i := 0; i < len(results); i++ {
		res := &results[i]
		if prevreponame == res.RepoName {
			res.HideRepoName = true
		}
		prevreponame = res.RepoName
	}

	return results, err
}

// TimesByMembers renders worktime by project member persons.
func TimesByMembers(ctx *context.Context) {
	// Run common functionality
	unixfrom, unixto, err := parseOrgTimes(ctx)
	if err != nil {
		return
	}

	// View variables
	ctx.Data["PageIsOrgTimes"] = true
	ctx.Data["AppSubURL"] = setting.AppSubURL

	// Set submenu tab
	ctx.Data["TabIsByMembers"] = true

	// Get the data from the DB
	results, err := getTimesByMembers(unixfrom, unixto, ctx.Org.Organization.ID)
	if err != nil {
		ctx.ServerError("getTimesByMembers", err)
		return
	}
	ctx.Data["results"] = results

	ctx.HTML(http.StatusOK, tplByMembers)
}

// getTimesByMembers gets the actual data from the DB to serve TimesByMembers.
func getTimesByMembers(unixfrom, unixto, orgid int64) (results []resultTimesByMembers, err error) {
	err = db.GetEngine(db.DefaultContext).
		Select("user.name, SUM(tracked_time.time) AS sum_time").
		Table("tracked_time").
		Join("INNER", "issue", "tracked_time.issue_id = issue.id").
		Join("INNER", "repository", "issue.repo_id = repository.id").
		Join("INNER", "user", "tracked_time.user_id = user.id").
		Where(builder.Eq{"repository.owner_id": orgid}).
		And(builder.Eq{"tracked_time.deleted": false}).
		And(builder.Gte{"tracked_time.created_unix": unixfrom}).
		And(builder.Lte{"tracked_time.created_unix": unixto}).
		GroupBy("user.id").
		OrderBy("sum_time DESC").
		Find(&results)
	return results, err
}
