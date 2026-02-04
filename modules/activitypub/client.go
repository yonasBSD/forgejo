// Copyright 2022 The Gitea Authors. All rights reserved.
// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

// TODO: Think about whether this should be moved to services/activitypub (compare to exosy/services/activitypub/client.go)
package activitypub

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	forgefed_model "forgejo.org/models/forgefed"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/log"
	"forgejo.org/modules/proxy"
	"forgejo.org/modules/setting"

	"github.com/42wim/httpsig"
	httpsign9421 "github.com/yaronf/httpsign"
)

const (
	// ActivityStreamsContentType const
	ActivityStreamsContentType = `application/ld+json; profile="https://www.w3.org/ns/activitystreams"`
	httpsigExpirationTime      = 60
)

func CurrentTime() string {
	return time.Now().UTC().Format(http.TimeFormat)
}

func containsRequiredHTTPHeaders(method string, headers []string) error {
	var hasRequestTarget, hasDate, hasDigest, hasHost bool
	for _, header := range headers {
		hasRequestTarget = hasRequestTarget || header == httpsig.RequestTarget
		hasDate = hasDate || header == "Date"
		hasDigest = hasDigest || header == "Digest"
		hasHost = hasHost || header == "Host"
	}
	if !hasRequestTarget {
		return fmt.Errorf("missing http header for %s: %s", method, httpsig.RequestTarget)
	} else if !hasDate {
		return fmt.Errorf("missing http header for %s: Date", method)
	} else if !hasHost {
		return fmt.Errorf("missing http header for %s: Host", method)
	} else if !hasDigest && method != http.MethodGet {
		return fmt.Errorf("missing http header for %s: Digest", method)
	}
	return nil
}

// Client struct
type ClientFactory struct {
	client      *http.Client
	algs        []httpsig.Algorithm
	digestAlg   httpsig.DigestAlgorithm
	getHeaders  []string
	postHeaders []string
}

// NewClient function
func NewClientFactory() (c *ClientFactory, err error) {
	return NewClientFactoryWithTimeout(5 * time.Second)
}

// NewClient function
func NewClientFactoryWithTimeout(timeout time.Duration) (c *ClientFactory, err error) {
	if err = containsRequiredHTTPHeaders(http.MethodGet, setting.Federation.GetHeaders); err != nil {
		return nil, err
	} else if err = containsRequiredHTTPHeaders(http.MethodPost, setting.Federation.PostHeaders); err != nil {
		return nil, err
	}

	c = &ClientFactory{
		client: &http.Client{
			Transport: &http.Transport{
				Proxy: proxy.Proxy(),
			},
			Timeout: timeout,
		},
		algs:        setting.HttpsigAlgs,
		digestAlg:   httpsig.DigestAlgorithm(setting.Federation.DigestAlgorithm),
		getHeaders:  setting.Federation.GetHeaders,
		postHeaders: setting.Federation.PostHeaders,
	}
	return c, err
}

type APClientFactory interface {
	WithKeys(ctx context.Context, user *user_model.User, pubID string) (APClient, error)
	WithKeysDirect(ctx context.Context, privateKey, pubID string) (APClient, error)
	WithPublicKeys(ctx context.Context, keys []ClientPublicKey) (APClient, error)
}

// ClientPublicKey is a convenience struct used to construct a verifying ClientKey
type ClientPublicKey struct {
	KeyID string
	Alg   setting.Algorithm
}

// NewClientPublicKey creates a new ClientPublicKey for verifying signatures.
func NewClientPublicKey(keyID string, alg setting.Algorithm) ClientPublicKey {
	return ClientPublicKey{keyID, alg}
}

func (k ClientPublicKey) PublicKeyBytes(ctx context.Context) ([]byte, error) {
	_, federatedUser, err := user_model.FindFederatedUserByKeyID(ctx, k.KeyID)
	if err == nil && federatedUser != nil && federatedUser.PublicKey.Valid {
		return federatedUser.PublicKey.V, nil
	} else if host, err := forgefed_model.FindFederationHostByKeyID(ctx, k.KeyID); err == nil && host != nil && host.PublicKey.Valid {
		return host.PublicKey.V, nil
	}
	return nil, fmt.Errorf("error finding public key for key ID: %v, error: %v", k.KeyID, err)
}

