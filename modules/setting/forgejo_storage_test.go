// SPDX-License-Identifier: MIT

//
// Tests verifying the Forgejo documentation on storage settings is correct
//
// https://forgejo.org/docs/v1.20/admin/storage/
//

package setting

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func testForgejoStoragePath(t *testing.T, appDataPath, iniStr string, loader func(rootCfg ConfigProvider) error, storagePtr **Storage, expectedPath string) {
	cfg, err := NewConfigProviderFromData(iniStr)
	assert.NoError(t, err)
	AppDataPath = appDataPath
	assert.NoError(t, loader(cfg))
	storage := *storagePtr

	assert.EqualValues(t, "local", storage.Type)
	assert.True(t, filepath.IsAbs(storage.Path))
	assert.EqualValues(t, filepath.Clean(expectedPath), filepath.Clean(storage.Path))
}

func TestForgejoDocs_StorageOverride(t *testing.T) {
	for _, c := range []struct {
		name     string
		basePath string
		section  string
		loader   func(rootCfg ConfigProvider) error
		storage  **Storage
	}{
		{"Attachements", "attachments", "attachment", loadAttachmentFrom, &Attachment.Storage},
		{"LFS", "lfs", "lfs", loadLFSFrom, &LFS.Storage},
		{"Avatars", "avatars", "avatar", loadAvatarsFrom, &Avatar.Storage},
		{"Repository avatars", "repo-avatars", "repo-avatar", loadRepoAvatarFrom, &RepoAvatar.Storage},
		{"Repository archives", "repo-archive", "repo-archive", loadRepoArchiveFrom, &RepoArchive.Storage},
		{"Packages", "packages", "packages", loadPackagesFrom, &Packages.Storage},
		{"Actions logs", "actions_log", "storage.actions_log", loadActionsFrom, &Actions.LogStorage},
		{"Actions Artifacts", "actions_artifacts", "actions.artifacts", loadActionsFrom, &Actions.ArtifactStorage},
	} {
		t.Run(c.name, func(t *testing.T) {
			testForgejoStoragePath(t, "/appdata", "", c.loader, c.storage, fmt.Sprintf("/appdata/%s", c.basePath))

			testForgejoStoragePath(t, "/appdata", `
[storage]
STORAGE_TYPE = local
PATH = /storagepath
`, c.loader, c.storage, fmt.Sprintf("/storagepath/%s", c.basePath))

			section := fmt.Sprintf(`
[%s]
STORAGE_TYPE = local
PATH = /sectionpath
`, c.section)
			testForgejoStoragePath(t, "/appdata", section, c.loader, c.storage, "/sectionpath")
		})
	}
}
