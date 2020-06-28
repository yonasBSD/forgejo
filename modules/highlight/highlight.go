// Copyright 2015 The Gogs Authors. All rights reserved.
// Copyright 2020 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package highlight

import (
	"bufio"
	"bytes"
	"path/filepath"
	"strings"
	"sync"

	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	"github.com/alecthomas/chroma/formatters/html"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
)

var (
	// For custom user mapping
	highlightMapping = map[string]string{}

	once sync.Once
)

// NewContext loads custom highlight map from local config
func NewContext() {
	once.Do(func() {
		keys := setting.Cfg.Section("highlight.mapping").Keys()
		for i := range keys {
			highlightMapping[keys[i].Name()] = keys[i].Value()
		}
	})
}

// Code returns a HTML version of code string with chroma syntax highlighting classes
func Code(fileName, code string) string {
	NewContext()
	// don't highlight over 25kb
	if len(code) > 25000 {
		return plainText(string(code), numLines)
	}
	formatter := html.New(html.WithClasses(true),
		html.WithLineNumbers(false),
		html.PreventSurroundingPre(true),
	)
	if formatter == nil {
		log.Error("Couldn't create chroma formatter")
		return code
	}

	htmlbuf := bytes.Buffer{}
	htmlw := bufio.NewWriter(&htmlbuf)

	if val, ok := highlightMapping[filepath.Ext(fileName)]; ok {
		//change file name to one with mapped extension so we look that up instead
		fileName = "mapped." + val
	}

	lexer := lexers.Match(fileName)
	if lexer == nil {
		lexer = lexers.Fallback
	}

	iterator, err := lexer.Tokenise(nil, string(code))
	if err != nil {
		log.Error("Can't tokenize code: %v", err)
		return code
	}
	// style not used for live site but need to pass something
	err = formatter.Format(htmlw, styles.GitHub, iterator)
	if err != nil {
		log.Error("Can't format code: %v", err)
		return code
	}

	htmlw.Flush()
	return htmlbuf.String()
}

// File returns map with line lumbers and HTML version of code with chroma syntax highlighting classes
func File(numLines int, fileName string, code []byte) map[int]string {
	NewContext()
	// don't highlight over 25kb
	if len(code) > 25000 {
		return plainText(string(code), numLines)
	}
	formatter := html.New(html.WithClasses(true),
		html.WithLineNumbers(false),
		html.PreventSurroundingPre(true),
	)

	if formatter == nil {
		log.Error("Couldn't create chroma formatter")
		return plainText(string(code), numLines)
	}

	htmlbuf := bytes.Buffer{}
	htmlw := bufio.NewWriter(&htmlbuf)

	if val, ok := highlightMapping[filepath.Ext(fileName)]; ok {
		fileName = "test." + val
	}

	lexer := lexers.Match(fileName)
	if lexer == nil {
		lexer = lexers.Analyse(string(code))
		if lexer == nil {
			lexer = lexers.Fallback
		}
	}

	iterator, err := lexer.Tokenise(nil, string(code))
	if err != nil {
		log.Error("Can't tokenize code: %v", err)
		return plainText(string(code), numLines)
	}

	err = formatter.Format(htmlw, styles.GitHub, iterator)
	if err != nil {
		log.Error("Can't format code: %v", err)
		return plainText(string(code), numLines)
	}

	htmlw.Flush()
	m := make(map[int]string, numLines)
	for k, v := range strings.SplitN(htmlbuf.String(), "\n", numLines) {
		line := k + 1
		m[line] = string(v)
	}
	return m
}

// return unhiglighted map
func plainText(code string, numLines int) map[int]string {
	m := make(map[int]string, numLines)
	for k, v := range strings.SplitN(string(code), "\n", numLines) {
		line := k + 1
		m[line] = string(v)
	}
	return m
}
