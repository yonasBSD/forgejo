// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package private_test

import (
	"net/http"
	"testing"

	asymkey_model "code.gitea.io/gitea/models/asymkey"
	auth_model "code.gitea.io/gitea/models/auth"
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/cache"
	"code.gitea.io/gitea/modules/json"
	"code.gitea.io/gitea/modules/private"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/test"
	"code.gitea.io/gitea/modules/web"
	private_router "code.gitea.io/gitea/routers/private"
	"code.gitea.io/gitea/services/contexttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServTOTPRecovery(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	require.NoError(t, cache.Init())

	t.Run("Disabled", func(t *testing.T) {
		ctx, resp := contexttest.MockPrivateContext(t, "/")
		private_router.ServTOTPRecovery(ctx)
		assert.Equal(t, http.StatusForbidden, resp.Result().StatusCode)

		var extra private.Response
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &extra))

		assert.Equal(t, "This feature is not enabled on this instance.", extra.UserMsg)
	})

	defer test.MockVariableValue(&setting.SSH.AllowTOTPRegeneration, true)()

	t.Run("Bad key id", func(t *testing.T) {
		ctx, resp := contexttest.MockPrivateContext(t, "/")
		web.SetForm(ctx, &private.SSHTOTPRecoveryOption{
			KeyID: 0,
		})

		private_router.ServTOTPRecovery(ctx)
		assert.Equal(t, http.StatusBadRequest, resp.Result().StatusCode)

		var extra private.Response
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &extra))

		assert.Equal(t, "Bad key id: 0", extra.UserMsg)
	})

	t.Run("Account not active", func(t *testing.T) {
		user := unittest.AssertExistsAndLoadBean(t, &user_model.User{}, "is_active = false AND prohibit_login = false")
		key := &asymkey_model.PublicKey{OwnerID: user.ID, Verified: true, Type: asymkey_model.KeyTypeUser}
		unittest.AssertSuccessfulInsert(t, key)

		ctx, resp := contexttest.MockPrivateContext(t, "/")
		web.SetForm(ctx, &private.SSHTOTPRecoveryOption{
			KeyID: key.ID,
		})

		private_router.ServTOTPRecovery(ctx)
		assert.Equal(t, http.StatusForbidden, resp.Result().StatusCode)

		var extra private.Response
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &extra))

		assert.Equal(t, "Your account is disabled.", extra.UserMsg)
	})

	t.Run("Account prohibited login", func(t *testing.T) {
		user := unittest.AssertExistsAndLoadBean(t, &user_model.User{}, "is_active = true AND prohibit_login = true")
		key := &asymkey_model.PublicKey{OwnerID: user.ID, Verified: true, Type: asymkey_model.KeyTypeUser}
		unittest.AssertSuccessfulInsert(t, key)

		ctx, resp := contexttest.MockPrivateContext(t, "/")
		web.SetForm(ctx, &private.SSHTOTPRecoveryOption{
			KeyID: key.ID,
		})

		private_router.ServTOTPRecovery(ctx)
		assert.Equal(t, http.StatusForbidden, resp.Result().StatusCode)

		var extra private.Response
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &extra))

		assert.Equal(t, "Your account is disabled.", extra.UserMsg)
	})

	t.Run("Dangling public key", func(t *testing.T) {
		key := &asymkey_model.PublicKey{OwnerID: 99999, Verified: true, Type: asymkey_model.KeyTypeUser}
		unittest.AssertSuccessfulInsert(t, key)

		ctx, resp := contexttest.MockPrivateContext(t, "/")
		web.SetForm(ctx, &private.SSHTOTPRecoveryOption{
			KeyID: key.ID,
		})

		private_router.ServTOTPRecovery(ctx)
		assert.Equal(t, http.StatusUnauthorized, resp.Result().StatusCode)

		var extra private.Response
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &extra))

		assert.Equal(t, "Cannot find user[id=99999] for key[id=4]", extra.UserMsg)
	})

	t.Run("Non-existent publickey", func(t *testing.T) {
		ctx, resp := contexttest.MockPrivateContext(t, "/")
		web.SetForm(ctx, &private.SSHTOTPRecoveryOption{
			KeyID: 99999,
		})

		private_router.ServTOTPRecovery(ctx)
		assert.Equal(t, http.StatusInternalServerError, resp.Result().StatusCode)

		var extra private.Response
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &extra))

		assert.Equal(t, "Cannot find key[id=99999]", extra.UserMsg)
	})

	t.Run("SSH key not verified", func(t *testing.T) {
		unittest.AssertExistsIf(t, true, &asymkey_model.PublicKey{ID: 1, OwnerID: 2}, "verified = false")

		ctx, resp := contexttest.MockPrivateContext(t, "/")
		web.SetForm(ctx, &private.SSHTOTPRecoveryOption{
			KeyID: 1,
		})

		private_router.ServTOTPRecovery(ctx)
		assert.Equal(t, http.StatusForbidden, resp.Result().StatusCode)

		var extra private.Response
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &extra))

		assert.Equal(t, "The SSH key is not verified.", extra.UserMsg)
	})

	t.Run("Incorrect SSH key type", func(t *testing.T) {
		verifiedIncorrectTypeKey := &asymkey_model.PublicKey{OwnerID: 2, Verified: true, Type: asymkey_model.KeyTypeDeploy}
		unittest.AssertSuccessfulInsert(t, verifiedIncorrectTypeKey)

		ctx, resp := contexttest.MockPrivateContext(t, "/")
		web.SetForm(ctx, &private.SSHTOTPRecoveryOption{
			KeyID: verifiedIncorrectTypeKey.ID,
		})

		private_router.ServTOTPRecovery(ctx)
		assert.Equal(t, http.StatusForbidden, resp.Result().StatusCode)

		var extra private.Response
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &extra))

		assert.Equal(t, "Not a correct SSH key type.", extra.UserMsg)
	})

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	verifiedKey := &asymkey_model.PublicKey{OwnerID: user.ID, Verified: true, Type: asymkey_model.KeyTypeUser, Name: "n0toose's evil key"}
	unittest.AssertSuccessfulInsert(t, verifiedKey)

	t.Run("No TOTP enrolled", func(t *testing.T) {
		ctx, resp := contexttest.MockPrivateContext(t, "/")
		web.SetForm(ctx, &private.SSHTOTPRecoveryOption{
			KeyID:    verifiedKey.ID,
			Password: "password",
		})

		private_router.ServTOTPRecovery(ctx)
		assert.Equal(t, http.StatusForbidden, resp.Result().StatusCode)

		var extra private.Response
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &extra))

		assert.Equal(t, "You are not enrolled into TOTP.", extra.UserMsg)
	})

	twoFactor := &auth_model.TwoFactor{UID: verifiedKey.OwnerID, ScratchSalt: "aaaaaaa", ScratchHash: "bbbbbbb"}
	unittest.AssertSuccessfulInsert(t, twoFactor)

	t.Run("Regeneration not allowed", func(t *testing.T) {
		ctx, resp := contexttest.MockPrivateContext(t, "/")
		web.SetForm(ctx, &private.SSHTOTPRecoveryOption{
			KeyID:    verifiedKey.ID,
			Password: "password",
		})

		private_router.ServTOTPRecovery(ctx)
		assert.Equal(t, http.StatusForbidden, resp.Result().StatusCode)

		var extra private.Response
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &extra))

		assert.Equal(t, "You have not allowed regeneration of the TOTP recovery code via SSH.", extra.UserMsg)
	})

	twoFactor.AllowRegenerationOverSSH = true
	_, err := db.GetEngine(db.DefaultContext).ID(twoFactor.ID).Cols("allow_regeneration_over_ssh").Update(twoFactor)
	require.NoError(t, err)

	t.Run("Normal", func(t *testing.T) {
		ctx, resp := contexttest.MockPrivateContext(t, "/")
		web.SetForm(ctx, &private.SSHTOTPRecoveryOption{
			KeyID:    verifiedKey.ID,
			Password: "password",
		})

		private_router.ServTOTPRecovery(ctx)
		assert.Equal(t, http.StatusOK, resp.Result().StatusCode)

		var extra private.TOTPRecovery
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &extra))

		twoFactorAfter := unittest.AssertExistsAndLoadBean(t, &auth_model.TwoFactor{ID: twoFactor.ID})
		assert.NotEmpty(t, extra.ScratchCode)
		assert.Equal(t, twoFactorAfter.ScratchHash, auth_model.HashToken(extra.ScratchCode, twoFactorAfter.ScratchSalt))
		assert.NotEqual(t, twoFactor.ScratchHash, twoFactorAfter.ScratchHash)
		assert.NotEqual(t, twoFactor.ScratchSalt, twoFactorAfter.ScratchSalt)
	})

	t.Run("Incorrect password", func(t *testing.T) {
		ctx, resp := contexttest.MockPrivateContext(t, "/")
		web.SetForm(ctx, &private.SSHTOTPRecoveryOption{
			KeyID:    verifiedKey.ID,
			Password: "aaaaaaaa",
		})

		private_router.ServTOTPRecovery(ctx)
		assert.Equal(t, http.StatusForbidden, resp.Result().StatusCode)

		var extra private.Response
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &extra))

		assert.Equal(t, "Incorrect password. You are allowed to retry in 5 minutes.", extra.UserMsg)
	})

	t.Run("Rate limited", func(t *testing.T) {
		ctx, resp := contexttest.MockPrivateContext(t, "/")
		web.SetForm(ctx, &private.SSHTOTPRecoveryOption{
			KeyID:    verifiedKey.ID,
			Password: "password",
		})

		private_router.ServTOTPRecovery(ctx)
		assert.Equal(t, http.StatusForbidden, resp.Result().StatusCode)

		var extra private.Response
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &extra))

		assert.Regexp(t, `You can retry in [0-9]{1,3} seconds\.`, extra.UserMsg)
	})
}
