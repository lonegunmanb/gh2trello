package sync

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lonegunmanb/gh2trello/config"
	"github.com/lonegunmanb/gh2trello/trello"

	"github.com/google/go-github/v69/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mock GitHub Client ---

type mockGitHubClient struct {
	getIssueFunc      func(ctx context.Context, owner, repo string, number int) (*github.Issue, error)
	searchIssuesFunc  func(ctx context.Context, query string) ([]*github.Issue, error)
	getEffectiveFunc  func(ctx context.Context, issue *github.Issue) time.Time
}

func (m *mockGitHubClient) GetIssue(ctx context.Context, owner, repo string, number int) (*github.Issue, error) {
	return m.getIssueFunc(ctx, owner, repo, number)
}

func (m *mockGitHubClient) SearchIssues(ctx context.Context, query string) ([]*github.Issue, error) {
	return m.searchIssuesFunc(ctx, query)
}

func (m *mockGitHubClient) GetEffectiveTime(ctx context.Context, issue *github.Issue) time.Time {
	return m.getEffectiveFunc(ctx, issue)
}

// --- Mock Trello Client ---

type mockTrelloClient struct {
	createCardFunc func(issue *github.Issue, issueListID, prListID string) (*trello.Card, error)
	findCardFunc   func(issueListID, prListID, url string) (*trello.Card, error)
	moveCardFunc   func(cardID, targetListID string) error
}

func (m *mockTrelloClient) CreateCardForIssue(issue *github.Issue, issueListID, prListID string) (*trello.Card, error) {
	return m.createCardFunc(issue, issueListID, prListID)
}

func (m *mockTrelloClient) FindCard(issueListID, prListID, url string) (*trello.Card, error) {
	if m.findCardFunc != nil {
		return m.findCardFunc(issueListID, prListID, url)
	}
	return nil, nil
}

func (m *mockTrelloClient) MoveCard(cardID, targetListID string) error {
	if m.moveCardFunc != nil {
		return m.moveCardFunc(cardID, targetListID)
	}
	return nil
}

// --- Helpers ---

func newTestConfig() *config.Config {
	return &config.Config{
		GitHubToken:       "gh_token",
		TrelloAPIKey:      "trello_key",
		TrelloAPIToken:    "trello_token",
		TrelloBoardID:     "board1",
		TrelloIssueListID: "issue_list",
		TrelloPRListID:    "pr_list",
		PollInterval:      "5m",
		Repos:             make(map[string]config.RepoConfig),
	}
}

func makeIssue(number int, title, htmlURL string, created time.Time) *github.Issue {
	return &github.Issue{
		Number:    github.Int(number),
		Title:     github.String(title),
		Body:      github.String("body"),
		State:     github.String("open"),
		HTMLURL:   github.String(htmlURL),
		CreatedAt: &github.Timestamp{Time: created},
		UpdatedAt: &github.Timestamp{Time: created},
		User:      &github.User{Login: github.String("user")},
	}
}

func makePR(number int, title, htmlURL string, created time.Time) *github.Issue {
	issue := makeIssue(number, title, htmlURL, created)
	issue.PullRequestLinks = &github.PullRequestLinks{
		URL:     github.String(fmt.Sprintf("https://api.github.com/repos/owner/repo/pulls/%d", number)),
		HTMLURL: github.String(htmlURL),
	}
	return issue
}

// =========================================================================
// 4.1 SyncWorkitem tests
// =========================================================================

func TestSyncWorkitem_Issue(t *testing.T) {
	created, _ := time.Parse(time.RFC3339, "2026-01-10T00:00:00Z")
	issue := makeIssue(42, "Bug fix", "https://github.com/owner/repo/issues/42", created)

	var createdInList string
	ghMock := &mockGitHubClient{
		getIssueFunc: func(_ context.Context, owner, repo string, number int) (*github.Issue, error) {
			assert.Equal(t, "owner", owner)
			assert.Equal(t, "repo", repo)
			assert.Equal(t, 42, number)
			return issue, nil
		},
	}
	trelloMock := &mockTrelloClient{
		findCardFunc: func(_, _, _ string) (*trello.Card, error) {
			return nil, nil
		},
		createCardFunc: func(i *github.Issue, issueListID, prListID string) (*trello.Card, error) {
			createdInList = issueListID
			return &trello.Card{ID: "card1"}, nil
		},
	}

	cfg := newTestConfig()
	engine := NewEngine(ghMock, trelloMock, cfg, "")

	err := engine.SyncWorkitem(t.Context(), "owner", "repo", 42)
	require.NoError(t, err)
	assert.Equal(t, "issue_list", createdInList)
}

