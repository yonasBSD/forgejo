// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package webhook

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strings"

	webhook_model "forgejo.org/models/webhook"
	"forgejo.org/modules/git"
	api "forgejo.org/modules/structs"
	"forgejo.org/modules/util"
	webhook_module "forgejo.org/modules/webhook"
	"forgejo.org/services/forms"
	"forgejo.org/services/webhook/shared"
)

type msteamsHandler struct{}

func (msteamsHandler) Type() webhook_module.HookType       { return webhook_module.MSTEAMS }
func (msteamsHandler) Metadata(*webhook_model.Webhook) any { return nil }
func (msteamsHandler) Icon(size int) template.HTML         { return shared.ImgIcon("msteams.png", size) }

func (msteamsHandler) UnmarshalForm(bind func(any)) forms.WebhookForm {
	var form struct {
		forms.WebhookCoreForm
		PayloadURL string `binding:"Required;ValidUrl"`
	}
	bind(&form)

	return forms.WebhookForm{
		WebhookCoreForm: form.WebhookCoreForm,
		URL:             form.PayloadURL,
		ContentType:     webhook_model.ContentTypeJSON,
		Secret:          "",
		HTTPMethod:      http.MethodPost,
		Metadata:        nil,
	}
}

type (
	// MSTeamsAction defines an action for an Adaptive Card, for example, opening a URL.
	MSTeamsAction struct {
		Type  string `json:"type"`  // Must be "Action.OpenUrl"
		Title string `json:"title"` // The button title
		URL   string `json:"url"`   // The URL to open (renamed from "Url" to "URL")
	}

	// MSTeamsTextBlock defines the Adaptive Card TextBlock element.
	MSTeamsTextBlock struct {
		Type         string         `json:"type"`                          // Must be "TextBlock"
		Text         string         `json:"text"`                          // The text content
		Size         string         `json:"size,omitempty"`                // e.g., "Small", "Default", "Medium", "Large", "ExtraLarge"
		Weight       string         `json:"weight,omitempty"`              // e.g., "Lighter", "Default", "Bolder"
		Color        string         `json:"color,omitempty"`               // text color, e.g., "Default", "Dark", "Light", "Accent", "Good", "Warning", "Attention"
		IsSubtle     bool           `json:"isSubtle,omitempty"`            // Optional: makes text less prominent
		Wrap         bool           `json:"wrap,omitempty"`                // Optional: enables text wrapping
		MaxLines     int            `json:"maxLines,omitempty"`            // Optional: maximum number of lines to display
		HorizAlign   string         `json:"horizontalAlignment,omitempty"` // e.g., "Left", "Center", "Right"
		Spacing      string         `json:"spacing,omitempty"`             // e.g., "None", "Small", "Default", "Medium", "Large", "ExtraLarge", "Padding"
		FontType     string         `json:"fontType,omitempty"`            // e.g., "Default", "Monospace"
		Style        string         `json:"style,omitempty"`               // e.g., "default","columnHeader","heading"
		SelectAction *MSTeamsAction `json:"selectAction,omitempty"`        // Optional: action when the TextBlock is clicked
	}

	// MSTeamsImage defines the Adaptive Card Image element.
	MSTeamsImage struct {
		Type         string         `json:"type"`                   // Must be "Image"
		URL          string         `json:"url"`                    // URL of the image
		Alt          string         `json:"altText"`                // Alt text for the image
		Size         string         `json:"size,omitempty"`         // e.g., "Auto", "Stretch", "Small", "Medium", "Large"
		Style        string         `json:"style,omitempty"`        // e.g., "Default", "Person", "RoundedCorners"
		SelectAction *MSTeamsAction `json:"selectAction,omitempty"` // Optional: action when the columnset is clicked
	}

	// MSTeamsColumn defines a column in an Adaptive Card ColumnSet.
	MSTeamsColumn struct {
		Type          string         `json:"type"`                               // Must be "Column"
		Items         []any          `json:"items,omitempty"`                    // Array of card elements (TextBlock, Image, etc.)
		Width         any            `json:"width,omitempty"`                    // "auto", "stretch", or number/string for fixed width
		Style         string         `json:"style,omitempty"`                    // e.g., "default", "emphasis"
		VerticalAlign string         `json:"verticalContentAlignment,omitempty"` // "top", "center", "bottom"
		Bleed         bool           `json:"bleed,omitempty"`                    // Optional: allow content to bleed outside padding
		Separator     bool           `json:"separator,omitempty"`                // Optional: draw a separating line
		Spacing       string         `json:"spacing,omitempty"`                  // e.g., "none", "small", "default", "medium", "large", "extraLarge", "padding"
		SelectAction  *MSTeamsAction `json:"selectAction,omitempty"`             // Optional: action when column is clicked
	}

	// MSTeamsColumnSet defines a row of columns in an Adaptive Card.
	MSTeamsColumnSet struct {
		Type         string          `json:"type"`                   // Must be "ColumnSet"
		Columns      []MSTeamsColumn `json:"columns"`                // Array of columns
		Spacing      string          `json:"spacing,omitempty"`      // e.g., "none", "small", "default", "medium", "large", "extraLarge", "padding"
		Separator    bool            `json:"separator,omitempty"`    // Optional: draw a separating line
		Bleed        bool            `json:"bleed,omitempty"`        // Optional: allow content to bleed outside padding
		SelectAction *MSTeamsAction  `json:"selectAction,omitempty"` // Optional: action when columnset is clicked
	}

	// MSTeamsFact represents a fact in an Adaptive Card FactSet.
	MSTeamsFact struct {
		Title string `json:"title"`
		Value string `json:"value"`
	}

	// MSTeamsFactSet represents the Adaptive Card FactSet element.
	MSTeamsFactSet struct {
		Type  string        `json:"type"`  // Must be "FactSet"
		Facts []MSTeamsFact `json:"facts"` // List of facts
	}

	// MSTeamsContainer corresponds to an Adaptive Card container.
	// It can hold one or more TextBlock elements and an optional FactSet.
	MSTeamsContainer struct {
		Type           string `json:"type"`                     // Must be "Container"
		Items          []any  `json:"items,omitempty"`          // Array of card elements (TextBlock, Image, FactSet, etc.)
		ShowBorder     bool   `json:"showBorder,omitempty"`     // Optional: draw a border around the container
		RoundedCorners bool   `json:"roundedCorners,omitempty"` // Optional: round the corners of the container
		Spacing        string `json:"spacing,omitempty"`        // e.g., "None", "ExtraSmall", "Small", "Default", "Medium", "Large", "ExtraLarge", "Padding"
		Bleed          bool   `json:"bleed,omitempty"`          // Optional: allow content to bleed outside padding
		Style          string `json:"style,omitempty"`          // color theme of the container, e.g. "default", "emphasis", "accent", "good", "attention", "warning"
	}

	MSTeamsOptions struct {
		Width string `json:"width"` // use "Full" to make card full width
	}

	// MSTeamsPayload represents the Adaptive Card payload.
	// Adaptive Cards use "body" for visual elements and "actions" for interactive buttons.
	MSTeamsPayload struct {
		Type    string             `json:"type"`              // Must be "AdaptiveCard"
		Schema  string             `json:"$schema"`           // e.g., "http://adaptivecards.io/schemas/adaptive-card.json"
		Version string             `json:"version"`           // e.g., "1.5"
		MsTeams MSTeamsOptions     `json:"msteams"`           // Optional: settings for Microsoft Teams
		Body    []MSTeamsContainer `json:"body"`              // Array of containers (sections)
		Actions []MSTeamsAction    `json:"actions,omitempty"` // Optional array of actions
	}
)