// ClientKey creates a new [ClientKey].
func (k ClientPublicKey) ClientKey(ctx context.Context) (*ClientKey, error) {
	keyBytes, err := k.PublicKeyBytes(ctx)
	if err != nil {
		return nil, err
	}
	clientKey := &ClientKey{
		pubKey:   keyBytes,
		pubKeyID: k.KeyID,
		alg:      k.Alg,
	}
	return clientKey, nil
}

// ClientKey represents a client key used to sign a HTTP request.
type ClientKey struct {
	privKey  []byte
	pubKey   []byte
	pubKeyID string
	alg      setting.Algorithm
}

// NewClientKey creates a new [ClientKey] from the provided parameters.
func NewClientKey(privKey []byte, pubKeyID string, alg setting.Algorithm) *ClientKey {
	return &ClientKey{
		privKey:  privKey,
		pubKeyID: pubKeyID,
		alg:      alg,
	}
}

// RSAPrivateKey attempts to parse an RSA private key from the [ClientKey].
func (k ClientKey) RSAPrivateKey() (*rsa.PrivateKey, error) {
	switch k.alg {
	case setting.AlgorithmRSARFC9421, setting.AlgorithmRSAPSSRFC9421, setting.AlgorithmRSASHA256CAVAGE, setting.AlgorithmRSASHA512CAVAGE:
		privPem, _ := pem.Decode(k.privKey)
		return x509.ParsePKCS1PrivateKey(privPem.Bytes)
	default:
		return nil, fmt.Errorf("invalid signing algorithm: %v", k.alg)
	}
}

// RSAPublicKey attempts to parse an RSA private key from the [ClientKey].
func (k ClientKey) RSAPublicKey() (*rsa.PublicKey, error) {
	if k.privKey == nil {
		switch k.alg {
		case setting.AlgorithmRSARFC9421, setting.AlgorithmRSAPSSRFC9421, setting.AlgorithmRSASHA256CAVAGE, setting.AlgorithmRSASHA512CAVAGE:
			key, err := x509.ParsePKIXPublicKey(k.pubKey)
			if err != nil {
				return nil, err
			}
			pk, ok := key.(*rsa.PublicKey)
			if !ok {
				return nil, fmt.Errorf("invalid RSA public key")
			}
			return pk, nil
		default:
			return nil, fmt.Errorf("invalid signing algorithm: %v", k.alg)
		}
	} else {
		priv, err := k.RSAPrivateKey()
		if err != nil {
			return nil, err
		}
		pk, ok := priv.Public().(*rsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("invalid RSA public key")
		}
		return pk, nil
	}
}

// ECDSAPrivateKey attempts to parse an ECDSA private key from the [ClientKey].
func (k ClientKey) ECDSAPrivateKey() (*ecdsa.PrivateKey, error) {
	switch k.alg {
	case setting.AlgorithmP256CAVAGE, setting.AlgorithmP256RFC9421, setting.AlgorithmP384CAVAGE, setting.AlgorithmP384RFC9421:
		privPem, _ := pem.Decode(k.privKey)
		pk, err := x509.ParsePKCS8PrivateKey(privPem.Bytes)
		if err != nil {
			return nil, err
		}
		ecdsaPriv, ok := pk.(*ecdsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("invalid ECDSA P256 private key")
		}
		return ecdsaPriv, nil
	default:
		return nil, fmt.Errorf("invalid ECDSA algorithm: %v", k.alg)
	}
}

// ECDSAPublicKey attempts to parse an ECDSA public key from the [ClientKey].
func (k ClientKey) ECDSAPublicKey() (*ecdsa.PublicKey, error) {
	if k.privKey == nil {
		pk, err := x509.ParsePKIXPublicKey(k.pubKey)
		if err != nil {
			return nil, err
		}

		switch k.alg {
		case setting.AlgorithmP256CAVAGE, setting.AlgorithmP256RFC9421,
			setting.AlgorithmP384CAVAGE, setting.AlgorithmP384RFC9421:
			k, ok := pk.(*ecdsa.PublicKey)
			if !ok {
				return nil, fmt.Errorf("invalid ECDSA public key")
			}
			return k, nil
		default:
			return nil, fmt.Errorf("invalid ECDSA algorithm: %v", k.alg)
		}
	} else {
		priv, err := k.ECDSAPrivateKey()
		if err != nil {
			return nil, err
		}
		pk, ok := priv.Public().(*ecdsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("invalid ECDSA public key")
		}
		return pk, nil
	}
}

