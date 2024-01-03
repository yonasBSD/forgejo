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
		item    ForgeLike
		want    []byte
		wantErr error
	}

	tests := map[string]testPair{
		"empty": {
			item: ForgeLike{},
			want: nil,
		},
		"with ID": {
			item: ForgeLike{
				Activity: ap.Activity{
					Actor:  ap.IRI("https://repo.prod.meissa.de/api/v1/activitypub/user-id/1"),
					Type:   "Like",
					Object: ap.IRI("https://codeberg.org/api/v1/activitypub/repository-id/1"),
				},
			},
			want: []byte(`{"type":"Like","actor":"https://repo.prod.meissa.de/api/v1/activitypub/user-id/1","object":"https://codeberg.org/api/v1/activitypub/repository-id/1"}`),
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

func Test_StarUnmarshalJSON(t *testing.T) {
	type testPair struct {
		item    []byte
		want    *ForgeLike
		wantErr error
	}

	tests := map[string]testPair{
		"with ID": {
			item: []byte(`{"type":"Like","actor":"https://repo.prod.meissa.de/api/activitypub/user-id/1","object":"https://codeberg.org/api/activitypub/repository-id/1"}`),
			want: &ForgeLike{
				Activity: ap.Activity{
					Actor:  ap.IRI("https://repo.prod.meissa.de/api/activitypub/user-id/1"),
					Type:   "Like",
					Object: ap.IRI("https://codeberg.org/api/activitypub/repository-id/1"),
				},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := new(ForgeLike)
			err := got.UnmarshalJSON(tt.item)
			if (err != nil || tt.wantErr != nil) && tt.wantErr.Error() != err.Error() {
				t.Errorf("UnmarshalJSON() error = \"%v\", wantErr \"%v\"", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UnmarshalJSON() got = %q, want %q", got, tt.want)
			}
		})
	}
}
