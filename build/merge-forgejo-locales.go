// Copyright 2022 The Forgejo Authors c/o Codeberg e.V.. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:build ignore

package main

import (
	"io/ioutil"
	"strings"

	"gopkg.in/ini.v1"
)

const (
	trimPrefix   = "gitea_"
	sourceFolder = "options/locales/"
)

// returns list of locales, still containing the file extension!
func generate_locale_list() []string {
	localeFiles, _ := ioutil.ReadDir(sourceFolder)
	locales := []string{}
	for _, localeFile := range localeFiles {
		if !localeFile.IsDir() && strings.HasPrefix(localeFile.Name(), trimPrefix) {
			locales = append(locales, strings.TrimPrefix(localeFile.Name(), trimPrefix))
		}
	}
	return locales
}

func main() {
	locales := generate_locale_list()
	var err error
	var localeFile *ini.File
	for _, locale := range locales {
		localeFile, err = ini.LooseLoad(sourceFolder+"gitea_"+locale, sourceFolder+"forgejo_"+locale)
		if err != nil {
			panic(err)
		}
		err = localeFile.SaveTo("options/locale/locale_" + locale)
		if err != nil {
			panic(err)
		}
	}
}
