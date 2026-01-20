// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package code

import (
	"bytes"
	"context"
	"html/template"
	"math"
	"slices"
	"sort"
	"strings"

	"forgejo.org/modules/highlight"
	"forgejo.org/modules/indexer/code/internal"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/timeutil"
	"forgejo.org/services/gitdiff"
)

// Result a search result to display
type Result struct {
	RepoID      int64
	Filename    string
	CommitID    string
	UpdatedUnix timeutil.TimeStamp
	Language    string
	Color       string
	Lines       []ResultLine
}

type ResultLine struct {
	Num              int
	FormattedContent template.HTML
}

type SearchResultLanguages = internal.SearchResultLanguages

type SearchOptions = internal.SearchOptions

// llu:TrKeysSuffix search.
var CodeSearchOptions = []string{"exact", "union", "fuzzy"}

type SearchMode = internal.CodeSearchMode

const (
	SearchModeExact = internal.CodeSearchModeExact
	SearchModeUnion = internal.CodeSearchModeUnion
	SearchModeFuzzy = internal.CodeSearchModeFuzzy
)

type Results []*Result

// Get the set of repo IDs from a list of search results
func (res Results) RepoIDs() []int64 {
	ids := make([]int64, len(res))
	for _, r := range res {
		if !slices.Contains(ids, r.RepoID) {
			ids = append(ids, r.RepoID)
		}
	}
	return ids
}

func indices(content string, selectionStartIndex, selectionEndIndex int) (int, int) {
	startIndex := selectionStartIndex
	numLinesBefore := 0
	for ; startIndex > 0; startIndex-- {
		if content[startIndex-1] == '\n' {
			if numLinesBefore == 1 {
				break
			}
			numLinesBefore++
		}
	}

	endIndex := selectionEndIndex
	numLinesAfter := 0
	for ; endIndex < len(content); endIndex++ {
		if content[endIndex] == '\n' {
			if numLinesAfter == 1 {
				break
			}
			numLinesAfter++
		}
	}

	return startIndex, endIndex
}

func writeStrings(buf *bytes.Buffer, strs ...string) error {
	for _, s := range strs {
		_, err := buf.WriteString(s)
		if err != nil {
			return err
		}
	}
	return nil
}

const (
	highlightTagStart = "<span class=\"search-highlight\">"
	highlightTagEnd   = "</span>"
)

func HighlightSearchResultCode(filename string, lineNums []int, highlightRanges [][3]int, code string) []ResultLine {
	hcd := gitdiff.NewHighlightCodeDiff()
	hcd.CollectUsedRunes(code)
	startTag, endTag := hcd.NextPlaceholder(), hcd.NextPlaceholder()
	hcd.PlaceholderTokenMap[startTag] = highlightTagStart
	hcd.PlaceholderTokenMap[endTag] = highlightTagEnd

	// we should highlight the whole code block first, otherwise it doesn't work well with multiple line highlighting
	hl, _ := highlight.Code(filename, "", code)
	conv := hcd.ConvertToPlaceholders(string(hl))
	convLines := strings.Split(conv, "\n")

	// each highlightRange is of the form [line number, start byte offset, end byte offset]
	for _, highlightRange := range highlightRanges {
		ln, start, end := highlightRange[0], highlightRange[1], highlightRange[2]
		line := convLines[ln]
		if line == "" || len(line) <= start || len(line) < end {
			continue
		}

		sr := strings.NewReader(line)
		sb := strings.Builder{}
		count := -1
		isOpen := false
		for r, size, err := sr.ReadRune(); err == nil; r, size, err = sr.ReadRune() {
			if token, ok := hcd.PlaceholderTokenMap[r];
			// token was not found
			!ok {
				count += size
			} else if
			// token was marked as used
			token == "" ||
				// the token is not an valid html tag emitted by chroma
				!(len(token) > 6 && (token[0:5] == "<span" || token[0:6] == "</span")) {
				count++
			} else if !isOpen {
				// open the tag only after all other placeholders
				sb.WriteRune(r)
				continue
			} else if isOpen && count < end {
				// if the tag is open, but a placeholder exists in between
				// close the tag
				sb.WriteRune(endTag)
				// write the placeholder
				sb.WriteRune(r)
				// reopen the tag
				sb.WriteRune(startTag)
				continue
			}

			switch {
			case count >= end:
				// if tag is not open, no need to close
				if !isOpen {
					break
				}
				sb.WriteRune(endTag)
				isOpen = false
			case count >= start:
				// if tag is open, do not open again
				if isOpen {
					break
				}
				isOpen = true
				sb.WriteRune(startTag)
			}

			sb.WriteRune(r)
		}
		if isOpen {
			sb.WriteRune(endTag)
		}
		convLines[ln] = sb.String()
	}
	conv = strings.Join(convLines, "\n")

	highlightedLines := strings.Split(hcd.Recover(conv), "\n")
	// The lineNums outputted by highlight.Code might not match the original lineNums, because "highlight" removes the last `\n`
	lines := make([]ResultLine, min(len(highlightedLines), len(lineNums)))
	for i := range len(lines) {
		lines[i].Num = lineNums[i]
		lines[i].FormattedContent = template.HTML(highlightedLines[i])
	}
	return lines
}

