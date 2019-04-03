// Copyright 2019 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package models

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"time"

	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	"github.com/Unknwon/com"
)

// LocalCopyPath returns the local repository temporary copy path.
func LocalCopyPath() string {
	if filepath.IsAbs(setting.Repository.Local.LocalCopyPath) {
		return setting.Repository.Local.LocalCopyPath
	}
	return path.Join(setting.AppDataPath, setting.Repository.Local.LocalCopyPath)
}

// CreateTemporaryPath creates a temporary path
func CreateTemporaryPath(prefix string) (string, error) {
	timeStr := com.ToStr(time.Now().Nanosecond()) // SHOULD USE SOMETHING UNIQUE
	basePath := filepath.Join(LocalCopyPath(), prefix+"-"+timeStr+".git")
	if err := os.MkdirAll(filepath.Dir(basePath), os.ModePerm); err != nil {
		log.Error("Unable to create temporary directory: %s (%v)", basePath, err)
		return "", fmt.Errorf("Failed to create dir %s: %v", basePath, err)
	}
	return basePath, nil
}

// WithTemporaryPath takes a prefix and callback function to run with a temporary path
func WithTemporaryPath(prefix string, callback func(string) error) error {
	basePath, err := CreateTemporaryPath(prefix)
	if err != nil {
		return err
	}
	defer func() {
		if _, err := os.Stat(basePath); !os.IsNotExist(err) {
			os.RemoveAll(basePath)
		}
	}()
	return callback(basePath)
}
