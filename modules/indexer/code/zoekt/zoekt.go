// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

//go:build unix

package zoekt

import (
	"bufio"
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"regexp/syntax"
	"slices"
	"strconv"
	"strings"

	repo_model "forgejo.org/models/repo"
	"forgejo.org/modules/analyze"
	"forgejo.org/modules/charset"
	"forgejo.org/modules/git"
	"forgejo.org/modules/gitrepo"
	"forgejo.org/modules/indexer/code/internal"
	indexer_internal "forgejo.org/modules/indexer/internal"
	inner_zoekt "forgejo.org/modules/indexer/internal/zoekt"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/typesniffer"

	"github.com/go-enry/go-enry/v2"
	"github.com/sourcegraph/zoekt"
	"github.com/sourcegraph/zoekt/index"
	"github.com/sourcegraph/zoekt/query"
)

const repoIndexerLatestVersion = 1

type Indexer struct {
	indexer_internal.Indexer // do not composite inner_zoekt.Indexer directly to avoid exposing too much
	inner                    *inner_zoekt.Indexer
	indexDir                 string
}

func NewIndexer(indexDir string) *Indexer {
	idxer := inner_zoekt.NewIndexer(indexDir, repoIndexerLatestVersion)
	return &Indexer{
		Indexer:  idxer,
		inner:    idxer,
		indexDir: indexDir,
	}
}

func newZoektIndexBuilder(indexDir string, repo *repo_model.Repository, targetSHA string) (*index.Builder, error) {
	opts := index.Options{
		IndexDir: indexDir,
		SizeMax:  int(setting.Indexer.MaxIndexerFileSize),
		IsDelta:  true,
		RepositoryDescription: zoekt.Repository{
			ID:   uint32(repo.ID),
			Name: strconv.FormatInt(repo.ID, 10),
			Branches: []zoekt.RepositoryBranch{
				{
					Name:    "HEAD",
					Version: targetSHA,
				},
			},
		},
	}

	if opts.IncrementalSkipIndexing() {
		return nil, nil
	}

	opts.SetDefaults()

	builder, err := index.NewBuilder(opts)
	if err != nil {
		return nil, fmt.Errorf("index.newZoektIndexBuilder: %w", err)
	}

	return builder, nil
}

func (b *Indexer) addDelete(builder *index.Builder, filename string) {
	builder.MarkFileAsChangedOrRemoved(filename)
}

func (b *Indexer) addUpdate(ctx context.Context, builder *index.Builder, batchWriter git.WriteCloserError, batchReader *bufio.Reader, update internal.FileUpdate, repo *repo_model.Repository) error {
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
		b.addDelete(builder, update.Filename)
		return nil
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
	} else if !typesniffer.DetectContentType(fileContents, update.Filename).IsText() {
		// FIXME: UTF-16 files will probably fail here
		return nil
	}

	if _, err = batchReader.Discard(1); err != nil {
		return err
	}

	builder.MarkFileAsChangedOrRemoved(update.Filename)

	// branches := []string{repo.DefaultBranch}
	branches := []string{"HEAD"}

	err = builder.Add(
		index.Document{
			Name:     update.Filename,
			Content:  charset.ToUTF8DropErrors(fileContents, charset.ConvertOpts{}),
			Branches: branches,
			Language: detectLanguage(update.Filename, fileContents),
		})
	if err != nil {
		return fmt.Errorf("error adding document with name %s: %w", update.Filename, err)
	}

	return nil
}

func detectLanguage(filename string, content []byte) string {
	lang := enry.GetLanguage(filename, content)
	if lang == "" {
		lang = "Plain Text"
	}
	return lang
}

