// Copyright 2022 The Gitea Authors. All rights reserved.
// Copyright 2023,2024,2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package activitypub_test

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"forgejo.org/models/db"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/activitypub"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const keyID = "http://localhost:3003/api/v1/activitypub/user-id/1#main-key"

func TestCurrentTime(t *testing.T) {
	date := activitypub.CurrentTime()
	_, err := time.Parse(http.TimeFormat, date)
	require.NoError(t, err)
	assert.Equal(t, "GMT", date[len(date)-3:])
}

/* ToDo: Set Up tests for http get requests

Set up an expected response for GET on api with user-id = 1:
{
  "@context": [
    "https://www.w3.org/ns/activitystreams",
    "https://w3id.org/security/v1"
  ],
  "id": "http://localhost:3000/api/v1/activitypub/user-id/1",
  "type": "Person",
  "icon": {
    "type": "Image",
    "mediaType": "image/png",
    "url": "http://localhost:3000/avatar/3120fd0edc57d5d41230013ad88232e2"
  },
  "url": "http://localhost:3000/me",
  "inbox": "http://localhost:3000/api/v1/activitypub/user-id/1/inbox",
  "outbox": "http://localhost:3000/api/v1/activitypub/user-id/1/outbox",
  "preferredUsername": "me",
  "publicKey": {
    "id": "http://localhost:3000/api/v1/activitypub/user-id/1#main-key",
    "owner": "http://localhost:3000/api/v1/activitypub/user-id/1",
    "publicKeyPem": "-----BEGIN PUBLIC KEY-----\nMIIBojANBgkqhkiG9w0BAQEFAAOCAY8AMIIBigKCAYEAo1VDZGWQBDTWKhpWiPQp\n7nD94UsKkcoFwDQVuxE3bMquKEHBomB4cwUnVou922YkL3AmSOr1sX2yJQGqnCLm\nOeKS74/mCIAoYlu0d75bqY4A7kE2VrQmQLZBbmpCTfrPqDaE6Mfm/kXaX7+hsrZS\n4bVvzZCYq8sjtRxdPk+9ku2QhvznwTRlWLvwHmFSGtlQYPRu+f/XqoVM/DVRA/Is\nwDk9yiNIecV+Isus0CBq1jGQkfuVNu1GK2IvcSg9MoDm3VH/tCayAP+xWm0g7sC8\nKay6Y/khvTvE7bWEKGQsJGvi3+4wITLVLVt+GoVOuCzdbhTV2CHBzn7h30AoZD0N\nY6eyb+Q142JykoHadcRwh1a36wgoG7E496wPvV3ST8xdiClca8cDNhOzCj8woY+t\nTFCMl32U3AJ4e/cAsxKRocYLZqc95dDqdNQiIyiRMMkf5NaA/QvelY4PmFuHC0WR\nVuJ4A3mcti2QLS9j0fSwSJdlfolgW6xaPgjdvuSQsgX1AgMBAAE=\n-----END PUBLIC KEY-----\n"
  }
}

Set up a user called "me" for all tests
*/

func TestClientKeyRSA(t *testing.T) {
	rsaKey, err := rsa.GenerateKey(rand.Reader, 3072)
	require.NoError(t, err)

	rsaPubKey, ok := rsaKey.Public().(*rsa.PublicKey)
	assert.True(t, ok)

	derKey := x509.MarshalPKCS1PrivateKey(rsaKey)
	bytes := bytes.NewBufferString("")
	derBlock := &pem.Block{Bytes: derKey}
	require.NoError(t, pem.Encode(bytes, derBlock))

	algs := []setting.Algorithm{
		setting.AlgorithmRSASHA256CAVAGE,
		setting.AlgorithmRSASHA512CAVAGE,
		setting.AlgorithmRSARFC9421,
		setting.AlgorithmRSAPSSRFC9421,
	}

	for _, alg := range algs {
		clientKey := activitypub.NewClientKey(bytes.Bytes(), keyID, alg)
		clientPrivKey, err := clientKey.RSAPrivateKey()
		require.NoError(t, err)
		assert.Equal(t, clientPrivKey, rsaKey)

		clientPubKey, err := clientKey.RSAPublicKey()
		require.NoError(t, err)
		assert.Equal(t, clientPubKey, rsaPubKey)
	}

	otherAlgs := []setting.Algorithm{
		setting.AlgorithmEd25519,
		setting.AlgorithmHMACSHA256,
		setting.AlgorithmP256CAVAGE,
		setting.AlgorithmP256RFC9421,
		setting.AlgorithmP384CAVAGE,
		setting.AlgorithmP384RFC9421,
		setting.AlgorithmNone,
	}

	for _, alg := range otherAlgs {
		clientKey := activitypub.NewClientKey(bytes.Bytes(), keyID, alg)
		_, err = clientKey.RSAPrivateKey()
		require.Error(t, err)
		_, err = clientKey.ECDSAPrivateKey()
		require.Error(t, err)
		_, err = clientKey.Ed25519PrivateKey()
		require.Error(t, err)

		_, err = clientKey.RSAPublicKey()
		require.Error(t, err)
		_, err = clientKey.ECDSAPublicKey()
		require.Error(t, err)
		_, err = clientKey.Ed25519PublicKey()
		require.Error(t, err)
	}
}

