// Copyright 2024 The forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package validation

import (
	"testing"

	"code.gitea.io/gitea/modules/timeutil"
)

func Test_ValidateNotEmpty_ForString(t *testing.T) {
	sut := ""
	if len(ValidateNotEmpty(sut, "dummyField")) == 0 {
		t.Errorf("sut should be invalid")
	}
	sut = "not empty"
	if res := ValidateNotEmpty(sut, "dummyField"); len(res) > 0 {
		t.Errorf("sut should be valid but was %q", res)
	}
}

func Test_ValidateNotEmpty_ForTimestamp(t *testing.T) {
	sut := timeutil.TimeStamp(0)
	if res := ValidateNotEmpty(sut, "dummyField"); len(res) == 0 {
		t.Errorf("sut should be invalid")
	}
	sut = timeutil.TimeStampNow()
	if res := ValidateNotEmpty(sut, "dummyField"); len(res) > 0 {
		t.Errorf("sut should be valid but was %q", res)
	}
}

func Test_ValidateMaxLen(t *testing.T) {
	sut := "0123456789"
	if len(ValidateMaxLen(sut, 9, "dummyField")) == 0 {
		t.Errorf("sut should be invalid")
	}
	sut = "0123456789"
	if res := ValidateMaxLen(sut, 11, "dummyField"); len(res) > 0 {
		t.Errorf("sut should be valid but was %q", res)
	}
}