// Copyright 2020 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package code

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/modules/analyze"
	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/charset"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/timeutil"

	"github.com/go-enry/go-enry/v2"
	"github.com/olivere/elastic/v7"
)

const (
	esRepoIndexerLatestVersion = 1
)

var (
	_ Indexer = &ElasticSearchIndexer{}
)

// ElasticSearchIndexer implements Indexer interface
type ElasticSearchIndexer struct {
	client           *elastic.Client
	indexerAliasName string
}

type elasticLogger struct {
	*log.Logger
}

func (l elasticLogger) Printf(format string, args ...interface{}) {
	_ = l.Logger.Log(2, l.Logger.GetLevel(), format, args...)
}

// NewElasticSearchIndexer creates a new elasticsearch indexer
func NewElasticSearchIndexer(url, indexerName string) (*ElasticSearchIndexer, bool, error) {
	opts := []elastic.ClientOptionFunc{
		elastic.SetURL(url),
		elastic.SetSniff(false),
		elastic.SetHealthcheckInterval(10 * time.Second),
		elastic.SetGzip(false),
	}

	logger := elasticLogger{log.GetLogger(log.DEFAULT)}

	if logger.GetLevel() == log.TRACE || logger.GetLevel() == log.DEBUG {
		opts = append(opts, elastic.SetTraceLog(logger))
	} else if logger.GetLevel() == log.ERROR || logger.GetLevel() == log.CRITICAL || logger.GetLevel() == log.FATAL {
		opts = append(opts, elastic.SetErrorLog(logger))
	} else if logger.GetLevel() == log.INFO || logger.GetLevel() == log.WARN {
		opts = append(opts, elastic.SetInfoLog(logger))
	}

	client, err := elastic.NewClient(opts...)
	if err != nil {
		return nil, false, err
	}

	indexer := &ElasticSearchIndexer{
		client:           client,
		indexerAliasName: indexerName,
	}
	exists, err := indexer.init()

	return indexer, exists, err
}

const (
	defaultMapping = `{
		"mappings": {
			"properties": {
				"repo_id": {
					"type": "long",
					"index": true
				},
				"content": {
					"type": "text",
					"index": true
				},
				"commit_id": {
					"type": "keyword",
					"index": true
				},
				"language": {
					"type": "keyword",
					"index": true
				},
				"updated_at": {
					"type": "long",
					"index": true
				}
			}
		}
	}`
)

func (b *ElasticSearchIndexer) realIndexerName() string {
	return fmt.Sprintf("%s_%d", b.indexerAliasName, esRepoIndexerLatestVersion)
}

// Init will initialize the indexer
func (b *ElasticSearchIndexer) init() (bool, error) {
	ctx := context.Background()
	exists, err := b.client.IndexExists(b.realIndexerName()).Do(ctx)
	if err != nil {
		return false, err
	}
	if !exists {
		var mapping = defaultMapping

		createIndex, err := b.client.CreateIndex(b.realIndexerName()).BodyString(mapping).Do(ctx)
		if err != nil {
			return false, err
		}
		if !createIndex.Acknowledged {
			return false, fmt.Errorf("create index %s with %s failed", b.realIndexerName(), mapping)
		}
	}

	// check version
	r, err := b.client.Aliases().Do(ctx)
	if err != nil {
		return false, err
	}

	realIndexerNames := r.IndicesByAlias(b.indexerAliasName)
	if len(realIndexerNames) < 1 {
		res, err := b.client.Alias().
			Add(b.realIndexerName(), b.indexerAliasName).
			Do(ctx)
		if err != nil {
			return false, err
		}
		if !res.Acknowledged {
			return false, fmt.Errorf("")
		}
	} else if len(realIndexerNames) >= 1 && realIndexerNames[0] < b.realIndexerName() {
		log.Warn("Found older gitea indexer named %s, but we will create a new one %s and keep the old NOT DELETED. You can delete the old version after the upgrade succeed.",
			realIndexerNames[0], b.realIndexerName())
		res, err := b.client.Alias().
			Remove(realIndexerNames[0], b.indexerAliasName).
			Add(b.realIndexerName(), b.indexerAliasName).
			Do(ctx)
		if err != nil {
			return false, err
		}
		if !res.Acknowledged {
			return false, fmt.Errorf("")
		}
	}

	return exists, nil
}