var (
	defaultStyle   = "default"   // default colour
	emphasisStyle  = "emphasis"  // darker
	accentStyle    = "accent"    // blue
	goodStyle      = "good"      // green
	attentionStyle = "attention" // red
	warningStyle   = "warning"   // yellow
)

func markdownLinkFormatter(url, text string) string {
	if url == "" {
		return text
	}
	return fmt.Sprintf(`[%s](%s)`, text, url)
}

// Create implements PayloadConvertor Create method
func (m msteamsConvertor) Create(p *api.CreatePayload) (MSTeamsPayload, error) {
	// created tag/branch
	refName := git.RefName(p.Ref).ShortName()

	return createMSTeamsPayload(
		p.Repo,
		p.Sender,
		fmt.Sprintf("created a new %s '%s'", p.RefType, refName),
		nil,
		p.Repo.HTMLURL+"/src/"+util.PathEscapeSegments(refName),
	), nil
}

// Delete implements PayloadConvertor Delete method
func (m msteamsConvertor) Delete(p *api.DeletePayload) (MSTeamsPayload, error) {
	// deleted tag/branch
	refName := git.RefName(p.Ref).ShortName()
	actionTitle := fmt.Sprintf("deleted %s '%s'", p.RefType, refName)

	bodySections := []MSTeamsContainer{
		{
			Type:  "Container",
			Style: attentionStyle,
			Items: []any{
				MSTeamsTextBlock{
					Type: "TextBlock",
					Text: fmt.Sprintf("The %s '%s' was deleted.", p.RefType, refName),
				},
			},
		},
	}

	return createMSTeamsPayload(
		p.Repo,
		p.Sender,
		actionTitle,
		bodySections,
		p.Repo.HTMLURL+"/src/"+util.PathEscapeSegments(refName),
	), nil
}