func searchResult(result *internal.SearchResult, startIndex, endIndex int) (*Result, error) {
	if setting.Indexer.RepoType == "zoekt" {
		return searchZoektResult(result)
	}
	return searchResultCommon(result, startIndex, endIndex)
}

func searchZoektResult(result *internal.SearchResult) (*Result, error) {
	// Sort matches by starting position
	sort.Slice(result.Matches, func(i, j int) bool {
		return result.Matches[i].Start < result.Matches[j].Start
	})

	// Split all the content into lines (according to the order)
	contentLines := strings.Split(result.Content, "\n")

	// Create map: string num -> string text
	lineMap := make(map[int]string, len(result.LineNumbers))
	for i, lineNum := range result.LineNumbers {
		if i < len(contentLines) {
			lineMap[lineNum] = contentLines[i]
		}
	}

	// Collect all line numbers to display
	var sortedLines []int
	for _, match := range result.Matches {
		for i := match.LineNumber - 1; i <= match.LineNumber+1; i++ {
			if i > 0 {
				sortedLines = append(sortedLines, i)
			}
		}
	}

	// Sort line numbers in ascending order
	sort.Ints(sortedLines)

	// Remove duplicate line numbers after sorting
	uniqueLines := sortedLines[:0]
	for i, line := range sortedLines {
		if i == 0 || line != sortedLines[i-1] {
			uniqueLines = append(uniqueLines, line)
		}
	}
	sortedLines = uniqueLines

	// Group lines into blocks (block break if distance > 2 lines)
	var blocks [][]int
	var currentBlock []int
	for i, line := range sortedLines {
		if i > 0 && line > sortedLines[i-1]+2 {
			if len(currentBlock) > 0 {
				blocks = append(blocks, currentBlock)
				currentBlock = nil
			}
		}
		currentBlock = append(currentBlock, line)
	}
	if len(currentBlock) > 0 {
		blocks = append(blocks, currentBlock)
	}

	var resultLines []ResultLine
	for _, block := range blocks {
		// Forming a text block from lines lineMap
		var blockLines []string
		for _, lineNum := range block {
			if lineText, ok := lineMap[lineNum]; ok {
				blockLines = append(blockLines, lineText)
			} else {
				// If there is no line, insert a blank one to preserve the numbering
				blockLines = append(blockLines, "")
			}
		}

		// Calculate line offsets in a block
		lineOffsets := make([]int, len(blockLines)+1)
		for i := 1; i <= len(blockLines); i++ {
			lineOffsets[i] = lineOffsets[i-1] + len(blockLines[i-1]) + 1
		}

		startLine := block[0]
		endLine := block[len(block)-1]

		// Form the entire block text for highlighting
		blockContent := strings.Join(blockLines, "\n")

		highlightByLine := make(map[int][][2]int)
		for _, match := range result.Matches {
			if match.LineNumber < startLine || match.LineNumber > endLine {
				continue
			}

			lineInBlock := -1
			for i, ln := range block {
				if ln == match.LineNumber {
					lineInBlock = i
					break
				}
			}
			if lineInBlock == -1 || lineInBlock >= len(blockLines) {
				continue
			}

			globalLineIdx := -1
			minDelta := math.MaxInt
			for i, num := range result.LineNumbers {
				if num != match.LineNumber {
					continue
				}
				delta := abs(result.LineOffsets[i] - match.Start)
				if delta < minDelta {
					minDelta = delta
					globalLineIdx = i
				}
			}

			if globalLineIdx == -1 {
				continue
			}

			lineOffsetInResult := result.LineOffsets[globalLineIdx]
			highlightStart := match.Start - lineOffsetInResult
			highlightEnd := match.End - lineOffsetInResult

			highlightByLine[lineInBlock] = append(highlightByLine[lineInBlock], [2]int{highlightStart, highlightEnd})
		}

		// merge overlapping ranges
		var highlightRanges [][3]int
		for lineIdx, ranges := range highlightByLine {
			if len(ranges) == 0 {
				continue
			}
			sort.Slice(ranges, func(i, j int) bool {
				return ranges[i][0] < ranges[j][0]
			})
			merged := make([][2]int, 0, len(ranges))
			current := ranges[0]
			for _, r := range ranges[1:] {
				if r[0] <= current[1] {
					if r[1] > current[1] {
						current[1] = r[1]
					}
				} else {
					merged = append(merged, current)
					current = r
				}
			}
			merged = append(merged, current)

			for _, r := range merged {
				highlightRanges = append(highlightRanges, [3]int{lineIdx, r[0], r[1]})
			}
		}

		highlightedLines := HighlightSearchResultCode(
			result.Filename,
			block,
			highlightRanges,
			blockContent,
		)
		resultLines = append(resultLines, highlightedLines...)
	}

	return &Result{
		RepoID:      result.RepoID,
		Filename:    result.Filename,
		CommitID:    result.CommitID,
		UpdatedUnix: result.UpdatedUnix,
		Language:    result.Language,
		Color:       result.Color,
		Lines:       resultLines,
	}, nil
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

func searchResultCommon(result *internal.SearchResult, startIndex, endIndex int) (*Result, error) {
	startLineNum := 1 + strings.Count(result.Content[:startIndex], "\n")

	var formattedLinesBuffer bytes.Buffer

	contentLines := strings.SplitAfter(result.Content[startIndex:endIndex], "\n")
	lineNums := make([]int, 0, len(contentLines))
	index := startIndex
	var highlightRanges [][3]int
	for i, line := range contentLines {
		var err error
		if index < result.EndIndex &&
			result.StartIndex < index+len(line) &&
			result.StartIndex < result.EndIndex {
			openActiveIndex := max(result.StartIndex-index, 0)
			closeActiveIndex := min(result.EndIndex-index, len(line))
			highlightRanges = append(highlightRanges, [3]int{i, openActiveIndex, closeActiveIndex})
			err = writeStrings(&formattedLinesBuffer,
				line[:openActiveIndex],
				line[openActiveIndex:closeActiveIndex],
				line[closeActiveIndex:],
			)
		} else {
			err = writeStrings(&formattedLinesBuffer, line)
		}
		if err != nil {
			return nil, err
		}

		lineNums = append(lineNums, startLineNum+i)
		index += len(line)
	}

	return &Result{
		RepoID:      result.RepoID,
		Filename:    result.Filename,
		CommitID:    result.CommitID,
		UpdatedUnix: result.UpdatedUnix,
		Language:    result.Language,
		Color:       result.Color,
		Lines:       HighlightSearchResultCode(result.Filename, lineNums, highlightRanges, formattedLinesBuffer.String()),
	}, nil
}

// PerformSearch perform a search on a repository
func PerformSearch(ctx context.Context, opts *SearchOptions) (int, Results, []*SearchResultLanguages, error) {
	if opts == nil || len(opts.Keyword) == 0 {
		return 0, nil, nil, nil
	}

	total, results, resultLanguages, err := (*globalIndexer.Load()).Search(ctx, opts)
	if err != nil {
		return 0, nil, nil, err
	}

	displayResults := make([]*Result, len(results))

	for i, result := range results {
		startIndex, endIndex := indices(result.Content, result.StartIndex, result.EndIndex)
		displayResults[i], err = searchResult(result, startIndex, endIndex)
		if err != nil {
			return 0, nil, nil, err
		}
	}
	return int(total), displayResults, resultLanguages, nil
}
