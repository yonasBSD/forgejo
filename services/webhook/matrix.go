// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package webhook

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	webhook_model "code.gitea.io/gitea/models/webhook"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/json"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	api "code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/modules/util"
	webhook_module "code.gitea.io/gitea/modules/webhook"
)

const matrixPayloadSizeLimit = 1024 * 64

// MatrixMeta contains the Matrix metadata
type MatrixMeta struct {
	HomeserverURL string `json:"homeserver_url"`
	Room          string `json:"room_id"`
	MessageType   int    `json:"message_type"`
}

var messageTypeText = map[int]string{
	1: "m.notice",
	2: "m.text",
}

// GetMatrixHook returns Matrix metadata
func GetMatrixHook(w *webhook_model.Webhook) *MatrixMeta {
	s := &MatrixMeta{}
	if err := json.Unmarshal([]byte(w.Meta), s); err != nil {
		log.Error("webhook.GetMatrixHook(%d): %v", w.ID, err)
	}
	return s
}

// MatrixLinkFormatter creates a link compatible with Matrix
func MatrixLinkFormatter(url, text string) string {
	return htmlLinkFormatter(url, text)
}

// MatrixLinkToRef Matrix-formatter link to a repo ref
func MatrixLinkToRef(repoURL, ref string) string {
	refName := git.RefName(ref).ShortName()
	switch {
	case strings.HasPrefix(ref, git.BranchPrefix):
		return MatrixLinkFormatter(repoURL+"/src/branch/"+util.PathEscapeSegments(refName), refName)
	case strings.HasPrefix(ref, git.TagPrefix):
		return MatrixLinkFormatter(repoURL+"/src/tag/"+util.PathEscapeSegments(refName), refName)
	default:
		return MatrixLinkFormatter(repoURL+"/src/commit/"+util.PathEscapeSegments(refName), refName)
	}
}

type matrixConvertor struct {
	URL     string
	MsgType string
}

var _ requestConvertor = matrixConvertor{}

// newMatrixConvertor returns a new convertor for Matrix https://matrix.org/
func newMatrixConvertor(w *webhook_model.Webhook) (requestConvertor, error) {
	meta := &MatrixMeta{}
	if err := json.Unmarshal([]byte(w.Meta), meta); err != nil {
		return nil, fmt.Errorf("GetMatrixPayload meta json: %w", err)
	}

	return matrixConvertor{
		URL:     w.URL,
		MsgType: messageTypeText[meta.MessageType],
	}, nil
}

func (m matrixConvertor) newRequest(text string, commits ...*api.PayloadCommit) (request, error) {
	payload := struct {
		Body          string               `json:"body"`
		MsgType       string               `json:"msgtype"`
		Format        string               `json:"format"`
		FormattedBody string               `json:"formatted_body"`
		Commits       []*api.PayloadCommit `json:"io.gitea.commits,omitempty"`
	}{
		Body:          getMessageBody(text),
		MsgType:       m.MsgType,
		Format:        "org.matrix.custom.html",
		FormattedBody: text,
		Commits:       commits,
	}

	body, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return request{}, err
	}

	txnID, err := getMatrixTxnID(body)
	if err != nil {
		return request{}, err
	}

	return request{
		Method: http.MethodPut,
		URL:    m.URL + "/" + txnID,
		Body:   body,
		Header: http.Header{
			"Content-Type": {"application/json"},
		},
		AddDefaultHeaders: true,
	}, nil
}

// Create implements requestConvertor Create method
func (m matrixConvertor) Create(p *api.CreatePayload) (request, error) {
	repoLink := MatrixLinkFormatter(p.Repo.HTMLURL, p.Repo.FullName)
	refLink := MatrixLinkToRef(p.Repo.HTMLURL, p.Ref)
	text := fmt.Sprintf("[%s:%s] %s created by %s", repoLink, refLink, p.RefType, p.Sender.UserName)

	return m.newRequest(text)
}

// Delete composes Matrix payload for delete a branch or tag.
func (m matrixConvertor) Delete(p *api.DeletePayload) (request, error) {
	refName := git.RefName(p.Ref).ShortName()
	repoLink := MatrixLinkFormatter(p.Repo.HTMLURL, p.Repo.FullName)
	text := fmt.Sprintf("[%s:%s] %s deleted by %s", repoLink, refName, p.RefType, p.Sender.UserName)

	return m.newRequest(text)
}

// Fork composes Matrix payload for forked by a repository.
func (m matrixConvertor) Fork(p *api.ForkPayload) (request, error) {
	baseLink := MatrixLinkFormatter(p.Forkee.HTMLURL, p.Forkee.FullName)
	forkLink := MatrixLinkFormatter(p.Repo.HTMLURL, p.Repo.FullName)
	text := fmt.Sprintf("%s is forked to %s", baseLink, forkLink)

	return m.newRequest(text)
}

// Issue implements requestConvertor Issue method
func (m matrixConvertor) Issue(p *api.IssuePayload) (request, error) {
	text, _, _, _ := getIssuesPayloadInfo(p, MatrixLinkFormatter, true)

	return m.newRequest(text)
}

