// Copyright 2019 The Gitea Authors. All rights reserved.
// Copyright 2018 Jonas Franz. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package base

import (
	"time"

	"code.gitea.io/gitea/modules/structs"
)

// Downloader downloads the site repo informations
type Downloader interface {
	GetRepoInfo() (*Repository, error)
	GetTopics() ([]string, error)
	GetMilestones() ([]*Milestone, error)
	GetReleases() ([]*Release, error)
	GetLabels() ([]*Label, error)
	GetIssues(page, perPage int) ([]*Issue, bool, error)
	GetComments(issueNumber int64) ([]*Comment, error)
	GetPullRequests(page, perPage int) ([]*PullRequest, error)
	GetRequestTimes() int
	GetRequestLimit() float32
}

// DownloaderFactory defines an interface to match a downloader implementation and create a downloader
type DownloaderFactory interface {
	Match(opts MigrateOptions) (bool, error)
	New(opts MigrateOptions) (Downloader, error)
	GitServiceType() structs.GitServiceType
}

var (
	_ Downloader = &LimitDownloader{}
)

// RetryDownloader retry the downloads
type RetryDownloader struct {
	Downloader
	RetryTimes int       // the total execute times
	RetryDelay int       // time to delay seconds
	startTime  time.Time // start time
}

// NewRetryDownloader creates a retry downloader
func NewRetryDownloader(downloader Downloader, retryTimes, retryDelay int) *RetryDownloader {
	return &RetryDownloader{
		Downloader: downloader,
		RetryTimes: retryTimes,
		RetryDelay: retryDelay,
		startTime:  time.Now(),
	}
}

// GetRequestTimes returns request times
func (d *RetryDownloader) GetRequestTimes() int {
	return d.Downloader.GetRequestTimes()
}

// GetRequestLimit returns the limitation of the http request times per seconds.
func (d *RetryDownloader) GetRequestLimit() float32 {
	return d.Downloader.GetRequestLimit()
}

// GetRepoInfo returns a repository information with retry
func (d *RetryDownloader) GetRepoInfo() (*Repository, error) {
	var (
		times = d.RetryTimes
		repo  *Repository
		err   error
	)
	for ; times > 0; times-- {
		if repo, err = d.Downloader.GetRepoInfo(); err == nil {
			return repo, nil
		}
		time.Sleep(time.Second * time.Duration(d.RetryDelay))
	}
	return nil, err
}

// GetTopics returns a repository's topics with retry
func (d *RetryDownloader) GetTopics() ([]string, error) {
	var (
		times  = d.RetryTimes
		topics []string
		err    error
	)
	for ; times > 0; times-- {
		if topics, err = d.Downloader.GetTopics(); err == nil {
			return topics, nil
		}
		time.Sleep(time.Second * time.Duration(d.RetryDelay))
	}
	return nil, err
}

// GetMilestones returns a repository's milestones with retry
func (d *RetryDownloader) GetMilestones() ([]*Milestone, error) {
	var (
		times      = d.RetryTimes
		milestones []*Milestone
		err        error
	)
	for ; times > 0; times-- {
		if milestones, err = d.Downloader.GetMilestones(); err == nil {
			return milestones, nil
		}
		time.Sleep(time.Second * time.Duration(d.RetryDelay))
	}
	return nil, err
}

// GetReleases returns a repository's releases with retry
func (d *RetryDownloader) GetReleases() ([]*Release, error) {
	var (
		times    = d.RetryTimes
		releases []*Release
		err      error
	)
	for ; times > 0; times-- {
		if releases, err = d.Downloader.GetReleases(); err == nil {
			return releases, nil
		}
		time.Sleep(time.Second * time.Duration(d.RetryDelay))
	}
	return nil, err
}

// GetLabels returns a repository's labels with retry
func (d *RetryDownloader) GetLabels() ([]*Label, error) {
	var (
		times  = d.RetryTimes
		labels []*Label
		err    error
	)
	for ; times > 0; times-- {
		if labels, err = d.Downloader.GetLabels(); err == nil {
			return labels, nil
		}
		time.Sleep(time.Second * time.Duration(d.RetryDelay))
	}
	return nil, err
}

