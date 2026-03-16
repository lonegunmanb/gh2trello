package trello

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/go-github/v69/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	client := NewClient("api_key", "token")
	require.NotNil(t, client)
	assert.Equal(t, "api_key", client.APIKey)
	assert.Equal(t, "token", client.Token)
	assert.Equal(t, "https://api.trello.com/1", client.BaseURL)
}

func TestFormatCardName(t *testing.T) {
	issue := &github.Issue{
		Title: github.String("Test Issue Title"),
	}
	name := FormatCardName(issue)
	assert.Equal(t, "Test Issue Title", name)
}

func TestFormatCardDesc_Issue(t *testing.T) {
	created, _ := time.Parse(time.RFC3339, "2026-01-01T00:00:00Z")
	updated, _ := time.Parse(time.RFC3339, "2026-01-15T00:00:00Z")

	issue := &github.Issue{
		Number:    github.Int(123),
		Title:     github.String("Test Issue"),
		Body:      github.String("This is the issue body"),
		State:     github.String("open"),
		HTMLURL:   github.String("https://github.com/owner/repo/issues/123"),
		CreatedAt: &github.Timestamp{Time: created},
		UpdatedAt: &github.Timestamp{Time: updated},
		User: &github.User{
			Login: github.String("testuser"),
		},
		Labels: []*github.Label{
			{Name: github.String("bug")},
			{Name: github.String("enhancement")},
		},
	}

	desc := FormatCardDesc(issue)

	assert.Contains(t, desc, "https://github.com/owner/repo/issues/123")
	assert.Contains(t, desc, "This is the issue body")
	assert.Contains(t, desc, "Raw JSON:")
	assert.Contains(t, desc, `"number": 123`)
	assert.Contains(t, desc, `"title": "Test Issue"`)
	assert.Contains(t, desc, `"state": "open"`)
	assert.Contains(t, desc, `"author": "testuser"`)
}

func TestFormatCardDesc_PR(t *testing.T) {
	created, _ := time.Parse(time.RFC3339, "2026-01-01T00:00:00Z")

	issue := &github.Issue{
		Number:    github.Int(456),
		Title:     github.String("Test PR"),
		Body:      github.String("PR description"),
		State:     github.String("open"),
		HTMLURL:   github.String("https://github.com/owner/repo/pull/456"),
		CreatedAt: &github.Timestamp{Time: created},
		UpdatedAt: &github.Timestamp{Time: created},
		User: &github.User{
			Login: github.String("prauthor"),
		},
		PullRequestLinks: &github.PullRequestLinks{
			URL:     github.String("https://api.github.com/repos/owner/repo/pulls/456"),
			HTMLURL: github.String("https://github.com/owner/repo/pull/456"),
		},
	}

	desc := FormatCardDesc(issue)
	assert.Contains(t, desc, `"is_pr": true`)
}

func TestFormatCardDesc_NoLabels(t *testing.T) {
	created, _ := time.Parse(time.RFC3339, "2026-01-01T00:00:00Z")
	issue := &github.Issue{
		Number:    github.Int(1),
		Title:     github.String("No Labels Issue"),
		Body:      github.String("body"),
		State:     github.String("open"),
		HTMLURL:   github.String("https://github.com/owner/repo/issues/1"),
		CreatedAt: &github.Timestamp{Time: created},
		UpdatedAt: &github.Timestamp{Time: created},
		User:      &github.User{Login: github.String("user")},
	}

	desc := FormatCardDesc(issue)
	assert.Contains(t, desc, "No Labels Issue")
	assert.Contains(t, desc, `"labels": []`)
}

func TestFormatCardDescFromPR(t *testing.T) {
	created, _ := time.Parse(time.RFC3339, "2026-02-01T00:00:00Z")
	updated, _ := time.Parse(time.RFC3339, "2026-02-10T00:00:00Z")

	pr := &github.PullRequest{
		Number:       github.Int(789),
		Title:        github.String("Add feature"),
		Body:         github.String("This PR adds a feature"),
		State:        github.String("open"),
		HTMLURL:      github.String("https://github.com/owner/repo/pull/789"),
		CreatedAt:    &github.Timestamp{Time: created},
		UpdatedAt:    &github.Timestamp{Time: updated},
		Additions:    github.Int(50),
		Deletions:    github.Int(10),
		ChangedFiles: github.Int(3),
		Head:         &github.PullRequestBranch{Ref: github.String("feature-branch")},
		Base:         &github.PullRequestBranch{Ref: github.String("main")},
		User:         &github.User{Login: github.String("contributor")},
		Labels: []*github.Label{
			{Name: github.String("feature")},
		},
	}

	desc := FormatCardDescFromPR(pr)
	assert.Contains(t, desc, "https://github.com/owner/repo/pull/789")
	assert.Contains(t, desc, "This PR adds a feature")
	assert.Contains(t, desc, `"is_pr": true`)
	assert.Contains(t, desc, `"head_branch": "feature-branch"`)
	assert.Contains(t, desc, `"base_branch": "main"`)
	assert.Contains(t, desc, `"additions": 50`)
	assert.Contains(t, desc, `"deletions": 10`)
	assert.Contains(t, desc, `"changed_files": 3`)
	assert.Contains(t, desc, `"author": "contributor"`)
}

