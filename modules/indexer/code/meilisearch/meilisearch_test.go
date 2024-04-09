package meilisearch

import (
	"testing"

	"code.gitea.io/gitea/modules/indexer/code/internal"
	inner_meilisearch "code.gitea.io/gitea/modules/indexer/internal/meilisearch"
	"code.gitea.io/gitea/modules/timeutil"

	"github.com/meilisearch/meilisearch-go"
	"github.com/stretchr/testify/assert"
)

func TestConvertHits(t *testing.T) {
	validResponse := &meilisearch.SearchResponse{
		Hits: []any{
			map[string]any{
				"id":         "3e_Z2V0aXQ",
				"commit_id":  "91617c30caf5ebd9f5219397001b8ec1ea62e5b2",
				"content":    "License text",
				"language":   "Markdown",
				"updated_at": float64(123311),
				"_matchesPosition": map[string]any{
					"content": []any{
						map[string]any{
							"start":  float64(23),
							"length": float64(18),
						},
					},
				},
			},
			map[string]any{
				"id":         "3e_UkVBRE1F",
				"commit_id":  "91617c30caf5ebd9f5219397001b8ec1ea62e5b2",
				"content":    "This is the README",
				"language":   "Markdown",
				"updated_at": float64(123321),
				"_matchesPosition": map[string]any{
					"content": []any{
						map[string]any{
							"start":  float64(3),
							"length": float64(3),
						},
						map[string]any{
							"start":  float64(5),
							"length": float64(2),
						},
					},
				},
			},
			map[string]any{
				"id":         "3e_bWFpbi5ycw",
				"commit_id":  "91617c30caf5ebd9f5219397001b8ec1ea62e5b2",
				"content":    `fn main() { println!("Hello World!");}`,
				"language":   "Rust",
				"updated_at": float64(123321),
				"_matchesPosition": map[string]any{
					"content": []any{
						map[string]any{
							"start":  float64(9),
							"length": float64(1),
						},
						map[string]any{
							"start":  float64(5),
							"length": float64(5),
						},
					},
				},
			},
		},
	}
	hits, err := convertHits(validResponse)
	assert.NoError(t, err)
	assert.EqualValues(t, []*internal.SearchResult{
		{
			RepoID:      122,
			StartIndex:  23,
			EndIndex:    41,
			Filename:    "getit",
			Content:     "License text",
			CommitID:    "91617c30caf5ebd9f5219397001b8ec1ea62e5b2",
			UpdatedUnix: timeutil.TimeStamp(123311),
			Language:    "Markdown",
			Color:       "#083fa1",
		},
		{
			RepoID:      122,
			StartIndex:  3,
			EndIndex:    6,
			Filename:    "README",
			Content:     "This is the README",
			CommitID:    "91617c30caf5ebd9f5219397001b8ec1ea62e5b2",
			UpdatedUnix: timeutil.TimeStamp(123321),
			Language:    "Markdown",
			Color:       "#083fa1",
		},
		{
			RepoID:      122,
			StartIndex:  9,
			EndIndex:    10,
			Filename:    "main.rs",
			Content:     `fn main() { println!("Hello World!");}`,
			CommitID:    "91617c30caf5ebd9f5219397001b8ec1ea62e5b2",
			UpdatedUnix: timeutil.TimeStamp(123321),
			Language:    "Rust",
			Color:       "#dea584",
		},
	}, hits)

	hits, err = convertHits(&meilisearch.SearchResponse{Hits: []any{map[string]any{}}})
	assert.Nil(t, hits)
	assert.ErrorIs(t, err, inner_meilisearch.ErrMalformedResponse)
}

func TestLanguageResults(t *testing.T) {
	validResponse := &meilisearch.SearchResponse{
		Hits: []any{
			map[string]any{
				"language": "Rust",
			},
			map[string]any{
				"language": "Rust",
			},
			map[string]any{
				"language": "Markdown",
			},
			map[string]any{
				"language": "Go",
			},
			map[string]any{
				"language": "",
			},
		},
	}

	languageResults := languageResults(validResponse)
	assert.EqualValues(t, []*internal.SearchResultLanguages{
		{Language: "Rust", Count: 2, Color: "#dea584"},
		{Language: "Go", Count: 1, Color: "#00ADD8"},
		{Language: "Markdown", Count: 1, Color: "#083fa1"},
	}, languageResults)
}
