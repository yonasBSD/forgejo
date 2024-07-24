// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"bytes"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	asymkey_model "code.gitea.io/gitea/models/asymkey"
	auth_model "code.gitea.io/gitea/models/auth"
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/setting"
	api "code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/modules/test"
	"code.gitea.io/gitea/modules/translation"
	"code.gitea.io/gitea/services/mailer"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

func doCheckRepositoryEmptyStatus(ctx APITestContext, isEmpty bool) func(*testing.T) {
	return doAPIGetRepository(ctx, func(t *testing.T, repository api.Repository) {
		assert.Equal(t, isEmpty, repository.Empty)
	})
}

func doAddChangesToCheckout(dstPath, filename string) func(*testing.T) {
	return func(t *testing.T) {
		require.NoError(t, os.WriteFile(filepath.Join(dstPath, filename), []byte(fmt.Sprintf("# Testing Repository\n\nOriginally created in: %s at time: %v", dstPath, time.Now())), 0o644))
		require.NoError(t, git.AddChanges(dstPath, true))
		signature := git.Signature{
			Email: "test@example.com",
			Name:  "test",
			When:  time.Now(),
		}
		require.NoError(t, git.CommitChanges(dstPath, git.CommitChangesOptions{
			Committer: &signature,
			Author:    &signature,
			Message:   "Initial Commit",
		}))
	}
}

func TestPushDeployKeyOnEmptyRepo(t *testing.T) {
	onGiteaRun(t, testPushDeployKeyOnEmptyRepo)
}

func testPushDeployKeyOnEmptyRepo(t *testing.T, u *url.URL) {
	forEachObjectFormat(t, func(t *testing.T, objectFormat git.ObjectFormat) {
		// OK login
		ctx := NewAPITestContext(t, "user2", "deploy-key-empty-repo-"+objectFormat.Name(), auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)

		keyname := fmt.Sprintf("%s-push", ctx.Reponame)
		u.Path = ctx.GitPath()

		t.Run("CreateEmptyRepository", doAPICreateRepository(ctx, true, objectFormat))

		t.Run("CheckIsEmpty", doCheckRepositoryEmptyStatus(ctx, true))

		withKeyFile(t, keyname, func(keyFile string) {
			t.Run("CreatePushDeployKey", doAPICreateDeployKey(ctx, keyname, keyFile, false))

			// Setup the testing repository
			dstPath := t.TempDir()

			t.Run("InitTestRepository", doGitInitTestRepository(dstPath, objectFormat))

			// Setup remote link
			sshURL := createSSHUrl(ctx.GitPath(), u)

			t.Run("AddRemote", doGitAddRemote(dstPath, "origin", sshURL))

			t.Run("SSHPushTestRepository", doGitPushTestRepository(dstPath, "origin", "master"))

			t.Run("CheckIsNotEmpty", doCheckRepositoryEmptyStatus(ctx, false))

			t.Run("DeleteRepository", doAPIDeleteRepository(ctx))
		})
	})
}

func TestKeyOnlyOneType(t *testing.T) {
	onGiteaRun(t, testKeyOnlyOneType)
}