// Fork implements PayloadConvertor Fork method
func (m msteamsConvertor) Fork(p *api.ForkPayload) (MSTeamsPayload, error) {
	actionTitle := fmt.Sprintf("forked %s to %s", p.Forkee.FullName, p.Repo.FullName)

	bodySections := []MSTeamsContainer{
		{
			Type:  "Container",
			Style: emphasisStyle,
			Items: []any{
				MSTeamsTextBlock{
					Type: "TextBlock",
					Text: fmt.Sprintf("Repository forked from %s to %s.", p.Forkee.FullName, p.Repo.FullName),
				},
			},
		},
	}

	return createMSTeamsPayload(
		p.Repo,
		p.Sender,
		actionTitle,
		bodySections,
		p.Repo.HTMLURL,
	), nil
}

// Push implements PayloadConvertor Push method
func (m msteamsConvertor) Push(p *api.PushPayload) (MSTeamsPayload, error) {
	var (
		branchName  = git.RefName(p.Ref).ShortName()
		actionTitle string
	)

	var titleLink string
	if p.TotalCommits == 1 {
		actionTitle = fmt.Sprintf("pushed 1 new commit to %s", branchName)
		titleLink = p.Commits[0].URL
	} else {
		actionTitle = fmt.Sprintf("pushed %d new commits to %s", p.TotalCommits, branchName)
		titleLink = p.CompareURL
	}
	if titleLink == "" {
		titleLink = p.Repo.HTMLURL + "/src/" + util.PathEscapeSegments(branchName)
	}

	var textBlocks []any
	limit := 5
	for i, commit := range p.Commits {
		if i >= limit {
			break
		}
		textBlocks = append(textBlocks, MSTeamsContainer{
			Type:       "Container",
			ShowBorder: true,
			Spacing:    "Small",
			Items: []any{
				MSTeamsTextBlock{
					Type: "TextBlock",
					Text: fmt.Sprintf("%s %s - %s", markdownLinkFormatter(commit.URL, commit.ID[:7]),
						strings.TrimRight(commit.Message, "\r\n"), commit.Author.Name),
					Wrap:     true,
					Size:     "Small",
					Spacing:  "None",
					MaxLines: 3,
				},
			},
		})
	}
	if len(p.Commits) > limit {
		var extraCommitText string
		if (len(p.Commits) - limit) == 1 {
			extraCommitText = "and 1 more commit"
		} else {
			extraCommitText = fmt.Sprintf("and %d more commits", len(p.Commits)-limit)
		}

		textBlocks = append(textBlocks, MSTeamsTextBlock{
			Type: "TextBlock",
			Text: extraCommitText,
			Size: "Small",
		})
	}

	bodySections := []MSTeamsContainer{
		{
			Type:  "Container",
			Items: textBlocks,
		},
	}

	return createMSTeamsPayload(
		p.Repo,
		p.Sender,
		actionTitle,
		bodySections,
		titleLink,
	), nil
}

