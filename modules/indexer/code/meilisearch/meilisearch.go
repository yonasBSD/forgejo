// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package meilisearch

import (
	"bufio"
	"cmp"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"

	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/modules/analyze"
	"code.gitea.io/gitea/modules/charset"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/indexer/code/internal"
	indexer_internal "code.gitea.io/gitea/modules/indexer/internal"
	inner_meilisearch "code.gitea.io/gitea/modules/indexer/internal/meilisearch"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/timeutil"
	"code.gitea.io/gitea/modules/typesniffer"

	"github.com/go-enry/go-enry/v2"
	"github.com/meilisearch/meilisearch-go"
)

const repoIndexerLatestVersion = 6

var _ internal.Indexer = &Indexer{}

// Indexer represents a bleve indexer implementation
type Indexer struct {
	waitForIndex             bool
	inner                    *inner_meilisearch.Indexer
	indexer_internal.Indexer // do not composite inner_meilisearch.Indexer directly to avoid exposing too much
}

// filenameIndexerID is specific to Meilisearch as it only allows: (a-z, A-Z, 0-9), hyphens (-), and underscores (_).
func filenameIndexerID(repoID int64, filename string) string {
	return strconv.FormatInt(repoID, 36) + "_" + base64.RawURLEncoding.EncodeToString([]byte(filename))
}

// parseIndexerID parses the indexerID and returns the repoID and filename.
func parseIndexerID(indexerID string) (int64, string) {
	index := strings.IndexByte(indexerID, '_')
	if index == -1 {
		log.Error("Unexpected ID in repo indexer: %s", indexerID)
	}

	repoID, _ := strconv.ParseInt(indexerID[:index], 36, 64)
	fileName, _ := base64.RawURLEncoding.DecodeString(indexerID[index+1:])
	return repoID, string(fileName)
}

// NewIndexer creates a new meilisearch indexer
func NewIndexer(url, apiKey, indexerName string, waitForIndex bool) *Indexer {
	settings := &meilisearch.Settings{
		// The default ranking rules of meilisearch are: ["words", "typo", "proximity", "attribute", "sort", "exactness"]
		// So even if we specify the sort order, it could not be respected because the priority of "sort" is so low.
		// So we need to specify the ranking rules to make sure the sort order is respected.
		// See https://www.meilisearch.com/docs/learn/core_concepts/relevancy
		RankingRules: []string{"sort", // make sure "sort" has the highest priority
			"words", "typo", "proximity", "attribute", "exactness"},

		SearchableAttributes: []string{
			"content",
		},
		DisplayedAttributes: []string{
			"id",
			"language",
			"content",
			"commit_id",
			"updated_at",
		},
		FilterableAttributes: []string{
			"repo_id",
			"language",
		},
		SortableAttributes: []string{
			"repo_id",
			"id",
		},
		Pagination: &meilisearch.Pagination{
			MaxTotalHits: inner_meilisearch.MaxTotalHits,
		},
	}

	inner := inner_meilisearch.NewIndexer(url, apiKey, indexerName, repoIndexerLatestVersion, settings)

	return &Indexer{
		waitForIndex: waitForIndex,
		Indexer:      inner,
		inner:        inner,
	}
}

