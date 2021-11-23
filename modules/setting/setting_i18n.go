// Copyright 2021 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package setting

var defaultI18nLangMap = map[string]string{
	"en-US": "English",
	"zh-CN": "简体中文",
	"zh-HK": "繁體中文（香港）",
	"zh-TW": "繁體中文（台灣）",
	"de-DE": "Deutsch",
	"fr-FR": "français",
	"nl-NL": "Nederlands",
	"lv-LV": "latviešu",
	"ru-RU": "русский",
	"uk-UA": "Українська",
	"ja-JP": "日本語",
	"es-ES": "español",
	"pt-BR": "português do Brasil",
	"pt-PT": "Português de Portugal",
	"pl-PL": "polski",
	"bg-BG": "български",
	"it-IT": "italiano",
	"fi-FI": "suomi",
	"tr-TR": "Türkçe",
	"cs-CZ": "čeština",
	"sr-SP": "српски",
	"sv-SE": "svenska",
	"ko-KR": "한국어",
	"el-GR": "ελληνικά",
	"fa-IR": "فارسی",
	"hu-HU": "magyar nyelv",
	"id-ID": "bahasa Indonesia",
	"ml-IN": "മലയാളം",
}

func defaultI18nLangs() (res []string) {
	for k := range defaultI18nLangMap {
		res = append(res, k)
	}
	return
}

func defaultI18nNames() (res []string) {
	for _, v := range defaultI18nLangMap {
		res = append(res, v)
	}
	return
}