// Issue implements PayloadConvertor Issue method
func (m msteamsConvertor) Issue(p *api.IssuePayload) (MSTeamsPayload, error) {
	issueIndex := fmt.Sprintf("#%d", p.Index)
	var actionTitle string
	style := emphasisStyle

	switch p.Action {
	case api.HookIssueOpened:
		actionTitle = fmt.Sprintf("opened issue %s", issueIndex)
		style = accentStyle
	case api.HookIssueClosed:
		actionTitle = fmt.Sprintf("closed issue %s", issueIndex)
		style = warningStyle
	case api.HookIssueReOpened:
		actionTitle = fmt.Sprintf("re-opened issue %s", issueIndex)
	case api.HookIssueEdited:
		actionTitle = fmt.Sprintf("edited issue %s", issueIndex)
	case api.HookIssueAssigned:
		list := make([]string, len(p.Issue.Assignees))
		for i, user := range p.Issue.Assignees {
			list[i] = user.UserName
		}
		actionTitle = fmt.Sprintf("assigned issue %s to %s", issueIndex, strings.Join(list, ", "))
		style = accentStyle
	case api.HookIssueUnassigned:
		actionTitle = fmt.Sprintf("unassigned issue %s", issueIndex)
	case api.HookIssueLabelUpdated:
		actionTitle = fmt.Sprintf("updated labels in issue %s", issueIndex)
	case api.HookIssueLabelCleared:
		actionTitle = fmt.Sprintf("cleared labels in issue %s", issueIndex)
	case api.HookIssueSynchronized:
		actionTitle = fmt.Sprintf("synchronized issue %s", issueIndex)
	case api.HookIssueMilestoned:
		actionTitle = fmt.Sprintf("milestoned issue %s to %s", issueIndex, p.Issue.Milestone.Title)
	case api.HookIssueDemilestoned:
		actionTitle = fmt.Sprintf("cleared milestone on issue %s", issueIndex)
	}

	bodySections := []MSTeamsContainer{
		{
			Type:  "Container",
			Style: style,
			Items: []any{
				MSTeamsTextBlock{
					Type:   "TextBlock",
					Text:   markdownLinkFormatter(p.Issue.HTMLURL, fmt.Sprintf("Issue %s: %s", issueIndex, p.Issue.Title)),
					Wrap:   true,
					Weight: "Bold",
				},
				MSTeamsTextBlock{
					Type: "TextBlock",
					Text: p.Issue.Body,
					Wrap: true,
				},
			},
		},
	}

	return createMSTeamsPayload(
		p.Repository,
		p.Sender,
		actionTitle,
		bodySections,
		p.Issue.HTMLURL,
	), nil
}

// IssueComment implements PayloadConvertor IssueComment method
func (m msteamsConvertor) IssueComment(p *api.IssueCommentPayload) (MSTeamsPayload, error) {
	issueIndex := fmt.Sprintf("#%d", p.Issue.Index)
	issueTitle := fmt.Sprintf("Comment %s on issue %s: %s", p.Action, issueIndex, p.Issue.Title)
	var actionTitle string
	var bodyContentTitle string
	var typ string
	style := emphasisStyle

	if p.IsPull {
		typ = "pull request"
		bodyContentTitle = markdownLinkFormatter(p.Comment.PRURL, issueTitle)
	} else {
		typ = "issue"
		bodyContentTitle = markdownLinkFormatter(p.Comment.IssueURL, issueTitle)
	}

	switch p.Action {
	case api.HookIssueCommentCreated:
		actionTitle = fmt.Sprintf("commented on %s %s", typ, issueIndex)
		style = accentStyle
	case api.HookIssueCommentEdited:
		actionTitle = fmt.Sprintf("edited comment on %s %s", typ, issueIndex)
	case api.HookIssueCommentDeleted:
		actionTitle = fmt.Sprintf("deleted comment on %s %s", typ, issueIndex)
		style = attentionStyle
	}

	bodySections := []MSTeamsContainer{
		{
			Type:  "Container",
			Style: style,
			Items: []any{
				MSTeamsTextBlock{
					Type:   "TextBlock",
					Text:   bodyContentTitle,
					Wrap:   true,
					Weight: "Bold",
				},
				MSTeamsTextBlock{
					Type: "TextBlock",
					Text: p.Comment.Body,
					Wrap: true,
				},
			},
		},
	}

	return createMSTeamsPayload(
		p.Repository,
		p.Sender,
		actionTitle,
		bodySections,
		p.Comment.HTMLURL,
	), nil
}

