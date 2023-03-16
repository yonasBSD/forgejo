// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package webhook

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	webhook_model "code.gitea.io/gitea/models/webhook"
	api "code.gitea.io/gitea/modules/structs"
	webhook_module "code.gitea.io/gitea/modules/webhook"
)

type request struct {
	Method string
	URL    string
	Body   []byte
	Header http.Header
	// AddDefaultHeaders will add the default X- headers to the request (event type, id, signature...)
	AddDefaultHeaders bool
}

// requestConvertor defines the interface to create serializable request from a system webhook payload
type requestConvertor interface {
	Create(*api.CreatePayload) (request, error)
	Delete(*api.DeletePayload) (request, error)
	Fork(*api.ForkPayload) (request, error)
	Issue(*api.IssuePayload) (request, error)
	IssueComment(*api.IssueCommentPayload) (request, error)
	Push(*api.PushPayload) (request, error)
	PullRequest(*api.PullRequestPayload) (request, error)
	Review(*api.PullRequestPayload, webhook_module.HookEventType) (request, error)
	Repository(*api.RepositoryPayload) (request, error)
	Release(*api.ReleasePayload) (request, error)
	Wiki(*api.WikiPayload) (request, error)
	Package(*api.PackagePayload) (request, error)
}

func convertRequest(rc requestConvertor, p api.Payloader, event webhook_module.HookEventType) (request, error) {
	switch event {
	case webhook_module.HookEventCreate:
		return rc.Create(p.(*api.CreatePayload))
	case webhook_module.HookEventDelete:
		return rc.Delete(p.(*api.DeletePayload))
	case webhook_module.HookEventFork:
		return rc.Fork(p.(*api.ForkPayload))
	case webhook_module.HookEventIssues, webhook_module.HookEventIssueAssign, webhook_module.HookEventIssueLabel, webhook_module.HookEventIssueMilestone:
		return rc.Issue(p.(*api.IssuePayload))
	case webhook_module.HookEventIssueComment, webhook_module.HookEventPullRequestComment:
		pl, ok := p.(*api.IssueCommentPayload)
		if ok {
			return rc.IssueComment(pl)
		}
		return rc.PullRequest(p.(*api.PullRequestPayload))
	case webhook_module.HookEventPush:
		return rc.Push(p.(*api.PushPayload))
	case webhook_module.HookEventPullRequest, webhook_module.HookEventPullRequestAssign, webhook_module.HookEventPullRequestLabel,
		webhook_module.HookEventPullRequestMilestone, webhook_module.HookEventPullRequestSync, webhook_module.HookEventPullRequestReviewRequest:
		return rc.PullRequest(p.(*api.PullRequestPayload))
	case webhook_module.HookEventPullRequestReviewApproved, webhook_module.HookEventPullRequestReviewRejected, webhook_module.HookEventPullRequestReviewComment:
		return rc.Review(p.(*api.PullRequestPayload), event)
	case webhook_module.HookEventRepository:
		return rc.Repository(p.(*api.RepositoryPayload))
	case webhook_module.HookEventRelease:
		return rc.Release(p.(*api.ReleasePayload))
	case webhook_module.HookEventWiki:
		return rc.Wiki(p.(*api.WikiPayload))
	case webhook_module.HookEventPackage:
		return rc.Package(p.(*api.PackagePayload))
	}
	return request{}, fmt.Errorf("newRequest unsupported event: %s", event)
}

func (r request) createHookTask(ctx context.Context, hookID int64, event webhook_module.HookEventType) (*webhook_model.HookTask, error) {
	requestHeader := ""
	if len(r.Header) > 0 {
		var header strings.Builder
		err := r.Header.Write(&header)
		if err != nil {
			return nil, fmt.Errorf("serialize request header for %d[%s]: %w", hookID, event, err)
		}
		requestHeader = header.String()
	}

	task := &webhook_model.HookTask{
		HookID:    hookID,
		EventType: event,

		RequestMethod:     r.Method,
		RequestURL:        r.URL,
		RequestHeader:     requestHeader,
		PayloadContent:    string(r.Body),
		AddDefaultHeaders: r.AddDefaultHeaders,
	}
	return webhook_model.CreateHookTask(ctx, task)
}
