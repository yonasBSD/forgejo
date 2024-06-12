// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOpenClosedSwitch(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, giteaURL *url.URL) {
		// Everything can be done in one test env and only one user is needed
		session := loginUser(t, "user2")

		testOCSwitchGlobalIssues(t, session, "0 open ii")

		testNewIssue(t, session, "user2", "repo1", "Switch test - issue 1", "")
		testOCSwitchGlobalIssues(t, session, "1 open i")

		testNewIssue(t, session, "user2", "repo1", "Switch test - issue 2", "")
		testOCSwitchGlobalIssues(t, session, "2 open ii")

		//testOCSwitchGlobalPulls(t, session)
		//testOCSwitchGlobalMilestones(t, session)
	})
}

func testOCSwitchGlobalIssues(t *testing.T, session *TestSession, expected string) {
	t.Helper()

	resp := session.MakeRequest(t, NewRequest(t, "GET", "/issues"), http.StatusOK)
	element := NewHTMLParser(t, resp.Body).Find(".list-header-toggle a[href*='state=open']")
	text := strings.TrimSpace(element.Text())
	assert.EqualValues(t, expected, text)
}

func testOCSwitchGlobalPulls(t *testing.T, session *TestSession) {
	t.Helper()
}

func testOCSwitchGlobalMilestones(t *testing.T, session *TestSession) {
	t.Helper()
}
