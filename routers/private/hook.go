// Copyright 2019 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Package private includes all internal routes. The package name internal is ideal but Golang is not allowed, so we use private as package name instead.
package private

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/private"
	"code.gitea.io/gitea/modules/repofiles"
	"code.gitea.io/gitea/modules/util"
	pull_service "code.gitea.io/gitea/services/pull"
	"gopkg.in/src-d/go-git.v4/plumbing"

	"gitea.com/macaron/macaron"
)

// HookPreReceive checks whether a individual commit is acceptable
func HookPreReceive(ctx *macaron.Context, opts private.HookOptions) {
	ownerName := ctx.Params(":owner")
	repoName := ctx.Params(":repo")
	repo, err := models.GetRepositoryByOwnerAndName(ownerName, repoName)
	if err != nil {
		log.Error("Unable to get repository: %s/%s Error: %v", ownerName, repoName, err)
		ctx.JSON(http.StatusInternalServerError, map[string]interface{}{
			"err": err.Error(),
		})
		return
	}
	repo.OwnerName = ownerName
	gitRepo, err := git.OpenRepository(repo.RepoPath())
	if err != nil {
		log.Error("Unable to get git repository for: %s/%s Error: %v", ownerName, repoName, err)
		ctx.JSON(http.StatusInternalServerError, map[string]interface{}{
			"err": err.Error(),
		})
		return
	}
	defer gitRepo.Close()

	for i := range opts.OldCommitIDs {
		oldCommitID := opts.OldCommitIDs[i]
		newCommitID := opts.NewCommitIDs[i]
		refFullName := opts.RefFullNames[i]

		branchName := strings.TrimPrefix(refFullName, git.BranchPrefix)
		protectBranch, err := models.GetProtectedBranchBy(repo.ID, branchName)
		if err != nil {
			log.Error("Unable to get protected branch: %s in %-v Error: %v", branchName, repo, err)
			ctx.JSON(500, map[string]interface{}{
				"err": err.Error(),
			})
			return
		}
		if protectBranch != nil && protectBranch.IsProtected() {
			// check and deletion
			if newCommitID == git.EmptySHA {
				log.Warn("Forbidden: Branch: %s in %-v is protected from deletion", branchName, repo)
				ctx.JSON(http.StatusForbidden, map[string]interface{}{
					"err": fmt.Sprintf("branch %s is protected from deletion", branchName),
				})
				return
			}

			// detect force push
			if git.EmptySHA != oldCommitID {
				env := os.Environ()
				if opts.GitAlternativeObjectDirectories != "" {
					env = append(env,
						private.GitAlternativeObjectDirectories+"="+opts.GitAlternativeObjectDirectories)
				}
				if opts.GitObjectDirectory != "" {
					env = append(env,
						private.GitObjectDirectory+"="+opts.GitObjectDirectory)
				}
				if opts.GitQuarantinePath != "" {
					env = append(env,
						private.GitQuarantinePath+"="+opts.GitQuarantinePath)
				}

				output, err := git.NewCommand("rev-list", "--max-count=1", oldCommitID, "^"+newCommitID).RunInDirWithEnv(repo.RepoPath(), env)
				if err != nil {
					log.Error("Unable to detect force push between: %s and %s in %-v Error: %v", oldCommitID, newCommitID, repo, err)
					ctx.JSON(http.StatusInternalServerError, map[string]interface{}{
						"err": fmt.Sprintf("Fail to detect force push: %v", err),
					})
					return
				} else if len(output) > 0 {
					log.Warn("Forbidden: Branch: %s in %-v is protected from force push", branchName, repo)
					ctx.JSON(http.StatusForbidden, map[string]interface{}{
						"err": fmt.Sprintf("branch %s is protected from force push", branchName),
					})
					return

				}
			}

			// Require signed commits
			if protectBranch.RequireSignedCommits {
				env := os.Environ()
				if opts.GitAlternativeObjectDirectories != "" {
					env = append(env,
						private.GitAlternativeObjectDirectories+"="+opts.GitAlternativeObjectDirectories)
				}
				if opts.GitObjectDirectory != "" {
					env = append(env,
						private.GitObjectDirectory+"="+opts.GitObjectDirectory)
				}
				if opts.GitQuarantinePath != "" {
					env = append(env,
						private.GitQuarantinePath+"="+opts.GitQuarantinePath)
				}

				stdoutReader, stdoutWriter, err := os.Pipe()
				if err != nil {
					log.Error("Unable to create pipe: %v", err)
					ctx.JSON(http.StatusInternalServerError, map[string]interface{}{
						"err": fmt.Sprintf("Unable to create pipe: %v", err),
					})
					return
				}
				defer func() {
					_ = stdoutReader.Close()
					_ = stdoutWriter.Close()
				}()

				stderr := new(bytes.Buffer)
				commits := make([]*git.Commit, 0, 10)

				var finalErr error
				if err := git.NewCommand("rev-list", oldCommitID+"..."+newCommitID).
					RunInDirTimeoutEnvFullPipelineFunc(env, -1, repo.RepoPath(),
						stdoutWriter, stderr, nil,
						func(ctx context.Context, cancel context.CancelFunc) {
							_ = stdoutWriter.Close()
							scanner := bufio.NewScanner(stdoutReader)
							for scanner.Scan() {
								line := scanner.Text()
								// TODO: Consider whether we really want to read these completely in to memory
								var commitStr string
								commitStr, finalErr = git.NewCommand("cat-file", "commit", line).RunInDirWithEnv(repo.RepoPath(), env)
								if finalErr != nil {
									cancel()
								}
								commits = append(commits, &git.Commit{
									ID:            plumbing.NewHash(line),
									CommitMessage: commitStr,
								})
							}
							_ = stdoutReader.Close()
						}); err != nil {
					if finalErr != nil {
						err = finalErr
					}
					log.Error("Unable to check commits from %s to %s in %-v: %v", oldCommitID, newCommitID, repo, err)
					ctx.JSON(http.StatusInternalServerError, map[string]interface{}{
						"err": fmt.Sprintf("Unable to check commits from %s to %s: %v", oldCommitID, newCommitID, err),
					})
					return
				}

				// TODO: We should batch read a few commits in at a time: use -n and --skip
				// Now parse the commitStrs
				for _, commit := range commits {
					payloadSB := new(strings.Builder)
					signatureSB := new(strings.Builder)
					messageSB := new(strings.Builder)
					message := false
					pgpsig := false

					scanner := bufio.NewScanner(strings.NewReader(commit.CommitMessage))
					for scanner.Scan() {
						line := scanner.Bytes()
						if pgpsig {
							if len(line) > 0 && line[0] == ' ' {
								line = bytes.TrimLeft(line, " ")
								_, _ = signatureSB.Write(line)
								_ = signatureSB.WriteByte('\n')
								continue
							} else {
								pgpsig = false
							}
						}

						if !message {
							trimmed := bytes.TrimSpace(line)
							if len(trimmed) == 0 {
								message = true
								_, _ = payloadSB.WriteString("\n")
								continue
							}

							split := bytes.SplitN(trimmed, []byte{' '}, 2)

							switch string(split[0]) {
							case "tree":
								commit.Tree = *git.NewTree(gitRepo, plumbing.NewHash(string(split[1])))
								_, _ = payloadSB.Write(line)
								_ = payloadSB.WriteByte('\n')
							case "parent":
								commit.Parents = append(commit.Parents, plumbing.NewHash(string(split[1])))
								_, _ = payloadSB.Write(line)
								_ = payloadSB.WriteByte('\n')
							case "author":
								commit.Author = &git.Signature{}
								commit.Author.Decode(split[1])
								_, _ = payloadSB.Write(line)
								_ = payloadSB.WriteByte('\n')
							case "committer":
								commit.Committer = &git.Signature{}
								commit.Committer.Decode(split[1])
								_, _ = payloadSB.Write(line)
								_ = payloadSB.WriteByte('\n')
							case "gpgsig":
								_, _ = signatureSB.Write(split[1])
								_ = signatureSB.WriteByte('\n')
								pgpsig = true
							}
						} else {
							_, _ = messageSB.Write(line)
							_ = messageSB.WriteByte('\n')
						}
					}
					commit.CommitMessage = messageSB.String()
					_, _ = payloadSB.WriteString(commit.CommitMessage)
					commit.Signature = &git.CommitGPGSignature{
						Signature: signatureSB.String(),
						Payload:   payloadSB.String(),
					}
					if len(commit.Signature.Signature) == 0 {
						commit.Signature = nil
					}

					verification := models.ParseCommitWithSignature(commit)
					if !verification.Verified {
						log.Warn("Forbidden: Branch: %s in %-v is protected from unverified commit %s", branchName, repo, commit.ID.String())
						ctx.JSON(http.StatusForbidden, map[string]interface{}{
							"err": fmt.Sprintf("branch %s is protected from unverified commit %s", branchName, commit.ID.String()),
						})
						return
					}
				}
			}

			canPush := false
			if opts.IsDeployKey {
				canPush = protectBranch.CanPush && (!protectBranch.EnableWhitelist || protectBranch.WhitelistDeployKeys)
			} else {
				canPush = protectBranch.CanUserPush(opts.UserID)
			}
			if !canPush && opts.ProtectedBranchID > 0 {
				// Manual merge
				pr, err := models.GetPullRequestByID(opts.ProtectedBranchID)
				if err != nil {
					log.Error("Unable to get PullRequest %d Error: %v", opts.ProtectedBranchID, err)
					ctx.JSON(http.StatusInternalServerError, map[string]interface{}{
						"err": fmt.Sprintf("Unable to get PullRequest %d Error: %v", opts.ProtectedBranchID, err),
					})
					return
				}
				user, err := models.GetUserByID(opts.UserID)
				if err != nil {
					log.Error("Unable to get User id %d Error: %v", opts.UserID, err)
					ctx.JSON(http.StatusInternalServerError, map[string]interface{}{
						"err": fmt.Sprintf("Unable to get User id %d Error: %v", opts.UserID, err),
					})
					return
				}
				perm, err := models.GetUserRepoPermission(repo, user)
				if err != nil {
					log.Error("Unable to get Repo permission of repo %s/%s of User %s", repo.OwnerName, repo.Name, user.Name, err)
					ctx.JSON(http.StatusInternalServerError, map[string]interface{}{
						"err": fmt.Sprintf("Unable to get Repo permission of repo %s/%s of User %s: %v", repo.OwnerName, repo.Name, user.Name, err),
					})
					return
				}
				allowedMerge, err := pull_service.IsUserAllowedToMerge(pr, perm, user)
				if err != nil {
					log.Error("Error calculating if allowed to merge: %v", err)
					ctx.JSON(http.StatusInternalServerError, map[string]interface{}{
						"err": fmt.Sprintf("Error calculating if allowed to merge: %v", err),
					})
					return
				}
				if !allowedMerge {
					log.Warn("Forbidden: User %d is not allowed to push to protected branch: %s in %-v and is not allowed to merge pr #%d", opts.UserID, branchName, repo, pr.Index)
					ctx.JSON(http.StatusForbidden, map[string]interface{}{
						"err": fmt.Sprintf("Not allowed to push to protected branch %s", branchName),
					})
					return
				}
				// Manual merge only allowed if PR is ready (even if admin)
				if err := pull_service.CheckPRReadyToMerge(pr); err != nil {
					if models.IsErrNotAllowedToMerge(err) {
						log.Warn("Forbidden: User %d is not allowed push to protected branch %s in %-v and pr #%d is not ready to be merged: %s", opts.UserID, branchName, repo, pr.Index, err.Error())
						ctx.JSON(http.StatusForbidden, map[string]interface{}{
							"err": fmt.Sprintf("Not allowed to push to protected branch %s and pr #%d is not ready to be merged: %s", branchName, opts.ProtectedBranchID, err.Error()),
						})
						return
					}
					log.Error("Unable to check if mergable: protected branch %s in %-v and pr #%d. Error: %v", opts.UserID, branchName, repo, pr.Index, err)
					ctx.JSON(http.StatusInternalServerError, map[string]interface{}{
						"err": fmt.Sprintf("Unable to get status of pull request %d. Error: %v", opts.ProtectedBranchID, err),
					})
				}
			} else if !canPush {
				log.Warn("Forbidden: User %d is not allowed to push to protected branch: %s in %-v", opts.UserID, branchName, repo)
				ctx.JSON(http.StatusForbidden, map[string]interface{}{
					"err": fmt.Sprintf("Not allowed to push to protected branch %s", branchName),
				})
				return
			}
		}
	}

	ctx.PlainText(http.StatusOK, []byte("ok"))
}

