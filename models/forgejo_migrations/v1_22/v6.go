// Copyright 2024 The Forgejo Authors c/o Codeberg e.V.. All rights reserved.
// SPDX-License-Identifier: MIT

package v1_22 //nolint

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	webhook_model "code.gitea.io/gitea/models/webhook"
	"code.gitea.io/gitea/modules/json"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/timeutil"
	webhook_module "code.gitea.io/gitea/modules/webhook"
	"code.gitea.io/gitea/services/cron"

	"xorm.io/xorm"
)

// HookTask represents a hook task.
// exact copy from current models/webhook/hooktask.go
//   - xorm:"-" fields deleted
//   - RequestContent restored (needed for the migration)
type HookTask struct {
	ID             int64  `xorm:"pk autoincr"`
	HookID         int64  `xorm:"index"`
	UUID           string `xorm:"unique"`
	PayloadContent string `xorm:"LONGTEXT"`
	EventType      webhook_module.HookEventType
	IsDelivered    bool
	Delivered      timeutil.TimeStampNano

	// one of GET, PUT, POST, DELETE
	// According to https://www.iana.org/assignments/http-methods/http-methods.xhtml, it
	// should be 17 chars (for UPDATEREDIRECTREF), however the storage overhead is likely not
	// worth the unlikely usage of such an http method.
	RequestMethod string `xorm:"VARCHAR(6)"`
	RequestURL    string `xorm:"request_url TEXT"`
	RequestHeader string `xorm:"LONGTEXT"` // http.Header as text (HTTP wire format)
	// AddDefaultHeaders will add the default X- headers to the request (event type, id, signature...)
	// can't be stored in the RequestHeader, because they depend on the UUID of the task.
	AddDefaultHeaders bool

	// History info.
	IsSucceed       bool
	ResponseContent string `xorm:"LONGTEXT"`
	// ResponseInfo    *HookResponse `xorm:"-"` // unused during the migration

	RequestContent string `xorm:"LONGTEXT"` // needed for the migration
}

func UpdateHookTaskTable(x *xorm.Engine) error {
	cleanupHooks := cron.GetTask("cleanup_hook_task_table")
	if cleanupHooks == nil {
		log.Warn("cleanup_hook_task_table not found, migration might take longer than needed")
	} else {
		cleanupHooks.Run()
	}
	// create missing columns
	if err := x.Sync(new(HookTask)); err != nil {
		return err
	}

	// migrate previous hook tasks
	err := batchProcess(x,
		make([]*HookTask, 0, 50),
		func(limit, start int) *xorm.Session {
			return x.OrderBy("id").Limit(limit, start)
		},
		func(sess *xorm.Session, hookTask *HookTask) error {
			hookTask.AddDefaultHeaders = true // as of the migration, all hook types add the event headers
			if !hookTask.IsDelivered {
				var req *http.Request

				if err := prepareRequest(hookTask, func(r *http.Request) {
					req = r
				}); err != nil {
					return err
				}

				hookTask.RequestMethod = req.Method
				hookTask.RequestURL = req.URL.String()
				var header strings.Builder
				if err := req.Header.Write(&header); err != nil {
					return err
				}
				hookTask.RequestHeader = header.String()

				if req.Body != nil && req.Method != http.MethodGet {
					buf, err := io.ReadAll(req.Body)
					if err != nil {
						return err
					}
					hookTask.PayloadContent = string(buf)
				} else {
					hookTask.PayloadContent = ""
				}
			} else {
				// hookTask has already been delivered: take the actual delivered values

				// HookRequest represents hook task request information.
				type HookRequest struct {
					URL        string            `json:"url"`
					HTTPMethod string            `json:"http_method"`
					Headers    map[string]string `json:"headers"`
				}

				hr := &HookRequest{}
				if err := json.Unmarshal([]byte(hookTask.RequestContent), hr); err != nil {
					return err
				}
				hookTask.RequestMethod = hr.HTTPMethod
				hookTask.RequestURL = hr.URL
				httpHeader := http.Header{}
				for k, v := range hr.Headers {
					httpHeader.Set(k, v)
				}
				var header strings.Builder
				if err := httpHeader.Write(&header); err != nil {
					return err
				}
				hookTask.RequestHeader = header.String()
			}

			// save in database
			count, err := sess.ID(hookTask.ID).Cols("request_method", "request_url", "request_header", "payload_content", "add_default_headers").Update(hookTask)
			if count != 1 || err != nil {
				return fmt.Errorf("unable to update hook_task[id=%d]: %d,%w", hookTask.ID, count, err)
			}
			return nil
		})
	if err != nil {
		return err
	}

	return nil
}

