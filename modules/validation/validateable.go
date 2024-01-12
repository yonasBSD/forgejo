// Copyright 2023 The forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package validation

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

type Validateable interface {
	Validate() []string
}

func IsValid(v Validateable) (bool, error) {
	if err := v.Validate(); len(err) > 0 {
		errString := strings.Join(err, "\n")
		return false, fmt.Errorf(errString)
	}

	return true, nil
}

func ValidateNotEmpty(value string, fieldName string) []string {
	if value == "" {
		return []string{fmt.Sprintf("Field %v may not be empty", fieldName)}
	}
	return []string{}
}

func ValidateMaxLen(value string, maxLen int, fieldName string) []string {
	if utf8.RuneCountInString(value) > maxLen {
		return []string{fmt.Sprintf("Value in field %v was longer than %v", fieldName, maxLen)}
	}
	return []string{}
}

func ValidateOneOf(value any, allowed []any) []string {
	for _, allowedElem := range allowed {
		if value == allowedElem {
			return []string{}
		}
	}
	return []string{fmt.Sprintf("Value %v is not contained in allowed values [%v]", value, allowed)}
}

func ValidateSuffix(str, suffix string) bool {
	return strings.HasSuffix(str, suffix)
}
