// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package install

import (
	"context"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"code.gitea.io/gitea/models/unittest"
	"code.gitea.io/gitea/modules/opentelemetry"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/test"

	"github.com/stretchr/testify/assert"
)

func TestRoutes(t *testing.T) {
	r := Routes()
	assert.NotNil(t, r)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	r.ServeHTTP(w, req)
	assert.EqualValues(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), `class="page-content install"`)

	w = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/no-such", nil)
	r.ServeHTTP(w, req)
	assert.EqualValues(t, 404, w.Code)

	w = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/assets/img/gitea.svg", nil)
	r.ServeHTTP(w, req)
	assert.EqualValues(t, 200, w.Code)
}

func TestMain(m *testing.M) {
	unittest.MainTest(m)
}

func TestOtelChi(t *testing.T) {
	ServiceName := "forgejo-otelchi" + fmt.Sprint(rand.Int())

	otelURL, ok := os.LookupEnv("TEST_OTEL_URL")
	if !ok {
		t.Skip("TEST_OTEL_URL not set")
	}
	traceEndpoint, err := url.Parse(otelURL)
	assert.NoError(t, err)

	defer test.MockVariableValue(&setting.OpenTelemetry.Resource.ServiceName, ServiceName)()
	defer test.MockVariableValue(&setting.OpenTelemetry.Traces.Endpoint, traceEndpoint)()
	defer test.MockVariableValue(&setting.OpenTelemetry.Traces.Insecure, true)()

	assert.NoError(t, opentelemetry.Init(context.Background()))
	r := Routes()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/e/img/gitea.svg", nil)
	r.ServeHTTP(w, req)

	traceEndpoint.Host = traceEndpoint.Hostname() + ":16686"
	traceEndpoint.Path = "/api/services"

	namePresent := false

	for i := range []int{1, 1, 2, 3, 5, 8} {
		resp, err := http.Get(traceEndpoint.String())
		assert.NoError(t, err)

		apiResponse, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)

		if strings.Contains(string(apiResponse), ServiceName) {
			namePresent = true
			break
		}
		time.Sleep(time.Duration(i) * time.Second)
	}
	assert.True(t, namePresent)
}
