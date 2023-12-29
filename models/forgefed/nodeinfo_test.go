// Copyright 2023 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	"fmt"
	"reflect"
	"testing"

	"code.gitea.io/gitea/modules/validation"
)

func Test_NodeInfoWellKnownUnmarshalJSON(t *testing.T) {
	type testPair struct {
		item    []byte
		want    NodeInfoWellKnown
		wantErr error
	}

	tests := map[string]testPair{
		"with href": {
			item: []byte(`{"links":[{"href":"https://federated-repo.prod.meissa.de/api/v1/nodeinfo","rel":"http://nodeinfo.diaspora.software/ns/schema/2.1"}]}`),
			want: NodeInfoWellKnown{
				Href: "https://federated-repo.prod.meissa.de/api/v1/nodeinfo",
			},
		},
		"empty": {
			item:    []byte(``),
			wantErr: fmt.Errorf("cannot parse JSON: cannot parse empty string; unparsed tail: \"\""),
		},
		// "with too long href": {
		// 	item:    []byte(`{"links":[{"href":"https://federated-repo.prod.meissa.de/api/v1/nodeinfohttps://federated-repo.prod.meissa.de/api/v1/nodeinfohttps://federated-repo.prod.meissa.de/api/v1/nodeinfohttps://federated-repo.prod.meissa.de/api/v1/nodeinfohttps://federated-repo.prod.meissa.de/api/v1/nodeinfohttps://federated-repo.prod.meissa.de/api/v1/nodeinfo","rel":"http://nodeinfo.diaspora.software/ns/schema/2.1"}]}`),
		// 	wantErr: fmt.Errorf("cannot parse JSON: cannot parse empty string; unparsed tail: \"\""),
		// },
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := NodeInfoWellKnownUnmarshalJSON(tt.item)
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

func Test_NodeInfoWellKnownValidate(t *testing.T) {
	sut := NodeInfoWellKnown{Href: "https://federated-repo.prod.meissa.de/api/v1/nodeinfo"}
	if b, err := validation.IsValid(sut); !b {
		t.Errorf("sut should be valid, %v, %v", sut, err)
	}

	sut = NodeInfoWellKnown{Href: "./federated-repo.prod.meissa.de/api/v1/nodeinfo"}
	if _, err := validation.IsValid(sut); err.Error() != "Href has to be absolute\nValue  is not contained in allowed values [[http https]]" {
		t.Errorf("validation error expected but was: %v\n", err)
	}

	sut = NodeInfoWellKnown{Href: "https://federated-repo.prod.meissa.de/api/v1/nodeinfo?alert=1"}
	if _, err := validation.IsValid(sut); err.Error() != "Href may not contain query" {
		t.Errorf("sut should be valid, %v, %v", sut, err)
	}
}
