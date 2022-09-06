// SPDX-License-Identifier: MIT

package driver

import (
	"context"
	"fmt"

	"code.gitea.io/gitea/models/db"
	issues_model "code.gitea.io/gitea/models/issues"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/timeutil"
	issue_service "code.gitea.io/gitea/services/issue"

	"lab.forgefriends.org/friendlyforgeformat/gof3/forges/common"
	"lab.forgefriends.org/friendlyforgeformat/gof3/format"
	"lab.forgefriends.org/friendlyforgeformat/gof3/util"
)

type Comment struct {
	issues_model.Comment
}

func CommentConverter(f *issues_model.Comment) *Comment {
	return &Comment{
		Comment: *f,
	}
}

func (o Comment) GetID() int64 {
	return o.Comment.ID
}

func (o *Comment) SetID(id int64) {
	o.Comment.ID = id
}

func (o *Comment) IsNil() bool {
	return o.ID == 0
}

func (o *Comment) Equals(other *Comment) bool {
	return o.Comment.ID == other.Comment.ID
}

func (o *Comment) ToFormat() *format.Comment {
	return &format.Comment{
		IssueIndex: o.Comment.IssueID,
		Common:     format.NewCommon(o.Comment.ID),
		PosterID:   format.NewUserReference(o.Comment.Poster.ID),
		Content:    o.Comment.Content,
		Created:    o.Comment.CreatedUnix.AsTime(),
		Updated:    o.Comment.UpdatedUnix.AsTime(),
	}
}

func (o *Comment) FromFormat(comment *format.Comment) {
	*o = Comment{
		Comment: issues_model.Comment{
			ID:       comment.GetID(),
			PosterID: comment.PosterID.GetID(),
			Poster: &user_model.User{
				ID: comment.PosterID.GetID(),
			},
			Content:     comment.Content,
			CreatedUnix: timeutil.TimeStamp(comment.Created.Unix()),
			UpdatedUnix: timeutil.TimeStamp(comment.Updated.Unix()),
		},
	}
}

type CommentProvider struct {
	g *Forgejo
}

func (o *CommentProvider) ToFormat(ctx context.Context, comment *Comment) *format.Comment {
	return comment.ToFormat()
}

func (o *CommentProvider) FromFormat(ctx context.Context, f *format.Comment) *Comment {
	var comment Comment
	comment.FromFormat(f)
	return &comment
}

func (o *CommentProvider) GetObjects(ctx context.Context, user *User, project *Project, commentable common.ContainerObjectInterface, page int) []*Comment {
	comments, err := issues_model.FindComments(ctx, &issues_model.FindCommentsOptions{
		ListOptions: db.ListOptions{Page: page, PageSize: o.g.perPage},
		RepoID:      project.GetID(),
		IssueID:     commentable.GetID(),
		Type:        issues_model.CommentTypeComment,
	})
	if err != nil {
		panic(fmt.Errorf("error while listing comment: %v", err))
	}

	return util.ConvertMap[*issues_model.Comment, *Comment](comments, CommentConverter)
}

func (o *CommentProvider) ProcessObject(ctx context.Context, user *User, project *Project, commentable common.ContainerObjectInterface, comment *Comment) {
	if err := comment.LoadIssue(ctx); err != nil {
		panic(err)
	}
	if err := comment.LoadPoster(ctx); err != nil {
		panic(err)
	}
}

func (o *CommentProvider) Get(ctx context.Context, user *User, project *Project, commentable common.ContainerObjectInterface, comment *Comment) *Comment {
	id := comment.GetID()
	c, err := issues_model.GetCommentByID(ctx, id)
	if issues_model.IsErrCommentNotExist(err) {
		return &Comment{}
	}
	if err != nil {
		panic(err)
	}

	co := CommentConverter(c)
	o.ProcessObject(ctx, user, project, commentable, co)
	return co
}

func (o *CommentProvider) Put(ctx context.Context, user *User, project *Project, commentable common.ContainerObjectInterface, comment *Comment) *Comment {
	var issue *issues_model.Issue
	switch c := commentable.(type) {
	case *PullRequest:
		issue = c.PullRequest.Issue
	case *Issue:
		issue = &c.Issue
	default:
		panic(fmt.Errorf("unexpected type %T", commentable))
	}
	c, err := issue_service.CreateIssueComment(ctx, o.g.GetDoer(), &project.Repository, issue, comment.Content, nil)
	if err != nil {
		panic(err)
	}
	return o.Get(ctx, user, project, commentable, CommentConverter(c))
}

func (o *CommentProvider) Delete(ctx context.Context, user *User, project *Project, commentable common.ContainerObjectInterface, comment *Comment) *Comment {
	c := o.Get(ctx, user, project, commentable, comment)
	if !c.IsNil() {
		err := issues_model.DeleteComment(ctx, &c.Comment)
		if err != nil {
			panic(err)
		}
	}
	return c
}