// Ed25519PrivateKey attempts to parse an Ed25519 private key from the [ClientKey].
func (k ClientKey) Ed25519PrivateKey() (*ed25519.PrivateKey, error) {
	if k.privKey == nil {
		return nil, fmt.Errorf("nil private key")
	}

	switch k.alg {
	case setting.AlgorithmEd25519:
		privPem, _ := pem.Decode(k.privKey)
		pk, err := x509.ParsePKCS8PrivateKey(privPem.Bytes)
		if err != nil {
			return nil, err
		}
		k, ok := pk.(ed25519.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("invalid Ed25519 private key")
		}
		return &k, nil
	default:
		return nil, fmt.Errorf("invalid Ed25519 algorithm: %v", k.alg)
	}
}

// Ed25519PublicKey attempts to parse an Ed25519 public key from the [ClientKey].
func (k ClientKey) Ed25519PublicKey() (*ed25519.PublicKey, error) {
	if k.privKey == nil {
		switch k.alg {
		case setting.AlgorithmEd25519:
			pk, err := x509.ParsePKIXPublicKey(k.pubKey)
			if err != nil {
				return nil, err
			}

			k, ok := pk.(ed25519.PublicKey)
			if !ok {
				return nil, fmt.Errorf("invalid Ed25519 public key")
			}
			return &k, nil
		default:
			return nil, fmt.Errorf("invalid Ed25519 algorithm: %v", k.alg)
		}
	} else {
		priv, err := k.Ed25519PrivateKey()
		if err != nil {
			return nil, err
		}
		pk, ok := priv.Public().(ed25519.PublicKey)
		if !ok {
			return nil, fmt.Errorf("invalid Ed25519 public key")
		}
		return &pk, nil
	}
}

type RFC9421Config struct {
	Signer   *httpsign9421.SignConfig
	Verifier *httpsign9421.VerifyConfig
}

func (k ClientKey) SignerRFC9421Ed25519(config RFC9421Config, fields httpsign9421.Fields) (*httpsign9421.Signer, error) {
	pk, err := k.Ed25519PrivateKey()
	if err != nil {
		return nil, err
	}
	return httpsign9421.NewEd25519Signer(*pk, config.Signer, fields)
}

func (k ClientKey) VerifierRFC9421Ed25519(config RFC9421Config, fields httpsign9421.Fields) (*httpsign9421.Verifier, error) {
	pk, err := k.Ed25519PublicKey()
	if err != nil {
		return nil, err
	}
	return httpsign9421.NewEd25519Verifier(*pk, config.Verifier, fields)
}

func (k ClientKey) SignerRFC9421HMACSHA256(config RFC9421Config, fields httpsign9421.Fields) (*httpsign9421.Signer, error) {
	return httpsign9421.NewHMACSHA256Signer(k.privKey, config.Signer, fields)
}

func (k ClientKey) VerifierRFC9421HMACSHA256(config RFC9421Config, fields httpsign9421.Fields) (*httpsign9421.Verifier, error) {
	return httpsign9421.NewHMACSHA256Verifier(k.pubKey, config.Verifier, fields)
}

func (k ClientKey) SignerRFC9421P256(config RFC9421Config, fields httpsign9421.Fields) (*httpsign9421.Signer, error) {
	pk, err := k.ECDSAPrivateKey()
	if err != nil {
		return nil, err
	}
	return httpsign9421.NewP256Signer(*pk, config.Signer, fields)
}

func (k ClientKey) VerifierRFC9421P256(config RFC9421Config, fields httpsign9421.Fields) (*httpsign9421.Verifier, error) {
	pk, err := k.ECDSAPublicKey()
	if err != nil {
		return nil, err
	}
	return httpsign9421.NewP256Verifier(*pk, config.Verifier, fields)
}

func (k ClientKey) SignerRFC9421P384(config RFC9421Config, fields httpsign9421.Fields) (*httpsign9421.Signer, error) {
	pk, err := k.ECDSAPrivateKey()
	if err != nil {
		return nil, err
	}
	return httpsign9421.NewP384Signer(*pk, config.Signer, fields)
}

func (k ClientKey) VerifierRFC9421P384(config RFC9421Config, fields httpsign9421.Fields) (*httpsign9421.Verifier, error) {
	pk, err := k.ECDSAPublicKey()
	if err != nil {
		return nil, err
	}
	return httpsign9421.NewP384Verifier(*pk, config.Verifier, fields)
}