// PullRequest implements PayloadConvertor PullRequest method
func (m msteamsConvertor) PullRequest(p *api.PullRequestPayload) (MSTeamsPayload, error) {
	prIndex := fmt.Sprintf("#%d", p.Index)
	prTitle := fmt.Sprintf("Pull request %s: %s", prIndex, p.PullRequest.Title)
	var actionTitle string
	var attachmentText string
	style := emphasisStyle

	switch p.Action {
	case api.HookIssueOpened:
		actionTitle = fmt.Sprintf("opened new pull request %s", prIndex)
		prTitle += " is opened"
		attachmentText = p.PullRequest.Body
		style = accentStyle
	case api.HookIssueClosed:
		if p.PullRequest.HasMerged {
			actionTitle = fmt.Sprintf("merged pull request %s", prIndex)
			prTitle += " is merged"
			style = goodStyle
		} else {
			actionTitle = fmt.Sprintf("closed pull request %s", prIndex)
			prTitle += " is closed"
			style = attentionStyle
		}
	case api.HookIssueReOpened:
		actionTitle = fmt.Sprintf("reopened pull request %s", prIndex)
		prTitle += " is reopened"
	case api.HookIssueEdited:
		actionTitle = fmt.Sprintf("edited pull request %s", prIndex)
		prTitle += " is edited"
		attachmentText = p.PullRequest.Body
	case api.HookIssueAssigned:
		list := make([]string, len(p.PullRequest.Assignees))
		for i, user := range p.PullRequest.Assignees {
			list[i] = user.UserName
		}
		actionTitle = fmt.Sprintf("assigned pull request %s to %s", prIndex,
			strings.Join(list, ", "))
		style = accentStyle
		prTitle += " is assigned to " + strings.Join(list, ", ")
	case api.HookIssueUnassigned:
		actionTitle = fmt.Sprintf("unassigned pull request %s", prIndex)
		prTitle += " is unassigned from you"
	case api.HookIssueLabelUpdated:
		actionTitle = fmt.Sprintf("updated labels on pull request %s", prIndex)
		prTitle += " is updated with labels"
	case api.HookIssueLabelCleared:
		actionTitle = fmt.Sprintf("cleared labels on pull request %s", prIndex)
		prTitle += " is cleared of labels"
	case api.HookIssueSynchronized:
		actionTitle = fmt.Sprintf("synchronized pull request %s", prIndex)
		prTitle += " is synchronized"
	case api.HookIssueMilestoned:
		actionTitle = fmt.Sprintf("milestoned pull request %s to %s", prIndex, p.PullRequest.Milestone.Title)
		prTitle += " is milestoned to " + p.PullRequest.Milestone.Title
	case api.HookIssueDemilestoned:
		actionTitle = fmt.Sprintf("cleared milestone on pull request %s", prIndex)
		prTitle += " is cleared of milestone"
	case api.HookIssueReviewed:
		actionTitle = fmt.Sprintf("reviewed pull request %s", prIndex)
		prTitle += " is reviewed"
		attachmentText = p.Review.Content
	case api.HookIssueReviewRequested:
		actionTitle = fmt.Sprintf("requested review on pull request %s", prIndex)
		prTitle += " is requested for review"
	case api.HookIssueReviewRequestRemoved:
		actionTitle = fmt.Sprintf("removed review request on pull request %s", prIndex)
		prTitle += " is removed from review request"
	}

	bodySections := []MSTeamsContainer{
		{
			Type:  "Container",
			Style: style,
			Items: []any{
				MSTeamsTextBlock{
					Type:   "TextBlock",
					Text:   markdownLinkFormatter(p.PullRequest.HTMLURL, prTitle),
					Wrap:   true,
					Weight: "Bold",
				},
			},
		},
	}
	if attachmentText != "" {
		bodySections[0].Items = append(bodySections[0].Items, MSTeamsTextBlock{
			Type: "TextBlock",
			Text: attachmentText,
			Wrap: true,
		})
	}

	return createMSTeamsPayload(
		p.Repository,
		p.Sender,
		actionTitle,
		bodySections,
		p.PullRequest.HTMLURL,
	), nil
}

