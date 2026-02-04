// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package activitypub

import (
	"fmt"
	"net/http"

	"forgejo.org/modules/activitypub"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
	app_context "forgejo.org/services/context"
	"forgejo.org/services/federation"

	"github.com/42wim/httpsig"
	httpsign9421 "github.com/yaronf/httpsign"
)

func verifyHTTPSignature(ctx app_context.APIContext) (authenticated bool, err error) {
	if !setting.Federation.SignatureEnforced {
		return true, nil
	}

	r := ctx.Req

	// 1. Figure out what key we need to verify
	v, err := httpsig.NewVerifier(r)
	if err != nil {
		log.Debug("For %q verification failed: %v", r.URL.Path, err)
		return false, err
	}

	log.Debug("Verify %q, signed by KeyId: %v", r.URL.Path, v.KeyId())
	signatureAlgorithm := httpsig.Algorithm(setting.Federation.SignatureAlgorithms[0])
	pubKey, err := federation.FindOrCreateFederatedUserKey(ctx, v.KeyId())
	if err != nil || pubKey == nil {
		pubKey, err = federation.FindOrCreateFederationHostKey(ctx, v.KeyId())
		if err != nil {
			log.Debug("For %q verification failed: %v", r.URL.Path, err)
			return false, err
		}
	}

	err = v.Verify(pubKey, signatureAlgorithm)
	if err != nil {
		log.Debug("For %q verification failed: %v", r.URL.Path, err)
		return false, err
	}
	return true, nil
}

func verifyHTTPMessageSignature(ctx app_context.APIContext) (authenticated bool, err error) {
	if !setting.Federation.SignatureEnforced {
		return true, nil
	}

	r := ctx.Req

	sigHeaders := r.Header.Values("Signature")
	if len(sigHeaders) == 0 {
		return false, fmt.Errorf("missing signature header")
	}
	var fields httpsign9421.Fields
	switch r.Method {
	case http.MethodGet:
		fields = httpsign9421.Headers(setting.Federation.GetHeadersRFC9421...)
	case http.MethodPost:
		fields = httpsign9421.Headers(setting.Federation.PostHeadersRFC9421...)
	default:
		return false, fmt.Errorf("unsupported request type: %v", r.Method)
	}
	sigNames, err := httpsign9421.RequestSignatureNames(r, false)
	if err != nil {
		return false, err
	}

	if len(sigNames) == 0 {
		return false, fmt.Errorf("no RFC 9421 signature headers found: %v", r.Header)
	}

	config := httpsign9421.NewVerifyConfig()
	config.SetAllowedAlgs(setting.Federation.SignatureAlgorithmsRFC9421)

	if err = activitypub.ValidateContentDigest(r.Header.Get("Content-Digest"), &r.Body, setting.Federation.DigestAlgorithms); err != nil {
		return false, fmt.Errorf("invalid HTTP Content-Digest header: %v", err)
	}

	for _, name := range sigNames {
		msgDetails, err := httpsign9421.RequestDetails(name, r)
		if err != nil {
			log.Warn("no details for signature: %v, error: %v", name, err)
			continue
		}

		alg, err := setting.AlgorithmFromString(msgDetails.Alg)
		if err != nil {
			log.Debug("invalid HTTP message signature algorithm: %v", err)
			continue
		}

		log.Debug("verifyHttpMessageSignatures signature: %v, alg: %v, key ID: %v", name, alg, msgDetails.KeyID)

		pubKey, err := federation.FindOrCreateFederatedUserKey(ctx, msgDetails.KeyID)
		if err != nil || pubKey == nil {
			_, err = federation.FindOrCreateFederationHostKey(ctx, msgDetails.KeyID)
			if err != nil {
				log.Debug("For %q verification failed: %v", r.URL.Path, err)
				return false, err
			}
		}

		clientKey, err := activitypub.NewClientPublicKey(msgDetails.KeyID, alg).ClientKey(ctx)
		if err != nil {
			log.Warn("error creating client key: %v", err)
			continue
		}
		verifier, err := clientKey.VerifierRFC9421(config, fields)
		if err != nil {
			log.Warn("error creating HTTP message signature verifier: %v", err)
			continue
		}
		if err = httpsign9421.VerifyRequest(name, *verifier, r); err == nil {
			// return on first verified signature
			return true, nil
		}
	}

	return false, fmt.Errorf("no valid HTTP Message Signature (RFC 9421) found")
}

// ReqHTTPSignature function
func ReqHTTPSignature() func(ctx *app_context.APIContext) {
	return func(ctx *app_context.APIContext) {
		var (
			authenticated bool
			err           error
		)

		if len(ctx.Req.Header.Get("Signature-Input")) != 0 {
			// RFC9421 includes the `Signature-Input` header,
			// draft-cavage-http-signatures does not
			authenticated, err = verifyHTTPMessageSignature(*ctx)
		} else {
			authenticated, err = verifyHTTPSignature(*ctx)
		}

		if err != nil {
			log.Warn("verifyHttpSignatures failed: %v", err)
			ctx.Error(http.StatusBadRequest, "reqSignature", "request signature verification failed")
		} else if !authenticated {
			ctx.Error(http.StatusForbidden, "reqSignature", "request signature verification failed")
		}
	}
}
