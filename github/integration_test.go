//go:build integration

package github

import (
	"context"
	"os"
	"testing"

	gogithub "github.com/google/go-github/v69/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testRepo = "lonegunmanb/gh2trello"
const testOwner = "lonegunmanb"
const testRepoName = "gh2trello"

// Test data created in the repo:
// #1: Open issue, label:bug,         created 2026-03-16T09:31:37Z
// #2: Open issue, label:enhancement, created 2026-03-16T09:31:45Z
// #3: Closed issue, no label,        created 2026-03-16T09:31:54Z
// #4: Open PR,                       created 2026-03-16T09:32:35Z
// #5: Closed PR,                     created 2026-03-16T09:32:55Z

func newIntegrationClient(t *testing.T) *Client {
	t.Helper()
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		t.Skip("GITHUB_TOKEN not set, skipping integration test")
	}
	return NewClient(token)
}

func searchTitles(issues []*gogithub.Issue, prefix string) []string {
	var titles []string
	for _, i := range issues {
		titles = append(titles, i.GetTitle())
	}
	return titles
}

func issueNumbers(issues []*gogithub.Issue) []int {
	var nums []int
	for _, i := range issues {
		nums = append(nums, i.GetNumber())
	}
	return nums
}

// 7.2.1 Search with is:issue returns only issues
func TestIntegration_SearchIsIssue(t *testing.T) {
	client := newIntegrationClient(t)
	ctx := context.Background()

	query := BuildSearchQuery(testRepo, "", "is:issue [Please ignore: acceptance test]")
	issues, err := client.SearchIssues(ctx, query)
	require.NoError(t, err)

	for _, issue := range issues {
		assert.Nil(t, issue.PullRequestLinks, "is:issue should not return PRs, got #%d", issue.GetNumber())
	}
	// Should find at least issues #1, #2, #3
	nums := issueNumbers(issues)
	assert.Contains(t, nums, 1)
	assert.Contains(t, nums, 2)
	assert.Contains(t, nums, 3)
	assert.NotContains(t, nums, 4, "PR #4 should not appear")
	assert.NotContains(t, nums, 5, "PR #5 should not appear")
}

// 7.2.2 Search with is:pr returns only PRs
func TestIntegration_SearchIsPR(t *testing.T) {
	client := newIntegrationClient(t)
	ctx := context.Background()

	query := BuildSearchQuery(testRepo, "", "is:pr [Please ignore: acceptance test]")
	issues, err := client.SearchIssues(ctx, query)
	require.NoError(t, err)

	for _, issue := range issues {
		assert.NotNil(t, issue.PullRequestLinks, "is:pr should only return PRs, got #%d", issue.GetNumber())
	}
	nums := issueNumbers(issues)
	assert.Contains(t, nums, 4)
	assert.Contains(t, nums, 5)
	assert.NotContains(t, nums, 1, "Issue #1 should not appear")
}

// 7.2.3 Search with label:bug filters correctly
func TestIntegration_SearchLabelBug(t *testing.T) {
	client := newIntegrationClient(t)
	ctx := context.Background()

	query := BuildSearchQuery(testRepo, "", "is:issue label:bug [Please ignore: acceptance test]")
	issues, err := client.SearchIssues(ctx, query)
	require.NoError(t, err)

	require.NotEmpty(t, issues)
	for _, issue := range issues {
		labelNames := make([]string, 0)
		for _, l := range issue.Labels {
			labelNames = append(labelNames, l.GetName())
		}
		assert.Contains(t, labelNames, "bug", "all results should have bug label, issue #%d", issue.GetNumber())
	}
	nums := issueNumbers(issues)
	assert.Contains(t, nums, 1, "Issue #1 has bug label")
	assert.NotContains(t, nums, 2, "Issue #2 has enhancement label, not bug")
}

// 7.2.4 Search with author: filters correctly
func TestIntegration_SearchAuthor(t *testing.T) {
	client := newIntegrationClient(t)
	ctx := context.Background()

	query := BuildSearchQuery(testRepo, "", "is:issue author:lonegunmanb [Please ignore: acceptance test]")
	issues, err := client.SearchIssues(ctx, query)
	require.NoError(t, err)

	require.NotEmpty(t, issues, "should find items authored by lonegunmanb")
	for _, issue := range issues {
		assert.Equal(t, "lonegunmanb", issue.GetUser().GetLogin(),
			"all results should be by lonegunmanb, got issue #%d", issue.GetNumber())
	}
}