func TestSyncWorkitem_PR(t *testing.T) {
	created, _ := time.Parse(time.RFC3339, "2026-01-10T00:00:00Z")
	pr := makePR(7, "Add feature", "https://github.com/owner/repo/pull/7", created)

	var receivedIssueListID, receivedPRListID string
	ghMock := &mockGitHubClient{
		getIssueFunc: func(_ context.Context, _, _ string, _ int) (*github.Issue, error) {
			return pr, nil
		},
	}
	trelloMock := &mockTrelloClient{
		findCardFunc: func(_, _, _ string) (*trello.Card, error) {
			return nil, nil
		},
		createCardFunc: func(i *github.Issue, issueListID, prListID string) (*trello.Card, error) {
			receivedIssueListID = issueListID
			receivedPRListID = prListID
			return &trello.Card{ID: "card2"}, nil
		},
	}

	cfg := newTestConfig()
	engine := NewEngine(ghMock, trelloMock, cfg, "")

	err := engine.SyncWorkitem(t.Context(), "owner", "repo", 7)
	require.NoError(t, err)
	assert.Equal(t, "issue_list", receivedIssueListID)
	assert.Equal(t, "pr_list", receivedPRListID)
}

func TestSyncWorkitem_NotFound(t *testing.T) {
	ghMock := &mockGitHubClient{
		getIssueFunc: func(_ context.Context, _, _ string, _ int) (*github.Issue, error) {
			return nil, fmt.Errorf("404 Not Found")
		},
	}
	trelloMock := &mockTrelloClient{}

	cfg := newTestConfig()
	engine := NewEngine(ghMock, trelloMock, cfg, "")

	err := engine.SyncWorkitem(t.Context(), "owner", "repo", 999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get issue")
}

func TestSyncWorkitem_ExistsInCorrectList(t *testing.T) {
	created, _ := time.Parse(time.RFC3339, "2026-01-10T00:00:00Z")
	issue := makeIssue(42, "Bug fix", "https://github.com/owner/repo/issues/42", created)

	createCalled := false
	moveCalled := false
	ghMock := &mockGitHubClient{
		getIssueFunc: func(_ context.Context, _, _ string, _ int) (*github.Issue, error) {
			return issue, nil
		},
	}
	trelloMock := &mockTrelloClient{
		findCardFunc: func(issueListID, _, _ string) (*trello.Card, error) {
			// Card already in the issue list (correct list)
			return &trello.Card{ID: "card1", IDList: issueListID}, nil
		},
		createCardFunc: func(_ *github.Issue, _, _ string) (*trello.Card, error) {
			createCalled = true
			return nil, nil
		},
		moveCardFunc: func(_, _ string) error {
			moveCalled = true
			return nil
		},
	}

	cfg := newTestConfig()
	engine := NewEngine(ghMock, trelloMock, cfg, "")

	err := engine.SyncWorkitem(t.Context(), "owner", "repo", 42)
	require.NoError(t, err)
	assert.False(t, createCalled, "Should not create card when already in correct list")
	assert.False(t, moveCalled, "Should not move card when already in correct list")
}

func TestSyncWorkitem_ExistsInWrongList(t *testing.T) {
	created, _ := time.Parse(time.RFC3339, "2026-01-10T00:00:00Z")
	// An issue that ended up in the PR list
	issue := makeIssue(42, "Bug fix", "https://github.com/owner/repo/issues/42", created)

	createCalled := false
	var movedToList string
	ghMock := &mockGitHubClient{
		getIssueFunc: func(_ context.Context, _, _ string, _ int) (*github.Issue, error) {
			return issue, nil
		},
	}
	trelloMock := &mockTrelloClient{
		findCardFunc: func(_, prListID, _ string) (*trello.Card, error) {
			// Card is in the PR list (wrong list for an issue)
			return &trello.Card{ID: "card1", IDList: prListID}, nil
		},
		createCardFunc: func(_ *github.Issue, _, _ string) (*trello.Card, error) {
			createCalled = true
			return nil, nil
		},
		moveCardFunc: func(cardID, targetListID string) error {
			assert.Equal(t, "card1", cardID)
			movedToList = targetListID
			return nil
		},
	}

	cfg := newTestConfig()
	engine := NewEngine(ghMock, trelloMock, cfg, "")

	err := engine.SyncWorkitem(t.Context(), "owner", "repo", 42)
	require.NoError(t, err)
	assert.False(t, createCalled, "Should not create card when it already exists")
	assert.Equal(t, "issue_list", movedToList, "Issue card should be moved to issue list")
}

func TestSyncWorkitem_TrelloCreateError(t *testing.T) {
	created, _ := time.Parse(time.RFC3339, "2026-01-10T00:00:00Z")
	issue := makeIssue(42, "Bug fix", "https://github.com/owner/repo/issues/42", created)

	ghMock := &mockGitHubClient{
		getIssueFunc: func(_ context.Context, _, _ string, _ int) (*github.Issue, error) {
			return issue, nil
		},
	}
	trelloMock := &mockTrelloClient{
		findCardFunc: func(_, _, _ string) (*trello.Card, error) {
			return nil, nil
		},
		createCardFunc: func(_ *github.Issue, _, _ string) (*trello.Card, error) {
			return nil, fmt.Errorf("trello error")
		},
	}

	cfg := newTestConfig()
	engine := NewEngine(ghMock, trelloMock, cfg, "")

	err := engine.SyncWorkitem(t.Context(), "owner", "repo", 42)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create Trello card")
}

// =========================================================================
// 4.2 SyncRepo tests
// =========================================================================

func TestSyncRepo_NewItemsCreated(t *testing.T) {
	t1, _ := time.Parse(time.RFC3339, "2026-01-15T00:00:00Z")
	t2, _ := time.Parse(time.RFC3339, "2026-01-20T00:00:00Z")

	issues := []*github.Issue{
		makeIssue(1, "Issue 1", "https://github.com/owner/repo/issues/1", t1),
		makePR(2, "PR 2", "https://github.com/owner/repo/pull/2", t2),
	}

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	cfg := newTestConfig()
	cfg.AddRepo("owner/repo", "2026-01-01T00:00:00Z", "")

	var searchQuery string
	ghMock := &mockGitHubClient{
		searchIssuesFunc: func(_ context.Context, query string) ([]*github.Issue, error) {
			searchQuery = query
			return issues, nil
		},
		getEffectiveFunc: func(_ context.Context, issue *github.Issue) time.Time {
			return issue.GetCreatedAt().Time
		},
	}

	var createdCards []string
	trelloMock := &mockTrelloClient{
		findCardFunc: func(_, _, _ string) (*trello.Card, error) {
			return nil, nil
		},
		createCardFunc: func(issue *github.Issue, _, _ string) (*trello.Card, error) {
			createdCards = append(createdCards, issue.GetTitle())
			return &trello.Card{ID: "card"}, nil
		},
	}

	engine := NewEngine(ghMock, trelloMock, cfg, configPath)

	count, err := engine.SyncRepo(t.Context(), "owner/repo")
	require.NoError(t, err)
	assert.Equal(t, 2, count)
	assert.Len(t, createdCards, 2)
	assert.Contains(t, searchQuery, "repo:owner/repo")
	assert.Contains(t, searchQuery, "updated:>=2026-01-01T00:00:00Z")

	// Verify watermark updated to latest time
	repoCfg, ok := cfg.GetRepo("owner/repo")
	require.True(t, ok)
	assert.Equal(t, "2026-01-20T00:00:00Z", repoCfg.Since)
}

func TestSyncRepo_NoNewItems(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	cfg := newTestConfig()
	cfg.AddRepo("owner/repo", "2026-01-01T00:00:00Z", "is:issue")

	ghMock := &mockGitHubClient{
		searchIssuesFunc: func(_ context.Context, _ string) ([]*github.Issue, error) {
			return []*github.Issue{}, nil
		},
	}
	trelloMock := &mockTrelloClient{}

	engine := NewEngine(ghMock, trelloMock, cfg, configPath)

	count, err := engine.SyncRepo(t.Context(), "owner/repo")
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	// Watermark should not change
	repoCfg, _ := cfg.GetRepo("owner/repo")
	assert.Equal(t, "2026-01-01T00:00:00Z", repoCfg.Since)
}

func TestSyncRepo_ExistingCardInCorrectList(t *testing.T) {
	t1, _ := time.Parse(time.RFC3339, "2026-01-15T00:00:00Z")
	t2, _ := time.Parse(time.RFC3339, "2026-01-20T00:00:00Z")

	issues := []*github.Issue{
		makeIssue(1, "Issue 1", "https://github.com/owner/repo/issues/1", t1),
		makeIssue(2, "Issue 2", "https://github.com/owner/repo/issues/2", t2),
	}

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	cfg := newTestConfig()
	cfg.AddRepo("owner/repo", "2026-01-01T00:00:00Z", "")

	ghMock := &mockGitHubClient{
		searchIssuesFunc: func(_ context.Context, _ string) ([]*github.Issue, error) {
			return issues, nil
		},
		getEffectiveFunc: func(_ context.Context, issue *github.Issue) time.Time {
			return issue.GetCreatedAt().Time
		},
	}

	moveCalled := false
	// Issue 1 already exists in the correct list; Issue 2 does not
	trelloMock := &mockTrelloClient{
		findCardFunc: func(issueListID, _, url string) (*trello.Card, error) {
			if url == "https://github.com/owner/repo/issues/1" {
				return &trello.Card{ID: "card1", IDList: issueListID}, nil
			}
			return nil, nil
		},
		moveCardFunc: func(_, _ string) error {
			moveCalled = true
			return nil
		},
		createCardFunc: func(issue *github.Issue, _, _ string) (*trello.Card, error) {
			assert.Equal(t, "Issue 2", issue.GetTitle(), "Only Issue 2 should be created")
			return &trello.Card{ID: "card"}, nil
		},
	}

	engine := NewEngine(ghMock, trelloMock, cfg, configPath)

	count, err := engine.SyncRepo(t.Context(), "owner/repo")
	require.NoError(t, err)
	assert.Equal(t, 1, count, "Only 1 new card should be created")
	assert.False(t, moveCalled, "Should not move card already in the correct list")

	// Watermark should still advance to latest time (including existing cards)
	repoCfg, _ := cfg.GetRepo("owner/repo")
	assert.Equal(t, "2026-01-20T00:00:00Z", repoCfg.Since)
}

func TestSyncRepo_ExistingCardInWrongList(t *testing.T) {
	t1, _ := time.Parse(time.RFC3339, "2026-01-15T00:00:00Z")

	// An issue card that ended up in the PR list
	issues := []*github.Issue{
		makeIssue(1, "Issue 1", "https://github.com/owner/repo/issues/1", t1),
	}

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	cfg := newTestConfig()
	cfg.AddRepo("owner/repo", "2026-01-01T00:00:00Z", "")

	ghMock := &mockGitHubClient{
		searchIssuesFunc: func(_ context.Context, _ string) ([]*github.Issue, error) {
			return issues, nil
		},
		getEffectiveFunc: func(_ context.Context, issue *github.Issue) time.Time {
			return issue.GetCreatedAt().Time
		},
	}

	var movedToList string
	createCalled := false
	trelloMock := &mockTrelloClient{
		findCardFunc: func(_, prListID, _ string) (*trello.Card, error) {
			// Card is in the wrong list (PR list instead of issue list)
			return &trello.Card{ID: "card1", IDList: prListID}, nil
		},
		moveCardFunc: func(cardID, targetListID string) error {
			assert.Equal(t, "card1", cardID)
			movedToList = targetListID
			return nil
		},
		createCardFunc: func(_ *github.Issue, _, _ string) (*trello.Card, error) {
			createCalled = true
			return &trello.Card{ID: "card"}, nil
		},
	}

	engine := NewEngine(ghMock, trelloMock, cfg, configPath)

	count, err := engine.SyncRepo(t.Context(), "owner/repo")
	require.NoError(t, err)
	assert.Equal(t, 0, count, "No new cards should be created when card is moved")
	assert.False(t, createCalled, "Should not create card when it already exists")
	assert.Equal(t, "issue_list", movedToList, "Issue card should be moved to issue list")
}

func TestSyncRepo_WatermarkFiltersOldItems(t *testing.T) {
	// Items with effectiveTime <= watermark should be skipped
	oldTime, _ := time.Parse(time.RFC3339, "2026-01-01T00:00:00Z")
	newTime, _ := time.Parse(time.RFC3339, "2026-01-20T00:00:00Z")

	issues := []*github.Issue{
		makeIssue(1, "Old Issue", "https://github.com/owner/repo/issues/1", oldTime),
		makeIssue(2, "New Issue", "https://github.com/owner/repo/issues/2", newTime),
	}

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	cfg := newTestConfig()
	cfg.AddRepo("owner/repo", "2026-01-10T00:00:00Z", "")

	ghMock := &mockGitHubClient{
		searchIssuesFunc: func(_ context.Context, _ string) ([]*github.Issue, error) {
			return issues, nil
		},
		getEffectiveFunc: func(_ context.Context, issue *github.Issue) time.Time {
			return issue.GetCreatedAt().Time
		},
	}

	var createdTitles []string
	trelloMock := &mockTrelloClient{
		findCardFunc: func(_, _, _ string) (*trello.Card, error) {
			return nil, nil
		},
		createCardFunc: func(issue *github.Issue, _, _ string) (*trello.Card, error) {
			createdTitles = append(createdTitles, issue.GetTitle())
			return &trello.Card{ID: "card"}, nil
		},
	}

	engine := NewEngine(ghMock, trelloMock, cfg, configPath)

	count, err := engine.SyncRepo(t.Context(), "owner/repo")
	require.NoError(t, err)
	assert.Equal(t, 1, count)
	assert.Equal(t, []string{"New Issue"}, createdTitles)
}

func TestSyncRepo_RepoNotInConfig(t *testing.T) {
	cfg := newTestConfig()
	engine := NewEngine(nil, nil, cfg, "")

	_, err := engine.SyncRepo(t.Context(), "unknown/repo")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found in config")
}

func TestSyncRepo_SearchError(t *testing.T) {
	cfg := newTestConfig()
	cfg.AddRepo("owner/repo", "", "")

	ghMock := &mockGitHubClient{
		searchIssuesFunc: func(_ context.Context, _ string) ([]*github.Issue, error) {
			return nil, fmt.Errorf("API rate limit")
		},
	}

	engine := NewEngine(ghMock, nil, cfg, "")

	_, err := engine.SyncRepo(t.Context(), "owner/repo")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to search issues")
}

func TestSyncRepo_ConfigSaved(t *testing.T) {
	t1, _ := time.Parse(time.RFC3339, "2026-02-01T00:00:00Z")
	issues := []*github.Issue{
		makeIssue(1, "Issue 1", "https://github.com/owner/repo/issues/1", t1),
	}

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	cfg := newTestConfig()
	cfg.AddRepo("owner/repo", "2026-01-01T00:00:00Z", "")

	ghMock := &mockGitHubClient{
		searchIssuesFunc: func(_ context.Context, _ string) ([]*github.Issue, error) {
			return issues, nil
		},
		getEffectiveFunc: func(_ context.Context, issue *github.Issue) time.Time {
			return issue.GetCreatedAt().Time
		},
	}
	trelloMock := &mockTrelloClient{
		findCardFunc: func(_, _, _ string) (*trello.Card, error) { return nil, nil },
		createCardFunc: func(_ *github.Issue, _, _ string) (*trello.Card, error) {
			return &trello.Card{ID: "card"}, nil
		},
	}

	engine := NewEngine(ghMock, trelloMock, cfg, configPath)

	_, err := engine.SyncRepo(t.Context(), "owner/repo")
	require.NoError(t, err)

	// Verify config was persisted to disk
	reloaded, err := config.Load(configPath)
	require.NoError(t, err)
	repoCfg, ok := reloaded.GetRepo("owner/repo")
	require.True(t, ok)
	assert.Equal(t, "2026-02-01T00:00:00Z", repoCfg.Since)
}

func TestSyncRepo_IssueAndPRDispatch(t *testing.T) {
	t1, _ := time.Parse(time.RFC3339, "2026-01-15T00:00:00Z")

	issues := []*github.Issue{
		makeIssue(1, "Bug", "https://github.com/owner/repo/issues/1", t1),
		makePR(2, "Feature", "https://github.com/owner/repo/pull/2", t1),
	}

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	cfg := newTestConfig()
	cfg.AddRepo("owner/repo", "2026-01-01T00:00:00Z", "")

	ghMock := &mockGitHubClient{
		searchIssuesFunc: func(_ context.Context, _ string) ([]*github.Issue, error) {
			return issues, nil
		},
		getEffectiveFunc: func(_ context.Context, issue *github.Issue) time.Time {
			return issue.GetCreatedAt().Time
		},
	}

	var issueListReceived, prListReceived string
	trelloMock := &mockTrelloClient{
		findCardFunc: func(issueListID, prListID, _ string) (*trello.Card, error) {
			issueListReceived = issueListID
			prListReceived = prListID
			return nil, nil
		},
		createCardFunc: func(_ *github.Issue, _, _ string) (*trello.Card, error) {
			return &trello.Card{ID: "card"}, nil
		},
	}

	engine := NewEngine(ghMock, trelloMock, cfg, configPath)

	count, err := engine.SyncRepo(t.Context(), "owner/repo")
	require.NoError(t, err)
	assert.Equal(t, 2, count)
	assert.Equal(t, "issue_list", issueListReceived, "FindCard should receive issue list as first arg")
	assert.Equal(t, "pr_list", prListReceived, "FindCard should receive PR list as second arg")
}

// =========================================================================
// 4.4 RunPolling tests
// =========================================================================

func TestRunPolling_ExecutesAndStops(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	cfg := newTestConfig()
	cfg.PollInterval = "50ms"
	cfg.AddRepo("owner/repo", "2026-01-01T00:00:00Z", "")

	syncCount := 0
	ghMock := &mockGitHubClient{
		searchIssuesFunc: func(_ context.Context, _ string) ([]*github.Issue, error) {
			syncCount++
			return []*github.Issue{}, nil
		},
	}
	trelloMock := &mockTrelloClient{}

	engine := NewEngine(ghMock, trelloMock, cfg, configPath)

	ctx, cancel := context.WithTimeout(t.Context(), 180*time.Millisecond)
	defer cancel()

	err := engine.RunPolling(ctx)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	// Should have run at least twice: once immediately + at least once on tick
	assert.GreaterOrEqual(t, syncCount, 2, "Should have polled at least twice")
}

func TestRunPolling_InvalidInterval(t *testing.T) {
	cfg := newTestConfig()
	cfg.PollInterval = "invalid"

	engine := NewEngine(nil, nil, cfg, "")

	err := engine.RunPolling(t.Context())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid poll interval")
}

func TestRunPolling_MultiplRepos(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	cfg := newTestConfig()
	cfg.PollInterval = "50ms"
	cfg.AddRepo("owner/repo1", "2026-01-01T00:00:00Z", "")
	cfg.AddRepo("owner/repo2", "2026-01-01T00:00:00Z", "")

	syncedRepos := make(map[string]int)
	ghMock := &mockGitHubClient{
		searchIssuesFunc: func(_ context.Context, query string) ([]*github.Issue, error) {
			// Extract repo name from query
			if len(query) > 5 {
				for _, r := range []string{"owner/repo1", "owner/repo2"} {
					if containsStr(query, r) {
						syncedRepos[r]++
					}
				}
			}
			return []*github.Issue{}, nil
		},
	}
	trelloMock := &mockTrelloClient{}

	engine := NewEngine(ghMock, trelloMock, cfg, configPath)

	ctx, cancel := context.WithTimeout(t.Context(), 80*time.Millisecond)
	defer cancel()

	_ = engine.RunPolling(ctx)
	assert.GreaterOrEqual(t, syncedRepos["owner/repo1"], 1, "repo1 should be synced")
	assert.GreaterOrEqual(t, syncedRepos["owner/repo2"], 1, "repo2 should be synced")
}

func TestSyncRepo_EmptySinceNoFilter(t *testing.T) {
	// When since is empty, all items should pass the watermark check
	t1, _ := time.Parse(time.RFC3339, "2026-01-01T00:00:00Z")
	issues := []*github.Issue{
		makeIssue(1, "Issue 1", "https://github.com/owner/repo/issues/1", t1),
	}

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	cfg := newTestConfig()
	cfg.Repos["owner/repo"] = config.RepoConfig{Since: "", Query: ""}

	ghMock := &mockGitHubClient{
		searchIssuesFunc: func(_ context.Context, _ string) ([]*github.Issue, error) {
			return issues, nil
		},
		getEffectiveFunc: func(_ context.Context, issue *github.Issue) time.Time {
			return issue.GetCreatedAt().Time
		},
	}
	trelloMock := &mockTrelloClient{
		findCardFunc: func(_, _, _ string) (*trello.Card, error) { return nil, nil },
		createCardFunc: func(_ *github.Issue, _, _ string) (*trello.Card, error) {
			return &trello.Card{ID: "card"}, nil
		},
	}

	engine := NewEngine(ghMock, trelloMock, cfg, configPath)

	count, err := engine.SyncRepo(t.Context(), "owner/repo")
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestSyncRepo_WithQuery(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	cfg := newTestConfig()
	cfg.AddRepo("owner/repo", "2026-01-01T00:00:00Z", "label:bug is:issue")

	// Save config first so Load works later
	require.NoError(t, config.Save(configPath, cfg))

	var capturedQuery string
	ghMock := &mockGitHubClient{
		searchIssuesFunc: func(_ context.Context, query string) ([]*github.Issue, error) {
			capturedQuery = query
			return []*github.Issue{}, nil
		},
	}
	trelloMock := &mockTrelloClient{}

	engine := NewEngine(ghMock, trelloMock, cfg, configPath)

	_, err := engine.SyncRepo(t.Context(), "owner/repo")
	require.NoError(t, err)
	assert.Contains(t, capturedQuery, "label:bug is:issue")
}

// --- helpers ---

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstr(s, substr)
}

func searchSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Ensure config file can be loaded after SyncRepo saves it
func TestSyncRepo_ConfigRoundTrip(t *testing.T) {
	t1, _ := time.Parse(time.RFC3339, "2026-03-01T00:00:00Z")
	issues := []*github.Issue{
		makeIssue(1, "Issue", "https://github.com/owner/repo/issues/1", t1),
	}

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	cfg := newTestConfig()
	cfg.AddRepo("owner/repo", "2026-01-01T00:00:00Z", "label:bug")
	require.NoError(t, config.Save(configPath, cfg))

	ghMock := &mockGitHubClient{
		searchIssuesFunc: func(_ context.Context, _ string) ([]*github.Issue, error) {
			return issues, nil
		},
		getEffectiveFunc: func(_ context.Context, issue *github.Issue) time.Time {
			return issue.GetCreatedAt().Time
		},
	}
	trelloMock := &mockTrelloClient{
		findCardFunc: func(_, _, _ string) (*trello.Card, error) { return nil, nil },
		createCardFunc: func(_ *github.Issue, _, _ string) (*trello.Card, error) {
			return &trello.Card{ID: "card"}, nil
		},
	}

	engine := NewEngine(ghMock, trelloMock, cfg, configPath)

	_, err := engine.SyncRepo(t.Context(), "owner/repo")
	require.NoError(t, err)

	// Load from disk and verify
	loaded, err := config.Load(configPath)
	require.NoError(t, err)

	rc, ok := loaded.GetRepo("owner/repo")
	require.True(t, ok)
	assert.Equal(t, "2026-03-01T00:00:00Z", rc.Since)
	assert.Equal(t, "label:bug", rc.Query)

	// Verify config file exists
	_, err = os.Stat(configPath)
	require.NoError(t, err)
}