func (k ClientKey) SignerRFC9421RSA(config RFC9421Config, fields httpsign9421.Fields) (*httpsign9421.Signer, error) {
	pk, err := k.RSAPrivateKey()
	if err != nil {
		return nil, err
	}
	return httpsign9421.NewRSASigner(*pk, config.Signer, fields)
}

func (k ClientKey) VerifierRFC9421RSA(config RFC9421Config, fields httpsign9421.Fields) (*httpsign9421.Verifier, error) {
	pk, err := k.RSAPublicKey()
	if err != nil {
		return nil, err
	}
	return httpsign9421.NewRSAVerifier(*pk, config.Verifier, fields)
}

func (k ClientKey) SignerRFC9421RSAPSS(config RFC9421Config, fields httpsign9421.Fields) (*httpsign9421.Signer, error) {
	pk, err := k.RSAPrivateKey()
	if err != nil {
		return nil, err
	}
	return httpsign9421.NewRSAPSSSigner(*pk, config.Signer, fields)
}

func (k ClientKey) VerifierRFC9421RSAPSS(config RFC9421Config, fields httpsign9421.Fields) (*httpsign9421.Verifier, error) {
	pk, err := k.RSAPublicKey()
	if err != nil {
		return nil, err
	}
	return httpsign9421.NewRSAPSSVerifier(*pk, config.Verifier, fields)
}

// SignerRFC9421 attempts to create a valid RFC 9421 HTTP Message signer.
//

func (k ClientKey) SignerRFC9421(config *httpsign9421.SignConfig, fields httpsign9421.Fields) (*httpsign9421.Signer, error) {
	if config == nil {
		return nil, fmt.Errorf("nil signer config")
	}

	config.SetKeyID(k.pubKeyID)

	signConfig := RFC9421Config{
		Signer: config,
	}

	switch k.alg {
	case setting.AlgorithmEd25519:
		return k.SignerRFC9421Ed25519(signConfig, fields)
	case setting.AlgorithmHMACSHA256:
		return k.SignerRFC9421HMACSHA256(signConfig, fields)
	case setting.AlgorithmP256RFC9421:
		return k.SignerRFC9421P256(signConfig, fields)
	case setting.AlgorithmP384RFC9421:
		return k.SignerRFC9421P384(signConfig, fields)
	case setting.AlgorithmRSARFC9421:
		return k.SignerRFC9421RSA(signConfig, fields)
	case setting.AlgorithmRSAPSSRFC9421:
		return k.SignerRFC9421RSAPSS(signConfig, fields)
	default:
		return nil, fmt.Errorf("invalid RFC 9421 signature algorithm: %v", k.alg)
	}
}

// VerifierRFC9421 attempts to create a valid RFC 9421 HTTP Message verifier.
//

func (k ClientKey) VerifierRFC9421(config *httpsign9421.VerifyConfig, fields httpsign9421.Fields) (*httpsign9421.Verifier, error) {
	if config == nil {
		return nil, fmt.Errorf("nil verifier config")
	}

	config.SetKeyID(k.pubKeyID)

	verifyConfig := RFC9421Config{
		Verifier: config,
	}

	switch k.alg {
	case setting.AlgorithmEd25519:
		return k.VerifierRFC9421Ed25519(verifyConfig, fields)
	case setting.AlgorithmHMACSHA256:
		return k.VerifierRFC9421HMACSHA256(verifyConfig, fields)
	case setting.AlgorithmP256RFC9421:
		return k.VerifierRFC9421P256(verifyConfig, fields)
	case setting.AlgorithmP384RFC9421:
		return k.VerifierRFC9421P384(verifyConfig, fields)
	case setting.AlgorithmRSARFC9421:
		return k.VerifierRFC9421RSA(verifyConfig, fields)
	case setting.AlgorithmRSAPSSRFC9421:
		return k.VerifierRFC9421RSAPSS(verifyConfig, fields)
	default:
		return nil, fmt.Errorf("invalid RFC 9421 signature algorithm: %v", k.alg)
	}
}

// Client struct
type Client struct {
	client      *http.Client
	algs        []httpsig.Algorithm
	digestAlg   httpsig.DigestAlgorithm
	getHeaders  []string
	postHeaders []string
	priv        *rsa.PrivateKey
	pubID       string
	clientKeys  []*ClientKey
	useRFC9421  bool
}