func testKeyOnlyOneType(t *testing.T, u *url.URL) {
	// Once a key is a user key we cannot use it as a deploy key
	// If we delete it from the user we should be able to use it as a deploy key
	reponame := "ssh-key-test-repo"
	username := "user2"
	u.Path = fmt.Sprintf("%s/%s.git", username, reponame)
	keyname := fmt.Sprintf("%s-push", reponame)

	// OK login
	ctx := NewAPITestContext(t, username, reponame, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)
	ctxWithDeleteRepo := NewAPITestContext(t, username, reponame, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)

	otherCtx := ctx
	otherCtx.Reponame = "ssh-key-test-repo-2"
	otherCtxWithDeleteRepo := ctxWithDeleteRepo
	otherCtxWithDeleteRepo.Reponame = otherCtx.Reponame

	failCtx := ctx
	failCtx.ExpectedCode = http.StatusUnprocessableEntity

	t.Run("CreateRepository", doAPICreateRepository(ctx, false, git.Sha1ObjectFormat))           // FIXME: use forEachObjectFormat
	t.Run("CreateOtherRepository", doAPICreateRepository(otherCtx, false, git.Sha1ObjectFormat)) // FIXME: use forEachObjectFormat

	withKeyFile(t, keyname, func(keyFile string) {
		var userKeyPublicKeyID int64
		t.Run("KeyCanOnlyBeUser", func(t *testing.T) {
			dstPath := t.TempDir()

			sshURL := createSSHUrl(ctx.GitPath(), u)

			t.Run("FailToClone", doGitCloneFail(sshURL))

			t.Run("CreateUserKey", doAPICreateUserKey(ctx, keyname, keyFile, func(t *testing.T, publicKey api.PublicKey) {
				userKeyPublicKeyID = publicKey.ID
			}))

			t.Run("FailToAddReadOnlyDeployKey", doAPICreateDeployKey(failCtx, keyname, keyFile, true))

			t.Run("FailToAddDeployKey", doAPICreateDeployKey(failCtx, keyname, keyFile, false))

			t.Run("Clone", doGitClone(dstPath, sshURL))

			t.Run("AddChanges", doAddChangesToCheckout(dstPath, "CHANGES1.md"))

			t.Run("Push", doGitPushTestRepository(dstPath, "origin", "master"))

			t.Run("DeleteUserKey", doAPIDeleteUserKey(ctx, userKeyPublicKeyID))
		})

		t.Run("KeyCanBeAnyDeployButNotUserAswell", func(t *testing.T) {
			dstPath := t.TempDir()

			sshURL := createSSHUrl(ctx.GitPath(), u)

			t.Run("FailToClone", doGitCloneFail(sshURL))

			// Should now be able to add...
			t.Run("AddReadOnlyDeployKey", doAPICreateDeployKey(ctx, keyname, keyFile, true))

			t.Run("Clone", doGitClone(dstPath, sshURL))

			t.Run("AddChanges", doAddChangesToCheckout(dstPath, "CHANGES2.md"))

			t.Run("FailToPush", doGitPushTestRepositoryFail(dstPath, "origin", "master"))

			otherSSHURL := createSSHUrl(otherCtx.GitPath(), u)
			dstOtherPath := t.TempDir()

			t.Run("AddWriterDeployKeyToOther", doAPICreateDeployKey(otherCtx, keyname, keyFile, false))

			t.Run("CloneOther", doGitClone(dstOtherPath, otherSSHURL))

			t.Run("AddChangesToOther", doAddChangesToCheckout(dstOtherPath, "CHANGES3.md"))

			t.Run("PushToOther", doGitPushTestRepository(dstOtherPath, "origin", "master"))

			t.Run("FailToCreateUserKey", doAPICreateUserKey(failCtx, keyname, keyFile))
		})

		t.Run("DeleteRepositoryShouldReleaseKey", func(t *testing.T) {
			otherSSHURL := createSSHUrl(otherCtx.GitPath(), u)
			dstOtherPath := t.TempDir()

			t.Run("DeleteRepository", doAPIDeleteRepository(ctxWithDeleteRepo))

			t.Run("FailToCreateUserKeyAsStillDeploy", doAPICreateUserKey(failCtx, keyname, keyFile))

			t.Run("MakeSureCloneOtherStillWorks", doGitClone(dstOtherPath, otherSSHURL))

			t.Run("AddChangesToOther", doAddChangesToCheckout(dstOtherPath, "CHANGES3.md"))

			t.Run("PushToOther", doGitPushTestRepository(dstOtherPath, "origin", "master"))

			t.Run("DeleteOtherRepository", doAPIDeleteRepository(otherCtxWithDeleteRepo))

			t.Run("RecreateRepository", doAPICreateRepository(ctxWithDeleteRepo, false, git.Sha1ObjectFormat)) // FIXME: use forEachObjectFormat

			t.Run("CreateUserKey", doAPICreateUserKey(ctx, keyname, keyFile, func(t *testing.T, publicKey api.PublicKey) {
				userKeyPublicKeyID = publicKey.ID
			}))

			dstPath := t.TempDir()

			sshURL := createSSHUrl(ctx.GitPath(), u)

			t.Run("Clone", doGitClone(dstPath, sshURL))

			t.Run("AddChanges", doAddChangesToCheckout(dstPath, "CHANGES1.md"))

			t.Run("Push", doGitPushTestRepository(dstPath, "origin", "master"))
		})

		t.Run("DeleteUserKeyShouldRemoveAbilityToClone", func(t *testing.T) {
			sshURL := createSSHUrl(ctx.GitPath(), u)

			t.Run("DeleteUserKey", doAPIDeleteUserKey(ctx, userKeyPublicKeyID))

			t.Run("FailToClone", doGitCloneFail(sshURL))
		})
	})
}

