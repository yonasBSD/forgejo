// Copyright 2023 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	"net/url"

	"code.gitea.io/gitea/modules/validation"
	"github.com/valyala/fastjson"
)

type (
	SourceType string
)

type SourceTypes []SourceType

const (
	ForgejoSourceType SourceType = "frogejo"
)

var KnownSourceTypes = SourceTypes{
	ForgejoSourceType,
}

// NodeInfo data type
// swagger:model
type NodeInfoWellKnown struct {
	Href string
}

func NodeInfoWellKnownUnmarshalJSON(data []byte) (NodeInfoWellKnown, error) {
	p := fastjson.Parser{}
	val, err := p.ParseBytes(data)
	if err != nil {
		return NodeInfoWellKnown{}, err
	}
	href := string(val.GetStringBytes("links", "0", "href"))
	return NodeInfoWellKnown{Href: href}, nil
}

func NewNodeInfoWellKnown(body []byte) (NodeInfoWellKnown, error) {
	result, err := NodeInfoWellKnownUnmarshalJSON(body)
	if err != nil {
		return NodeInfoWellKnown{}, err
	}

	if valid, outcome := validation.IsValid(result); !valid {
		return NodeInfoWellKnown{}, outcome
	}

	return NodeInfoWellKnown{}, nil
}

// Validate collects error strings in a slice and returns this
func (node NodeInfoWellKnown) Validate() []string {
	var result []string
	result = append(result, validation.ValidateNotEmpty(node.Href, "Href")...)

	parsedUrl, err := url.Parse(node.Href)
	if err != nil {
		result = append(result, err.Error())
		return result
	}

	if parsedUrl.Host == "" {
		result = append(result, "Href has to be absolute")
	}

	result = append(result, validation.ValidateOneOf(parsedUrl.Scheme, []string{"http", "https"})...)

	if parsedUrl.RawQuery != "" {
		result = append(result, "Href may not contain query")
	}

	return result
}
