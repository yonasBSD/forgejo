// Copyright 2019 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package private

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"code.gitea.io/gitea/modules/setting"
)

// Git environment variables
const (
	GitAlternativeObjectDirectories = "GIT_ALTERNATE_OBJECT_DIRECTORIES"
	GitObjectDirectory              = "GIT_OBJECT_DIRECTORY"
	GitQuarantinePath               = "GIT_QUARANTINE_PATH"
)

// HookOptions represents the options for the Hook calls
type HookOptions struct {
	OldCommitIDs                    []string
	NewCommitIDs                    []string
	RefFullNames                    []string
	UserID                          int64
	UserName                        string
	GitObjectDirectory              string
	GitAlternativeObjectDirectories string
	GitQuarantinePath               string
	ProtectedBranchID               int64
	IsDeployKey                     bool
}

// HookPostReceiveResult represents an individual result from PostReceive
type HookPostReceiveResult struct {
	Message bool
	Create  bool
	Branch  string
	URL     string
	Err     string
}

// HookPreReceive check whether the provided commits are allowed
func HookPreReceive(ownerName, repoName string, opts HookOptions) (int, string) {
	reqURL := setting.LocalURL + fmt.Sprintf("api/internal/hook/pre-receive/%s/%s",
		url.PathEscape(ownerName),
		url.PathEscape(repoName),
	)
	req := newInternalRequest(reqURL, "POST")
	req = req.Header("Content-Type", "application/json")
	jsonBytes, _ := json.Marshal(opts)
	req.Body(jsonBytes)
	resp, err := req.Response()
	if err != nil {
		return http.StatusInternalServerError, fmt.Sprintf("Unable to contact gitea: %v", err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return resp.StatusCode, decodeJSONError(resp).Err
	}

	return http.StatusOK, ""
}

// HookPostReceive updates services and users
func HookPostReceive(ownerName, repoName string, opts HookOptions) ([]HookPostReceiveResult, string) {
	reqURL := setting.LocalURL + fmt.Sprintf("api/internal/hook/post-receive/%s/%s",
		url.PathEscape(ownerName),
		url.PathEscape(repoName),
	)

	req := newInternalRequest(reqURL, "POST")
	req = req.Header("Content-Type", "application/json")
	jsonBytes, _ := json.Marshal(opts)
	req.Body(jsonBytes)
	resp, err := req.Response()
	if err != nil {
		return nil, fmt.Sprintf("Unable to contact gitea: %v", err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, decodeJSONError(resp).Err
	}
	res := []HookPostReceiveResult{}
	_ = json.NewDecoder(resp.Body).Decode(&res)

	return res, ""
}
