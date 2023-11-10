// Copyright 2023 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	"reflect"
	"testing"

	ap "github.com/go-ap/activitypub"
)

func Test_StarMarshalJSON(t *testing.T) {
	type testPair struct {
		item    Star
		want    []byte
		wantErr error
	}

	tests := map[string]testPair{
		"empty": {
			item: Star{},
			want: nil,
		},
		"with ID": {
			item: Star{
				Source: "forgejo",
				Activity: ap.Activity{
					ID:   "https://repo.prod.meissa.de/api/activitypub/user-id/1",
					Type: "Star",
					Object: ap.Object{
						ID: "https://codeberg.org/api/activitypub/repository-id/1",
					},
				},
			},
			want: []byte(`{
			"type": "Star",
			"source": "forgejo",
			"actor": "https://repo.prod.meissa.de/api/activitypub/user-id/1",
			"object": "https://codeberg.org/api/activitypub/repository-id/1"
		  }`),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := tt.item.MarshalJSON()
			if (err != nil || tt.wantErr != nil) && tt.wantErr.Error() != err.Error() {
				t.Errorf("MarshalJSON() error = \"%v\", wantErr \"%v\"", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MarshalJSON() got = %q, want %q", got, tt.want)
			}
		})
	}
}
