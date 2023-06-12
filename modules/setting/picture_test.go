// Copyright 2023 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDisableGravatar(t *testing.T) {
	assert.True(t, GetDefaultDisableGravatar())

	cfg, err := NewConfigProviderFromData(``)
	assert.NoError(t, err)
	loadPictureFrom(cfg)

	assert.True(t, DisableGravatar)
}