// this function has been copy/pasted from services/webhook/deliver.go
// the callback argument is there to prevent having to change the `return` statements
func prepareRequest(t *HookTask, callback func(*http.Request)) error {
	w, err := webhook_model.GetWebhookByID(context.Background(), t.HookID)
	if err != nil {
		return err
	}

	var req *http.Request

	switch w.HTTPMethod {
	case "":
		log.Info("HTTP Method for webhook %s empty, setting to POST as default", w.URL)
		fallthrough
	case http.MethodPost:
		switch w.ContentType {
		case webhook_model.ContentTypeJSON:
			req, err = http.NewRequest("POST", w.URL, strings.NewReader(t.PayloadContent))
			if err != nil {
				return err
			}

			req.Header.Set("Content-Type", "application/json")
		case webhook_model.ContentTypeForm:
			forms := url.Values{
				"payload": []string{t.PayloadContent},
			}

			req, err = http.NewRequest("POST", w.URL, strings.NewReader(forms.Encode()))
			if err != nil {
				return err
			}

			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
	case http.MethodGet:
		u, err := url.Parse(w.URL)
		if err != nil {
			return fmt.Errorf("unable to deliver webhook task[%d] as cannot parse webhook url %s: %w", t.ID, w.URL, err)
		}
		vals := u.Query()
		vals["payload"] = []string{t.PayloadContent}
		u.RawQuery = vals.Encode()
		req, err = http.NewRequest("GET", u.String(), nil)
		if err != nil {
			return fmt.Errorf("unable to deliver webhook task[%d] as unable to create HTTP request for webhook url %s: %w", t.ID, w.URL, err)
		}
	case http.MethodPut:
		switch w.Type {
		case webhook_module.MATRIX:
			txnID, err := getMatrixTxnID([]byte(t.PayloadContent))
			if err != nil {
				return err
			}
			url := fmt.Sprintf("%s/%s", w.URL, url.PathEscape(txnID))
			req, err = http.NewRequest("PUT", url, strings.NewReader(t.PayloadContent))
			if err != nil {
				return fmt.Errorf("unable to deliver webhook task[%d] as cannot create matrix request for webhook url %s: %w", t.ID, w.URL, err)
			}
		default:
			return fmt.Errorf("invalid http method for webhook task[%d] in webhook %s: %v", t.ID, w.URL, w.HTTPMethod)
		}
	default:
		return fmt.Errorf("invalid http method for webhook task[%d] in webhook %s: %v", t.ID, w.URL, w.HTTPMethod)
	}

	callback(req)

	return nil
}

// getMatrixTxnID computes the transaction ID to ensure idempotency
func getMatrixTxnID(payload []byte) (string, error) {
	h := sha1.New()
	_, err := h.Write(payload)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

func batchProcess[T any](x *xorm.Engine, buf []T, query func(limit, start int) *xorm.Session, process func(*xorm.Session, T) error) error {
	size := cap(buf)
	start := 0
	for {
		err := query(size, start).Find(&buf)
		if err != nil {
			return err
		}
		if len(buf) == 0 {
			return nil
		}

		err = func() error {
			sess := x.NewSession()
			defer sess.Close()
			if err := sess.Begin(); err != nil {
				return fmt.Errorf("unable to allow start session. Error: %w", err)
			}
			for _, record := range buf {
				if err := process(sess, record); err != nil {
					return err
				}
			}
			return sess.Commit()
		}()
		if err != nil {
			return err
		}

		if len(buf) < size {
			return nil
		}
		start += size
		buf = buf[:0]
	}
}