func (b *ElasticSearchIndexer) addUpdate(sha string, update fileUpdate, repo *models.Repository) ([]elastic.BulkableRequest, error) {
	stdout, err := git.NewCommand("cat-file", "-s", update.BlobSha).
		RunInDir(repo.RepoPath())
	if err != nil {
		return nil, err
	}
	if size, err := strconv.Atoi(strings.TrimSpace(stdout)); err != nil {
		return nil, fmt.Errorf("Misformatted git cat-file output: %v", err)
	} else if int64(size) > setting.Indexer.MaxIndexerFileSize {
		return []elastic.BulkableRequest{b.addDelete(update.Filename, repo)}, nil
	}

	fileContents, err := git.NewCommand("cat-file", "blob", update.BlobSha).
		RunInDirBytes(repo.RepoPath())
	if err != nil {
		return nil, err
	} else if !base.IsTextFile(fileContents) {
		// FIXME: UTF-16 files will probably fail here
		return nil, nil
	}

	id := filenameIndexerID(repo.ID, update.Filename)

	return []elastic.BulkableRequest{
		elastic.NewBulkIndexRequest().
			Index(b.indexerAliasName).
			Id(id).
			Doc(map[string]interface{}{
				"repo_id":    repo.ID,
				"content":    string(charset.ToUTF8DropErrors(fileContents)),
				"commit_id":  sha,
				"language":   analyze.GetCodeLanguage(update.Filename, fileContents),
				"updated_at": timeutil.TimeStampNow(),
			}),
	}, nil
}

func (b *ElasticSearchIndexer) addDelete(filename string, repo *models.Repository) elastic.BulkableRequest {
	id := filenameIndexerID(repo.ID, filename)
	return elastic.NewBulkDeleteRequest().
		Index(b.indexerAliasName).
		Id(id)
}

// Index will save the index data
func (b *ElasticSearchIndexer) Index(repo *models.Repository, sha string, changes *repoChanges) error {
	reqs := make([]elastic.BulkableRequest, 0)
	for _, update := range changes.Updates {
		updateReqs, err := b.addUpdate(sha, update, repo)
		if err != nil {
			return err
		}
		if len(updateReqs) > 0 {
			reqs = append(reqs, updateReqs...)
		}
	}

	for _, filename := range changes.RemovedFilenames {
		reqs = append(reqs, b.addDelete(filename, repo))
	}

	if len(reqs) > 0 {
		_, err := b.client.Bulk().
			Index(b.indexerAliasName).
			Add(reqs...).
			Do(context.Background())
		return err
	}
	return nil
}

// Delete deletes indexes by ids
func (b *ElasticSearchIndexer) Delete(repoID int64) error {
	_, err := b.client.DeleteByQuery(b.indexerAliasName).
		Query(elastic.NewTermsQuery("repo_id", repoID)).
		Do(context.Background())
	return err
}