// NewRequest function
func (cf *ClientFactory) WithKeysDirect(ctx context.Context, privateKey, pubID string) (APClient, error) {
	privPem, _ := pem.Decode([]byte(privateKey))
	privParsed, err := x509.ParsePKCS1PrivateKey(privPem.Bytes)
	if err != nil {
		return nil, err
	}

	clientKeys := []*ClientKey{
		NewClientKey([]byte(privateKey), pubID, setting.AlgorithmRSASHA256CAVAGE),
		NewClientKey([]byte(privateKey), pubID, setting.AlgorithmRSARFC9421),
	}

	c := Client{
		client:      cf.client,
		algs:        cf.algs,
		digestAlg:   cf.digestAlg,
		getHeaders:  cf.getHeaders,
		postHeaders: cf.postHeaders,
		priv:        privParsed,
		pubID:       pubID,
		clientKeys:  clientKeys,
		useRFC9421:  setting.Federation.UseRFC9421,
	}
	return &c, nil
}

func (cf *ClientFactory) WithKeys(ctx context.Context, user *user_model.User, pubID string) (APClient, error) {
	priv, err := GetPrivateKey(ctx, user)
	if err != nil {
		return nil, err
	}
	return cf.WithKeysDirect(ctx, priv, pubID)
}

// WithPublicKeys creates a new [Client] for verifying signatures.
func (cf *ClientFactory) WithPublicKeys(ctx context.Context, pubKeys []ClientPublicKey) (APClient, error) {
	clientKeys := make([]*ClientKey, len(pubKeys))

	for i, key := range pubKeys {
		clientKey, err := key.ClientKey(ctx)
		if err != nil {
			return nil, err
		}
		clientKeys[i] = clientKey
	}

	c := Client{
		client:      cf.client,
		algs:        cf.algs,
		digestAlg:   cf.digestAlg,
		getHeaders:  cf.getHeaders,
		postHeaders: cf.postHeaders,
		clientKeys:  clientKeys,
		useRFC9421:  setting.Federation.UseRFC9421,
	}
	return &c, nil
}

// NewRequest function
func (c *Client) newRequest(method string, b []byte, to string) (req *http.Request, err error) {
	buf := bytes.NewBuffer(b)
	req, err = http.NewRequest(method, to, buf)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Accept", "application/json, "+ActivityStreamsContentType)
	req.Header.Add("Date", CurrentTime())
	req.Header.Add("Host", req.URL.Host)
	req.Header.Add("User-Agent", "Gitea/"+setting.AppVer)
	req.Header.Add("Content-Type", ActivityStreamsContentType)

	return req, err
}

// SignersRFC9421 attempts to create a list of valid RFC 9421 HTTP Message signers.
func (c *Client) SignersRFC9421(config *httpsign9421.SignConfig, fields httpsign9421.Fields) ([]httpsign9421.Signer, error) {
	if config == nil {
		return nil, fmt.Errorf("nil signer config")
	}
	var (
		err     error
		signer  *httpsign9421.Signer
		signers []httpsign9421.Signer
	)
	for _, key := range c.clientKeys {
		signer, err = key.SignerRFC9421(config, fields)
		if err != nil {
			log.Warn("Invalid RFC 9421 signer: %v", err)
			continue
		}
		signers = append(signers, *signer)
	}

	return signers, nil
}

// ContentDigest calculates the SHA256 + SHA512 content-digest header value.
func ContentDigest(b *io.ReadCloser, digestAlgs []string) (string, error) {
	digest, err := httpsign9421.GenerateContentDigestHeader(b, digestAlgs)
	if err != nil {
		return "", fmt.Errorf("error generating RFC 9421 content-digest: %v", err)
	}
	return digest, nil
}

// ValidateContentDigest validates the `Content-Digest` header.
//
// `digestAlgs` represents the list of expected digest algorithms, e.g. "sha-256", "sha-512"
func ValidateContentDigest(header string, b *io.ReadCloser, digestAlgs []string) error {
	if b == nil {
		return fmt.Errorf("nil content-digest body I/O reader")
	}
	body := bytes.NewBufferString("")
	if _, err := body.ReadFrom(*b); err != nil {
		return err
	}
	*b = io.NopCloser(bytes.NewBufferString(body.String()))
	bodyReader := io.NopCloser(body)

	return httpsign9421.ValidateContentDigestHeader([]string{header}, &bodyReader, digestAlgs)
}

