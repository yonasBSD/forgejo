// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"testing"

	"code.gitea.io/gitea/modules/validation"
)

func Test_FederatedRepoValidation(t *testing.T) {
	sut := FederatedRepo{
		RepoID:           12,
		ExternalID:       "12",
		FederationHostID: 1,
	}
	if res, err := validation.IsValid(sut); !res {
		t.Errorf("sut should be valid but was %q", err)
	}

	sut = FederatedRepo{
		ExternalID:       "12",
		FederationHostID: 1,
	}
	if res, _ := validation.IsValid(sut); res {
		t.Errorf("sut should be invalid")
	}
}