// Index will save the index data
func (b *Indexer) Index(ctx context.Context, repo *repo_model.Repository, sha string, changes *internal.RepoChanges) error {
	builder, err := newZoektIndexBuilder(b.indexDir, repo, sha)
	if err != nil {
		return fmt.Errorf("error creating builder: %w", err)
	}

	if builder == nil {
		// skip indexing when there is no change
		return nil
	}

	if len(changes.Updates) > 0 {
		r, err := gitrepo.OpenRepository(ctx, repo)
		if err != nil {
			return err
		}
		defer r.Close()
		batch, err := r.NewBatch(ctx)
		if err != nil {
			return err
		}
		defer batch.Close()
		for _, update := range changes.Updates {
			err := b.addUpdate(ctx, builder, batch.Writer, batch.Reader, update, repo)
			if err != nil {
				return err
			}
		}
	}

	for _, filename := range changes.RemovedFilenames {
		b.addDelete(builder, filename)
	}

	return builder.Finish()
}

// Delete entries by repoId
func (b *Indexer) Delete(ctx context.Context, repoID int64) error {
	repoPathPrefix := strconv.FormatInt(repoID, 10)

	// remove all {repoId}_v{N}.{X}.zoekt or {repoId}_v{N}.{X}.zoekt.meta where X is %05d formatted int in b.indexDir
	pattern := repoPathPrefix + "_v*.[0-9][0-9][0-9][0-9][0-9].zoekt*"
	matches, err := filepath.Glob(filepath.Join(b.indexDir, pattern))
	if err != nil {
		return fmt.Errorf("finding files to delete: %w", err)
	}

	for _, filePath := range matches {
		if err := os.Remove(filePath); err != nil {
			log.Error("failed to delete %s: %v", filePath, err)
		}
	}

	tmpPattern := repoPathPrefix + "_v*.tmp"
	tmpMatches, err := filepath.Glob(filepath.Join(b.indexDir, tmpPattern))
	if err != nil {
		return fmt.Errorf("finding temp files to delete: %w", err)
	}

	for _, filePath := range tmpMatches {
		if err := os.Remove(filePath); err != nil {
			log.Error("failed to delete temp file %s: %v", filePath, err)
		}
	}

	return nil
}

func TransToZoektContentQueryString(s string) string {
	return fmt.Sprintf("content:\"%s\"", s)
}

// generateZoektQuery creates a Zoekt query object based on search options
func (b *Indexer) generateZoektQuery(_ context.Context, opts *internal.SearchOptions) (query.Q, error) {
	keyword := opts.Keyword
	var contentQuery query.Q
	var err error

	// Zoekt does not support true fuzzy search.
	// CodeSearchModeFuzzy is therefore treated as a union (OR) search
	// to preserve previous behavior.
	switch opts.Mode {
	case internal.CodeSearchModeUnion, internal.CodeSearchModeFuzzy:
		fields := strings.Fields(keyword)
		if len(fields) == 0 {
			return nil, errors.New("empty keyword")
		}
		contentQuery, err = query.Parse(
			TransToZoektContentQueryString(QuoteMeta(fields[0])),
		)
		if err != nil {
			return nil, err
		}
		for _, f := range fields[1:] {
			q, err := query.Parse(
				TransToZoektContentQueryString(QuoteMeta(f)),
			)
			if err != nil {
				return nil, err
			}
			contentQuery = query.NewOr(contentQuery, q)
		}
	default:
		// Exact match
		contentQuery, err = query.Parse(
			TransToZoektContentQueryString(QuoteMeta(keyword)),
		)
		if err != nil {
			return nil, err
		}
	}

	finalQuery := contentQuery

	if len(opts.RepoIDs) > 0 {
		repoIDs := make([]uint32, len(opts.RepoIDs))
		for i, id := range opts.RepoIDs {
			repoIDs[i] = uint32(id)
		}
		finalQuery = query.NewAnd(finalQuery, query.NewRepoIDs(repoIDs...))
	}

	if opts.Filename != "" {
		prefix := strings.TrimPrefix(opts.Filename, "/")

		re, err := syntax.Parse(
			"^"+regexp.QuoteMeta(prefix),
			syntax.Perl,
		)
		if err != nil {
			return nil, err
		}

		fileQuery := &query.Regexp{
			Regexp:   re,
			FileName: true,
			Content:  false,
		}

		finalQuery = query.NewAnd(finalQuery, fileQuery)
	}

	return finalQuery, nil
}