func TestClientKeyECDSA(t *testing.T) {
	for _, curve := range []elliptic.Curve{elliptic.P256(), elliptic.P384()} {
		ecdsaKey, err := ecdsa.GenerateKey(curve, rand.Reader)
		require.NoError(t, err)

		ecdsaPubKey, ok := ecdsaKey.Public().(*ecdsa.PublicKey)
		assert.True(t, ok)

		derKey, err := x509.MarshalPKCS8PrivateKey(ecdsaKey)
		require.NoError(t, err)
		bytes := bytes.NewBufferString("")
		derBlock := &pem.Block{Bytes: derKey}
		require.NoError(t, pem.Encode(bytes, derBlock))

		algs := []setting.Algorithm{
			setting.AlgorithmP256CAVAGE,
			setting.AlgorithmP256RFC9421,
			setting.AlgorithmP384CAVAGE,
			setting.AlgorithmP384RFC9421,
		}

		for _, alg := range algs {
			clientKey := activitypub.NewClientKey(bytes.Bytes(), keyID, alg)
			clientPrivKey, err := clientKey.ECDSAPrivateKey()
			require.NoError(t, err)
			assert.Equal(t, clientPrivKey, ecdsaKey)

			clientPubKey, err := clientKey.ECDSAPublicKey()
			require.NoError(t, err)
			assert.Equal(t, clientPubKey, ecdsaPubKey)
		}

		otherAlgs := []setting.Algorithm{
			setting.AlgorithmRSASHA256CAVAGE,
			setting.AlgorithmRSASHA512CAVAGE,
			setting.AlgorithmRSARFC9421,
			setting.AlgorithmRSAPSSRFC9421,
			setting.AlgorithmEd25519,
			setting.AlgorithmHMACSHA256,
			setting.AlgorithmNone,
		}

		for _, alg := range otherAlgs {
			clientKey := activitypub.NewClientKey(bytes.Bytes(), keyID, alg)
			_, err = clientKey.ECDSAPrivateKey()
			require.Error(t, err)
			_, err = clientKey.RSAPrivateKey()
			require.Error(t, err)
			_, err = clientKey.Ed25519PrivateKey()
			require.Error(t, err)

			_, err = clientKey.ECDSAPublicKey()
			require.Error(t, err)
			_, err = clientKey.RSAPublicKey()
			require.Error(t, err)
			_, err = clientKey.Ed25519PublicKey()
			require.Error(t, err)
		}
	}
}

func TestClientKeyEd25519(t *testing.T) {
	ed25519PubKey, ed25519Key, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	derKey, err := x509.MarshalPKCS8PrivateKey(ed25519Key)
	require.NoError(t, err)
	bytes := bytes.NewBufferString("")
	derBlock := &pem.Block{Bytes: derKey}
	require.NoError(t, pem.Encode(bytes, derBlock))

	alg := setting.AlgorithmEd25519

	clientKey := activitypub.NewClientKey(bytes.Bytes(), keyID, alg)
	clientPrivKey, err := clientKey.Ed25519PrivateKey()
	require.NoError(t, err)
	assert.Equal(t, clientPrivKey, &ed25519Key)

	clientPubKey, err := clientKey.Ed25519PublicKey()
	require.NoError(t, err)
	assert.Equal(t, clientPubKey, &ed25519PubKey)

	otherAlgs := []setting.Algorithm{
		setting.AlgorithmP256CAVAGE,
		setting.AlgorithmP256RFC9421,
		setting.AlgorithmP384CAVAGE,
		setting.AlgorithmP384RFC9421,
		setting.AlgorithmRSASHA256CAVAGE,
		setting.AlgorithmRSASHA512CAVAGE,
		setting.AlgorithmRSARFC9421,
		setting.AlgorithmRSAPSSRFC9421,
		setting.AlgorithmHMACSHA256,
		setting.AlgorithmNone,
	}

	for _, alg := range otherAlgs {
		clientKey := activitypub.NewClientKey(bytes.Bytes(), keyID, alg)
		_, err = clientKey.ECDSAPrivateKey()
		require.Error(t, err)
		_, err = clientKey.RSAPrivateKey()
		require.Error(t, err)
		_, err = clientKey.Ed25519PrivateKey()
		require.Error(t, err)

		_, err = clientKey.ECDSAPublicKey()
		require.Error(t, err)
		_, err = clientKey.RSAPublicKey()
		require.Error(t, err)
		_, err = clientKey.Ed25519PublicKey()
		require.Error(t, err)
	}
}

