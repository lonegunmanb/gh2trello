package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/google/go-github/v69/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestClient creates a Client with BaseURL pointed at a test server.
func newTestClient(t *testing.T, handler http.Handler) (*Client, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(handler)
	ghClient := github.NewClient(nil)
	baseURL, _ := url.Parse(server.URL + "/")
	ghClient.BaseURL = baseURL
	return &Client{Client: ghClient}, server
}

func TestNewClient(t *testing.T) {
	client := NewClient("test_token")
	require.NotNil(t, client)
	require.NotNil(t, client.Client)
}

func TestBuildSearchQuery_WithAllParams(t *testing.T) {
	query := BuildSearchQuery("owner/repo", "2026-01-01T00:00:00Z", "label:bug is:issue")
	assert.Equal(t, "repo:owner/repo updated:>=2026-01-01T00:00:00Z label:bug is:issue", query)
}

func TestBuildSearchQuery_RepoAndSinceOnly(t *testing.T) {
	query := BuildSearchQuery("owner/repo", "2026-01-01T00:00:00Z", "")
	assert.Equal(t, "repo:owner/repo updated:>=2026-01-01T00:00:00Z", query)
}

func TestBuildSearchQuery_EmptySince(t *testing.T) {
	query := BuildSearchQuery("owner/repo", "", "is:open")
	assert.Equal(t, "repo:owner/repo is:open", query)
}

func TestBuildSearchQuery_RepoOnly(t *testing.T) {
	query := BuildSearchQuery("owner/repo", "", "")
	assert.Equal(t, "repo:owner/repo", query)
}

func TestIsPR_WithPullRequestLinks(t *testing.T) {
	issue := &github.Issue{
		PullRequestLinks: &github.PullRequestLinks{
			URL:     github.String("https://api.github.com/repos/o/r/pulls/1"),
			HTMLURL: github.String("https://github.com/o/r/pull/1"),
		},
	}
	assert.True(t, IsPR(issue))
}

func TestIsPR_WithoutPullRequestLinks(t *testing.T) {
	issue := &github.Issue{
		Number: github.Int(1),
		Title:  github.String("Test Issue"),
	}
	assert.False(t, IsPR(issue))
}

func TestGetLastActivityTime_UpdatedAfterCreated(t *testing.T) {
	created, _ := time.Parse(time.RFC3339, "2026-01-01T00:00:00Z")
	updated, _ := time.Parse(time.RFC3339, "2026-01-15T00:00:00Z")
	issue := &github.Issue{
		CreatedAt: &github.Timestamp{Time: created},
		UpdatedAt: &github.Timestamp{Time: updated},
	}
	assert.Equal(t, updated, GetLastActivityTime(issue))
}

func TestGetLastActivityTime_CreatedAfterUpdated(t *testing.T) {
	created, _ := time.Parse(time.RFC3339, "2026-01-15T00:00:00Z")
	updated, _ := time.Parse(time.RFC3339, "2026-01-01T00:00:00Z")
	issue := &github.Issue{
		CreatedAt: &github.Timestamp{Time: created},
		UpdatedAt: &github.Timestamp{Time: updated},
	}
	assert.Equal(t, created, GetLastActivityTime(issue))
}

func TestGetLastActivityTime_SameTime(t *testing.T) {
	created, _ := time.Parse(time.RFC3339, "2026-01-01T00:00:00Z")
	issue := &github.Issue{
		CreatedAt: &github.Timestamp{Time: created},
		UpdatedAt: &github.Timestamp{Time: created},
	}
	assert.Equal(t, created, GetLastActivityTime(issue))
}

// --- parseHTMLURL tests ---

func TestParseHTMLURL_IssueURL(t *testing.T) {
	owner, repo, number, err := parseHTMLURL("https://github.com/lonegunmanb/gh2trello/issues/42")
	require.NoError(t, err)
	assert.Equal(t, "lonegunmanb", owner)
	assert.Equal(t, "gh2trello", repo)
	assert.Equal(t, 42, number)
}

func TestParseHTMLURL_PullURL(t *testing.T) {
	owner, repo, number, err := parseHTMLURL("https://github.com/owner/repo/pull/7")
	require.NoError(t, err)
	assert.Equal(t, "owner", owner)
	assert.Equal(t, "repo", repo)
	assert.Equal(t, 7, number)
}

func TestParseHTMLURL_InvalidPrefix(t *testing.T) {
	_, _, _, err := parseHTMLURL("https://gitlab.com/owner/repo/issues/1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid GitHub URL")
}

