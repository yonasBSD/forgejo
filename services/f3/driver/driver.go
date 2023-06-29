// SPDX-License-Identifier: MIT

package driver

import (
	"fmt"

	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/services/migrations"

	"lab.forgefriends.org/friendlyforgeformat/gof3"
	"lab.forgefriends.org/friendlyforgeformat/gof3/forges/common"
	"lab.forgefriends.org/friendlyforgeformat/gof3/forges/driver"
	"lab.forgefriends.org/friendlyforgeformat/gof3/format"
)

type Options struct {
	gof3.Options

	Doer *user_model.User
}

type Forgejo struct {
	perPage int
	options *Options
}

func (o *Forgejo) GetPerPage() int {
	return o.perPage
}

func (o *Forgejo) GetOptions() gof3.OptionsInterface {
	return o.options
}

func (o *Forgejo) SetOptions(options gof3.OptionsInterface) {
	var ok bool
	o.options, ok = options.(*Options)
	if !ok {
		panic(fmt.Errorf("unexpected type %T", options))
	}
}

func (o *Forgejo) GetLogger() *gof3.Logger {
	return o.GetOptions().GetLogger()
}

func (o *Forgejo) Init(options gof3.OptionsInterface) {
	o.SetOptions(options)
	o.perPage = setting.ItemsPerPage
}

func (o *Forgejo) GetDirectory() string {
	return o.options.GetDirectory()
}

func (o *Forgejo) GetDoer() *user_model.User {
	return o.options.Doer
}

func (o *Forgejo) GetNewMigrationHTTPClient() gof3.NewMigrationHTTPClientFun {
	return migrations.NewMigrationHTTPClient
}

func (o *Forgejo) SupportGetRepoComments() bool {
	return false
}

func (o *Forgejo) GetProvider(name string, parent common.ProviderInterface) common.ProviderInterface {
	var parentImpl any
	if parent != nil {
		parentImpl = parent.GetImplementation()
	}
	switch name {
	case driver.ProviderUser:
		return driver.NewProvider[UserProvider, *UserProvider, User, *User, format.User, *format.User](driver.ProviderUser, NewProvider[UserProvider](o))
	case driver.ProviderProject:
		return driver.NewProviderWithParentOne[ProjectProvider, *ProjectProvider, Project, *Project, format.Project, *format.Project, User, *User](driver.ProviderProject, NewProvider[ProjectProvider, *ProjectProvider](o))
	case driver.ProviderMilestone:
		return driver.NewProviderWithParentOneTwo[MilestoneProvider, *MilestoneProvider, Milestone, *Milestone, format.Milestone, *format.Milestone, User, *User, Project, *Project](driver.ProviderMilestone, NewProviderWithProjectProvider[MilestoneProvider](o, parentImpl.(*ProjectProvider)))
	case driver.ProviderIssue:
		return driver.NewProviderWithParentOneTwo[IssueProvider, *IssueProvider, Issue, *Issue, format.Issue, *format.Issue, User, *User, Project, *Project](driver.ProviderIssue, NewProviderWithProjectProvider[IssueProvider](o, parentImpl.(*ProjectProvider)))
	case driver.ProviderPullRequest:
		return driver.NewProviderWithParentOneTwo[PullRequestProvider, *PullRequestProvider, PullRequest, *PullRequest, format.PullRequest, *format.PullRequest, User, *User, Project, *Project](driver.ProviderPullRequest, NewProviderWithProjectProvider[PullRequestProvider](o, parentImpl.(*ProjectProvider)))
	case driver.ProviderReview:
		return driver.NewProviderWithParentOneTwoThree[ReviewProvider, *ReviewProvider, Review, *Review, format.Review, *format.Review, User, *User, Project, *Project, PullRequest, *PullRequest](driver.ProviderReview, NewProvider[ReviewProvider](o))
	case driver.ProviderRepository:
		return driver.NewProviderWithParentOneTwo[RepositoryProvider, *RepositoryProvider, Repository, *Repository, format.Repository, *format.Repository, User, *User, Project, *Project](driver.ProviderRepository, NewProvider[RepositoryProvider](o))
	case driver.ProviderTopic:
		return driver.NewProviderWithParentOneTwo[TopicProvider, *TopicProvider, Topic, *Topic, format.Topic, *format.Topic, User, *User, Project, *Project](driver.ProviderTopic, NewProvider[TopicProvider](o))
	case driver.ProviderLabel:
		return driver.NewProviderWithParentOneTwo[LabelProvider, *LabelProvider, Label, *Label, format.Label, *format.Label, User, *User, Project, *Project](driver.ProviderLabel, NewProviderWithProjectProvider[LabelProvider](o, parentImpl.(*ProjectProvider)))
	case driver.ProviderRelease:
		return driver.NewProviderWithParentOneTwo[ReleaseProvider, *ReleaseProvider, Release, *Release, format.Release, *format.Release, User, *User, Project, *Project](driver.ProviderRelease, NewProvider[ReleaseProvider](o))
	case driver.ProviderAsset:
		return driver.NewProviderWithParentOneTwoThree[AssetProvider, *AssetProvider, Asset, *Asset, format.ReleaseAsset, *format.ReleaseAsset, User, *User, Project, *Project, Release, *Release](driver.ProviderAsset, NewProvider[AssetProvider](o))
	case driver.ProviderComment:
		return driver.NewProviderWithParentOneTwoThreeInterface[CommentProvider, *CommentProvider, Comment, *Comment, format.Comment, *format.Comment, User, *User, Project, *Project](driver.ProviderComment, NewProvider[CommentProvider](o))
	case driver.ProviderReaction:
		return driver.NewProviderWithParentOneTwoRest[ReactionProvider, *ReactionProvider, Reaction, *Reaction, format.Reaction, *format.Reaction, User, *User, Project, *Project](driver.ProviderReaction, NewProvider[ReactionProvider](o))
	default:
		panic(fmt.Sprintf("unknown provider name %s", name))
	}
}

func (o Forgejo) Finish() {
}
