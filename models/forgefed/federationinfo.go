// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	"time"

	"code.gitea.io/gitea/modules/timeutil"
	"code.gitea.io/gitea/modules/validation"
)

// FederationInfo data type
// swagger:model
type FederationInfo struct {
	ID             int64              `xorm:"pk autoincr"`
	HostFqdn       string             `xorm:"host_fqdn UNIQUE INDEX VARCHAR(255) NOT NULL"`
	NodeInfo       NodeInfo           `xorm:"extends NOT NULL"`
	LatestActivity time.Time          `xorm:"NOT NULL"`
	Create         timeutil.TimeStamp `xorm:"created"`
	Updated        timeutil.TimeStamp `xorm:"updated"`
}

// Validate collects error strings in a slice and returns this
func (info FederationInfo) Validate() []string {
	var result []string
	result = append(result, validation.ValidateNotEmpty(string(info.HostFqdn), "HostFqdn")...)
	result = append(result, validation.ValidateMaxLen(string(info.HostFqdn), 255, "HostFqdn")...)
	result = append(result, info.NodeInfo.Validate()...)

	return result
}
