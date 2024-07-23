// Copyright 2024 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package auth

import (
	"fmt"
	"net/http"
	"testing"

	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/services/contexttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestForgotPassword(t *testing.T) {
	type testCaseDef struct {
		disable     bool
		hasMail     bool
		hasInput    bool
		notFound    bool
		dataDisable any
		dataRequest any
		dataEmail   any
	}

	mailAddr := "nya@nya.nya"
	testCases := []testCaseDef{
		{disable: false, hasMail: false, hasInput: false, dataDisable: true},
		{disable: false, hasMail: false, hasInput: true, dataDisable: true},
		{disable: false, hasMail: true, hasInput: false, dataRequest: true, dataEmail: ""},
		{disable: false, hasMail: true, hasInput: true, dataRequest: true, dataEmail: mailAddr},
		{disable: true, hasMail: false, hasInput: false, notFound: true},
		{disable: true, hasMail: false, hasInput: true, notFound: true},
		{disable: true, hasMail: true, hasInput: false, notFound: true},
		{disable: true, hasMail: true, hasInput: true, notFound: true},
	}
	for _, testCase := range testCases {
		descr := fmt.Sprintf(
			"disable:%v mail:%v input:%v",
			testCase.disable, testCase.hasMail, testCase.hasInput,
		)

		t.Run(descr, func(t *testing.T) {
			setting.IsProd = false // so we can check the target page
			ctx, resp := contexttest.MockContext(t, "/user/forgot_password")
			setting.DisablePasswordRecovery = testCase.disable
			if testCase.hasMail {
				setting.MailService = &setting.Mailer{}
			} else {
				setting.MailService = nil
			}
			if testCase.hasInput {
				ctx.SetFormString("email", mailAddr)
			}
			checkDataValue := func(id string, expect any) {
				value, found := ctx.Data[id]
				require.Equal(t, expect != nil, found)
				if expect != nil {
					assert.EqualValues(t, expect, value)
				}
			}

			ForgotPasswd(ctx)

			if testCase.notFound {
				assert.Equal(t, http.StatusNotFound, resp.Code)
			} else {
				assert.Equal(t, http.StatusOK, resp.Code)
				checkDataValue("TemplateName", tplForgotPassword)
			}
			checkDataValue("IsResetDisable", testCase.dataDisable)
			checkDataValue("IsResetRequest", testCase.dataRequest)
			checkDataValue("Email", testCase.dataEmail)
		})
	}
}