// Review implements PayloadConvertor Review method
func (m msteamsConvertor) Review(p *api.PullRequestPayload, event webhook_module.HookEventType) (MSTeamsPayload, error) {
	var actionTitle string
	var bodySections []MSTeamsContainer
	style := emphasisStyle

	if p.Action == api.HookIssueReviewed {
		action, err := parseHookPullRequestEventType(event)
		if err != nil {
			return MSTeamsPayload{}, err
		}

		actionTitle = fmt.Sprintf("reviewed %s on pull request #%d", action, p.Index)
		//p.PullRequest.Title

		switch event {
		case webhook_module.HookEventPullRequestReviewApproved:
			style = goodStyle
		case webhook_module.HookEventPullRequestReviewRejected:
			style = attentionStyle
		}

		bodySections = []MSTeamsContainer{
			{
				Type:  "Container",
				Style: style,
				Items: []any{
					MSTeamsTextBlock{
						Type: "TextBlock",
						Text: markdownLinkFormatter(
							p.PullRequest.HTMLURL,
							fmt.Sprintf("Review %s on pull request #%d: %s", action, p.Index, p.PullRequest.Title),
						),
						Wrap:   true,
						Weight: "Bold",
					},
					MSTeamsTextBlock{
						Type: "TextBlock",
						Text: p.Review.Content,
						Wrap: true,
					},
				},
			},
		}
	}

	return createMSTeamsPayload(
		p.Repository,
		p.Sender,
		actionTitle,
		bodySections,
		p.PullRequest.HTMLURL,
	), nil
}

// Repository implements PayloadConvertor Repository method
func (m msteamsConvertor) Repository(p *api.RepositoryPayload) (MSTeamsPayload, error) {
	var actionTitle, url string
	// style := emphasisStyle
	switch p.Action {
	case api.HookRepoCreated:
		actionTitle = fmt.Sprintf("created new repository %s", p.Repository.FullName)
		url = p.Repository.HTMLURL
		// style = goodStyle
	case api.HookRepoDeleted:
		actionTitle = fmt.Sprintf("deleted repository %s", p.Repository.FullName)
		// style = attentionStyle
	}

	return createMSTeamsPayload(
		p.Repository,
		p.Sender,
		actionTitle,
		nil,
		url,
	), nil
}

// Wiki implements PayloadConvertor Wiki method
func (m msteamsConvertor) Wiki(p *api.WikiPayload) (MSTeamsPayload, error) {
	pageLink := p.Repository.HTMLURL + "/wiki/" + url.PathEscape(p.Page)
	var actionTitle string
	style := emphasisStyle

	switch p.Action {
	case api.HookWikiCreated:
		actionTitle = fmt.Sprintf("created new wiki page '%s'", p.Page)
		style = goodStyle
	case api.HookWikiEdited:
		actionTitle = fmt.Sprintf("edited wiki page '%s'", p.Page)
	case api.HookWikiDeleted:
		actionTitle = fmt.Sprintf("deleted wiki page '%s'", p.Page)
		style = attentionStyle
	}

	bodySections := []MSTeamsContainer{
		{
			Type:  "Container",
			Style: style,
			Items: []any{
				MSTeamsTextBlock{
					Type: "TextBlock",
					Text: markdownLinkFormatter(pageLink, fmt.Sprintf("Wiki page '%s' is %s", p.Page, p.Action)),
					Wrap: true,
				},
			},
		},
	}

	if p.Action != api.HookWikiDeleted && p.Comment != "" {
		actionTitle += fmt.Sprintf(" with comment")
		bodySections[0].Items = append(bodySections[0].Items, MSTeamsTextBlock{
			Type: "TextBlock",
			Text: "Comment: " + p.Comment,
			Wrap: true,
		})
	}

	return createMSTeamsPayload(
		p.Repository,
		p.Sender,
		actionTitle,
		bodySections,
		pageLink,
	), nil
}

// Release implements PayloadConvertor Release method
func (m msteamsConvertor) Release(p *api.ReleasePayload) (MSTeamsPayload, error) {
	var actionTitle string
	// var style string

	switch p.Action {
	case api.HookReleasePublished:
		actionTitle = fmt.Sprintf("published release %s", p.Release.TagName)
		// style = goodStyle
	case api.HookReleaseUpdated:
		actionTitle = fmt.Sprintf("updated release %s", p.Release.TagName)
		// style = emphasisStyle
	case api.HookReleaseDeleted:
		actionTitle = fmt.Sprintf("deleted release %s", p.Release.TagName)
		// style = attentionStyle
	}

	return createMSTeamsPayload(
		p.Repository,
		p.Sender,
		actionTitle,
		nil,
		p.Release.HTMLURL,
	), nil
}

