package actions

import (
	"bytes"
	"net/url"
	"strconv"

	actions_model "code.gitea.io/gitea/models/actions"
	"code.gitea.io/gitea/models/perm"
	"code.gitea.io/gitea/models/perm/access"
	"code.gitea.io/gitea/modules/actions"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/json"
	"code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/modules/webhook"
	context_module "code.gitea.io/gitea/services/context"
	"code.gitea.io/gitea/services/convert"
	"github.com/nektos/act/pkg/jobparser"
	act_model "github.com/nektos/act/pkg/model"
)

func ManualRunWorkflow(ctx *context_module.Context) {
	workflowID := ctx.FormString("workflow")
	if len(workflowID) == 0 {
		ctx.ServerError("workflow", nil)
		return
	}

	ref := ctx.FormString("ref")
	if len(ref) == 0 {
		ctx.ServerError("ref", nil)
		return
	}

	if empty, err := ctx.Repo.GitRepo.IsEmpty(); err != nil {
		ctx.ServerError("IsEmpty", err)
		return
	} else if empty {
		ctx.NotFound("IsEmpty", nil)
		return
	}

	commit, err := ctx.Repo.GitRepo.GetCommit(ref)
	if err != nil {
		ctx.ServerError("GetCommit", err)
		return
	}

	entries, err := actions.ListWorkflows(commit)
	if err != nil {
		ctx.ServerError("ListWorkflows", err)
		return
	}

	var workflowEntry *git.TreeEntry
	for _, entry := range entries {
		if entry.Name() == workflowID {
			workflowEntry = entry
			break
		}
	}
	if workflowEntry == nil {
		ctx.NotFound("workflow in ListWorkflows", nil)
		return
	}

	content, err := actions.GetContentFromEntry(workflowEntry)
	if err != nil {
		ctx.ServerError("GetContentFromEntry", err)
		return
	}
	wf, err := act_model.ReadWorkflow(bytes.NewReader(content))
	if err != nil {
		ctx.ServerError("ReadWorkflow", err)
		return
	}

	fullWorkflowID := ".forgejo/workflows/" + workflowID

	title := wf.Name
	if len(title) < 1 {
		title = fullWorkflowID
	}

	location := ctx.Repo.RepoLink + "/actions?workflow=" + url.QueryEscape(workflowID) +
		"&actor=" + url.QueryEscape(ctx.FormString("actor")) +
		"&status=" + url.QueryEscape(ctx.FormString("status"))

	inputs := make(map[string]string)
	if workflowDispatch := wf.WorkflowDispatchConfig(); workflowDispatch != nil {
		for key, input := range workflowDispatch.Inputs {
			formKey := "inputs[" + key + "]"
			val := ctx.FormString(formKey)
			if len(val) == 0 {
				val = input.Default
				if len(val) == 0 {
					if input.Required {
						name := input.Description
						if len(name) == 0 {
							name = key
						}
						ctx.Flash.Error(ctx.Locale.Tr("actions.workflow.dispatch.input_required", name))
						ctx.Redirect(location)
						return
					}
					continue
				}
			} else {
				switch input.Type {
				case "boolean":
					// Since "boolean" inputs are rendered as a checkbox in html, the value inside the form is "on"
					val = strconv.FormatBool(val == "on")
				}
			}
			inputs[key] = val
		}
	}

	payload := &structs.WorkflowDispatchPayload{
		Inputs:     inputs,
		Ref:        ref,
		Repository: convert.ToRepo(ctx, ctx.Repo.Repository, access.Permission{AccessMode: perm.AccessModeNone}),
		Sender:     convert.ToUser(ctx, ctx.Doer, nil),
		Workflow:   fullWorkflowID,
	}

	p, err := json.Marshal(payload)
	if err != nil {
		ctx.ServerError("json.Marshal", err)
		return
	}

	run := &actions_model.ActionRun{
		Title:         title,
		RepoID:        ctx.Repo.Repository.ID,
		Repo:          ctx.Repo.Repository,
		OwnerID:       ctx.Repo.Repository.OwnerID,
		WorkflowID:    workflowID,
		TriggerUserID: ctx.Doer.ID,
		TriggerUser:   ctx.Doer,
		Ref:           ref,
		CommitSHA:     commit.ID.String(),
		Event:         webhook.HookEventWorkflowDispatch,
		EventPayload:  string(p),
		TriggerEvent:  string(webhook.HookEventWorkflowDispatch),
		Status:        actions_model.StatusWaiting,
	}

	vars, err := actions_model.GetVariablesOfRun(ctx, run)
	if err != nil {
		ctx.ServerError("GetVariablesOfRun", err)
		return
	}

	jobs, err := jobparser.Parse(content, jobparser.WithVars(vars))
	if err != nil {
		ctx.ServerError("jobparser.Parse", err)
		return
	}

	if err := actions_model.InsertRun(ctx, run, jobs); err != nil {
		ctx.ServerError("InsertRun", err)
		return
	}

	// forward to the page of the run which was just created
	ctx.Flash.Info(ctx.Locale.Tr("actions.workflow.dispatch.success"))
	ctx.Redirect(location)
}
