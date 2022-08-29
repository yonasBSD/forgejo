// Copyright 2022 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package options

import (
	"errors"
	"os"
	"path"
	"strings"

	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/util"
)

// GetRepoInitFile returns repository init files
func GetRepoInitFile(tp, name string) ([]byte, error) {
	cleanedName := strings.TrimLeft(path.Clean("/"+name), "/")
	relPath := path.Join("options", tp, cleanedName)

	// Use custom file when available.
	customPath := path.Join(setting.CustomPath, relPath)
	isFile, err := util.IsFile(customPath)
	if err != nil {
		log.Error("Unable to check if %s is a file. Error: %v", customPath, err)
	}
	if isFile {
		return os.ReadFile(customPath)
	}

	switch tp {
	case "readme":
		return Readme(cleanedName)
	case "gitignore":
		return Gitignore(cleanedName)
	case "license":
		return License(cleanedName)
	case "label":
		return Labels(cleanedName)
	default:
		return []byte{}, errors.New("invalid init file type")
	}
}
