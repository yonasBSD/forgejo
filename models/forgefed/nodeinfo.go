// Copyright 2023 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	"fmt"
	"net/url"

	"code.gitea.io/gitea/modules/validation"

	"github.com/valyala/fastjson"
)

type (
	SourceType string
)

const (
	ForgejoSourceType SourceType = "forgejo"
	GiteaSourceType   SourceType = "gitea"
)

var KnownSourceTypes = []any{
	ForgejoSourceType, GiteaSourceType,
}

// ------------------------------------------------ NodeInfoWellKnown ------------------------------------------------

// NodeInfo data type
// swagger:model
type NodeInfoWellKnown struct {
	Href string
}

// Factory function for PersonID. Created struct is asserted to be valid
func NewNodeInfoWellKnown(body []byte) (NodeInfoWellKnown, error) {
	result, err := NodeInfoWellKnownUnmarshalJSON(body)
	if err != nil {
		return NodeInfoWellKnown{}, err
	}

	if valid, err := validation.IsValid(result); !valid {
		return NodeInfoWellKnown{}, err
	}

	return result, nil
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

	result = append(result, validation.ValidateOneOf(parsedUrl.Scheme, []any{"http", "https"})...)

	if parsedUrl.RawQuery != "" {
		result = append(result, "Href may not contain query")
	}

	return result
}

func (id ActorID) AsWellKnownNodeInfoUri() string {
	wellKnownPath := ".well-known/nodeinfo"
	var result string
	if id.Port == "" {
		result = fmt.Sprintf("%s://%s/%s", id.Schema, id.Host, wellKnownPath)
	} else {
		result = fmt.Sprintf("%s://%s:%s/%s", id.Schema, id.Host, id.Port, wellKnownPath)
	}
	return result
}

// ------------------------------------------------ NodeInfo ------------------------------------------------

// NodeInfo data type
// swagger:model
type NodeInfo struct {
	ID     int64 `xorm:"pk autoincr"`
	Source SourceType
}

func NodeInfoUnmarshalJSON(data []byte) (NodeInfo, error) {
	p := fastjson.Parser{}
	val, err := p.ParseBytes(data)
	if err != nil {
		return NodeInfo{}, err
	}
	source := string(val.GetStringBytes("software", "name"))
	result := NodeInfo{}
	result.Source = SourceType(source)
	return result, nil
}

func NewNodeInfo(body []byte) (NodeInfo, error) {
	result, err := NodeInfoUnmarshalJSON(body)
	if err != nil {
		return NodeInfo{}, err
	}

	if valid, err := validation.IsValid(result); !valid {
		return NodeInfo{}, err
	}
	return result, nil
}

// Validate collects error strings in a slice and returns this
func (node NodeInfo) Validate() []string {
	var result []string
	result = append(result, validation.ValidateNotEmpty(string(node.Source), "source")...)
	result = append(result, validation.ValidateOneOf(node.Source, KnownSourceTypes)...)

	return result
}
