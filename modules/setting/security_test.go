// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDisablePasswordRecoverySet(t *testing.T) {
	type testCaseDef struct {
		title    string
		input    string
		expected bool
	}

	testCases := []testCaseDef{
		{title: "default", input: "", expected: false},
		{title: "set to true", input: "DISABLE_PASSWORD_RECOVERY=true", expected: true},
		{title: "set to false", input: "DISABLE_PASSWORD_RECOVERY=false", expected: false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.title, func(t *testing.T) {
			cfg, err := NewConfigProviderFromData("[security]\n" + testCase.input)
			require.NoError(t, err)
			loadSecurityFrom(cfg)
			assert.EqualValues(t, testCase.expected, DisablePasswordRecovery)
		})
	}
}