// Do makes an HTTP request to an ActivityPub server.
func (c *Client) Do(req *http.Request) (resp *http.Response, err error) {
	if req == nil {
		return nil, fmt.Errorf("nil ActivityPub request")
	}

	return c.client.Do(req)
}

// Post constructs a POST request with forgejo/gitea specific headers, and makes the request to the server.
func (c *Client) Post(b []byte, to string) (resp *http.Response, err error) {
	req, err := c.PostRequest(b, to)
	if err != nil {
		return nil, err
	}

	return c.Do(req)
}

// KeyID gets the key ID for the ActivityPub public key.
func (c *Client) KeyID() string {
	return c.pubID
}

// Create an http POST request with forgejo/gitea specific headers
//
//nolint:dupl
func (c *Client) PostRequest(b []byte, to string) (req *http.Request, err error) {
	if req, err = c.newRequest(http.MethodPost, b, to); err != nil {
		return nil, err
	}

	if c.pubID != "" {
		if c.useRFC9421 {
			config := httpsign9421.NewSignConfig().SignCreated(true)
			fields := httpsign9421.Headers(setting.Federation.PostHeadersRFC9421...)

			signers, err := c.SignersRFC9421(config, fields)
			if err != nil {
				return nil, err
			}

			req.Header.Set("Created", fmt.Sprintf("%d", time.Now().Unix()))
			digest, err := ContentDigest(&req.Body, setting.Federation.DigestAlgorithms)
			if err != nil {
				return nil, err
			}
			req.Header.Set("Content-Digest", digest)

			for i, signer := range signers {
				sigName := fmt.Sprintf("sig%d", i+1)
				signatureInput, signature, err := httpsign9421.SignRequest(sigName, signer, req)
				if err != nil {
					return nil, err
				}
				req.Header.Set("Signature-Input", signatureInput)
				req.Header.Set("Signature", signature)
			}
		} else {
			signer, _, err := httpsig.NewSigner(c.algs, c.digestAlg, c.postHeaders, httpsig.Signature, httpsigExpirationTime)
			if err != nil {
				return nil, err
			}
			if err := signer.SignRequest(c.priv, c.pubID, req, b); err != nil {
				return nil, err
			}
		}
	}

	return req, nil
}

// Create an http GET request with forgejo/gitea specific headers
func (c *Client) Get(to string) (resp *http.Response, err error) {
	req, err := c.GetRequest(to)
	if err != nil {
		return nil, err
	}

	return c.Do(req)
}

// Create an http GET request with forgejo/gitea specific headers
//
//nolint:dupl
func (c *Client) GetRequest(to string) (req *http.Request, err error) {
	if req, err = c.newRequest(http.MethodGet, nil, to); err != nil {
		return nil, err
	}

	if c.pubID != "" {
		if c.useRFC9421 {
			config := httpsign9421.NewSignConfig().SignCreated(true)
			fields := httpsign9421.Headers(setting.Federation.GetHeadersRFC9421...)

			signers, err := c.SignersRFC9421(config, fields)
			if err != nil {
				return nil, err
			}

			req.Header.Set("Created", fmt.Sprintf("%d", time.Now().Unix()))
			digest, err := ContentDigest(&req.Body, setting.Federation.DigestAlgorithms)
			if err != nil {
				return nil, err
			}
			req.Header.Set("Content-Digest", digest)

			for i, signer := range signers {
				sigName := fmt.Sprintf("sig%d", i+1)
				signatureInput, signature, err := httpsign9421.SignRequest(sigName, signer, req)
				if err != nil {
					return nil, err
				}
				req.Header.Set("Signature-Input", signatureInput)
				req.Header.Set("Signature", signature)
			}
		} else {
			signer, _, err := httpsig.NewSigner(c.algs, c.digestAlg, c.getHeaders, httpsig.Signature, httpsigExpirationTime)
			if err != nil {
				return nil, err
			}
			if err := signer.SignRequest(c.priv, c.pubID, req, nil); err != nil {
				return nil, err
			}
		}
	}

	return req, nil
}

