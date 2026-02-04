// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"testing"
	"time"

	"forgejo.org/models/db"
	"forgejo.org/models/forgefed"
	"forgejo.org/models/unittest"
	"forgejo.org/models/user"
	"forgejo.org/modules/activitypub"
	fm "forgejo.org/modules/forgefed"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"
	"forgejo.org/routers"
	"forgejo.org/services/contexttest"
	"forgejo.org/services/federation"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFederationHttpSigValidation(t *testing.T) {
	defer test.MockVariableValue(&setting.Federation.Enabled, true)()
	defer test.MockVariableValue(&testWebRoutes, routers.NormalRoutes())()

	err := federation.Init()
	require.NoError(t, err)

	mock := test.NewFederationServerMock()
	federatedSrv := mock.DistantServer(t)
	followUser := mock.Persons[0]
	followURL := followUser.FederationID(federatedSrv.URL)
	followKeyID := followUser.KeyID(federatedSrv.URL)

	onApplicationRun(t, func(t *testing.T, u *url.URL) {
		userID := 2
		userURL := fmt.Sprintf("%sapi/v1/activitypub/user-id/%d", u, userID)

		user1 := unittest.AssertExistsAndLoadBean(t, &user.User{ID: 1})
		_ = unittest.AssertExistsAndLoadBean(t, &user.User{ID: 2})

		ctx, _ := contexttest.MockAPIContext(t, userURL)
		clientFactory, err := activitypub.NewClientFactoryWithTimeout(60 * time.Second)
		require.NoError(t, err)

		apClient, err := clientFactory.WithKeys(ctx, user1, user1.KeyID())
		require.NoError(t, err)

		followClient, err := clientFactory.WithKeysDirect(ctx, followUser.PrivKey, followKeyID)
		require.NoError(t, err)

		// Unsigned request
		t.Run("UnsignedRequest", func(t *testing.T) {
			req := NewRequest(t, "GET", userURL)
			MakeRequest(t, req, http.StatusBadRequest)
		})

		// Signed CAVAGE GET request
		t.Run("SignedGetCAVAGERequest", func(t *testing.T) {
			assert.False(t, apClient.GetRFC9421())

			req, err := apClient.GetRequest(userURL)
			require.NoError(t, err)

			sig := req.Header.Get("Signature")

			assert.NotEmpty(t, sig)
			assert.Contains(t, sig, `algorithm="hs2019"`)

			expKeyID := fmt.Sprintf(`keyId="%v"`, apClient.KeyID())
			assert.Contains(t, sig, expKeyID)

			expHeaders := fmt.Sprintf(`headers="%v"`, apClient.SignedHeaders(http.MethodGet))
			assert.Contains(t, sig, expHeaders)

			assert.Contains(t, sig, "signature=")

			resp, err := apClient.Do(req)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			resp, err = apClient.Get(userURL)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			resp, err = apClient.Get(followURL)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
		})

		// Signed CAVAGE GET request (failure tests)
		t.Run("SignedGetCAVAGERequestFailures", func(t *testing.T) {
			assert.False(t, apClient.GetRFC9421())

			req, err := apClient.GetRequest(userURL)
			require.NoError(t, err)

			sig := req.Header.Get("Signature")

			assert.NotEmpty(t, sig)
			assert.Contains(t, sig, `algorithm="hs2019"`)

			// empty signature
			req.Header.Set("Signature", "")
			resp, err := apClient.Do(req)
			require.NoError(t, err)
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

			// empty algorithm
			re := regexp.MustCompile(`algorithm="[a-zA-Z0-9_\-]*"`)
			badSigAlg := re.ReplaceAllString(sig, ``)
			req.Header.Set("Signature", badSigAlg)
			resp, err = apClient.Do(req)
			require.NoError(t, err)
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

			// bad key ID
			re = regexp.MustCompile(`keyId=".*"`)
			badKeyID := re.ReplaceAllString(sig, `keyId="https://bad.key/id#main"`)
			req.Header.Set("Signature", badKeyID)
			resp, err = apClient.Do(req)
			require.NoError(t, err)
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

			// bad signature
			re = regexp.MustCompile(`signature=".*"`)
			badSig := re.ReplaceAllString(sig, `signature="badSignature"`)
			req.Header.Set("Signature", badSig)
			resp, err = apClient.Do(req)
			require.NoError(t, err)
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})

		// Signed RFC 9421 GET request
		t.Run("SignedGetRFC9421Request", func(t *testing.T) {
			apClient.SetRFC9421(true)
			assert.True(t, apClient.GetRFC9421())

			req, err := apClient.GetRequest(userURL)
			require.NoError(t, err)

			sigInput := req.Header.Get("Signature-Input")
			sig := req.Header.Get("Signature")

			assert.NotEmpty(t, sigInput)
			assert.NotEmpty(t, sig)

			assert.Contains(t, sigInput, `alg="rsa-v1_5-sha256"`)

			expKeyID := fmt.Sprintf(`keyid="%v"`, apClient.KeyID())
			assert.Contains(t, sigInput, expKeyID)

			expHeaders := fmt.Sprintf(`sig1=(%v)`, apClient.SignedHeaders(http.MethodGet))
			assert.Contains(t, sigInput, expHeaders)

			resp, err := apClient.Do(req)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			resp, err = apClient.Get(userURL)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			resp, err = apClient.Get(followURL)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
		})

		// Signed RFC 9421 GET request (failure tests)
		t.Run("SignedGetRFC9421RequestFailures", func(t *testing.T) {
			apClient.SetRFC9421(true)
			assert.True(t, apClient.GetRFC9421())

			req, err := apClient.GetRequest(userURL)
			require.NoError(t, err)

			// assert valid POST request
			resp, err := apClient.Do(req)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			sig := req.Header.Get("Signature")
			sigInput := req.Header.Get("Signature-Input")

			assert.NotEmpty(t, sig)

			// empty signature
			req.Header.Set("Signature", "")
			resp, err = apClient.Do(req)
			require.NoError(t, err)
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

			// empty algorithm
			re := regexp.MustCompile(`alg="[a-zA-Z0-9_\-]*"`)
			badSigAlg := re.ReplaceAllString(sigInput, ``)
			req.Header.Set("Signature-Input", badSigAlg)
			resp, err = apClient.Do(req)
			require.NoError(t, err)
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

			// bad key ID
			re = regexp.MustCompile(`keyid=".*"`)
			badKeyID := re.ReplaceAllString(sig, `keyid="https://bad.key/id#main"`)
			req.Header.Set("Signature-Input", badKeyID)
			resp, err = apClient.Do(req)
			require.NoError(t, err)
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

			// bad signature
			req.Header.Set("Signature", "badSignature")
			resp, err = apClient.Do(req)
			require.NoError(t, err)
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})

		// Signed CAVAGE POST request
		t.Run("SignedPostCAVAGERequest", func(t *testing.T) {
			apClient.SetRFC9421(false)
			followClient.SetRFC9421(false)

			assert.False(t, apClient.GetRFC9421())
			assert.False(t, followClient.GetRFC9421())

			followActivity, err := fm.NewForgeFollow(followURL, userURL)
			require.NoError(t, err)

			followJSON, err := followActivity.MarshalJSON()
			require.NoError(t, err)

			req, err := followClient.PostRequest(followJSON, fmt.Sprintf("%v/inbox", userURL))
			require.NoError(t, err)

			sig := req.Header.Get("Signature")

			assert.NotEmpty(t, sig)
			assert.Contains(t, sig, `algorithm="hs2019"`)

			expKeyID := fmt.Sprintf(`keyId="%v"`, followClient.KeyID())
			assert.Contains(t, sig, expKeyID)

			expHeaders := fmt.Sprintf(`headers="%v"`, followClient.SignedHeaders(http.MethodPost))
			assert.Contains(t, sig, expHeaders)

			assert.Contains(t, sig, "signature=")

			resp, err := followClient.Do(req)
			require.NoError(t, err)
			assert.Equal(t, http.StatusAccepted, resp.StatusCode)

			resp, err = followClient.Post(followJSON, fmt.Sprintf("%v/inbox", followURL))
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
		})

		// Signed CAVAGE POST request (failure tests)
		t.Run("SignedPostCAVAGERequestFailures", func(t *testing.T) {
			apClient.SetRFC9421(false)
			followClient.SetRFC9421(false)

			assert.False(t, apClient.GetRFC9421())
			assert.False(t, followClient.GetRFC9421())

			followActivity, err := fm.NewForgeFollow(followURL, userURL)
			require.NoError(t, err)

			followJSON, err := followActivity.MarshalJSON()
			require.NoError(t, err)

			req, err := followClient.PostRequest(followJSON, fmt.Sprintf("%v/inbox", userURL))
			require.NoError(t, err)

			sig := req.Header.Get("Signature")

			assert.NotEmpty(t, sig)
			assert.Contains(t, sig, `algorithm="hs2019"`)

			// assert valid follow request
			resp, err := followClient.Do(req)
			require.NoError(t, err)
			assert.Equal(t, http.StatusNoContent, resp.StatusCode)

			// empty signature
			req.Header.Set("Signature", "")
			resp, err = followClient.Do(req)
			require.NoError(t, err)
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

			// empty algorithm
			re := regexp.MustCompile(`algorithm="[a-zA-Z0-9_\-]*"`)
			badSigAlg := re.ReplaceAllString(sig, ``)
			req.Header.Set("Signature", badSigAlg)
			_, err = followClient.Do(req)
			require.Error(t, err)

			// bad key ID
			re = regexp.MustCompile(`keyId=".*"`)
			badKeyID := re.ReplaceAllString(sig, `keyId="https://bad.key/id#main"`)
			req.Header.Set("Signature", badKeyID)
			_, err = followClient.Do(req)
			require.Error(t, err)

			// bad signature
			re = regexp.MustCompile(`signature=".*"`)
			badSig := re.ReplaceAllString(sig, `signature="badSignature"`)
			req.Header.Set("Signature", badSig)
			_, err = followClient.Do(req)
			require.Error(t, err)
		})

		// Signed RFC 9421 POST request
		t.Run("SignedPostRFC9421Request", func(t *testing.T) {
			apClient.SetRFC9421(true)
			followClient.SetRFC9421(true)

			assert.True(t, apClient.GetRFC9421())
			assert.True(t, followClient.GetRFC9421())

			followActivity, err := fm.NewForgeFollow(followURL, userURL)
			require.NoError(t, err)

			followJSON, err := followActivity.MarshalJSON()
			require.NoError(t, err)

			req, err := followClient.PostRequest(followJSON, fmt.Sprintf("%v/inbox", userURL))
			require.NoError(t, err)

			sigInput := req.Header.Get("Signature-Input")
			sig := req.Header.Get("Signature")

			assert.NotEmpty(t, sigInput)
			assert.NotEmpty(t, sig)

			assert.Contains(t, sigInput, `alg="rsa-v1_5-sha256"`)

			expKeyID := fmt.Sprintf(`keyid="%v"`, followClient.KeyID())
			assert.Contains(t, sigInput, expKeyID)

			expHeaders := fmt.Sprintf(`sig1=(%v)`, followClient.SignedHeaders(http.MethodPost))
			assert.Contains(t, sigInput, expHeaders)

			resp, err := followClient.Do(req)
			require.NoError(t, err)
			assert.Equal(t, http.StatusNoContent, resp.StatusCode)

			resp, err = followClient.Post(followJSON, fmt.Sprintf("%v/inbox", followURL))
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
		})

		// Signed RFC 9421 POST request (failure tests)
		t.Run("SignedPostRFC9421RequestFailures", func(t *testing.T) {
			apClient.SetRFC9421(true)
			followClient.SetRFC9421(true)

			assert.True(t, apClient.GetRFC9421())
			assert.True(t, followClient.GetRFC9421())

			followActivity, err := fm.NewForgeFollow(followURL, userURL)
			require.NoError(t, err)

			activityJSON, err := followActivity.MarshalJSON()
			require.NoError(t, err)

			req, err := followClient.PostRequest(activityJSON, fmt.Sprintf("%v/inbox", userURL))
			require.NoError(t, err)

			// assert valid POST request
			resp, err := apClient.Do(req)
			require.NoError(t, err)
			assert.Equal(t, http.StatusNoContent, resp.StatusCode)

			sig := req.Header.Get("Signature")
			sigInput := req.Header.Get("Signature-Input")

			assert.NotEmpty(t, sig)

			// empty signature
			req.Header.Set("Signature", "")
			resp, err = apClient.Do(req)
			require.NoError(t, err)
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

			// empty algorithm
			re := regexp.MustCompile(`alg="[a-zA-Z0-9_\-]*"`)
			badSigAlg := re.ReplaceAllString(sigInput, `alg=""`)
			req.Header.Set("Signature-Input", badSigAlg)
			_, err = apClient.Do(req)
			require.Error(t, err)

			// bad key ID
			re = regexp.MustCompile(`keyid=".*"`)
			badKeyID := re.ReplaceAllString(sig, `keyid="https://bad.key/id#main"`)
			req.Header.Set("Signature-Input", badKeyID)
			_, err = apClient.Do(req)
			require.Error(t, err)

			// bad signature
			req.Header.Set("Signature", "badSignature")
			_, err = apClient.Do(req)
			require.Error(t, err)
		})

		// HACK HACK HACK: the host part of the URL gets set to which IP forgejo is
		// listening on, NOT localhost, which is the Domain given to forgejo which
		// is then used for eg. the keyID all requests
		applicationKeyID := fmt.Sprintf("%sapi/v1/activitypub/actor#main-key", setting.AppURL)
		actorKeyID := fmt.Sprintf("%sapi/v1/activitypub/user-id/1#main-key", setting.AppURL)

		// Check for cached public keys
		t.Run("ValidateCaches", func(t *testing.T) {
			host, err := forgefed.FindFederationHostByKeyID(db.DefaultContext, applicationKeyID)
			require.NoError(t, err)
			assert.NotNil(t, host)
			assert.True(t, host.PublicKey.Valid)

			_, user, err := user.FindFederatedUserByKeyID(db.DefaultContext, actorKeyID)
			require.NoError(t, err)
			assert.NotNil(t, user)
			assert.True(t, user.PublicKey.Valid)
		})

		t.Run("ValidateActorFromKeyID", func(t *testing.T) {
			_, err := federation.NewActorIDFromKeyID(ctx, actorKeyID)
			require.NoError(t, err)

			_, err = federation.NewActorIDFromKeyID(ctx, "http://bad.url/%^&")
			require.Error(t, err)
		})

		// Disable signature validation
		defer test.MockVariableValue(&setting.Federation.SignatureEnforced, false)()

		// Unsigned request
		t.Run("SignatureValidationDisabled", func(t *testing.T) {
			req := NewRequest(t, "GET", userURL)
			MakeRequest(t, req, http.StatusOK)
		})
	})
}
