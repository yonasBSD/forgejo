package webhook

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"

	webhook_model "code.gitea.io/gitea/models/webhook"
	"code.gitea.io/gitea/modules/log"
	api "code.gitea.io/gitea/modules/structs"
	webhook_module "code.gitea.io/gitea/modules/webhook"

	"github.com/minio/sha256-simd"
)

type defaultConvertor struct {
	Webhook *webhook_model.Webhook
}

var _ requestConvertor = defaultConvertor{}

// newDefaultConvertor returns a new default convertor
func newDefaultConvertor(w *webhook_model.Webhook) (requestConvertor, error) {
	return defaultConvertor{
		Webhook: w,
	}, nil
}

// Create implements requestConvertor.
func (dc defaultConvertor) newRequest(p api.Payloader) (request, error) {
	payload, err := p.JSONPayload()
	if err != nil {
		return request{}, err
	}

	w := dc.Webhook
	switch w.HTTPMethod {
	case "":
		log.Info("HTTP Method for for webhook %s[%d] %s empty, setting to POST as default", w.Type, w.ID, w.URL)
		fallthrough
	case http.MethodPost:
		switch w.ContentType {
		case webhook_model.ContentTypeJSON:
			return request{
				Method: "POST",
				URL:    w.URL,
				Body:   payload,
				Header: http.Header{
					"Content-Type": {"application/json"},
				},
				AddDefaultHeaders: true,
			}, nil
		case webhook_model.ContentTypeForm:
			forms := url.Values{
				"payload": []string{string(payload)},
			}
			return request{
				Method: "POST",
				URL:    w.URL,
				Body:   []byte(forms.Encode()),
				Header: http.Header{
					"Content-Type": {"application/x-www-form-urlencoded"},
				},
				AddDefaultHeaders: true,
			}, nil
		default:
			return request{}, fmt.Errorf("invalid content type for webhook %s[%d] %s: %v", w.Type, w.ID, w.URL, w.ContentType)
		}
	case http.MethodGet:
		u, err := url.Parse(w.URL)
		if err != nil {
			return request{}, fmt.Errorf("invalid URL for webhook %s[%d] %s: %w", w.Type, w.ID, w.URL, err)
		}
		vals := u.Query()
		vals["payload"] = []string{string(payload)}
		u.RawQuery = vals.Encode()

		return request{
			Method:            "GET",
			URL:               u.String(),
			Body:              nil,
			AddDefaultHeaders: true,
		}, nil
	default:
		return request{}, fmt.Errorf("invalid http method for webhook %s[%d] %s: %v", w.Type, w.ID, w.URL, w.HTTPMethod)
	}
}

// Create implements requestConvertor.
func (dc defaultConvertor) Create(p *api.CreatePayload) (request, error) {
	return dc.newRequest(p)
}

// Delete implements requestConvertor.
func (dc defaultConvertor) Delete(p *api.DeletePayload) (request, error) {
	return dc.newRequest(p)
}

// Fork implements requestConvertor.
func (dc defaultConvertor) Fork(p *api.ForkPayload) (request, error) {
	return dc.newRequest(p)
}

// Issue implements requestConvertor.
func (dc defaultConvertor) Issue(p *api.IssuePayload) (request, error) {
	return dc.newRequest(p)
}

// IssueComment implements requestConvertor.
func (dc defaultConvertor) IssueComment(p *api.IssueCommentPayload) (request, error) {
	return dc.newRequest(p)
}

// Package implements requestConvertor.
func (dc defaultConvertor) Package(p *api.PackagePayload) (request, error) {
	return dc.newRequest(p)
}

// PullRequest implements requestConvertor.
func (dc defaultConvertor) PullRequest(p *api.PullRequestPayload) (request, error) {
	return dc.newRequest(p)
}

// Push implements requestConvertor.
func (dc defaultConvertor) Push(p *api.PushPayload) (request, error) {
	return dc.newRequest(p)
}

// Release implements requestConvertor.
func (dc defaultConvertor) Release(p *api.ReleasePayload) (request, error) {
	return dc.newRequest(p)
}

// Repository implements requestConvertor.
func (dc defaultConvertor) Repository(p *api.RepositoryPayload) (request, error) {
	return dc.newRequest(p)
}

// Review implements requestConvertor.
func (dc defaultConvertor) Review(p *api.PullRequestPayload, event webhook_module.HookEventType) (request, error) {
	return dc.newRequest(p)
}

// Wiki implements requestConvertor.
func (dc defaultConvertor) Wiki(p *api.WikiPayload) (request, error) {
	return dc.newRequest(p)
}

func addDefaultHeaders(req *http.Request, w *webhook_model.Webhook, t *webhook_model.HookTask) {
	var signatureSHA1 string
	var signatureSHA256 string
	if len(w.Secret) > 0 {
		sig1 := hmac.New(sha1.New, []byte(w.Secret))
		sig256 := hmac.New(sha256.New, []byte(w.Secret))
		_, err := io.MultiWriter(sig1, sig256).Write([]byte(t.PayloadContent))
		if err != nil {
			log.Error("prepareWebhooks.sigWrite: %v", err)
		}
		signatureSHA1 = hex.EncodeToString(sig1.Sum(nil))
		signatureSHA256 = hex.EncodeToString(sig256.Sum(nil))
	}

	event := t.EventType.Event()
	eventType := string(t.EventType)
	req.Header.Add("X-Forgejo-Delivery", t.UUID)
	req.Header.Add("X-Forgejo-Event", event)
	req.Header.Add("X-Forgejo-Event-Type", eventType)
	req.Header.Add("X-Forgejo-Signature", signatureSHA256)
	req.Header.Add("X-Gitea-Delivery", t.UUID)
	req.Header.Add("X-Gitea-Event", event)
	req.Header.Add("X-Gitea-Event-Type", eventType)
	req.Header.Add("X-Gitea-Signature", signatureSHA256)
	req.Header.Add("X-Gogs-Delivery", t.UUID)
	req.Header.Add("X-Gogs-Event", event)
	req.Header.Add("X-Gogs-Event-Type", eventType)
	req.Header.Add("X-Gogs-Signature", signatureSHA256)
	req.Header.Add("X-Hub-Signature", "sha1="+signatureSHA1)
	req.Header.Add("X-Hub-Signature-256", "sha256="+signatureSHA256)
	req.Header["X-GitHub-Delivery"] = []string{t.UUID}
	req.Header["X-GitHub-Event"] = []string{event}
	req.Header["X-GitHub-Event-Type"] = []string{eventType}
}
