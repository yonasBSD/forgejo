// Copyright 2022 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package math

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

// Extension is a math extension
type Extension struct {
	enabled                bool
	inlineStartDelimRender string
	inlineEndDelimRender   string
	blockStartDelimRender  string
	blockEndDelimRender    string
	parseDollarInline      bool
	parseDollarBlock       bool
}

// Option is the interface Options should implement
type Option interface {
	SetOption(e *Extension)
}

type extensionFunc func(e *Extension)

func (fn extensionFunc) SetOption(e *Extension) {
	fn(e)
}

// Enabled enables or disables this extension
func Enabled(enable ...bool) Option {
	value := true
	if len(enable) > 0 {
		value = enable[0]
	}
	return extensionFunc(func(e *Extension) {
		e.enabled = value
	})
}

// WithInlineDollarParser enables or disables the parsing of $...$
func WithInlineDollarParser(enable ...bool) Option {
	value := true
	if len(enable) > 0 {
		value = enable[0]
	}
	return extensionFunc(func(e *Extension) {
		e.parseDollarInline = value
	})
}

// WithBlockDollarParser enables or disables the parsing of $$...$$
func WithBlockDollarParser(enable ...bool) Option {
	value := true
	if len(enable) > 0 {
		value = enable[0]
	}
	return extensionFunc(func(e *Extension) {
		e.parseDollarBlock = value
	})
}

// WithInlineDelimRender sets the start and end strings for the rendered inline delimiters
func WithInlineDelimRender(start, end string) Option {
	return extensionFunc(func(e *Extension) {
		e.inlineStartDelimRender = start
		e.inlineEndDelimRender = end
	})
}

// WithBlockDelimRender sets the start and end strings for the rendered block delimiters
func WithBlockDelimRender(start, end string) Option {
	return extensionFunc(func(e *Extension) {
		e.blockStartDelimRender = start
		e.blockEndDelimRender = end
	})
}

// Math represents a math extension with default rendered delimiters
var Math = &Extension{
	enabled:                true,
	inlineStartDelimRender: `\(`,
	inlineEndDelimRender:   `\)`,
	blockStartDelimRender:  `\[`,
	blockEndDelimRender:    `\]`,
	parseDollarBlock:       true,
}

// NewExtension creates a new math extension with the provided options
func NewExtension(opts ...Option) *Extension {
	r := &Extension{
		enabled:                true,
		inlineStartDelimRender: `\(`,
		inlineEndDelimRender:   `\)`,
		blockStartDelimRender:  `\[`,
		blockEndDelimRender:    `\]`,
		parseDollarBlock:       true,
	}

	for _, o := range opts {
		o.SetOption(r)
	}
	return r
}

// Extend extends goldmark with our parsers and renderers
func (e *Extension) Extend(m goldmark.Markdown) {
	if !e.enabled {
		return
	}

	m.Parser().AddOptions(parser.WithBlockParsers(
		util.Prioritized(NewBlockParser(e.parseDollarBlock), 701),
	))

	inlines := []util.PrioritizedValue{
		util.Prioritized(NewInlineBracketParser(), 501),
	}
	if e.parseDollarInline {
		inlines = append(inlines, util.Prioritized(NewInlineDollarParser(), 501))
	}
	m.Parser().AddOptions(parser.WithInlineParsers(inlines...))

	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(NewBlockRenderer(e.blockStartDelimRender, e.blockEndDelimRender), 501),
		util.Prioritized(NewInlineRenderer(e.inlineStartDelimRender, e.inlineEndDelimRender), 502),
	))
}