func TestParseHTMLURL_TooFewSegments(t *testing.T) {
	_, _, _, err := parseHTMLURL("https://github.com/owner/repo")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid GitHub URL format")
}

func TestParseHTMLURL_NonNumericNumber(t *testing.T) {
	_, _, _, err := parseHTMLURL("https://github.com/owner/repo/issues/abc")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid issue/PR number")
}

// --- GetIssue tests ---

func TestGetIssue_Success(t *testing.T) {
	created, _ := time.Parse(time.RFC3339, "2026-01-01T00:00:00Z")
	client, server := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/repos/owner/repo/issues/42", r.URL.Path)
		issue := github.Issue{
			Number:    github.Int(42),
			Title:     github.String("Test Issue"),
			State:     github.String("open"),
			CreatedAt: &github.Timestamp{Time: created},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(issue)
	}))
	defer server.Close()

	issue, err := client.GetIssue(t.Context(), "owner", "repo", 42)
	require.NoError(t, err)
	assert.Equal(t, 42, issue.GetNumber())
	assert.Equal(t, "Test Issue", issue.GetTitle())
}

func TestGetIssue_NotFound(t *testing.T) {
	client, server := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"message":"Not Found"}`)
	}))
	defer server.Close()

	_, err := client.GetIssue(t.Context(), "owner", "repo", 999)
	assert.Error(t, err)
}

// --- SearchIssues tests ---

func TestSearchIssues_Success(t *testing.T) {
	client, server := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/search/issues", r.URL.Path)
		assert.Contains(t, r.URL.Query().Get("q"), "repo:owner/repo")
		result := github.IssuesSearchResult{
			Total:             github.Int(2),
			IncompleteResults: github.Bool(false),
			Issues: []*github.Issue{
				{Number: github.Int(1), Title: github.String("Issue 1")},
				{Number: github.Int(2), Title: github.String("Issue 2")},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}))
	defer server.Close()

	issues, err := client.SearchIssues(t.Context(), "repo:owner/repo")
	require.NoError(t, err)
	assert.Len(t, issues, 2)
	assert.Equal(t, "Issue 1", issues[0].GetTitle())
	assert.Equal(t, "Issue 2", issues[1].GetTitle())
}

func TestSearchIssues_Empty(t *testing.T) {
	client, server := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		result := github.IssuesSearchResult{
			Total:             github.Int(0),
			IncompleteResults: github.Bool(false),
			Issues:            []*github.Issue{},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}))
	defer server.Close()

	issues, err := client.SearchIssues(t.Context(), "repo:owner/repo")
	require.NoError(t, err)
	assert.Empty(t, issues)
}

func TestSearchIssues_Error(t *testing.T) {
	client, server := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		fmt.Fprint(w, `{"message":"Validation Failed"}`)
	}))
	defer server.Close()

	_, err := client.SearchIssues(t.Context(), "invalid query")
	assert.Error(t, err)
}

// --- GetComments tests ---

func TestGetComments_Success(t *testing.T) {
	t1, _ := time.Parse(time.RFC3339, "2026-01-10T00:00:00Z")
	t2, _ := time.Parse(time.RFC3339, "2026-01-20T00:00:00Z")
	client, server := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/repos/owner/repo/issues/5/comments", r.URL.Path)
		comments := []*github.IssueComment{
			{ID: github.Int64(1), Body: github.String("comment 1"), CreatedAt: &github.Timestamp{Time: t1}},
			{ID: github.Int64(2), Body: github.String("comment 2"), CreatedAt: &github.Timestamp{Time: t2}},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(comments)
	}))
	defer server.Close()

	comments, err := client.GetComments(t.Context(), "owner", "repo", 5)
	require.NoError(t, err)
	assert.Len(t, comments, 2)
	assert.Equal(t, "comment 1", comments[0].GetBody())
}

func TestGetComments_Empty(t *testing.T) {
	client, server := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[]`)
	}))
	defer server.Close()

	comments, err := client.GetComments(t.Context(), "owner", "repo", 5)
	require.NoError(t, err)
	assert.Empty(t, comments)
}