func TestTOTPRecoveryCodes(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 5})
		ctx := NewAPITestContext(t, user.Name, "repo4", auth_model.AccessTokenScopeWriteUser, auth_model.AccessTokenScopeWriteRepository)

		withKeyFile(t, "totp", func(keyFile string) {
			keyName := "test-key-totp"
			t.Run("CreateUserKey", func(t *testing.T) {
				doAPICreateUserKey(ctx, keyName, keyFile)(t)

				publicKey := unittest.AssertExistsAndLoadBean(t, &asymkey_model.PublicKey{OwnerID: user.ID})
				publicKey.Verified = true
				_, err := db.GetEngine(db.DefaultContext).ID(publicKey.ID).Cols("verified").Update(publicKey)
				require.NoError(t, err)
			})

			privateKeyBytes, err := os.ReadFile(keyFile)
			require.NoError(t, err)

			privateKey, err := ssh.ParsePrivateKey(privateKeyBytes)
			require.NoError(t, err)

			config := &ssh.ClientConfig{
				User:            setting.SSH.BuiltinServerUser,
				HostKeyCallback: ssh.InsecureIgnoreHostKey(),
				Auth:            []ssh.AuthMethod{ssh.PublicKeys(privateKey)},
			}

			client, err := ssh.Dial("tcp", net.JoinHostPort(setting.SSH.ListenHost, strconv.Itoa(setting.SSH.ListenPort)), config)
			require.NoError(t, err)
			defer client.Close()

			t.Run("Not enrolled into TOTP", func(t *testing.T) {
				session, err := client.NewSession()
				require.NoError(t, err)
				defer session.Close()

				var b bytes.Buffer
				session.Stderr = &b
				session.Stdin = strings.NewReader("y\npassword\n")
				require.NoError(t, session.Start("totp_recovery_codes"))

				require.Error(t, session.Wait())
				assert.EqualValues(t, "Forgejo: You are not enrolled into TOTP.\n", b.String())
			})

			twoFactor := &auth_model.TwoFactor{UID: user.ID, ScratchSalt: "aaaaaaa", ScratchHash: "bbbbbbb"}
			unittest.AssertSuccessfulInsert(t, twoFactor)

			t.Run("Not allowed regeneration", func(t *testing.T) {
				session, err := client.NewSession()
				require.NoError(t, err)
				defer session.Close()

				var b bytes.Buffer
				session.Stderr = &b
				session.Stdin = strings.NewReader("y\npassword\n")
				require.NoError(t, session.Start("totp_recovery_codes"))

				require.Error(t, session.Wait())
				assert.EqualValues(t, "Forgejo: You have not allowed regeneration of the TOTP recovery code via SSH.\n", b.String())
			})

			twoFactor.AllowRegenerationOverSSH = true
			_, err = db.GetEngine(db.DefaultContext).ID(twoFactor.ID).Cols("allow_regeneration_over_ssh").Update(twoFactor)
			require.NoError(t, err)

			t.Run("Normal", func(t *testing.T) {
				called := false
				defer test.MockVariableValue(&mailer.SendAsync, func(msgs ...*mailer.Message) {
					assert.Len(t, msgs, 1)
					assert.Equal(t, user.EmailTo(), msgs[0].To)
					assert.EqualValues(t, translation.NewLocale("en-US").Tr("mail.totp_regenerated_via_ssh.subject"), msgs[0].Subject)
					assert.Contains(t, msgs[0].Body, keyName)
					called = true
				})()

				session, err := client.NewSession()
				require.NoError(t, err)
				defer session.Close()

				var stdOutBuf bytes.Buffer
				session.Stdout = &stdOutBuf
				session.Stdin = strings.NewReader("aaaa\nbbbbb\ny\npassword\n")
				require.NoError(t, session.Start("totp_recovery_codes"))
				assert.NoError(t, session.Wait())

				twoFactorAfter := unittest.AssertExistsAndLoadBean(t, &auth_model.TwoFactor{ID: twoFactor.ID})
				scratchCode := strings.TrimFunc(regexp.MustCompile(`"[A-Z0-9]+"`).FindString(stdOutBuf.String()), func(r rune) bool { return r == '"' })
				assert.Equal(t, twoFactorAfter.ScratchHash, auth_model.HashToken(scratchCode, twoFactorAfter.ScratchSalt))
				assert.NotEqual(t, twoFactor.ScratchHash, twoFactorAfter.ScratchHash)
				assert.NotEqual(t, twoFactor.ScratchSalt, twoFactorAfter.ScratchSalt)

				assert.True(t, called)
			})

			t.Run("Incorrect password", func(t *testing.T) {
				session, err := client.NewSession()
				require.NoError(t, err)
				defer session.Close()

				var b bytes.Buffer
				session.Stderr = &b
				session.Stdin = strings.NewReader("y\nblahaj00!\n")
				require.NoError(t, session.Start("totp_recovery_codes"))

				require.Error(t, session.Wait())
				assert.EqualValues(t, "Forgejo: Incorrect password. You are allowed to retry in 5 minutes.\n", b.String())
			})

			t.Run("Rate limimted", func(t *testing.T) {
				session, err := client.NewSession()
				require.NoError(t, err)
				defer session.Close()

				var b bytes.Buffer
				session.Stderr = &b
				session.Stdin = strings.NewReader("y\npassword\n")
				require.NoError(t, session.Start("totp_recovery_codes"))

				require.Error(t, session.Wait())
				assert.Regexp(t, `Forgejo: You can retry in [0-9]{1,3} seconds\.`, b.String())
			})

			t.Run("Exit", func(t *testing.T) {
				session, err := client.NewSession()
				require.NoError(t, err)
				defer session.Close()

				var stdOutBuf bytes.Buffer
				session.Stdout = &stdOutBuf
				session.Stdin = strings.NewReader("no\n")
				require.NoError(t, session.Start("totp_recovery_codes"))
				assert.NoError(t, session.Wait())

				assert.Contains(t, stdOutBuf.String(), "No new TOTP recovery code has been generated. The existing one will remain valid.")
			})
		})

		withKeyFile(t, "totp-deploy", func(keyFile string) {
			t.Run("CreateDeployKey", func(t *testing.T) {
				doAPICreateDeployKey(ctx, "totp-deploy", keyFile, false)(t)

				publicKey := unittest.AssertExistsAndLoadBean(t, &asymkey_model.PublicKey{Name: "totp-deploy", Type: asymkey_model.KeyTypeDeploy})
				publicKey.Verified = true
				_, err := db.GetEngine(db.DefaultContext).ID(publicKey.ID).Cols("verified").Update(publicKey)
				require.NoError(t, err)
			})

			privateKeyBytes, err := os.ReadFile(keyFile)
			require.NoError(t, err)

			privateKey, err := ssh.ParsePrivateKey(privateKeyBytes)
			require.NoError(t, err)

			config := &ssh.ClientConfig{
				User:            setting.SSH.BuiltinServerUser,
				HostKeyCallback: ssh.InsecureIgnoreHostKey(),
				Auth:            []ssh.AuthMethod{ssh.PublicKeys(privateKey)},
			}

			client, err := ssh.Dial("tcp", net.JoinHostPort(setting.SSH.ListenHost, strconv.Itoa(setting.SSH.ListenPort)), config)
			require.NoError(t, err)
			defer client.Close()

			t.Run("Incorrect SSH key type", func(t *testing.T) {
				session, err := client.NewSession()
				require.NoError(t, err)
				defer session.Close()

				var b bytes.Buffer
				session.Stderr = &b
				session.Stdin = strings.NewReader("y\npassword\n")
				require.NoError(t, session.Start("totp_recovery_codes"))

				require.Error(t, session.Wait())
				assert.EqualValues(t, "Forgejo: Not a correct SSH key type.\n", b.String())
			})
		})
	})
}
