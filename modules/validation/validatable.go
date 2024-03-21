// Copyright 2023 The forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package validation

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"code.gitea.io/gitea/modules/timeutil"
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

func ValidateNotEmpty(value any, fieldName string) []string {
	isValid := true
	switch v := value.(type) {
	case string:
		if v == "" {
			isValid = false
		}
	case timeutil.TimeStamp:
		if v.IsZero() {
			isValid = false
		}
	case int64:
		if v == 0 {
			isValid = false
		}
	default:
		isValid = false
	}

	if isValid {
		return []string{}
	}
	return []string{fmt.Sprintf("Field %v should not be empty", fieldName)}
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