func (m msteamsConvertor) Package(p *api.PackagePayload) (MSTeamsPayload, error) {
	var actionTitle string
	packageName := p.Package.Name + ":" + p.Package.Version
	// var style string

	switch p.Action {
	case api.HookPackageCreated:
		// style = goodStyle
		actionTitle = fmt.Sprintf("created package %s", packageName)
	case api.HookPackageDeleted:
		// style = attentionStyle
		actionTitle = fmt.Sprintf("deleted package %s", packageName)
	}

	return createMSTeamsPayload(
		p.Repository,
		p.Sender,
		actionTitle,
		nil,
		p.Package.HTMLURL,
	), nil
}

func (m msteamsConvertor) Action(p *api.ActionPayload) (MSTeamsPayload, error) {
	var actionTitle string

	runTitle := fmt.Sprintf("%s #%d", p.Run.Title, p.Run.ID)

	switch p.Action {
	case api.HookActionFailure:
		actionTitle = fmt.Sprintf("started action %s has failed on %s", runTitle, p.Run.PrettyRef)
	case api.HookActionRecover:
		actionTitle = fmt.Sprintf("started action %s has recovered on %s", runTitle, p.Run.PrettyRef)
	case api.HookActionSuccess:
		actionTitle = fmt.Sprintf("started action %s succeeded on %s", runTitle, p.Run.PrettyRef)
	}

	// TODO: is TriggerUser correct here?
	// if you'd like to test these proprietary services, see the discussion on: https://codeberg.org/forgejo/forgejo/pulls/7508
	return createMSTeamsPayload(
		p.Run.Repo,
		p.Run.TriggerUser,
		actionTitle,
		nil,
		p.Run.HTMLURL,
	), nil
}

func createMSTeamsPayload(r *api.Repository, s *api.User, actionTitle string, bodySections []MSTeamsContainer, actionTarget string) MSTeamsPayload {
	// determine displayed username
	var displayName string
	if s.FullName != "" {
		// bold the full name if it's set
		displayName = fmt.Sprintf("**%s** (%s)", s.FullName, s.UserName)
	} else {
		// otherwise just use the username
		displayName = fmt.Sprintf("**%s**", s.UserName)
	}
	// Header: title
	headerSection := MSTeamsContainer{
		Type: "Container",
		Items: []any{
			MSTeamsTextBlock{
				Type:     "TextBlock",
				Text:     fmt.Sprintf("💬 Update | [%s](%s)", r.FullName, r.HTMLURL),
				Weight:   "Bolder",
				Size:     "Small",
				IsSubtle: true,
			},
		},
	}

	// Sender info section
	senderSection := MSTeamsContainer{
		Type: "Container",
		Items: []any{
			MSTeamsColumnSet{
				Type: "ColumnSet",
				Columns: []MSTeamsColumn{
					{
						Type:  "Column",
						Width: "auto",
						Items: []any{
							MSTeamsImage{
								Type:  "Image",
								URL:   s.AvatarURL, // use the sender's avatar URL
								Alt:   "Avatar of " + displayName,
								Size:  "Small",
								Style: "Person",
							},
						},
					},
					{
						Type:  "Column",
						Width: "stretch",
						Items: []any{
							MSTeamsTextBlock{
								Type:   "TextBlock",
								Text:   markdownLinkFormatter(s.HTMLURL, displayName) + " " + actionTitle,
								Weight: "Default",
								Size:   "Default",
							},
						},
					},
				},
			},
		},
	}

	// Combine sections in order
	body := []MSTeamsContainer{headerSection, senderSection}
	body = append(body, bodySections...)

	// Create action button for navigation
	actionButton := MSTeamsAction{
		Type:  "Action.OpenUrl",
		Title: "View in Forgejo",
		URL:   actionTarget,
	}

	return MSTeamsPayload{
		Type:    "AdaptiveCard",
		Schema:  "http://adaptivecards.io/schemas/adaptive-card.json",
		Version: "1.5",
		MsTeams: MSTeamsOptions{
			Width: "Full",
		},
		Body:    body,
		Actions: []MSTeamsAction{actionButton},
	}
}

type msteamsConvertor struct{}

var _ shared.PayloadConvertor[MSTeamsPayload] = msteamsConvertor{}

func (msteamsHandler) NewRequest(ctx context.Context, w *webhook_model.Webhook, t *webhook_model.HookTask) (*http.Request, []byte, error) {
	return shared.NewJSONRequest(msteamsConvertor{}, w, t, true)
}
