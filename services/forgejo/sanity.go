// SPDX-License-Identifier: MIT

package forgejo

import (
	"fmt"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/forgejo/semver"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"

	"github.com/hashicorp/go-version"
)

var (
	ForgejoV5DatabaseVersion = int64(260)
	ForgejoV4DatabaseVersion = int64(244)
)

var logFatal = log.Fatal

func fatal(err error) error {
	logFatal("%v", err)
	return err
}

func PreMigrationSanityChecks(e db.Engine, dbVersion int64, cfg setting.ConfigProvider) error {
	return v1TOv5_0_1Included(e, cfg)
}

func v1TOv5_0_1Included(e db.Engine, cfg setting.ConfigProvider) error {
	previousServerVersion, err := semver.GetVersionWithEngine(e)
	if err != nil {
		return err
	}
	//
	// The sanity check needs to be done for all versions up to v5.0.1
	// included.
	//
	upper, err := version.NewVersion("v5.0.1")
	if err != nil {
		return err
	}
	if previousServerVersion.GreaterThan(upper) {
		return nil
	}
	originalCfg, err := cfg.PrepareSaving()
	if err != nil {
		return err
	}
	if originalCfg.Section("storage").HasKey("PATH") {
		return fatal(fmt.Errorf("[storage].PATH is set and needs to be manually fixed. Please read https://forgejo.org/2023-08-release-v1-20-3-0/ for instructions"))
	}
	return nil
}
