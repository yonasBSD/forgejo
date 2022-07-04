// Copyright 2022 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:generate go run invisible/generate.go -v -o ./invisible_gen.go

//go:generate go run ambiguous/generate.go -v -o ./ambiguous_gen.go ambiguous/ambiguous.json

package charset

import (
	"io"
	"strings"

	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/translation"
)

// EscapeControlHTML escapes the unicode control sequences in a provided html document
func EscapeControlHTML(text string, locale translation.Locale) (escaped EscapeStatus, output string) {
	sb := &strings.Builder{}
	outputStream := &HTMLStreamerWriter{Writer: sb}
	streamer := NewEscapeStreamer(locale, outputStream).(*escapeStreamer)

	if err := StreamHTML(strings.NewReader(text), streamer); err != nil {
		streamer.escaped.HasError = true
		log.Error("Error whilst escaping: %v", err)
	}
	output = sb.String()
	escaped = streamer.escaped
	return
}

// EscapeControlReaders escapes the unicode control sequences in a provider reader and writer in a locale and returns the findings as an EscapeStatus and the escaped []byte
func EscapeControlReader(reader io.Reader, writer io.Writer, locale translation.Locale) (escaped EscapeStatus, err error) {
	outputStream := &HTMLStreamerWriter{Writer: writer}
	streamer := NewEscapeStreamer(locale, outputStream).(*escapeStreamer)

	if err = StreamHTML(reader, streamer); err != nil {
		streamer.escaped.HasError = true
		log.Error("Error whilst escaping: %v", err)
	}
	escaped = streamer.escaped
	return
}

// EscapeControlString escapes the unicode control sequences in a provided string and returns the findings as an EscapeStatus and the escaped string
func EscapeControlString(text string, locale translation.Locale) (escaped EscapeStatus, output string) {
	sb := &strings.Builder{}
	outputStream := &HTMLStreamerWriter{Writer: sb}
	streamer := NewEscapeStreamer(locale, outputStream).(*escapeStreamer)

	if err := streamer.Text(text); err != nil {
		streamer.escaped.HasError = true
		log.Error("Error whilst escaping: %v", err)
	}
	return streamer.escaped, sb.String()
}
