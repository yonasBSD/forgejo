// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package meilisearch

import (
	"errors"
	"fmt"
	"strings"

	"code.gitea.io/gitea/modules/log"
)

// VersionedIndexName returns the full index name with version
func (i *Indexer) VersionedIndexName() string {
	return versionedIndexName(i.indexName, i.version)
}

func versionedIndexName(indexName string, version int) string {
	if version == 0 {
		// Old index name without version
		return indexName
	}

	// The format of the index name is <index_name>_v<version>, not <index_name>.v<version> like elasticsearch.
	// Because meilisearch does not support "." in index name, it should contain only alphanumeric characters, hyphens (-) and underscores (_).
	// See https://www.meilisearch.com/docs/learn/core_concepts/indexes#index-uid

	return fmt.Sprintf("%s_v%d", indexName, version)
}

func (i *Indexer) checkOldIndexes() {
	for v := 0; v < i.version; v++ {
		indexName := versionedIndexName(i.indexName, v)
		_, err := i.Client.GetIndex(indexName)
		if err == nil {
			log.Warn("Found older meilisearch index named %q, Gitea will keep the old NOT DELETED. You can delete the old version after the upgrade succeed.", indexName)
		}
	}
}

// TODO: Should be made configurable.
const MaxTotalHits = 10000

// ErrMalformedResponse is never expected as we initialize the indexer ourself and so define the types.
var ErrMalformedResponse = errors.New("meilisearch returned unexpected malformed content")

func DoubleQuoteKeyword(k string) string {
	kp := strings.Split(k, " ")
	parts := 0
	for i := range kp {
		part := strings.Trim(kp[i], "\"")
		if part != "" {
			kp[parts] = fmt.Sprintf(`"%s"`, part)
			parts++
		}
	}
	return strings.Join(kp[:parts], " ")
}