func TestGetComments_Error(t *testing.T) {
	client, server := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"message":"Not Found"}`)
	}))
	defer server.Close()

	_, err := client.GetComments(t.Context(), "owner", "repo", 999)
	assert.Error(t, err)
}

// --- GetLatestCommentTime tests ---

func TestGetLatestCommentTime_WithComments(t *testing.T) {
	t1, _ := time.Parse(time.RFC3339, "2026-01-10T00:00:00Z")
	t2, _ := time.Parse(time.RFC3339, "2026-01-20T00:00:00Z")
	t3, _ := time.Parse(time.RFC3339, "2026-01-15T00:00:00Z")
	client, server := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		comments := []*github.IssueComment{
			{ID: github.Int64(1), CreatedAt: &github.Timestamp{Time: t1}},
			{ID: github.Int64(2), CreatedAt: &github.Timestamp{Time: t2}},
			{ID: github.Int64(3), CreatedAt: &github.Timestamp{Time: t3}},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(comments)
	}))
	defer server.Close()

	latest, hasComments := client.GetLatestCommentTime(t.Context(), "owner", "repo", 1)
	assert.True(t, hasComments)
	assert.Equal(t, t2, latest)
}

func TestGetLatestCommentTime_NoComments(t *testing.T) {
	client, server := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[]`)
	}))
	defer server.Close()

	_, hasComments := client.GetLatestCommentTime(t.Context(), "owner", "repo", 1)
	assert.False(t, hasComments)
}

func TestGetLatestCommentTime_Error(t *testing.T) {
	client, server := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"message":"server error"}`)
	}))
	defer server.Close()

	_, hasComments := client.GetLatestCommentTime(t.Context(), "owner", "repo", 1)
	assert.False(t, hasComments)
}

// --- GetEffectiveTime tests ---

func TestGetEffectiveTime_CommentLaterThanCreated(t *testing.T) {
	created, _ := time.Parse(time.RFC3339, "2026-01-01T00:00:00Z")
	commentTime, _ := time.Parse(time.RFC3339, "2026-01-20T00:00:00Z")

	mux := http.NewServeMux()
	mux.HandleFunc("/repos/owner/repo/issues/10/comments", func(w http.ResponseWriter, r *http.Request) {
		comments := []*github.IssueComment{
			{ID: github.Int64(1), CreatedAt: &github.Timestamp{Time: commentTime}},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(comments)
	})
	client, server := newTestClient(t, mux)
	defer server.Close()

	issue := &github.Issue{
		Number:    github.Int(10),
		HTMLURL:   github.String("https://github.com/owner/repo/issues/10"),
		CreatedAt: &github.Timestamp{Time: created},
	}

	result := client.GetEffectiveTime(t.Context(), issue)
	assert.Equal(t, commentTime, result)
}

func TestGetEffectiveTime_CreatedLaterThanComment(t *testing.T) {
	created, _ := time.Parse(time.RFC3339, "2026-01-20T00:00:00Z")
	commentTime, _ := time.Parse(time.RFC3339, "2026-01-01T00:00:00Z")

	mux := http.NewServeMux()
	mux.HandleFunc("/repos/owner/repo/issues/10/comments", func(w http.ResponseWriter, r *http.Request) {
		comments := []*github.IssueComment{
			{ID: github.Int64(1), CreatedAt: &github.Timestamp{Time: commentTime}},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(comments)
	})
	client, server := newTestClient(t, mux)
	defer server.Close()

	issue := &github.Issue{
		Number:    github.Int(10),
		HTMLURL:   github.String("https://github.com/owner/repo/issues/10"),
		CreatedAt: &github.Timestamp{Time: created},
	}

	result := client.GetEffectiveTime(t.Context(), issue)
	assert.Equal(t, created, result)
}

func TestGetEffectiveTime_NoComments(t *testing.T) {
	created, _ := time.Parse(time.RFC3339, "2026-01-01T00:00:00Z")

	mux := http.NewServeMux()
	mux.HandleFunc("/repos/owner/repo/issues/10/comments", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[]`)
	})
	client, server := newTestClient(t, mux)
	defer server.Close()

	issue := &github.Issue{
		Number:    github.Int(10),
		HTMLURL:   github.String("https://github.com/owner/repo/issues/10"),
		CreatedAt: &github.Timestamp{Time: created},
	}

	result := client.GetEffectiveTime(t.Context(), issue)
	assert.Equal(t, created, result)
}

func TestGetEffectiveTime_InvalidURL_FallsBackToCreated(t *testing.T) {
	created, _ := time.Parse(time.RFC3339, "2026-01-01T00:00:00Z")

	client, server := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not make any HTTP request")
	}))
	defer server.Close()

	issue := &github.Issue{
		Number:    github.Int(10),
		HTMLURL:   github.String("https://gitlab.com/owner/repo/issues/10"),
		CreatedAt: &github.Timestamp{Time: created},
	}

	result := client.GetEffectiveTime(t.Context(), issue)
	assert.Equal(t, created, result)
}