// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package swagger

import (
	activities_model "code.gitea.io/gitea/models/activities"
	api "code.gitea.io/gitea/modules/structs"
)

// User
// swagger:response User
type swaggerResponseUser struct {
	// in:body
	Body api.User `json:"body"`
}

// UserList
// swagger:response UserList
type swaggerResponseUserList struct {
	// in:body
	Body []api.User `json:"body"`
}

// EmailList
// swagger:response EmailList
type swaggerResponseEmailList struct {
	// in:body
	Body []api.Email `json:"body"`
}

// swagger:model EditUserOption
type swaggerModelEditUserOption struct {
	// in:body
	Options api.EditUserOption
}

// UserHeatmapData
// swagger:response UserHeatmapData
type swaggerResponseUserHeatmapData struct {
	// in:body
	Body []activities_model.UserHeatmapData `json:"body"`
}

// UserSettings
// swagger:response UserSettings
type swaggerResponseUserSettings struct {
	// in:body
	Body api.UserSettings `json:"body"`
}

// StarList
// swagger:response StarList
type swaggerResponseStarList struct {
	// in:body
	Body api.StarList `json:"body"`
}

// StarListSlice
// swagger:response StarListSlice
type swaggerResponseStarListSlice struct {
	// in:body
	Body []api.StarList `json:"body"`
}

// StarListRepoInfo
// swagger:response StarListRepoInfo
type swaggerResponseStarListRepoInfo struct {
	// in:body
	Body []api.StarListRepoInfo `json:"body"`
}
