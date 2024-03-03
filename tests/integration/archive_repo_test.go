// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"archive/zip"
	"bytes"
	"io"
	"net/http"
	"net/url"
	"testing"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/test"
	"code.gitea.io/gitea/services/repository/archiver"
	"code.gitea.io/gitea/tests"
	"github.com/stretchr/testify/assert"
)

func TestLFSArchive(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		session := loginUser(t, "user2")

		cleanup := func(t *testing.T) {
			t.Helper()

			assert.NoError(t, archiver.DeleteRepositoryArchives(db.DefaultContext))
		}

		t.Run("LFS disabled", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			defer test.MockVariableValue(&setting.LFS.StartServer, false)()
			defer cleanup(t)

			req := NewRequest(t, "GET", "/user2/lfs/archive/master.zip")
			resp := session.MakeRequest(t, req, http.StatusOK)

			r, err := zip.NewReader(bytes.NewReader(resp.Body.Bytes()), int64(resp.Body.Len()))
			assert.NoError(t, err)
			assert.Len(t, r.File, 9)

			// Test README.md
			assert.EqualValues(t, "lfs/subdir/README.md", r.File[8].Name)
			file8, err := r.File[8].Open()
			assert.NoError(t, err)
			defer file8.Close()
			fBytes, err := io.ReadAll(file8)
			assert.NoError(t, err)
			assert.EqualValues(t, "version https://git-lfs.github.com/spec/v1\noid sha256:9d172e5c64b4f0024b9901ec6afe9ea052f3c9b6ff9f4b07956d8c48c86fca82\nsize 25\n", string(fBytes))

			// Test jpeg.jpg
			assert.EqualValues(t, "lfs/jpeg.jpg", r.File[5].Name)
			file5, err := r.File[5].Open()
			assert.NoError(t, err)
			defer file5.Close()
			fBytes, err = io.ReadAll(file5)
			assert.NoError(t, err)
			assert.EqualValues(t, "version https://git-lfs.github.com/spec/v1\noid sha256:0b8d8b5f15046343fd32f451df93acc2bdd9e6373be478b968e4cad6b6647351\nsize 107\n", string(fBytes))

			// Test CONTRIBUTING.md
			assert.EqualValues(t, "lfs/CONTRIBUTING.md", r.File[2].Name)
			file2, err := r.File[2].Open()
			assert.NoError(t, err)
			defer file2.Close()
			fBytes, err = io.ReadAll(file2)
			assert.NoError(t, err)
			assert.EqualValues(t, "version https://git-lfs.github.com/spec/v1\noid sha256:7b6b2c88dba9f760a1a58469b67fee2b698ef7e9399c4ca4f34a14ccbe39f623\nsize 27\n", string(fBytes))
		})

		t.Run("Private repository", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			defer test.MockVariableValue(&setting.LFS.StartServer, true)()
			defer cleanup(t)

			req := NewRequest(t, "GET", "/user2/lfs/archive/master.zip")
			resp := session.MakeRequest(t, req, http.StatusOK)

			r, err := zip.NewReader(bytes.NewReader(resp.Body.Bytes()), int64(resp.Body.Len()))
			assert.NoError(t, err)
			assert.Len(t, r.File, 9)

			// Test README.md
			assert.EqualValues(t, "lfs/subdir/README.md", r.File[8].Name)
			file8, err := r.File[8].Open()
			assert.NoError(t, err)
			defer file8.Close()
			fBytes, err := io.ReadAll(file8)
			assert.NoError(t, err)
			assert.EqualValues(t, "# Testing READMEs in LFS\n", string(fBytes))

			// Test jpeg.jpg
			assert.EqualValues(t, "lfs/jpeg.jpg", r.File[5].Name)
			file5, err := r.File[5].Open()
			assert.NoError(t, err)
			defer file5.Close()
			fBytes, err = io.ReadAll(file5)
			assert.NoError(t, err)
			assert.EqualValues(t, "\xff\xd8\xff\xdb\x00C\x00\x03\x02\x02\x02\x02\x02\x03\x02\x02\x02\x03\x03\x03\x03\x04\x06\x04\x04\x04\x04\x04\b\x06\x06\x05\x06\t\b\n\n\t\b\t\t\n\f\x0f\f\n\v\x0e\v\t\t\r\x11\r\x0e\x0f\x10\x10\x11\x10\n\f\x12\x13\x12\x10\x13\x0f\x10\x10\x10\xff\xc9\x00\v\b\x00\x01\x00\x01\x01\x01\x11\x00\xff\xcc\x00\x06\x00\x10\x10\x05\xff\xda\x00\b\x01\x01\x00\x00?\x00\xd2\xcf \xff\xd9", string(fBytes))

			// Test CONTRIBUTING.md
			assert.EqualValues(t, "lfs/CONTRIBUTING.md", r.File[2].Name)
			file2, err := r.File[2].Open()
			assert.NoError(t, err)
			defer file2.Close()
			fBytes, err = io.ReadAll(file2)
			assert.NoError(t, err)
			assert.EqualValues(t, "# Testing documents in LFS\n", string(fBytes))
		})
	})
}