func TestFormatCardDescFromPR_NoLabels(t *testing.T) {
	created, _ := time.Parse(time.RFC3339, "2026-02-01T00:00:00Z")
	pr := &github.PullRequest{
		Number:       github.Int(10),
		Title:        github.String("No labels PR"),
		Body:         github.String("body"),
		State:        github.String("open"),
		HTMLURL:      github.String("https://github.com/owner/repo/pull/10"),
		CreatedAt:    &github.Timestamp{Time: created},
		UpdatedAt:    &github.Timestamp{Time: created},
		Additions:    github.Int(0),
		Deletions:    github.Int(0),
		ChangedFiles: github.Int(0),
		Head:         &github.PullRequestBranch{Ref: github.String("dev")},
		Base:         &github.PullRequestBranch{Ref: github.String("main")},
		User:         &github.User{Login: github.String("u")},
	}

	desc := FormatCardDescFromPR(pr)
	assert.Contains(t, desc, `"labels": []`)
}

// --- newTestTrelloClient helper ---

func newTestTrelloClient(server *httptest.Server) *Client {
	client := NewClient("api_key", "api_token")
	client.BaseURL = server.URL
	return client
}

func TestCreateCard_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "api_key", r.URL.Query().Get("key"))
		assert.Equal(t, "api_token", r.URL.Query().Get("token"))

		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "list123", body["idList"])
		assert.Equal(t, "Test Card", body["name"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"card123","name":"Test Card","desc":"Description","idList":"list123"}`))
	}))
	defer server.Close()

	client := newTestTrelloClient(server)
	card, err := client.CreateCard("list123", "Test Card", "Description")
	require.NoError(t, err)
	assert.Equal(t, "card123", card.ID)
	assert.Equal(t, "Test Card", card.Name)
}

func TestCreateCard_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"Invalid list"}`))
	}))
	defer server.Close()

	client := newTestTrelloClient(server)
	_, err := client.CreateCard("invalid_list", "Test", "Desc")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create card")
}

func TestCreateCardForIssue_IssueGoesToIssueList(t *testing.T) {
	created, _ := time.Parse(time.RFC3339, "2026-01-01T00:00:00Z")
	var receivedListID string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		receivedListID = body["idList"]

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"card1","name":"Bug fix","desc":"desc","idList":"issue_list"}`))
	}))
	defer server.Close()

	client := newTestTrelloClient(server)
	issue := &github.Issue{
		Number:    github.Int(1),
		Title:     github.String("Bug fix"),
		Body:      github.String("Fix the bug"),
		State:     github.String("open"),
		HTMLURL:   github.String("https://github.com/owner/repo/issues/1"),
		CreatedAt: &github.Timestamp{Time: created},
		UpdatedAt: &github.Timestamp{Time: created},
		User:      &github.User{Login: github.String("user")},
	}

	card, err := client.CreateCardForIssue(issue, "issue_list", "pr_list")
	require.NoError(t, err)
	assert.Equal(t, "card1", card.ID)
	assert.Equal(t, "issue_list", receivedListID, "Issue should be sent to issue list")
}

func TestCreateCardForIssue_PRGoesToPRList(t *testing.T) {
	created, _ := time.Parse(time.RFC3339, "2026-01-01T00:00:00Z")
	var receivedListID string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		receivedListID = body["idList"]

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"card2","name":"Add feature","desc":"desc","idList":"pr_list"}`))
	}))
	defer server.Close()

	client := newTestTrelloClient(server)
	issue := &github.Issue{
		Number:    github.Int(2),
		Title:     github.String("Add feature"),
		Body:      github.String("Feature PR"),
		State:     github.String("open"),
		HTMLURL:   github.String("https://github.com/owner/repo/pull/2"),
		CreatedAt: &github.Timestamp{Time: created},
		UpdatedAt: &github.Timestamp{Time: created},
		User:      &github.User{Login: github.String("user")},
		PullRequestLinks: &github.PullRequestLinks{
			URL:     github.String("https://api.github.com/repos/owner/repo/pulls/2"),
			HTMLURL: github.String("https://github.com/owner/repo/pull/2"),
		},
	}

	card, err := client.CreateCardForIssue(issue, "issue_list", "pr_list")
	require.NoError(t, err)
	assert.Equal(t, "card2", card.ID)
	assert.Equal(t, "pr_list", receivedListID, "PR should be sent to PR list")
}

func TestGetCardsInList_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/lists/list1/cards")
		assert.Equal(t, "desc", r.URL.Query().Get("fields"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"id":"c1","desc":"desc1"},{"id":"c2","desc":"desc2"}]`))
	}))
	defer server.Close()

	client := newTestTrelloClient(server)
	cards, err := client.GetCardsInList("list1")
	require.NoError(t, err)
	assert.Len(t, cards, 2)
	assert.Equal(t, "c1", cards[0].ID)
}

func TestGetCardsInList_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"list not found"}`))
	}))
	defer server.Close()

	client := newTestTrelloClient(server)
	_, err := client.GetCardsInList("bad_list")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get cards")
}

func TestCardExists_Found(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"id":"card1","desc":"https://github.com/owner/repo/issues/123\n\nbody"}]`))
	}))
	defer server.Close()

	client := newTestTrelloClient(server)
	exists, err := client.CardExists("list123", "https://github.com/owner/repo/issues/123")
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestCardExists_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"id":"card1","desc":"https://github.com/other/repo/issues/456"}]`))
	}))
	defer server.Close()

	client := newTestTrelloClient(server)
	exists, err := client.CardExists("list123", "https://github.com/owner/repo/issues/123")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestCardExists_EmptyList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
	}))
	defer server.Close()

	client := newTestTrelloClient(server)
	exists, err := client.CardExists("list123", "https://github.com/owner/repo/issues/123")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestCardExists_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`server error`))
	}))
	defer server.Close()

	client := newTestTrelloClient(server)
	_, err := client.CardExists("list123", "https://github.com/owner/repo/issues/123")
	assert.Error(t, err)
}