func (m *Indexer) Index(ctx context.Context, repo *repo_model.Repository, sha string, changes *internal.RepoChanges) error {
	if len(changes.Updates) > 0 {
		// Now because of some insanity with git cat-file not immediately failing if not run in a valid git directory we need to run git rev-parse first!
		if err := git.EnsureValidGitRepository(ctx, repo.RepoPath()); err != nil {
			log.Error("Unable to open git repo: %s for %-v: %v", repo.RepoPath(), repo, err)
			return err
		}

		batchWriter, batchReader, cancel := git.CatFileBatch(ctx, repo.RepoPath())
		defer cancel()

		for _, update := range changes.Updates {
			err := m.addUpdate(ctx, batchWriter, batchReader, sha, update, repo)
			if err != nil {
				return err
			}
		}
		cancel()
	}

	for _, filename := range changes.RemovedFilenames {
		_, err := m.inner.Client.Index(m.inner.VersionedIndexName()).DeleteDocument(filenameIndexerID(repo.ID, filename))
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *Indexer) addUpdate(ctx context.Context, batchWriter git.WriteCloserError, batchReader *bufio.Reader, sha string, update internal.FileUpdate, repo *repo_model.Repository) error {
	// Ignore vendored files in code search
	if setting.Indexer.ExcludeVendored && analyze.IsVendor(update.Filename) {
		return nil
	}

	size := update.Size
	var err error
	if !update.Sized {
		var stdout string
		stdout, _, err = git.NewCommand(ctx, "cat-file", "-s").AddDynamicArguments(update.BlobSha).RunStdString(&git.RunOpts{Dir: repo.RepoPath()})
		if err != nil {
			return err
		}
		if size, err = strconv.ParseInt(strings.TrimSpace(stdout), 10, 64); err != nil {
			return fmt.Errorf("misformatted git cat-file output: %w", err)
		}
	}

	if size > setting.Indexer.MaxIndexerFileSize {
		// First check if the document was indexed.
		if err := m.inner.Client.Index(m.inner.VersionedIndexName()).GetDocument(filenameIndexerID(repo.ID, update.Filename), nil, nil); err == nil {
			_, err := m.inner.Client.Index(m.inner.VersionedIndexName()).DeleteDocument(filenameIndexerID(repo.ID, update.Filename))
			if err != nil {
				return err
			}
		}
	}

	if _, err := batchWriter.Write([]byte(update.BlobSha + "\n")); err != nil {
		return err
	}

	_, _, size, err = git.ReadBatchLine(batchReader)
	if err != nil {
		return err
	}

	fileContents, err := io.ReadAll(io.LimitReader(batchReader, size))
	if err != nil {
		return err
	} else if !typesniffer.DetectContentType(fileContents).IsText() {
		// FIXME: UTF-16 files will probably fail here
		return nil
	}

	if _, err = batchReader.Discard(1); err != nil {
		return err
	}

	id := filenameIndexerID(repo.ID, update.Filename)
	task, err := m.inner.Client.Index(m.inner.VersionedIndexName()).AddDocuments(map[string]any{
		"id":         id,
		"repo_id":    repo.ID,
		"content":    string(charset.ToUTF8DropErrors(fileContents, charset.ConvertOpts{})),
		"commit_id":  sha,
		"language":   analyze.GetCodeLanguage(update.Filename, fileContents),
		"updated_at": timeutil.TimeStampNow(),
	})

	if m.waitForIndex {
		if _, err := m.inner.Client.WaitForTask(task.TaskUID); err != nil {
			return err
		}
	}
	return err
}

func (m *Indexer) Delete(ctx context.Context, repoID int64) error {
	_, err := m.inner.Client.Index(m.inner.VersionedIndexName()).DeleteDocumentsByFilter(fmt.Sprintf("repo_id = %d", repoID))
	if err != nil {
		return err
	}
	return nil
}

func convertHits(searchRes *meilisearch.SearchResponse) ([]*internal.SearchResult, error) {
	hits := make([]*internal.SearchResult, 0, len(searchRes.Hits))
	for _, hit := range searchRes.Hits {
		hit, ok := hit.(map[string]any)
		if !ok {
			return nil, fmt.Errorf(`field "hit" is not type "map[string]any": %w`, inner_meilisearch.ErrMalformedResponse)
		}
		id, ok := hit["id"].(string)
		if !ok {
			return nil, fmt.Errorf(`field "id" is not type "string": %w`, inner_meilisearch.ErrMalformedResponse)
		}
		commitID, ok := hit["commit_id"].(string)
		if !ok {
			return nil, fmt.Errorf(`field "commit_id" is not type "string": %w`, inner_meilisearch.ErrMalformedResponse)
		}
		content, ok := hit["content"].(string)
		if !ok {
			return nil, fmt.Errorf(`field "content" is not type "string": %w`, inner_meilisearch.ErrMalformedResponse)
		}
		language, ok := hit["language"].(string)
		if !ok {
			return nil, fmt.Errorf(`field "language" is not type "string": %w`, inner_meilisearch.ErrMalformedResponse)
		}
		updatedAt, ok := hit["updated_at"].(float64)
		if !ok {
			return nil, fmt.Errorf(`field "update_at" is not a number: %w`, inner_meilisearch.ErrMalformedResponse)
		}
		matches, ok := hit["_matchesPosition"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf(`field "_matchesPosition" is not type "map[string]any": %w`, inner_meilisearch.ErrMalformedResponse)
		}

		contentMatches, ok := matches["content"].([]any)
		if !ok {
			return nil, fmt.Errorf(`field "content" in field "_matchesPosition" is not type "[]any": %w`, inner_meilisearch.ErrMalformedResponse)
		}

		if len(contentMatches) == 0 {
			return nil, fmt.Errorf(`field "content" in field "_matchesPosition" is empty: %w`, inner_meilisearch.ErrMalformedResponse)
		}

		// TODO: Highlight multiple matches.
		firstMatch := contentMatches[0].(map[string]any)
		startIndex := int(firstMatch["start"].(float64))
		matchLength := int(firstMatch["length"].(float64))

		repoID, fileName := parseIndexerID(id)
		hits = append(hits, &internal.SearchResult{
			RepoID:      repoID,
			Filename:    fileName,
			CommitID:    commitID,
			Content:     content,
			UpdatedUnix: timeutil.TimeStamp(updatedAt),
			Language:    language,
			StartIndex:  startIndex,
			EndIndex:    startIndex + matchLength,
			Color:       enry.GetColor(language),
		})

	}
	return hits, nil
}

func languageResults(searchRes *meilisearch.SearchResponse) []*internal.SearchResultLanguages {
	languages := map[string]int{}
	for _, hit := range searchRes.Hits {
		hit := hit.(map[string]any)
		language := hit["language"].(string)
		if language != "" {
			languages[language]++
		}
	}

	searchResultLanguages := make([]*internal.SearchResultLanguages, 0, len(languages))
	for language, count := range languages {
		searchResultLanguages = append(searchResultLanguages, &internal.SearchResultLanguages{
			Language: language,
			Color:    enry.GetColor(language),
			Count:    count,
		})
	}

	// Sort by count descending, then by name (a-z).
	slices.SortFunc(searchResultLanguages, func(a, b *internal.SearchResultLanguages) int {
		if n := cmp.Compare(b.Count, a.Count); n != 0 {
			return n
		}
		return cmp.Compare(strings.ToLower(a.Language), strings.ToLower(b.Language))
	})

	return searchResultLanguages
}

func (m *Indexer) Search(ctx context.Context, opts *internal.SearchOptions) (int64, []*internal.SearchResult, []*internal.SearchResultLanguages, error) {
	query := inner_meilisearch.FilterAnd{}

	if opts.Language != "" {
		query.And(inner_meilisearch.FilterEq(fmt.Sprintf("language = '%s'", opts.Language)))
	}

	query.And(inner_meilisearch.NewFilterIn("repo_id", opts.RepoIDs...))

	keyword := opts.Keyword
	if !opts.IsKeywordFuzzy {
		// to make it non fuzzy ("typo tolerance" in meilisearch terms), we have to quote the keyword(s)
		// https://www.meilisearch.com/docs/reference/api/search#phrase-search
		keyword = inner_meilisearch.DoubleQuoteKeyword(keyword)
	}

	start, pageSize := opts.GetSkipTake()
	searchRes, err := m.inner.Client.Index(m.inner.VersionedIndexName()).Search(keyword, &meilisearch.SearchRequest{
		Filter:              query.Statement(),
		Sort:                []string{"repo_id:desc", "id:desc"},
		Offset:              int64(start),
		Limit:               int64(pageSize),
		ShowMatchesPosition: true,
		MatchingStrategy:    "all",
	})
	if err != nil {
		return 0, nil, nil, err
	}

	hits, err := convertHits(searchRes)

	return searchRes.EstimatedTotalHits, hits, nil, err
}
