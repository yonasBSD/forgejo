// Copyright 2020 The Gitea Authors. All rights reserved.
// Copyright 2015 Kenneth Shaw
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// +build ignore

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/format"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	gemojiURL         = "https://raw.githubusercontent.com/github/gemoji/master/db/emoji.json"
	maxUnicodeVersion = 12
)

var (
	flagOut = flag.String("o", "modules/emoji/emoji_data.go", "out")
)

// Gemoji is a set of emoji data.
type Gemoji []Emoji

// Emoji represents a single emoji and associated data.
type Emoji struct {
	Emoji          string   `json:"emoji"`
	Description    string   `json:"description,omitempty"`
	Aliases        []string `json:"aliases"`
	UnicodeVersion string   `json:"unicode_version,omitempty"`
	SkinTones      bool     `json:"skin_tones,omitempty"`
}

// Don't include some fields in JSON
func (e Emoji) MarshalJSON() ([]byte, error) {
	type emoji Emoji
	x := emoji(e)
	x.UnicodeVersion = ""
	x.Description = ""
	x.SkinTones = false
	return json.Marshal(x)
}

func main() {
	var err error

	flag.Parse()

	// generate data
	buf, err := generate()
	if err != nil {
		log.Fatal(err)
	}

	// write
	err = ioutil.WriteFile(*flagOut, buf, 0644)
	if err != nil {
		log.Fatal(err)
	}
}

var replacer = strings.NewReplacer(
	"main.Gemoji", "Gemoji",
	"main.Emoji", "\n",
	"}}", "},\n}",
	", Description:", ", ",
	", Aliases:", ", ",
	", UnicodeVersion:", ", ",
	", SkinTones:", ", ",
)

var emojiRE = regexp.MustCompile(`\{Emoji:"([^"]*)"`)

func generate() ([]byte, error) {
	var err error

	// load gemoji data
	res, err := http.Get(gemojiURL)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	// read all
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	// unmarshal
	var data Gemoji
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	var skinTones = make(map[string]string)

	skinTones["\U0001f3fb"] = "Light Skin Tone"
	skinTones["\U0001f3fc"] = "Medium-Light Skin Tone"
	skinTones["\U0001f3fd"] = "Medium Skin Tone"
	skinTones["\U0001f3fe"] = "Medium-Dark Skin Tone"
	skinTones["\U0001f3ff"] = "Dark Skin Tone"

	var tmp Gemoji

	//filter out emoji that require greater than max unicode version
	for i := range data {
		val, _ := strconv.ParseFloat(data[i].UnicodeVersion, 64)
		if int(val) <= maxUnicodeVersion {
			tmp = append(tmp, data[i])
		}
	}
	data = tmp

	sort.Slice(data, func(i, j int) bool {
		return data[i].Aliases[0] < data[j].Aliases[0]
	})

	aliasMap := make(map[string]int, len(data))

	for i, e := range data {
		if e.Emoji == "" || len(e.Aliases) == 0 {
			continue
		}
		for _, a := range e.Aliases {
			if a == "" {
				continue
			}
			aliasMap[a] = i
		}
	}

	// gitea customizations
	i, ok := aliasMap["tada"]
	if ok {
		data[i].Aliases = append(data[i].Aliases, "hooray")
	}
	i, ok = aliasMap["laughing"]
	if ok {
		data[i].Aliases = append(data[i].Aliases, "laugh")
	}

	// write a JSON file to use with tribute (write before adding skin tones since we can't support them there yet)
	file, _ := json.Marshal(data)
	_ = ioutil.WriteFile("assets/emoji.json", file, 0644)

	// Add skin tones to emoji that support it
	var (
		s              []string
		newEmoji       string
		newDescription string
		newData        Emoji
	)

	for i := range data {
		if data[i].SkinTones {
			for k, v := range skinTones {
				s = strings.Split(data[i].Emoji, "")

				if utf8.RuneCountInString(data[i].Emoji) == 1 {
					s = append(s, k)
				} else {
					// insert into slice after first element because all emoji that support skin tones
					// have that modifier placed at this spot
					s = append(s, "")
					copy(s[2:], s[1:])
					s[1] = k
				}

				newEmoji = strings.Join(s, "")
				newDescription = data[i].Description + ": " + v
				newAlias := data[i].Aliases[0] + "_" + strings.ReplaceAll(v, " ", "_")

				newData = Emoji{newEmoji, newDescription, []string{newAlias}, "12.0", false}
				data = append(data, newData)
			}
		}
	}

	// add header
	str := replacer.Replace(fmt.Sprintf(hdr, gemojiURL, data))

	// change the format of the unicode string
	str = emojiRE.ReplaceAllStringFunc(str, func(s string) string {
		var err error
		s, err = strconv.Unquote(s[len("{Emoji:"):])
		if err != nil {
			panic(err)
		}
		return "{" + strconv.QuoteToASCII(s)
	})

	// format
	return format.Source([]byte(str))
}

const hdr = `
// Copyright 2020 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package emoji

// Code generated by gen.go. DO NOT EDIT.
// Sourced from %s
//
var GemojiData = %#v
`