// HookPostReceive updates services and users
func HookPostReceive(ctx *macaron.Context, opts private.HookOptions) {
	ownerName := ctx.Params(":owner")
	repoName := ctx.Params(":repo")

	var repo *models.Repository
	updates := make([]*repofiles.PushUpdateOptions, 0, len(opts.OldCommitIDs))
	wasEmpty := false

	for i := range opts.OldCommitIDs {
		refFullName := opts.RefFullNames[i]
		branch := opts.RefFullNames[i]
		if strings.HasPrefix(branch, git.BranchPrefix) {
			branch = strings.TrimPrefix(branch, git.BranchPrefix)
		} else {
			branch = strings.TrimPrefix(branch, git.TagPrefix)
		}

		// Only trigger activity updates for changes to branches or
		// tags.  Updates to other refs (eg, refs/notes, refs/changes,
		// or other less-standard refs spaces are ignored since there
		// may be a very large number of them).
		if strings.HasPrefix(refFullName, git.BranchPrefix) || strings.HasPrefix(refFullName, git.TagPrefix) {
			if repo == nil {
				var err error
				repo, err = models.GetRepositoryByOwnerAndName(ownerName, repoName)
				if err != nil {
					log.Error("Failed to get repository: %s/%s Error: %v", ownerName, repoName, err)
					ctx.JSON(http.StatusInternalServerError, private.HookPostReceiveResult{
						Err: fmt.Sprintf("Failed to get repository: %s/%s Error: %v", ownerName, repoName, err),
					})
					return
				}
				if repo.OwnerName == "" {
					repo.OwnerName = ownerName
				}
				wasEmpty = repo.IsEmpty
			}

			option := repofiles.PushUpdateOptions{
				RefFullName:  refFullName,
				OldCommitID:  opts.OldCommitIDs[i],
				NewCommitID:  opts.NewCommitIDs[i],
				Branch:       branch,
				PusherID:     opts.UserID,
				PusherName:   opts.UserName,
				RepoUserName: ownerName,
				RepoName:     repoName,
			}
			updates = append(updates, &option)
			if repo.IsEmpty && branch == "master" && strings.HasPrefix(refFullName, git.BranchPrefix) {
				// put the master branch first
				copy(updates[1:], updates)
				updates[0] = &option
			}
		}
	}

	if repo != nil && len(updates) > 0 {
		if err := repofiles.PushUpdates(repo, updates); err != nil {
			log.Error("Failed to Update: %s/%s Total Updates: %d", ownerName, repoName, len(updates))
			for i, update := range updates {
				log.Error("Failed to Update: %s/%s Update: %d/%d: Branch: %s", ownerName, repoName, i, len(updates), update.Branch)
			}
			log.Error("Failed to Update: %s/%s Error: %v", ownerName, repoName, err)

			ctx.JSON(http.StatusInternalServerError, private.HookPostReceiveResult{
				Err: fmt.Sprintf("Failed to Update: %s/%s Error: %v", ownerName, repoName, err),
			})
			return
		}
	}

	results := make([]private.HookPostReceiveBranchResult, 0, len(opts.OldCommitIDs))

	// We have to reload the repo in case its state is changed above
	repo = nil
	var baseRepo *models.Repository

	for i := range opts.OldCommitIDs {
		refFullName := opts.RefFullNames[i]
		newCommitID := opts.NewCommitIDs[i]

		branch := git.RefEndName(opts.RefFullNames[i])

		if newCommitID != git.EmptySHA && strings.HasPrefix(refFullName, git.BranchPrefix) {
			if repo == nil {
				var err error
				repo, err = models.GetRepositoryByOwnerAndName(ownerName, repoName)
				if err != nil {
					log.Error("Failed to get repository: %s/%s Error: %v", ownerName, repoName, err)
					ctx.JSON(http.StatusInternalServerError, private.HookPostReceiveResult{
						Err:          fmt.Sprintf("Failed to get repository: %s/%s Error: %v", ownerName, repoName, err),
						RepoWasEmpty: wasEmpty,
					})
					return
				}
				if repo.OwnerName == "" {
					repo.OwnerName = ownerName
				}

				if !repo.AllowsPulls() {
					// We can stop there's no need to go any further
					ctx.JSON(http.StatusOK, private.HookPostReceiveResult{
						RepoWasEmpty: wasEmpty,
					})
					return
				}
				baseRepo = repo

				if repo.IsFork {
					if err := repo.GetBaseRepo(); err != nil {
						log.Error("Failed to get Base Repository of Forked repository: %-v Error: %v", repo, err)
						ctx.JSON(http.StatusInternalServerError, private.HookPostReceiveResult{
							Err:          fmt.Sprintf("Failed to get Base Repository of Forked repository: %-v Error: %v", repo, err),
							RepoWasEmpty: wasEmpty,
						})
						return
					}
					baseRepo = repo.BaseRepo
				}
			}

			if !repo.IsFork && branch == baseRepo.DefaultBranch {
				results = append(results, private.HookPostReceiveBranchResult{})
				continue
			}

			pr, err := models.GetUnmergedPullRequest(repo.ID, baseRepo.ID, branch, baseRepo.DefaultBranch)
			if err != nil && !models.IsErrPullRequestNotExist(err) {
				log.Error("Failed to get active PR in: %-v Branch: %s to: %-v Branch: %s Error: %v", repo, branch, baseRepo, baseRepo.DefaultBranch, err)
				ctx.JSON(http.StatusInternalServerError, private.HookPostReceiveResult{
					Err: fmt.Sprintf(
						"Failed to get active PR in: %-v Branch: %s to: %-v Branch: %s Error: %v", repo, branch, baseRepo, baseRepo.DefaultBranch, err),
					RepoWasEmpty: wasEmpty,
				})
				return
			}

			if pr == nil {
				if repo.IsFork {
					branch = fmt.Sprintf("%s:%s", repo.OwnerName, branch)
				}
				results = append(results, private.HookPostReceiveBranchResult{
					Message: true,
					Create:  true,
					Branch:  branch,
					URL:     fmt.Sprintf("%s/compare/%s...%s", baseRepo.HTMLURL(), util.PathEscapeSegments(baseRepo.DefaultBranch), util.PathEscapeSegments(branch)),
				})
			} else {
				results = append(results, private.HookPostReceiveBranchResult{
					Message: true,
					Create:  false,
					Branch:  branch,
					URL:     fmt.Sprintf("%s/pulls/%d", baseRepo.HTMLURL(), pr.Index),
				})
			}
		}
	}
	ctx.JSON(http.StatusOK, private.HookPostReceiveResult{
		Results:      results,
		RepoWasEmpty: wasEmpty,
	})
}

