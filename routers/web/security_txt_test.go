// Copyright 2023 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package web

import (
	"regexp"
	"testing"
	"time"
)

func extractLines(message, pattern string) []string {
	ptn := regexp.MustCompile(pattern)
	return ptn.FindAllString(message, -1)
}

func TestSecurityTxt(t *testing.T) {
	// Contact: is required and value MUST be https:// or mailto:
	{
		contacts := extractLines(securityTxtContent, `(?m:^Contact: .+$)`)
		if contacts == nil {
			t.Error("Error: \"Contact: \" field is required")
		}
		for _, contact := range contacts {
			match, err := regexp.MatchString("Contact: (https:)|(mailto:)", contact)
			if !match {
				t.Error("Error in line ", contact, "\n\"Contact:\" field have incorrect format")
			}
			if err != nil {
				t.Error("Error in line ", contact, err)
			}
		}
	}
	// Expires is required
	{
		expires := extractLines(securityTxtContent, `(?m:^Expires: .+$)`)
		if expires == nil {
			t.Error("Error: \"Expires: \" field is required")
		}
		if len(expires) != 1 {
			t.Error("Error: \"Expires: \" MUST be single")
		}
		expRe := regexp.MustCompile(`Expires: (.*)`)
		expSlice := expRe.FindStringSubmatch(expires[0])
		if len(expSlice) != 2 {
			t.Error("Error: \"Expires: \" have no value")
		}
		expValue := expSlice[1]
		expTime, err := time.Parse(time.RFC3339, expValue)
		if err != nil {
			t.Error("Error parsing Expires value", expValue, err)
		}
		if time.Now().AddDate(0, 2, 0).After(expTime) {
			t.Error("Error: Expires date time almost in the past", expTime)
		}
	}
}
