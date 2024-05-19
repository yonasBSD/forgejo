// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"bytes"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"mime/multipart"
	"net/http"
	"testing"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/packages"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/packages/rubygems"
	"code.gitea.io/gitea/tests"

	"github.com/stretchr/testify/assert"
)

func TestPackageRubyGems(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	packageName := "gitea"
	packageVersion := "1.0.5"
	packageFilename := "gitea-1.0.5.gem"
	packageDependency := "runtime-dep:>= 1.2.0&< 2.0"
	rubyRequirements := "ruby:>= 2.3.0"
	sep := "---"

	gemContent, _ := base64.StdEncoding.DecodeString(`bWV0YWRhdGEuZ3oAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAADAwMDA0NDQAMDAwMDAw
MAAwMDAwMDAwADAwMDAwMDAxMDQxADE0MTEwNzcyMzY2ADAxMzQ0MQAgMAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAB1c3RhcgAwMHdoZWVsAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAd2hlZWwAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAwMDAwMDAwADAwMDAw
MDAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAf
iwgA9vQjYQID1VVNb9QwEL37V5he9pRsmlJAFlQckCoOXAriQIUix5nNmsYf2JOqKwS/nYmz2d3Q
qqCCKpFdadfjmfdm5nmcLMv4k9DXm6Wrv4BCcQ5GiPcelF5pJVE7y6w0IHirESS7hhDJJu4I+jhu
Mc53Tsd5kZ8y30lcuWAEH2KY7HHtQhQs4+cJkwwuwNdeB6JhtbaNDoLTL1MQsFJrqQnr8jNrJJJH
WZTHWfEiK094UYj0zYvp4Z9YAx5sA1ZpSCS3M30zeWwo2bG60FvUBjIKJts2GwMW76r0Yr9NzjN3
YhwsGX2Ozl4dpcWwvK9d43PQtDIv9igvHwSyIIwFmXHjqTqxLY8MPkCADmQk80p2EfZ6VbM6/ue6
/1D0Bq7/qeA/zh6W82leHmhFWUHn/JbsEfT6q7QbiCpoj8l0QcEUFLmX6kq2wBEiMjBSd+Pwt7T5
Ot0kuXYMbkD1KOuOBnWYb7hBsAP4bhlkFRqnqpWefMZ/pHCn6+WIFGq2dgY8EQq+RvRRLJcTyZJ1
WhHqGPTu7QdmACXdJFLwb9+ZdxErbSPKrqsMxJhAWCJ1qaqRdtu6yktcT/STsamG0qp7rsa5EL/K
MBua30uw4ynzExqYWRJDfx8/kQWN3PwsDh2jYLr1W+pZcAmCs9splvnz/Flesqhbq21bXcGG/OLh
+2fv/JTF3hgZyCW9OaZjxoZjdnBGfgKpxZyJ1QYAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAZGF0
YS50YXIuZ3oAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAADAwMDA0NDQAMDAwMDAwMAAw
MDAwMDAwADAwMDAwMDAwMjQyADE0MTEwNzcyMzY2ADAxMzM2MQAgMAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAB1c3RhcgAwMHdoZWVsAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAd2hlZWwAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAwMDAwMDAwADAwMDAwMDAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAfiwgA
9vQjYQID7M/NCsMgDABgz32KrA/QxersK/Q17ExXIcyhlr7+HLv1sJ02KPhBCPk5JOyn881nsl2c
xI+gRDRaC3zbZ8RBCamlxGHolTFlX11kLwDFH6wp21hO2RYi/rD3bb5/7iCubFOCMbBtABzNkIjn
bvGlAnisOUE7EnOALUR2p7b06e6aV4iqqqrquJ4AAAD//wMA+sA/NQAIAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAGNoZWNr
c3Vtcy55YW1sLmd6AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAwMDAwNDQ0ADAwMDAwMDAAMDAw
MDAwMAAwMDAwMDAwMDQ1MAAxNDExMDc3MjM2NgAwMTQ2MTIAIDAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAdXN0YXIAMDB3aGVlbAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAHdoZWVsAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAMDAwMDAwMAAwMDAwMDAwAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAH4sIAPb0
I2ECA2WQOa4UQAxE8znFXGCQ21vbPyMj5wRuL0Qk6EecnmZCyKyy9FSvXq/X4/u3ryj68Xg+f/Zn
VHzGlx+/P57qvU4XxWalBKftSXOgCjNYkdRycrC5Axem+W4HqS12PNEv7836jF9vnlHxwSyxKY+y
go0cPblyHzkrZ4HF1GSVhe7mOOoasXNk2fnbUxb+19Pp9tobD/QlJKMX7y204PREh6nQ5hG9Alw6
x4TnmtA+aekGfm6wAseog2LSgpR4Q7cYnAH3K4qAQa6A6JCC1gpuY7P+9YxE5SZ+j0eVGbaBTwBQ
iIqRUyyzLCoFCBdYNWxniapTavD97blXTzFvgoVoAsKBAtlU48cdaOmeZDpwV01OtcGwjscfeUrY
B9QBAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA`)
	checksum := fmt.Sprintf("%x", sha256.Sum256(gemContent))

	root := fmt.Sprintf("/api/packages/%s/rubygems", user.Name)

	uploadFile := func(t *testing.T, expectedStatus int) {
		req := NewRequestWithBody(t, "POST", fmt.Sprintf("%s/api/v1/gems", root), bytes.NewReader(gemContent)).
			AddBasicAuth(user.Name)
		MakeRequest(t, req, expectedStatus)
	}

	t.Run("Upload", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		uploadFile(t, http.StatusCreated)

		pvs, err := packages.GetVersionsByPackageType(db.DefaultContext, user.ID, packages.TypeRubyGems)
		assert.NoError(t, err)
		assert.Len(t, pvs, 1)

		pd, err := packages.GetPackageDescriptor(db.DefaultContext, pvs[0])
		assert.NoError(t, err)
		assert.NotNil(t, pd.SemVer)
		assert.IsType(t, &rubygems.Metadata{}, pd.Metadata)
		assert.Equal(t, packageName, pd.Package.Name)
		assert.Equal(t, packageVersion, pd.Version.Version)

		pfs, err := packages.GetFilesByVersionID(db.DefaultContext, pvs[0].ID)
		assert.NoError(t, err)
		assert.Len(t, pfs, 1)
		assert.Equal(t, packageFilename, pfs[0].Name)
		assert.True(t, pfs[0].IsLead)

		pb, err := packages.GetBlobByID(db.DefaultContext, pfs[0].BlobID)
		assert.NoError(t, err)
		assert.Equal(t, int64(4608), pb.Size)
	})

	t.Run("UploadExists", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		uploadFile(t, http.StatusConflict)
	})

	t.Run("Download", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "GET", fmt.Sprintf("%s/gems/%s", root, packageFilename)).
			AddBasicAuth(user.Name)
		resp := MakeRequest(t, req, http.StatusOK)

		assert.Equal(t, gemContent, resp.Body.Bytes())

		pvs, err := packages.GetVersionsByPackageType(db.DefaultContext, user.ID, packages.TypeRubyGems)
		assert.NoError(t, err)
		assert.Len(t, pvs, 1)
		assert.Equal(t, int64(1), pvs[0].DownloadCount)
	})

	t.Run("DownloadGemspec", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "GET", fmt.Sprintf("%s/quick/Marshal.4.8/%sspec.rz", root, packageFilename)).
			AddBasicAuth(user.Name)
		resp := MakeRequest(t, req, http.StatusOK)

		b, _ := base64.StdEncoding.DecodeString(`eJxi4Si1EndPzbWyCi5ITc5My0xOLMnMz2M8zMIRLeGpxGWsZ6RnzGbF5hqSyempxJWeWZKayGbN
EBJqJQjWFZZaVJyZnxfN5qnEZahnoGcKkjTwVBJyB6lUKEhMzk5MTwULGngqcRaVJlWCONEMBp5K
DGAWSKc7zFhPJamg0qRK99TcYphehZLU4hKInFhGSUlBsZW+PtgZepn5+iDxECRzDUDGcfh6hoA4
gAAAAP//MS06Gw==`)
		assert.Equal(t, b, resp.Body.Bytes())

		pvs, err := packages.GetVersionsByPackageType(db.DefaultContext, user.ID, packages.TypeRubyGems)
		assert.NoError(t, err)
		assert.Len(t, pvs, 1)
		assert.Equal(t, int64(1), pvs[0].DownloadCount)
	})

	t.Run("EnumeratePackages", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		enumeratePackages := func(t *testing.T, endpoint string, expectedContent []byte) {
			req := NewRequest(t, "GET", fmt.Sprintf("%s/%s", root, endpoint)).
				AddBasicAuth(user.Name)
			resp := MakeRequest(t, req, http.StatusOK)

			assert.Equal(t, expectedContent, resp.Body.Bytes())
		}

		b, _ := base64.StdEncoding.DecodeString(`H4sICAAAAAAA/3NwZWNzLjQuOABi4Yhmi+bwVOJKzyxJTWSzYnMNCbUSdE/NtbIKSy0qzszPi2bzVOIy1DPQM2WzZgjxVOIsKk2qBDEBAQAA///xOEYKOwAAAA==`)
		enumeratePackages(t, "specs.4.8.gz", b)
		b, _ = base64.StdEncoding.DecodeString(`H4sICAAAAAAA/2xhdGVzdF9zcGVjcy40LjgAYuGIZovm8FTiSs8sSU1ks2JzDQm1EnRPzbWyCkstKs7Mz4tm81TiMtQz0DNls2YI8VTiLCpNqgQxAQEAAP//8ThGCjsAAAA=`)
		enumeratePackages(t, "latest_specs.4.8.gz", b)
		b, _ = base64.StdEncoding.DecodeString(`H4sICAAAAAAA/3ByZXJlbGVhc2Vfc3BlY3MuNC44AGLhiGYABAAA//9snXr5BAAAAA==`)
		enumeratePackages(t, "prerelease_specs.4.8.gz", b)
	})

	t.Run("PackageInfo", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "GET", fmt.Sprintf("%s/info/%s", root, packageName)).
			AddBasicAuth(user.Name)
		resp := MakeRequest(t, req, http.StatusOK)
		expected := fmt.Sprintf("%s\n%s %s|checksum:%s,%s\n",
			sep, packageVersion, packageDependency, checksum, rubyRequirements)
		assert.Equal(t, expected, resp.Body.String())
	})
	t.Run("Versions", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		versionsReq := NewRequest(t, "GET", fmt.Sprintf("%s/versions", root)).
			AddBasicAuth(user.Name)
		versionsResp := MakeRequest(t, versionsReq, http.StatusOK)
		infoReq := NewRequest(t, "GET", fmt.Sprintf("%s/info/%s", root, packageName)).
			AddBasicAuth(user.Name)
		infoResp := MakeRequest(t, infoReq, http.StatusOK)

		expected := fmt.Sprintf("%s\n%s %s %x\n",
			sep, packageName, packageVersion, md5.Sum(infoResp.Body.Bytes()))
		assert.Equal(t, expected, versionsResp.Body.String())
	})
	t.Run("Delete", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		body := bytes.Buffer{}
		writer := multipart.NewWriter(&body)
		writer.WriteField("gem_name", packageName)
		writer.WriteField("version", packageVersion)
		writer.Close()

		req := NewRequestWithBody(t, "DELETE", fmt.Sprintf("%s/api/v1/gems/yank", root), &body).
			SetHeader("Content-Type", writer.FormDataContentType()).
			AddBasicAuth(user.Name)
		MakeRequest(t, req, http.StatusOK)

		pvs, err := packages.GetVersionsByPackageType(db.DefaultContext, user.ID, packages.TypeRubyGems)
		assert.NoError(t, err)
		assert.Empty(t, pvs)
	})

	t.Run("NonExistingGem", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "GET", fmt.Sprintf("%s/info/%s", root, packageName)).
			AddBasicAuth(user.Name)
		_ = MakeRequest(t, req, http.StatusNotFound)
	})
	t.Run("EmptyVersions", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "GET", fmt.Sprintf("%s/versions", root)).
			AddBasicAuth(user.Name)
		resp := MakeRequest(t, req, http.StatusOK)
		assert.Equal(t, sep+"\n", resp.Body.String())
	})
}