// excludeVendored filters out search results whose filenames indicate they are vendored dependencies.
func excludeVendored(results []*internal.SearchResult) []*internal.SearchResult {
	filtered := make([]*internal.SearchResult, 0, len(results))
	for _, res := range results {
		if !analyze.IsVendor(res.Filename) {
			filtered = append(filtered, res)
		}
	}
	return filtered
}

// paginateResults returns a slice of results starting from `skip` index up to `take` number of items.
func paginateResults[T any](results []T, skip, take int) []T {
	if skip >= len(results) {
		return nil
	}
	end := min(skip+take, len(results))
	return results[skip:end]
}

func getSearchResultLanguages(searchResult *zoekt.SearchResult) []*internal.SearchResultLanguages {
	languages := make(map[string]int)

	for _, file := range searchResult.Files {
		lang := file.Language
		if lang == "" {
			lang = "Plain Text"
		}
		languages[lang]++
	}

	searchResultLanguages := make([]*internal.SearchResultLanguages, 0, len(languages))

	for lang, count := range languages {
		searchResultLanguages = append(searchResultLanguages, &internal.SearchResultLanguages{
			Language: lang,
			Count:    count,
			Color:    enry.GetColor(lang),
		})
	}

	slices.SortFunc(searchResultLanguages, func(a, b *internal.SearchResultLanguages) int {
		if a.Count != b.Count {
			return cmp.Compare(b.Count, a.Count)
		}
		return cmp.Compare(a.Language, b.Language)
	})

	return searchResultLanguages
}

func convertZoektResult(files []zoekt.FileMatch) []*internal.SearchResult {
	results := make([]*internal.SearchResult, 0, len(files))

	for _, f := range files {
		content := string(f.Content)
		lines := strings.Split(content, "\n")

		var (
			contentLines []string
			lineNumbers  []int
			lineOffsets  []int
			matches      []internal.Match
		)

		offset := 0

		for lineIdx, line := range lines {
			lineNum := lineIdx + 1

			contentLines = append(contentLines, line)
			lineNumbers = append(lineNumbers, lineNum)
			lineOffsets = append(lineOffsets, offset)

			for _, lm := range f.LineMatches {
				if lm.LineNumber != lineNum {
					continue
				}
				for _, frag := range lm.LineFragments {
					start := int(frag.Offset)
					end := start + frag.MatchLength

					matches = append(matches, internal.Match{
						Start:      start,
						End:        end,
						LineNumber: lineNum,
					})
				}
			}

			offset += len(line) + 1
		}

		if len(matches) == 0 {
			continue
		}

		lang := f.Language
		if lang == "" {
			lang = "Plain Text"
		}

		results = append(results, &internal.SearchResult{
			RepoID:      int64(f.RepositoryID),
			Filename:    f.FileName,
			CommitID:    f.Version,
			Content:     strings.Join(contentLines, "\n"),
			Language:    lang,
			Color:       enry.GetColor(lang),
			Matches:     matches,
			LineNumbers: lineNumbers,
			LineOffsets: lineOffsets,
		})
	}

	return results
}

func (b *Indexer) Search(ctx context.Context, opts *internal.SearchOptions) (int64, []*internal.SearchResult, []*internal.SearchResultLanguages, error) {
	q, err := b.generateZoektQuery(ctx, opts)
	if err != nil {
		return 0, nil, nil, err
	}

	result, err := b.inner.Searcher.Search(ctx, q, &zoekt.SearchOptions{
		Whole: true,
	})
	if err != nil {
		return 0, nil, nil, err
	}

	allHits := convertZoektResult(result.Files)

	searchResultsLanguages := getSearchResultLanguages(result)

	if opts.Language != "" {
		allHits = slices.DeleteFunc(allHits, func(r *internal.SearchResult) bool {
			return r.Language != opts.Language
		})
	}

	if setting.Indexer.ExcludeVendored {
		allHits = excludeVendored(allHits)
	}

	skip, take := opts.GetSkipTake()
	pagedHits := paginateResults(allHits, skip, take)

	total := int64(len(allHits))

	return total, pagedHits, searchResultsLanguages, nil
}
