// Copyright 2023 The forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package activitypub

import (
	"testing"
)

func Test_ActorParser(t *testing.T) {
	type testPair struct {
		item    string
		want    ActorData
		wantErr error
	}

	tests := map[string]testPair{
		"empty": {
			item: "",
			want: ActorData{},
		},
		"withActorID": {
			item: "https://repo.prod.meissa.de/api/v1/activitypub/user-id/1",
			want: ActorData{
				schema: "https://",
				userId: "1",
				host:   "repo.prod.meissa.de",
				port:   "",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := tt.want.parseActor(tests[name].item)

			if (err != nil || tt.wantErr != nil) && tt.wantErr.Error() != err.Error() {
				t.Errorf("parseActor() error = \"%v\", wantErr \"%v\"", err, tt.wantErr)
				return
			}
		})
	}
}