// GetIssues returns a repository's issues with retry
func (d *RetryDownloader) GetIssues(page, perPage int) ([]*Issue, bool, error) {
	var (
		times  = d.RetryTimes
		issues []*Issue
		isEnd  bool
		err    error
	)
	for ; times > 0; times-- {
		if issues, isEnd, err = d.Downloader.GetIssues(page, perPage); err == nil {
			return issues, isEnd, nil
		}
		time.Sleep(time.Second * time.Duration(d.RetryDelay))
	}
	return nil, false, err
}

// GetComments returns a repository's comments with retry
func (d *RetryDownloader) GetComments(issueNumber int64) ([]*Comment, error) {
	var (
		times    = d.RetryTimes
		comments []*Comment
		err      error
	)
	for ; times > 0; times-- {
		if comments, err = d.Downloader.GetComments(issueNumber); err == nil {
			return comments, nil
		}
		time.Sleep(time.Second * time.Duration(d.RetryDelay))
	}
	return nil, err
}

// GetPullRequests returns a repository's pull requests with retry
func (d *RetryDownloader) GetPullRequests(page, perPage int) ([]*PullRequest, error) {
	var (
		times = d.RetryTimes
		prs   []*PullRequest
		err   error
	)
	for ; times > 0; times-- {
		if prs, err = d.Downloader.GetPullRequests(page, perPage); err == nil {
			return prs, nil
		}
		time.Sleep(time.Second * time.Duration(d.RetryDelay))
	}
	return nil, err
}

var (
	_ Downloader = &LimitDownloader{}
)

// LimitDownloader limit the downloads
type LimitDownloader struct {
	Downloader
	startTime time.Time // start time
}

// NewLimitDownloader creates a limit downloader
func NewLimitDownloader(downloader Downloader) *LimitDownloader {
	return &LimitDownloader{
		Downloader: downloader,
		startTime:  time.Now(),
	}
}

// GetRequestTimes returns request times
func (d *LimitDownloader) GetRequestTimes() int {
	return d.Downloader.GetRequestTimes()
}

// GetRequestLimit returns the limitation of the http request times per seconds.
func (d *LimitDownloader) GetRequestLimit() float32 {
	return d.Downloader.GetRequestLimit()
}

func (d *LimitDownloader) sleep() {
	secs := int(time.Now().Sub(d.startTime) / time.Second)
	if secs > 0 && float32(d.GetRequestTimes()/secs) > d.GetRequestLimit() {
		time.Sleep(time.Second * time.Duration(int(float32(d.GetRequestTimes())/d.GetRequestLimit())-secs))
	}
}

// GetRepoInfo returns a repository information with retry
func (d *LimitDownloader) GetRepoInfo() (*Repository, error) {
	d.sleep()
	return d.Downloader.GetRepoInfo()
}

// GetTopics returns a repository's topics with retry
func (d *LimitDownloader) GetTopics() ([]string, error) {
	d.sleep()
	return d.Downloader.GetTopics()
}

// GetMilestones returns a repository's milestones with retry
func (d *LimitDownloader) GetMilestones() ([]*Milestone, error) {
	d.sleep()
	return d.Downloader.GetMilestones()
}

// GetReleases returns a repository's releases with retry
func (d *LimitDownloader) GetReleases() ([]*Release, error) {
	d.sleep()
	return d.Downloader.GetReleases()
}

// GetLabels returns a repository's labels with retry
func (d *LimitDownloader) GetLabels() ([]*Label, error) {
	d.sleep()
	return d.Downloader.GetLabels()
}

// GetIssues returns a repository's issues with retry
func (d *LimitDownloader) GetIssues(page, perPage int) ([]*Issue, bool, error) {
	d.sleep()
	return d.Downloader.GetIssues(page, perPage)
}

// GetComments returns a repository's comments with retry
func (d *LimitDownloader) GetComments(issueNumber int64) ([]*Comment, error) {
	d.sleep()
	return d.Downloader.GetComments(issueNumber)
}

// GetPullRequests returns a repository's pull requests with retry
func (d *LimitDownloader) GetPullRequests(page, perPage int) ([]*PullRequest, error) {
	d.sleep()
	return d.Downloader.GetPullRequests(page, perPage)
}
