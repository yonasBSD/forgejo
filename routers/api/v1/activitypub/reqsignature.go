// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package activitypub

import (
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/url"

	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/activitypub"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	gitea_context "code.gitea.io/gitea/services/context"

	ap "github.com/go-ap/activitypub"
	"github.com/go-fed/httpsig"
)

func getPublicKeyFromResponse(b []byte, keyID *url.URL) (p crypto.PublicKey, err error) {
	person := ap.PersonNew(ap.IRI(keyID.String()))
	err = person.UnmarshalJSON(b)
	if err != nil {
		return nil, fmt.Errorf("ActivityStreams type cannot be converted to one known to have publicKey property: %w", err)
	}
	pubKey := person.PublicKey
	if pubKey.ID.String() != keyID.String() {
		return nil, fmt.Errorf("cannot find publicKey with id: %s in %s", keyID, string(b))
	}
	pubKeyPem := pubKey.PublicKeyPem
	block, _ := pem.Decode([]byte(pubKeyPem))
	if block == nil || block.Type != "PUBLIC KEY" {
		return nil, fmt.Errorf("could not decode publicKeyPem to PUBLIC KEY pem block type")
	}
	p, err = x509.ParsePKIXPublicKey(block.Bytes)
	return p, err
}

func fetch(ctx *gitea_context.APIContext, iri *url.URL) (b []byte, err error) {
	client, err := activitypub.NewClient(ctx, user_model.NewAPActorUser(), user_model.APActorUserAPActorID())
	if err != nil {
		return nil, err
	}
	return client.GetBody(iri.String())
}

func verifyHTTPSignatures(ctx *gitea_context.APIContext) (authenticated bool, err error) {
	r := ctx.Req

	// 1. Figure out what key we need to verify
	v, err := httpsig.NewVerifier(r)
	if err != nil {
		return false, err
	}
	ID := v.KeyId()
	idIRI, err := url.Parse(ID)
	if err != nil {
		return false, err
	}
	// 2. Fetch the public key of the other actor
	b, err := fetch(ctx, idIRI)
	if err != nil {
		return false, err
	}
	pubKey, err := getPublicKeyFromResponse(b, idIRI)
	if err != nil {
		return false, err
	}
	// 3. Verify the other actor's key
	algo := httpsig.Algorithm(setting.Federation.Algorithms[0])
	authenticated = v.Verify(pubKey, algo) == nil
	return authenticated, err
}

// ReqHTTPSignature function
func ReqHTTPSignature() func(ctx *gitea_context.APIContext) {
	return func(ctx *gitea_context.APIContext) {
		if authenticated, err := verifyHTTPSignatures(ctx); err != nil {
			log.Warn("verifyHttpSignatures failed: %v", err)
			ctx.Error(http.StatusBadRequest, "reqSignature", "request signature verification failed")
		} else if !authenticated {
			ctx.Error(http.StatusForbidden, "reqSignature", "request signature verification failed")
		}
	}
}
