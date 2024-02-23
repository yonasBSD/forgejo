// Copyright Earl Warren <contact@earl-warren.org>
// Copyright Loïc Dachary <loic@dachary.org>
// SPDX-License-Identifier: MIT

package options

import (
	"net/http"

	"lab.forgefriends.org/friendlyforgeformat/gof3/options"
	"lab.forgefriends.org/friendlyforgeformat/gof3/options/cli"
	"lab.forgefriends.org/friendlyforgeformat/gof3/options/logger"
)

type NewMigrationHTTPClientFun func() *http.Client

type Options struct {
	options.Options
	logger.OptionsLogger
	cli.OptionsCLI

	NewMigrationHTTPClient NewMigrationHTTPClientFun
}

func (o *Options) GetNewMigrationHTTPClient() NewMigrationHTTPClientFun {
	return o.NewMigrationHTTPClient
}

func (o *Options) SetNewMigrationHTTPClient(fun NewMigrationHTTPClientFun) {
	o.NewMigrationHTTPClient = fun
}
