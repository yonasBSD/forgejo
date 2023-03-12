// Copyright 2023 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT
package issue

import (
	"context"

	issues_model "code.gitea.io/gitea/models/issues"
	user_model "code.gitea.io/gitea/models/user"
)

// CreateIssueReaction creates a reaction on issue.
func CreateIssueReaction(ctx context.Context, doer *user_model.User, issue *issues_model.Issue, content string) (*issues_model.Reaction, error) {
	// Check if the doer is blocked by the issue's poster.
	if user_model.IsBlocked(ctx, issue.PosterID, doer.ID) {
		return nil, user_model.ErrBlockedByUser
	}

	return issues_model.CreateReaction(ctx, &issues_model.ReactionOptions{
		Type:    content,
		DoerID:  doer.ID,
		IssueID: issue.ID,
	})
}

// CreateCommentReaction creates a reaction on comment.
func CreateCommentReaction(ctx context.Context, doer *user_model.User, issue *issues_model.Issue, comment *issues_model.Comment, content string) (*issues_model.Reaction, error) {
	// Check if the doer is blocked by the issue's poster or by the comment's poster.
	if user_model.IsBlocked(ctx, comment.PosterID, doer.ID) || user_model.IsBlocked(ctx, issue.PosterID, doer.ID) {
		return nil, user_model.ErrBlockedByUser
	}

	return issues_model.CreateReaction(ctx, &issues_model.ReactionOptions{
		Type:      content,
		DoerID:    doer.ID,
		IssueID:   issue.ID,
		CommentID: comment.ID,
	})
}