func convertResult(searchResult *elastic.SearchResult, kw string, pageSize int) (int64, []*SearchResult, []*SearchResultLanguages, error) {
	hits := make([]*SearchResult, 0, pageSize)
	for _, hit := range searchResult.Hits.Hits {
		// FIXME: There is no way to get the position the keyword on the content currently on the same request.
		// So we get it from content, this may made the query slower. See
		// https://discuss.elastic.co/t/fetching-position-of-keyword-in-matched-document/94291
		var startIndex, endIndex int = -1, -1
		c, ok := hit.Highlight["content"]
		if ok && len(c) > 0 {
			var subStr = make([]rune, 0, len(kw))
			startIndex = strings.IndexFunc(c[0], func(r rune) bool {
				if len(subStr) >= len(kw) {
					subStr = subStr[1:]
				}
				subStr = append(subStr, r)
				return strings.EqualFold(kw, string(subStr))
			})
			if startIndex > -1 {
				endIndex = startIndex + len(kw)
			} else {
				panic(fmt.Sprintf("1===%#v", hit.Highlight))
			}
		} else {
			panic(fmt.Sprintf("2===%#v", hit.Highlight))
		}

		repoID, fileName := parseIndexerID(hit.Id)
		var res = make(map[string]interface{})
		if err := json.Unmarshal(hit.Source, &res); err != nil {
			return 0, nil, nil, err
		}

		language := res["language"].(string)

		hits = append(hits, &SearchResult{
			RepoID:      repoID,
			Filename:    fileName,
			CommitID:    res["commit_id"].(string),
			Content:     res["content"].(string),
			UpdatedUnix: timeutil.TimeStamp(res["updated_at"].(float64)),
			Language:    language,
			StartIndex:  startIndex,
			EndIndex:    endIndex,
			Color:       enry.GetColor(language),
		})
	}

	return searchResult.TotalHits(), hits, extractAggs(searchResult), nil
}

func extractAggs(searchResult *elastic.SearchResult) []*SearchResultLanguages {
	var searchResultLanguages []*SearchResultLanguages
	agg, found := searchResult.Aggregations.Terms("language")
	if found {
		searchResultLanguages = make([]*SearchResultLanguages, 0, 10)

		for _, bucket := range agg.Buckets {
			searchResultLanguages = append(searchResultLanguages, &SearchResultLanguages{
				Language: bucket.Key.(string),
				Color:    enry.GetColor(bucket.Key.(string)),
				Count:    int(bucket.DocCount),
			})
		}
	}
	return searchResultLanguages
}

// Search searches for codes and language stats by given conditions.
func (b *ElasticSearchIndexer) Search(repoIDs []int64, language, keyword string, page, pageSize int) (int64, []*SearchResult, []*SearchResultLanguages, error) {
	kwQuery := elastic.NewMultiMatchQuery(keyword, "content")
	query := elastic.NewBoolQuery()
	query = query.Must(kwQuery)
	if len(repoIDs) > 0 {
		var repoStrs = make([]interface{}, 0, len(repoIDs))
		for _, repoID := range repoIDs {
			repoStrs = append(repoStrs, repoID)
		}
		repoQuery := elastic.NewTermsQuery("repo_id", repoStrs...)
		query = query.Must(repoQuery)
	}

	var (
		start       int
		kw          = "<em>" + keyword + "</em>"
		aggregation = elastic.NewTermsAggregation().Field("language").Size(10).OrderByCountDesc()
	)

	if page > 0 {
		start = (page - 1) * pageSize
	}

	if len(language) == 0 {
		searchResult, err := b.client.Search().
			Index(b.indexerAliasName).
			Aggregation("language", aggregation).
			Query(query).
			Highlight(elastic.NewHighlight().Field("content")).
			Sort("repo_id", true).
			From(start).Size(pageSize).
			Do(context.Background())
		if err != nil {
			return 0, nil, nil, err
		}

		return convertResult(searchResult, kw, pageSize)
	}

	langQuery := elastic.NewMatchQuery("language", language)
	countResult, err := b.client.Search().
		Index(b.indexerAliasName).
		Aggregation("language", aggregation).
		Query(query).
		Size(0). // We only needs stats information
		Do(context.Background())
	if err != nil {
		return 0, nil, nil, err
	}

	query = query.Must(langQuery)
	searchResult, err := b.client.Search().
		Index(b.indexerAliasName).
		Query(query).
		Highlight(elastic.NewHighlight().Field("content")).
		Sort("repo_id", true).
		From(start).Size(pageSize).
		Do(context.Background())
	if err != nil {
		return 0, nil, nil, err
	}

	total, hits, _, err := convertResult(searchResult, kw, pageSize)

	return total, hits, extractAggs(countResult), err
}

// Close implements indexer
func (b *ElasticSearchIndexer) Close() {}
