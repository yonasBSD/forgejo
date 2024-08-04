// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package activitypub

import (
	"net/http"

	activities_model "code.gitea.io/gitea/models/activities"
	"code.gitea.io/gitea/modules/activitypub"
	"code.gitea.io/gitea/modules/forgefed"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/services/context"
	"code.gitea.io/gitea/services/convert"
	"code.gitea.io/gitea/services/federation"

	ap "github.com/go-ap/activitypub"
	"github.com/go-ap/jsonld"
)

// Person function returns the Person actor for a user
func Person(ctx *context.APIContext) {
	// swagger:operation GET /activitypub/user-id/{user-id} activitypub activitypubPerson
	// ---
	// summary: Returns the Person actor for a user
	// produces:
	// - application/json
	// parameters:
	// - name: user-id
	//   in: path
	//   description: user ID of the user
	//   type: integer
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/ActivityPub"

	person, err := convert.ToActivityPubPerson(ctx, ctx.ContextUser)
	if err != nil {
		ctx.ServerError("convert.ToActivityPubPerson", err)
		return
	}

	binary, err := jsonld.WithContext(jsonld.IRI(ap.ActivityBaseURI), jsonld.IRI(ap.SecurityContextURI)).Marshal(person)
	if err != nil {
		ctx.ServerError("MarshalJSON", err)
		return
	}
	ctx.Resp.Header().Add("Content-Type", activitypub.ActivityStreamsContentType)
	ctx.Resp.WriteHeader(http.StatusOK)
	if _, err = ctx.Resp.Write(binary); err != nil {
		log.Error("write to resp err: %v", err)
	}
}

// PersonInbox function handles the incoming data for a user inbox
func PersonInbox(ctx *context.APIContext) {
	// swagger:operation POST /activitypub/user-id/{user-id}/inbox activitypub activitypubPersonInbox
	// ---
	// summary: Send to the inbox
	// produces:
	// - application/json
	// parameters:
	// - name: user-id
	//   in: path
	//   description: user ID of the user
	//   type: integer
	//   required: true
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"

	form := web.GetForm(ctx)
	federation.ProcessPersonInbox(ctx, form)
}

func getActivity(ctx *context.APIContext, id int64) (*forgefed.ForgeUserActivity, error) {
	action, err := activities_model.GetActivityByID(ctx, id)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "GetActivityByID", err.Error())
		return nil, err
	}

	if action.UserID != action.ActUserID || action.ActUserID != ctx.ContextUser.ID {
		ctx.NotFound()
		return nil, err
	}

	actions := activities_model.ActionList{action}
	if err := actions.LoadAttributes(ctx); err != nil {
		ctx.Error(http.StatusInternalServerError, "action.LoadAttributes", err.Error())
		return nil, err
	}

	activity, err := convert.ActionToForgeUserActivity(ctx, actions[0])
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "ActionToForgeUserActivity", err.Error())
		return nil, err
	}

	return &activity, nil
}

func PersonActivity(ctx *context.APIContext) {
	// /activitypub/user-id/{user-id}/activities/{activity-id}/activity

	id := ctx.ParamsInt64("activity-id")
	activity, err := getActivity(ctx, id)
	if err != nil {
		return
	}

	binary, err := jsonld.WithContext(jsonld.IRI(ap.ActivityBaseURI), jsonld.IRI(ap.SecurityContextURI)).Marshal(activity)
	if err != nil {
		ctx.ServerError("MarshalJSON", err)
		return
	}
	ctx.Resp.Header().Add("Content-Type", activitypub.ActivityStreamsContentType)
	ctx.Resp.WriteHeader(http.StatusOK)
	if _, err = ctx.Resp.Write(binary); err != nil {
		log.Error("write to resp err: %v", err)
	}
}

func PersonActivityNote(ctx *context.APIContext) {
	// /activitypub/user-id/{user-id}/activities/{activity-id}

	id := ctx.ParamsInt64("activity-id")
	activity, err := getActivity(ctx, id)
	if err != nil {
		return
	}

	binary, err := jsonld.WithContext(jsonld.IRI(ap.ActivityBaseURI), jsonld.IRI(ap.SecurityContextURI)).Marshal(activity.Object)
	if err != nil {
		ctx.ServerError("MarshalJSON", err)
		return
	}
	ctx.Resp.Header().Add("Content-Type", activitypub.ActivityStreamsContentType)
	ctx.Resp.WriteHeader(http.StatusOK)
	if _, err = ctx.Resp.Write(binary); err != nil {
		log.Error("write to resp err: %v", err)
	}
}
