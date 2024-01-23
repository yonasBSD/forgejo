// Copyright Earl Warren <contact@earl-warren.org>
// Copyright Loïc Dachary <loic@dachary.org>
// SPDX-License-Identifier: MIT

package driver

import (
	"net/http"

	forgejo_options "code.gitea.io/gitea/services/f3/driver/options"
	"lab.forgefriends.org/friendlyforgeformat/gof3/options"
)

func newOptions() options.Interface {
	o := &forgejo_options.Options{}
	o.SetName(Name)
	o.SetNewMigrationHTTPClient(func() *http.Client { return &http.Client{} })
	return o
}
