// Copyright 2023 The forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package validation

import (
	"fmt"
	"strings"
)

type ValidationFunctions interface {
	Validate() []string
	IsValid() (bool, error)
}

type Validateable struct {
	ValidationFunctions
}

func IsValid(v any) (bool, error) {
	if err := Validate(v); len(err) > 0 {
		errString := strings.Join(err, "\n")
		return false, fmt.Errorf(errString)
	}

	return true, nil
}

func Validate(v any) []string {
	var result = []string{}
	return result
}
