// Copyright 2021 The Gitea Authors. All rights reserved.
// Copyright 2023,2024,2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package setting

import (
	"fmt"

	"forgejo.org/modules/log"

	"github.com/42wim/httpsig"
)

type Algorithm string

const (
	AlgorithmEd25519         Algorithm = "ed25519"
	AlgorithmHMACSHA256      Algorithm = "hmac-sha256"
	AlgorithmP256CAVAGE      Algorithm = Algorithm(string(httpsig.ECDSA_SHA256))
	AlgorithmP256RFC9421     Algorithm = "ecdsa-p256-sha256"
	AlgorithmP384CAVAGE      Algorithm = Algorithm(string(httpsig.ECDSA_SHA384))
	AlgorithmP384RFC9421     Algorithm = "ecdsa-p384-sha384"
	AlgorithmRSASHA512CAVAGE Algorithm = Algorithm(string(httpsig.RSA_SHA512))
	AlgorithmRSAPSSRFC9421   Algorithm = "rsa-pss-sha512"
	AlgorithmRSASHA256CAVAGE Algorithm = Algorithm(string(httpsig.RSA_SHA256))
	AlgorithmRSARFC9421      Algorithm = "rsa-v1_5-sha256"
	AlgorithmNone            Algorithm = "none"
)

// AlgorithmFromString attempts to convert a string to a valid [Algortihm] variant.
func AlgorithmFromString(algString string) (Algorithm, error) {
	switch alg := Algorithm(algString); alg {
	case AlgorithmEd25519, AlgorithmHMACSHA256, AlgorithmP256CAVAGE, AlgorithmP256RFC9421, AlgorithmP384CAVAGE, AlgorithmP384RFC9421, AlgorithmRSASHA512CAVAGE, AlgorithmRSAPSSRFC9421, AlgorithmRSASHA256CAVAGE, AlgorithmRSARFC9421:
		return alg, nil
	default:
		return AlgorithmNone, fmt.Errorf("invalid signature algorithm: %v", algString)
	}
}

// Federation settings
var (
	Federation = struct {
		Enabled                    bool     `ini:"ENABLED"`
		ShareUserStatistics        bool     `ini:"SHARE_USER_STATISTICS"`
		MaxSize                    int64    `ini:"MAX_SIZE"`
		SignatureAlgorithms        []string `ini:"SIGNATURE_ALGORITHMS"`
		SignatureAlgorithmsRFC9421 []string `ini:"SIGNATURE_ALGORITHMS_RFC9421"`
		DigestAlgorithm            string   `ini:"DIGEST_ALGORITHM"`
		DigestAlgorithms           []string `ini:"DIGEST_ALGORITHMS"`
		GetHeaders                 []string `ini:"GET_HEADERS"`
		GetHeadersRFC9421          []string `ini:"GET_HEADERS_RFC9421"`
		PostHeaders                []string `ini:"POST_HEADERS"`
		PostHeadersRFC9421         []string `ini:"POST_HEADERS_RFC9421"`
		SignatureEnforced          bool     `ini:"SIGNATURE_ENFORCED"`
		UseRFC9421                 bool     `ini:"USE_RFC9421"`
	}{
		Enabled:                    false,
		ShareUserStatistics:        true,
		MaxSize:                    4,
		SignatureAlgorithms:        []string{"rsa-sha256", "rsa-sha512", "ed25519"},
		SignatureAlgorithmsRFC9421: []string{string(AlgorithmRSARFC9421)},
		DigestAlgorithm:            "SHA-256",
		DigestAlgorithms:           []string{"sha-256", "sha-512"},
		GetHeaders:                 []string{"(request-target)", "Date", "Host"},
		GetHeadersRFC9421:          []string{"@method", "@target-uri", "Content-Digest", "Created"},
		PostHeaders:                []string{"(request-target)", "Date", "Host", "Digest"},
		PostHeadersRFC9421:         []string{"@method", "@target-uri", "Content-Digest", "Created"},
		SignatureEnforced:          true,
		UseRFC9421:                 false,
	}
)

var (
	// HttpsigAlgs is a constant slice of httpsig algorithm objects
	HttpsigAlgs []httpsig.Algorithm
	// Httpsig9421Algs is a constant slice of httpsig algorithm objects
	HttpsigAlgsRFC9421 []Algorithm
)

func loadFederationFrom(rootCfg ConfigProvider) {
	if err := rootCfg.Section("federation").MapTo(&Federation); err != nil {
		log.Fatal("Failed to map Federation settings: %v", err)
	} else if !httpsig.IsSupportedDigestAlgorithm(Federation.DigestAlgorithm) {
		log.Fatal("unsupported digest algorithm: %s", Federation.DigestAlgorithm)
		return
	}

	// Get MaxSize in bytes instead of MiB
	Federation.MaxSize = 1 << 20 * Federation.MaxSize

	HttpsigAlgs = make([]httpsig.Algorithm, len(Federation.SignatureAlgorithms))
	for i, alg := range Federation.SignatureAlgorithms {
		HttpsigAlgs[i] = httpsig.Algorithm(alg)
	}

	HttpsigAlgsRFC9421 = make([]Algorithm, len(Federation.SignatureAlgorithmsRFC9421))
	for i, alg := range Federation.SignatureAlgorithmsRFC9421 {
		HttpsigAlgsRFC9421[i] = Algorithm(alg)
	}
}
