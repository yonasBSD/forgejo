// Copyright 2023 The forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package activitypub

import (
	"testing"
)

func Test_ActorParser(t *testing.T) {
	type testPair struct {
		item string
		want ActorID
	}

	tests := map[string]testPair{
		"empty": {
			item: "",
			want: ActorID{},
		},
		"withValidActorID": {
			item: "https://repo.prod.meissa.de/api/v1/activitypub/user-id/1",
			want: ActorID{
				schema: "https",
				userId: "1",
				path:   "/api/v1/activitypub/user-id/1",
				host:   "repo.prod.meissa.de",
				port:   "",
			},
		},
		"withInvalidActorID": {
			item: "https://repo.prod.meissa.de/api/activitypub/user-id/1",
			want: ActorID{
				schema: "https",
				userId: "1",
				path:   "/api/v1/activitypub/user-id/1",
				host:   "repo.prod.meissa.de",
				port:   "",
			},
		},
	}

	for name, _ := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := ParseActorID(tests[name].item)

			if err != nil {
				t.Errorf("parseActor() error = \"%v\"", err)
				return
			}

		})
	}
}
