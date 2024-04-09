// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT
package meilisearch

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDoubleQuoteKeyword(t *testing.T) {
	assert.EqualValues(t, "", DoubleQuoteKeyword(""))
	assert.EqualValues(t, `"a" "b" "c"`, DoubleQuoteKeyword("a b c"))
	assert.EqualValues(t, `"a" "d" "g"`, DoubleQuoteKeyword("a  d g"))
	assert.EqualValues(t, `"a" "d" "g"`, DoubleQuoteKeyword("a  d g"))
	assert.EqualValues(t, `"a" "d" "g"`, DoubleQuoteKeyword(`a  "" "d" """g`))
}
