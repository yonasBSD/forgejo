// Copyright 2024 The forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	"testing"
	"time"

	"code.gitea.io/gitea/modules/validation"
)

func Test_FederationInfoValidation(t *testing.T) {
	sut := FederationInfo{
		HostFqdn: "host.do.main",
		NodeInfo: NodeInfo{
			Source: "forgejo",
		},
		LatestActivity: time.Now(),
	}
	if res, err := validation.IsValid(sut); !res {
		t.Errorf("sut should be valid but was %q", err)
	}

	sut = FederationInfo{
		HostFqdn:       "host.do.main",
		NodeInfo:       NodeInfo{},
		LatestActivity: time.Now(),
	}
	if res, _ := validation.IsValid(sut); res {
		t.Errorf("sut should be invalid")
	}
}
