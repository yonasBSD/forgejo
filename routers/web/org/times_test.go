// Copyright 2014 The Gogs Authors. All rights reserved.
// Copyright 2022 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package org

import (
	"path/filepath"
	"testing"

	"code.gitea.io/gitea/models/unittest"
	"code.gitea.io/gitea/modules/setting"
	"github.com/stretchr/testify/assert"
)

func init() {
	setting.SetCustomPathAndConf("", "", "")
	setting.LoadForTest()
}

// TestMain sets up the test DB.
func TestMain(m *testing.M) {
	unittest.MainTest(m, &unittest.TestOptions{
		GiteaRootPath: filepath.Join("..", "..", ".."),
		FixtureFiles: []string{
			"user.yml",
			"org_user.yml",
			"repository.yml",
			"issue.yml",
			"milestone.yml",
			"tracked_time.yml",
		},
	})
}

// TestTimesPrepareDB prepares the database for the following tests.
func TestTimesPrepareDB(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
}

// TestTimesByRepos tests TimesByRepos functionality
func TestTimesByRepos(t *testing.T) {
	kases := []struct {
		name     string
		unixfrom int64
		unixto   int64
		orgid    int64
		expected []resultTimesByRepos
	}{
		{
			name:     "Full sum for org 1",
			unixfrom: 0,
			unixto:   9223372036854775807,
			orgid:    1,
			expected: []resultTimesByRepos(nil),
		},
		{
			name:     "Full sum for org 2",
			unixfrom: 0,
			unixto:   9223372036854775807,
			orgid:    2,
			expected: []resultTimesByRepos{
				{
					Name:    "repo1",
					SumTime: 4083,
				},
				{
					Name:    "repo2",
					SumTime: 75,
				},
			},
		},
		{
			name:     "Simple time bound",
			unixfrom: 946684801,
			unixto:   946684802,
			orgid:    2,
			expected: []resultTimesByRepos{
				{
					Name:    "repo1",
					SumTime: 3662,
				},
			},
		},
		{
			name:     "Both times inclusive",
			unixfrom: 946684801,
			unixto:   946684801,
			orgid:    2,
			expected: []resultTimesByRepos{
				{
					Name:    "repo1",
					SumTime: 3661,
				},
			},
		},
		{
			name:     "Should ignore deleted",
			unixfrom: 947688814,
			unixto:   947688815,
			orgid:    2,
			expected: []resultTimesByRepos{
				{
					Name:    "repo2",
					SumTime: 71,
				},
			},
		},
	}

	// Run test kases
	for _, kase := range kases {
		t.Run(kase.name, func(t *testing.T) {
			results, err := getTimesByRepos(kase.unixfrom, kase.unixto, kase.orgid)
			assert.NoError(t, err)
			assert.Equal(t, kase.expected, results)
		})
	}
}

// TestTimesByMilestones tests TimesByMilestones functionality
func TestTimesByMilestones(t *testing.T) {
	kases := []struct {
		name     string
		unixfrom int64
		unixto   int64
		orgid    int64
		expected []resultTimesByMilestones
	}{
		{
			name:     "Full sum for org 1",
			unixfrom: 0,
			unixto:   9223372036854775807,
			orgid:    1,
			expected: []resultTimesByMilestones(nil),
		},
		{
			name:     "Full sum for org 2",
			unixfrom: 0,
			unixto:   9223372036854775807,
			orgid:    2,
			expected: []resultTimesByMilestones{
				{
					RepoName:     "repo1",
					Name:         "",
					ID:           "",
					SumTime:      401,
					HideRepoName: false,
				},
				{
					RepoName:     "repo1",
					Name:         "milestone1",
					ID:           "1",
					SumTime:      3682,
					HideRepoName: true,
				},
				{
					RepoName:     "repo2",
					Name:         "",
					ID:           "",
					SumTime:      75,
					HideRepoName: false,
				},
			},
		},
		{
			name:     "Simple time bound",
			unixfrom: 946684801,
			unixto:   946684802,
			orgid:    2,
			expected: []resultTimesByMilestones{
				{
					RepoName:     "repo1",
					Name:         "milestone1",
					ID:           "1",
					SumTime:      3662,
					HideRepoName: false,
				},
			},
		},
		{
			name:     "Both times inclusive",
			unixfrom: 946684801,
			unixto:   946684801,
			orgid:    2,
			expected: []resultTimesByMilestones{
				{
					RepoName:     "repo1",
					Name:         "milestone1",
					ID:           "1",
					SumTime:      3661,
					HideRepoName: false,
				},
			},
		},
		{
			name:     "Should ignore deleted",
			unixfrom: 947688814,
			unixto:   947688815,
			orgid:    2,
			expected: []resultTimesByMilestones{
				{
					RepoName:     "repo2",
					Name:         "",
					ID:           "",
					SumTime:      71,
					HideRepoName: false,
				},
			},
		},
	}

	// Run test kases
	for _, kase := range kases {
		t.Run(kase.name, func(t *testing.T) {
			results, err := getTimesByMilestones(kase.unixfrom, kase.unixto, kase.orgid)
			assert.NoError(t, err)
			assert.Equal(t, kase.expected, results)
		})
	}
}

// TestTimesByMembers tests TimesByMembers functionality
func TestTimesByMembers(t *testing.T) {
	kases := []struct {
		name     string
		unixfrom int64
		unixto   int64
		orgid    int64
		expected []resultTimesByMembers
	}{
		{
			name:     "Full sum for org 1",
			unixfrom: 0,
			unixto:   9223372036854775807,
			orgid:    1,
			expected: []resultTimesByMembers(nil),
		},
		{
			// Test case: Sum of times forever in org no. 2
			name:     "Full sum for org 2",
			unixfrom: 0,
			unixto:   9223372036854775807,
			orgid:    2,
			expected: []resultTimesByMembers{
				{
					Name:    "user2",
					SumTime: 3666,
				},
				{
					Name:    "user1",
					SumTime: 491,
				},
			},
		},
		{
			name:     "Simple time bound",
			unixfrom: 946684801,
			unixto:   946684802,
			orgid:    2,
			expected: []resultTimesByMembers{
				{
					Name:    "user2",
					SumTime: 3662,
				},
			},
		},
		{
			name:     "Both times inclusive",
			unixfrom: 946684801,
			unixto:   946684801,
			orgid:    2,
			expected: []resultTimesByMembers{
				{
					Name:    "user2",
					SumTime: 3661,
				},
			},
		},
		{
			name:     "Should ignore deleted",
			unixfrom: 947688814,
			unixto:   947688815,
			orgid:    2,
			expected: []resultTimesByMembers{
				{
					Name:    "user1",
					SumTime: 71,
				},
			},
		},
	}

	// Run test kases
	for _, kase := range kases {
		t.Run(kase.name, func(t *testing.T) {
			results, err := getTimesByMembers(kase.unixfrom, kase.unixto, kase.orgid)
			assert.NoError(t, err)
			assert.Equal(t, kase.expected, results)
		})
	}
}
