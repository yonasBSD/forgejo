// SPDX-License-Identifier: MIT

package util

import (
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/log"
	base "code.gitea.io/gitea/modules/migration"
	"code.gitea.io/gitea/services/f3/driver"

	"lab.forgefriends.org/friendlyforgeformat/gof3"
	f3_forges "lab.forgefriends.org/friendlyforgeformat/gof3/forges"
	"lab.forgefriends.org/friendlyforgeformat/gof3/forges/f3"
)

func ToF3Logger(messenger base.Messenger) gof3.Logger {
	if messenger == nil {
		messenger = func(string, ...interface{}) {}
	}
	return gof3.Logger{
		Message:  messenger,
		Trace:    log.Trace,
		Debug:    log.Debug,
		Info:     log.Info,
		Warn:     log.Warn,
		Error:    log.Error,
		Critical: log.Critical,
		Fatal:    log.Fatal,
	}
}

func ForgejoForgeRoot(features gof3.Features, doer *user_model.User) *f3_forges.ForgeRoot {
	forgeRoot := f3_forges.NewForgeRootFromDriver(&driver.Forgejo{}, &driver.Options{
		Options: gof3.Options{
			Features: features,
			Logger:   ToF3Logger(nil),
		},
		Doer: doer,
	})
	return forgeRoot
}

func F3ForgeRoot(features gof3.Features, directory string) *f3_forges.ForgeRoot {
	forgeRoot := f3_forges.NewForgeRoot(&f3.Options{
		Options: gof3.Options{
			Configuration: gof3.Configuration{
				Directory: directory,
			},
			Features: features,
			Logger:   ToF3Logger(nil),
		},
		Remap: true,
	})
	return forgeRoot
}