// SetDefaultBranch updates the default branch
func SetDefaultBranch(ctx *macaron.Context) {
	ownerName := ctx.Params(":owner")
	repoName := ctx.Params(":repo")
	branch := ctx.Params(":branch")
	repo, err := models.GetRepositoryByOwnerAndName(ownerName, repoName)
	if err != nil {
		log.Error("Failed to get repository: %s/%s Error: %v", ownerName, repoName, err)
		ctx.JSON(http.StatusInternalServerError, map[string]interface{}{
			"Err": fmt.Sprintf("Failed to get repository: %s/%s Error: %v", ownerName, repoName, err),
		})
		return
	}
	if repo.OwnerName == "" {
		repo.OwnerName = ownerName
	}

	repo.DefaultBranch = branch
	gitRepo, err := git.OpenRepository(repo.RepoPath())
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, map[string]interface{}{
			"Err": fmt.Sprintf("Failed to get git repository: %s/%s Error: %v", ownerName, repoName, err),
		})
		return
	}
	if err := gitRepo.SetDefaultBranch(repo.DefaultBranch); err != nil {
		if !git.IsErrUnsupportedVersion(err) {
			gitRepo.Close()
			ctx.JSON(http.StatusInternalServerError, map[string]interface{}{
				"Err": fmt.Sprintf("Unable to set default branch onrepository: %s/%s Error: %v", ownerName, repoName, err),
			})
			return
		}
	}
	gitRepo.Close()

	if err := repo.UpdateDefaultBranch(); err != nil {
		ctx.JSON(http.StatusInternalServerError, map[string]interface{}{
			"Err": fmt.Sprintf("Unable to set default branch onrepository: %s/%s Error: %v", ownerName, repoName, err),
		})
		return
	}
	ctx.PlainText(200, []byte("success"))
}