func TestClientCtx(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	pubID := "myGpgId"
	cf, err := activitypub.NewClientFactory()
	log.Debug("ClientFactory: %v\nError: %v", cf, err)
	require.NoError(t, err)

	c, err := cf.WithKeys(db.DefaultContext, user, pubID)

	log.Debug("Client: %v\nError: %v", c, err)
	require.NoError(t, err)
	_ = activitypub.NewContext(db.DefaultContext, cf)
}

/* TODO: bring this test to work or delete
func TestActivityPubSignedGet(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1, Name: "me"})
	pubID := "myGpgId"
	c, err := NewClient(db.DefaultContext, user, pubID)
	require.NoError(t, err)

	expected := "TestActivityPubSignedGet"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Regexp(t, regexp.MustCompile("^"+setting.Federation.DigestAlgorithm), r.Header.Get("Digest"))
		assert.Contains(t, r.Header.Get("Signature"), pubID)
		assert.Equal(t, r.Header.Get("Content-Type"), ActivityStreamsContentType)
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Equal(t, expected, string(body))
		fmt.Fprint(w, expected)
	}))
	defer srv.Close()

	r, err := c.Get(srv.URL)
	require.NoError(t, err)
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	require.NoError(t, err)
	assert.Equal(t, expected, string(body))

}
*/

func TestActivityPubSignedPost(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	pubID := "https://example.com/pubID"
	cf, err := activitypub.NewClientFactory()
	require.NoError(t, err)
	c, err := cf.WithKeys(db.DefaultContext, user, pubID)
	require.NoError(t, err)

	expected := "BODY"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Regexp(t, "^"+setting.Federation.DigestAlgorithm, r.Header.Get("Digest"))
		assert.Contains(t, r.Header.Get("Signature"), pubID)
		assert.Equal(t, activitypub.ActivityStreamsContentType, r.Header.Get("Content-Type"))
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Equal(t, expected, string(body))
		fmt.Fprint(w, expected)
	}))
	defer srv.Close()

	r, err := c.Post([]byte(expected), srv.URL)
	require.NoError(t, err)
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	require.NoError(t, err)
	assert.Equal(t, expected, string(body))
}

func TestActivityPubSignedPostRFC9421(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	pubID := "https://example.com/pubID"
	digestAlgs := []string{"sha-256", "sha-512"}
	cf, err := activitypub.NewClientFactory()
	require.NoError(t, err)
	c, err := cf.WithKeys(db.DefaultContext, user, pubID)
	c.SetRFC9421(true)
	require.NoError(t, err)

	expected := "BODY"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Regexp(t, "^"+strings.ToLower(setting.Federation.DigestAlgorithm), r.Header.Get("Content-Digest"))
		for _, input := range setting.Federation.PostHeadersRFC9421 {
			assert.Contains(t, r.Header.Get("Signature-Input"), strings.ToLower(input))
		}
		assert.Equal(t, activitypub.ActivityStreamsContentType, r.Header.Get("Content-Type"))
		require.NoError(t, activitypub.ValidateContentDigest(r.Header.Get("Content-Digest"), &r.Body, digestAlgs))
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Equal(t, expected, string(body))
		fmt.Fprint(w, expected)
	}))
	defer srv.Close()

	r, err := c.Post([]byte(expected), srv.URL)
	require.NoError(t, err)
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	require.NoError(t, err)
	assert.Equal(t, expected, string(body))
}

func TestActivityPubSignedGetRFC9421(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	pubID := "https://example.com/pubID"
	digestAlgs := []string{"sha-256", "sha-512"}
	cf, err := activitypub.NewClientFactory()
	require.NoError(t, err)
	c, err := cf.WithKeys(db.DefaultContext, user, pubID)
	c.SetRFC9421(true)
	require.NoError(t, err)

	expected := ""
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Regexp(t, "^"+strings.ToLower(setting.Federation.DigestAlgorithm), r.Header.Get("Content-Digest"))
		for _, input := range setting.Federation.PostHeadersRFC9421 {
			assert.Contains(t, r.Header.Get("Signature-Input"), strings.ToLower(input))
		}
		assert.Equal(t, activitypub.ActivityStreamsContentType, r.Header.Get("Content-Type"))
		require.NoError(t, activitypub.ValidateContentDigest(r.Header.Get("Content-Digest"), &r.Body, digestAlgs))
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Equal(t, expected, string(body))
		fmt.Fprint(w, expected)
	}))
	defer srv.Close()

	r, err := c.Get(srv.URL)
	require.NoError(t, err)
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	require.NoError(t, err)
	assert.Equal(t, expected, string(body))
}