// IssueComment implements requestConvertor IssueComment method
func (m matrixConvertor) IssueComment(p *api.IssueCommentPayload) (request, error) {
	text, _, _ := getIssueCommentPayloadInfo(p, MatrixLinkFormatter, true)

	return m.newRequest(text)
}

// Wiki implements requestConvertor Wiki method
func (m matrixConvertor) Wiki(p *api.WikiPayload) (request, error) {
	text, _, _ := getWikiPayloadInfo(p, MatrixLinkFormatter, true)

	return m.newRequest(text)
}

// Release implements requestConvertor Release method
func (m matrixConvertor) Release(p *api.ReleasePayload) (request, error) {
	text, _ := getReleasePayloadInfo(p, MatrixLinkFormatter, true)

	return m.newRequest(text)
}

// Push implements requestConvertor Push method
func (m matrixConvertor) Push(p *api.PushPayload) (request, error) {
	var commitDesc string

	if p.TotalCommits == 1 {
		commitDesc = "1 commit"
	} else {
		commitDesc = fmt.Sprintf("%d commits", p.TotalCommits)
	}

	repoLink := MatrixLinkFormatter(p.Repo.HTMLURL, p.Repo.FullName)
	branchLink := MatrixLinkToRef(p.Repo.HTMLURL, p.Ref)
	text := fmt.Sprintf("[%s] %s pushed %s to %s:<br>", repoLink, p.Pusher.UserName, commitDesc, branchLink)

	// for each commit, generate a new line text
	for i, commit := range p.Commits {
		text += fmt.Sprintf("%s: %s - %s", MatrixLinkFormatter(commit.URL, commit.ID[:7]), commit.Message, commit.Author.Name)
		// add linebreak to each commit but the last
		if i < len(p.Commits)-1 {
			text += "<br>"
		}

	}

	return m.newRequest(text, p.Commits...)
}

// PullRequest implements requestConvertor PullRequest method
func (m matrixConvertor) PullRequest(p *api.PullRequestPayload) (request, error) {
	text, _, _, _ := getPullRequestPayloadInfo(p, MatrixLinkFormatter, true)

	return m.newRequest(text)
}

// Review implements requestConvertor Review method
func (m matrixConvertor) Review(p *api.PullRequestPayload, event webhook_module.HookEventType) (request, error) {
	senderLink := MatrixLinkFormatter(setting.AppURL+url.PathEscape(p.Sender.UserName), p.Sender.UserName)
	title := fmt.Sprintf("#%d %s", p.Index, p.PullRequest.Title)
	titleLink := MatrixLinkFormatter(p.PullRequest.HTMLURL, title)
	repoLink := MatrixLinkFormatter(p.Repository.HTMLURL, p.Repository.FullName)
	var text string

	switch p.Action {
	case api.HookIssueReviewed:
		action, err := parseHookPullRequestEventType(event)
		if err != nil {
			return request{}, err
		}

		text = fmt.Sprintf("[%s] Pull request review %s: %s by %s", repoLink, action, titleLink, senderLink)
	}

	return m.newRequest(text)
}

// Repository implements requestConvertor Repository method
func (m matrixConvertor) Repository(p *api.RepositoryPayload) (request, error) {
	senderLink := MatrixLinkFormatter(setting.AppURL+p.Sender.UserName, p.Sender.UserName)
	repoLink := MatrixLinkFormatter(p.Repository.HTMLURL, p.Repository.FullName)
	var text string

	switch p.Action {
	case api.HookRepoCreated:
		text = fmt.Sprintf("[%s] Repository created by %s", repoLink, senderLink)
	case api.HookRepoDeleted:
		text = fmt.Sprintf("[%s] Repository deleted by %s", repoLink, senderLink)
	}
	return m.newRequest(text)
}

func (m matrixConvertor) Package(p *api.PackagePayload) (request, error) {
	senderLink := MatrixLinkFormatter(setting.AppURL+p.Sender.UserName, p.Sender.UserName)
	packageLink := MatrixLinkFormatter(p.Package.HTMLURL, p.Package.Name)
	var text string

	switch p.Action {
	case api.HookPackageCreated:
		text = fmt.Sprintf("[%s] Package published by %s", packageLink, senderLink)
	case api.HookPackageDeleted:
		text = fmt.Sprintf("[%s] Package deleted by %s", packageLink, senderLink)
	}

	return m.newRequest(text)
}

var urlRegex = regexp.MustCompile(`<a [^>]*?href="([^">]*?)">(.*?)</a>`)

func getMessageBody(htmlText string) string {
	htmlText = urlRegex.ReplaceAllString(htmlText, "[$2]($1)")
	htmlText = strings.ReplaceAll(htmlText, "<br>", "\n")
	return htmlText
}

// getMatrixTxnID computes the transaction ID to ensure idempotency
func getMatrixTxnID(payload []byte) (string, error) {
	if len(payload) >= matrixPayloadSizeLimit {
		return "", fmt.Errorf("getMatrixTxnID: payload size %d > %d", len(payload), matrixPayloadSizeLimit)
	}

	h := sha1.New()
	_, err := h.Write(payload)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