// Create an http GET request with forgejo/gitea specific headers
func (c *Client) GetBody(uri string) ([]byte, error) {
	response, err := c.Get(uri)
	if err != nil {
		return nil, err
	}
	log.Debug("Client: got status: %v", response.Status)
	if response.StatusCode != 200 {
		err = fmt.Errorf("got non 200 status code for id: %v", uri)
		return nil, err
	}
	defer response.Body.Close()
	if response.ContentLength > setting.Federation.MaxSize {
		return nil, fmt.Errorf("Request returned %d bytes (max allowed incoming size: %d bytes)", response.ContentLength, setting.Federation.MaxSize)
	} else if response.ContentLength == -1 {
		log.Warn("Request to %v returned an unknown content length, response may be truncated to %d bytes", uri, setting.Federation.MaxSize)
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, setting.Federation.MaxSize))
	if err != nil {
		return nil, err
	}

	log.Debug("Client: got body: %v", charLimiter(string(body), 120))
	return body, nil
}

// SetRFC9421 sets whether to sign requests using the RFC 9421 HTTP Message Signing algorithm.
func (c *Client) SetRFC9421(use bool) {
	c.useRFC9421 = use
}

// GetRFC9421 gets whether the RFC 9421 HTTP Message Signing algorithm is used to sign requests.
func (c *Client) GetRFC9421() bool {
	return c.useRFC9421
}

func (c *Client) SignedHeaders(method string) string {
	var ret string

	switch method {
	case http.MethodGet:
		if c.GetRFC9421() {
			ret = fmt.Sprintf(`"%v"`, strings.Join(setting.Federation.GetHeadersRFC9421, `" "`))
		} else {
			ret = strings.Join(setting.Federation.GetHeaders, " ")
		}
	case http.MethodPost:
		if c.GetRFC9421() {
			ret = fmt.Sprintf(`"%v"`, strings.Join(setting.Federation.PostHeadersRFC9421, `" "`))
		} else {
			ret = strings.Join(setting.Federation.PostHeaders, " ")
		}
	default:
		ret = ""
	}

	return strings.ToLower(ret)
}

// Limit number of characters in a string (useful to prevent log injection attacks and overly long log outputs)
// Thanks to https://www.socketloop.com/tutorials/golang-characters-limiter-example
func charLimiter(s string, limit int) string {
	reader := strings.NewReader(s)
	buff := make([]byte, limit)
	n, _ := io.ReadAtLeast(reader, buff, limit)
	if n != 0 {
		return fmt.Sprint(string(buff), "...")
	}
	return s
}

type APClient interface {
	newRequest(method string, b []byte, to string) (req *http.Request, err error)
	Post(b []byte, to string) (resp *http.Response, err error)
	PostRequest(b []byte, to string) (req *http.Request, err error)
	Get(to string) (resp *http.Response, err error)
	GetRequest(to string) (req *http.Request, err error)
	GetBody(uri string) ([]byte, error)
	Do(req *http.Request) (resp *http.Response, err error)
	GetRFC9421() bool
	SetRFC9421(use bool)
	KeyID() string
	SignedHeaders(method string) string
}

// contextKey is a value for use with context.WithValue.
type contextKey struct {
	name string
}

// clientFactoryContextKey is a context key. It is used with context.Value() to get the current Food for the context
var (
	clientFactoryContextKey                 = &contextKey{"clientFactory"}
	_                       APClientFactory = &ClientFactory{}
)

// Context represents an activitypub client factory context
type Context struct {
	context.Context
	e APClientFactory
}

func NewContext(ctx context.Context, e APClientFactory) *Context {
	return &Context{
		Context: ctx,
		e:       e,
	}
}

// APClientFactory represents an activitypub client factory
func (ctx *Context) APClientFactory() APClientFactory {
	return ctx.e
}

// provides APClientFactory
type GetAPClient interface {
	GetClientFactory() APClientFactory
}

// GetClientFactory will get an APClientFactory from this context or returns the default implementation
func GetClientFactory(ctx context.Context) (APClientFactory, error) {
	if e := getClientFactory(ctx); e != nil {
		return e, nil
	}
	return NewClientFactory()
}

// getClientFactory will get an APClientFactory from this context or return nil
func getClientFactory(ctx context.Context) APClientFactory {
	if clientFactory, ok := ctx.(APClientFactory); ok {
		return clientFactory
	}
	clientFactoryInterface := ctx.Value(clientFactoryContextKey)
	if clientFactoryInterface != nil {
		return clientFactoryInterface.(GetAPClient).GetClientFactory()
	}
	return nil
}
