// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	"fmt"
	"time"

	"code.gitea.io/gitea/modules/timeutil"
	"code.gitea.io/gitea/modules/validation"
)

// FederationHost data type
// swagger:model
type FederationHost struct {
	ID int64 `xorm:"pk autoincr"`
	// TODO: implement a toLower here & add a toLowerValidation
	HostFqdn       string             `xorm:"host_fqdn UNIQUE INDEX VARCHAR(255) NOT NULL"`
	NodeInfo       NodeInfo           `xorm:"extends NOT NULL"`
	LatestActivity time.Time          `xorm:"NOT NULL"`
	Create         timeutil.TimeStamp `xorm:"created"`
	Updated        timeutil.TimeStamp `xorm:"updated"`
}

// Factory function for PersonID. Created struct is asserted to be valid
func NewFederationHost(nodeInfo NodeInfo, hostFqdn string) (FederationHost, error) {
	result := FederationHost{
		HostFqdn: hostFqdn,
		NodeInfo: nodeInfo,
	}
	if valid, err := validation.IsValid(result); !valid {
		return FederationHost{}, err
	}
	return result, nil
}

// Validate collects error strings in a slice and returns this
func (host FederationHost) Validate() []string {
	var result []string
	result = append(result, validation.ValidateNotEmpty(host.HostFqdn, "HostFqdn")...)
	result = append(result, validation.ValidateMaxLen(host.HostFqdn, 255, "HostFqdn")...)
	result = append(result, host.NodeInfo.Validate()...)
	if !host.LatestActivity.IsZero() && host.LatestActivity.After(time.Now().Add(10*time.Minute)) {
		result = append(result, fmt.Sprintf("Latest Activity may not be far futurer: %v", host.LatestActivity))
	}

	return result
}