// 7.2.5 Search with state:open / state:closed filters correctly
func TestIntegration_SearchStateOpen(t *testing.T) {
	client := newIntegrationClient(t)
	ctx := context.Background()

	query := BuildSearchQuery(testRepo, "", "state:open is:issue [Please ignore: acceptance test]")
	issues, err := client.SearchIssues(ctx, query)
	require.NoError(t, err)

	for _, issue := range issues {
		assert.Equal(t, "open", issue.GetState(), "all results should be open, got #%d", issue.GetNumber())
	}
	nums := issueNumbers(issues)
	assert.Contains(t, nums, 1)
	assert.Contains(t, nums, 2)
	assert.NotContains(t, nums, 3, "Closed issue #3 should not appear")
}

func TestIntegration_SearchStateClosed(t *testing.T) {
	client := newIntegrationClient(t)
	ctx := context.Background()

	query := BuildSearchQuery(testRepo, "", "state:closed is:issue [Please ignore: acceptance test]")
	issues, err := client.SearchIssues(ctx, query)
	require.NoError(t, err)

	for _, issue := range issues {
		assert.Equal(t, "closed", issue.GetState(), "all results should be closed, got #%d", issue.GetNumber())
	}
	nums := issueNumbers(issues)
	assert.Contains(t, nums, 3, "Closed issue #3 should appear")
	assert.NotContains(t, nums, 1, "Open issue #1 should not appear")
}

// 7.2.6 Search with combined filters works
func TestIntegration_SearchCombinedFilters(t *testing.T) {
	client := newIntegrationClient(t)
	ctx := context.Background()

	query := BuildSearchQuery(testRepo, "", "is:issue state:open label:bug [Please ignore: acceptance test]")
	issues, err := client.SearchIssues(ctx, query)
	require.NoError(t, err)

	nums := issueNumbers(issues)
	assert.Contains(t, nums, 1, "Open bug issue #1 should match")
	assert.NotContains(t, nums, 2, "Enhancement issue #2 should not match")
	assert.NotContains(t, nums, 3, "Closed issue #3 should not match")
	assert.NotContains(t, nums, 4, "PR #4 should not match")
}

// 7.2.7 Search with created:>= time filter works
func TestIntegration_SearchCreatedDate(t *testing.T) {
	client := newIntegrationClient(t)
	ctx := context.Background()

	// All test data was created on 2026-03-16.
	// Use created:>=2026-03-16 to match them.
	query := BuildSearchQuery(testRepo, "", "is:issue created:>=2026-03-16 [Please ignore: acceptance test]")
	issues, err := client.SearchIssues(ctx, query)
	require.NoError(t, err)
	require.NotEmpty(t, issues, "should find items created on or after 2026-03-16")

	nums := issueNumbers(issues)
	assert.Contains(t, nums, 1)
	assert.Contains(t, nums, 2)

	// Use a future date to get no results
	query2 := BuildSearchQuery(testRepo, "", "is:issue created:>=2099-01-01 [Please ignore: acceptance test]")
	issues2, err := client.SearchIssues(ctx, query2)
	require.NoError(t, err)
	assert.Empty(t, issues2, "no items should be created in 2099")
}

// 7.2.8 Watermark logic - only items newer than watermark are returned
func TestIntegration_WatermarkLogic(t *testing.T) {
	client := newIntegrationClient(t)
	ctx := context.Background()

	// All test items were created around 09:31-09:32 UTC on 2026-03-16.
	// Use updated:>= with a time before they were created.
	query := BuildSearchQuery(testRepo, "2026-03-16T09:31:00Z", "is:issue [Please ignore: acceptance test]")
	issues, err := client.SearchIssues(ctx, query)
	require.NoError(t, err)
	require.NotEmpty(t, issues, "should find items updated after 09:31:00Z")

	// All 5 items should be found (all updated after 09:31:00Z)
	nums := issueNumbers(issues)
	assert.Contains(t, nums, 1)
	assert.Contains(t, nums, 2)

	// Use a future time — nothing should match
	query2 := BuildSearchQuery(testRepo, "2099-01-01T00:00:00Z", "is:issue [Please ignore: acceptance test]")
	issues2, err := client.SearchIssues(ctx, query2)
	require.NoError(t, err)
	assert.Empty(t, issues2, "no items should be updated after 2099")
}
