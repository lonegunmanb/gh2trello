package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_Structure(t *testing.T) {
	// Test 1.1.1: Config struct has all required fields
	cfg := &Config{
		GitHubToken:       "gh_token",
		TrelloAPIKey:      "trello_key",
		TrelloAPIToken:    "trello_token",
		TrelloBoardID:     "board_id",
		TrelloIssueListID: "issue_list",
		TrelloPRListID:    "pr_list",
		PollInterval:      "5m",
		Repos:             make(map[string]RepoConfig),
	}

	assert.Equal(t, "gh_token", cfg.GitHubToken)
	assert.Equal(t, "trello_key", cfg.TrelloAPIKey)
	assert.Equal(t, "trello_token", cfg.TrelloAPIToken)
	assert.Equal(t, "board_id", cfg.TrelloBoardID)
	assert.Equal(t, "issue_list", cfg.TrelloIssueListID)
	assert.Equal(t, "pr_list", cfg.TrelloPRListID)
	assert.Equal(t, "5m", cfg.PollInterval)
	assert.NotNil(t, cfg.Repos)
}

func TestRepoConfig_Structure(t *testing.T) {
	// Test 1.1.2: RepoConfig struct has since and query fields
	repoCfg := RepoConfig{
		Since: "2026-01-01T00:00:00Z",
		Query: "label:bug",
	}

	assert.Equal(t, "2026-01-01T00:00:00Z", repoCfg.Since)
	assert.Equal(t, "label:bug", repoCfg.Query)
}

func TestConfig_MarshalUnmarshal(t *testing.T) {
	// Test 1.1.3: JSON marshal/unmarshal
	original := &Config{
		GitHubToken:       "gh_token",
		TrelloAPIKey:      "trello_key",
		TrelloAPIToken:    "trello_token",
		TrelloBoardID:     "board_id",
		TrelloIssueListID: "issue_list",
		TrelloPRListID:    "pr_list",
		PollInterval:      "10m",
		Repos: map[string]RepoConfig{
			"owner/repo1": {
				Since: "2026-01-01T00:00:00Z",
				Query: "label:bug",
			},
			"owner/repo2": {
				Since: "",
				Query: "",
			},
		},
	}

	// Marshal
	data, err := json.Marshal(original)
	require.NoError(t, err)

	// Unmarshal
	var restored Config
	err = json.Unmarshal(data, &restored)
	require.NoError(t, err)

	// Verify
	assert.Equal(t, original.GitHubToken, restored.GitHubToken)
	assert.Equal(t, original.TrelloAPIKey, restored.TrelloAPIKey)
	assert.Equal(t, original.TrelloAPIToken, restored.TrelloAPIToken)
	assert.Equal(t, original.TrelloBoardID, restored.TrelloBoardID)
	assert.Equal(t, original.TrelloIssueListID, restored.TrelloIssueListID)
	assert.Equal(t, original.TrelloPRListID, restored.TrelloPRListID)
	assert.Equal(t, original.PollInterval, restored.PollInterval)
	assert.Equal(t, len(original.Repos), len(restored.Repos))
}

func TestConfig_DefaultValues(t *testing.T) {
	// Test 1.1.4: Default values when fields are missing
	// Write a config file with missing fields
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	jsonContent := `{
  "github_token": "",
  "trello_api_key": ""
}`
	err := os.WriteFile(configPath, []byte(jsonContent), 0644)
	require.NoError(t, err)

	// Load should apply defaults
	cfg, err := Load(configPath)
	require.NoError(t, err)

	// poll_interval should default to 5m
	assert.Equal(t, DefaultPollInterval, cfg.PollInterval)
	// repos should be initialized to empty map
	assert.NotNil(t, cfg.Repos)
}

func TestLoad_ExistingFile(t *testing.T) {
	// Test 1.2.1 & 1.2.3: Load existing config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	jsonContent := `{
  "github_token": "my_token",
  "trello_api_key": "key",
  "trello_api_token": "token",
  "trello_board_id": "board123",
  "trello_issue_list_id": "list1",
  "trello_pr_list_id": "list2",
  "poll_interval": "3m",
  "repos": {
    "owner/repo1": {
      "since": "2026-01-01T00:00:00Z",
      "query": "label:bug"
    }
  }
}`
	err := os.WriteFile(configPath, []byte(jsonContent), 0644)
	require.NoError(t, err)

	cfg, err := Load(configPath)
	require.NoError(t, err)

	assert.Equal(t, "my_token", cfg.GitHubToken)
	assert.Equal(t, "key", cfg.TrelloAPIKey)
	assert.Equal(t, "token", cfg.TrelloAPIToken)
	assert.Equal(t, "board123", cfg.TrelloBoardID)
	assert.Equal(t, "list1", cfg.TrelloIssueListID)
	assert.Equal(t, "list2", cfg.TrelloPRListID)
	assert.Equal(t, "3m", cfg.PollInterval)
	assert.Len(t, cfg.Repos, 1)
}

func TestLoad_NonExistentFile(t *testing.T) {
	// Test 1.2.2: Load non-existent file returns empty config
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nonexistent.json")

	cfg, err := Load(configPath)
	require.NoError(t, err)

	// Should return empty config with defaults
	assert.Equal(t, DefaultPollInterval, cfg.PollInterval)
	assert.NotNil(t, cfg.Repos)
	assert.Empty(t, cfg.Repos)
}

func TestSave_AndReload(t *testing.T) {
	// Test 1.2.3: Save config, reload, verify round-trip
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	original := &Config{
		GitHubToken:       "token123",
		TrelloAPIKey:      "key456",
		TrelloAPIToken:    "token789",
		TrelloBoardID:     "boardabc",
		TrelloIssueListID: "list1",
		TrelloPRListID:    "list2",
		PollInterval:      "7m",
		Repos: map[string]RepoConfig{
			"test/repo": {
				Since: "2026-03-01T00:00:00Z",
				Query: "is:issue",
			},
		},
	}

	// Save
	err := Save(configPath, original)
	require.NoError(t, err)

	// Reload
	reloaded, err := Load(configPath)
	require.NoError(t, err)

	// Verify
	assert.Equal(t, original.GitHubToken, reloaded.GitHubToken)
	assert.Equal(t, original.TrelloAPIKey, reloaded.TrelloAPIKey)
	assert.Equal(t, original.TrelloAPIToken, reloaded.TrelloAPIToken)
	assert.Equal(t, original.TrelloBoardID, reloaded.TrelloBoardID)
	assert.Equal(t, original.TrelloIssueListID, reloaded.TrelloIssueListID)
	assert.Equal(t, original.TrelloPRListID, reloaded.TrelloPRListID)
	assert.Equal(t, original.PollInterval, reloaded.PollInterval)
	assert.Equal(t, len(original.Repos), len(reloaded.Repos))
}

func TestAddRepo_NewRepo(t *testing.T) {
	// Test 1.3.1 & 1.3.2: Add new repo with since = current time when empty
	cfg := &Config{
		Repos: make(map[string]RepoConfig),
	}

	cfg.AddRepo("owner/repo", "", "label:bug")

	repoCfg, ok := cfg.GetRepo("owner/repo")
	require.True(t, ok)
	assert.NotEmpty(t, repoCfg.Since) // Should be set to current time
	assert.Equal(t, "label:bug", repoCfg.Query)
}

func TestAddRepo_ExistingRepo(t *testing.T) {
	// Test 1.3.3: Add repo that already exists → updates
	cfg := &Config{
		Repos: map[string]RepoConfig{
			"owner/repo": {
				Since: "2026-01-01T00:00:00Z",
				Query: "old_query",
			},
		},
	}

	cfg.AddRepo("owner/repo", "2026-02-01T00:00:00Z", "new_query")

	repoCfg, ok := cfg.GetRepo("owner/repo")
	require.True(t, ok)
	assert.Equal(t, "2026-02-01T00:00:00Z", repoCfg.Since)
	assert.Equal(t, "new_query", repoCfg.Query)
}

func TestAddRepo_WithSince(t *testing.T) {
	// Test 1.3.4: Add new repo with since specified
	cfg := &Config{
		Repos: make(map[string]RepoConfig),
	}

	cfg.AddRepo("owner/repo", "2026-01-01T00:00:00Z", "label:enhancement")

	repoCfg, ok := cfg.GetRepo("owner/repo")
	require.True(t, ok)
	assert.Equal(t, "2026-01-01T00:00:00Z", repoCfg.Since)
	assert.Equal(t, "label:enhancement", repoCfg.Query)
}

func TestDeleteRepo_Existing(t *testing.T) {
	// Test 1.4.1 & 1.4.2: Delete existing repo
	cfg := &Config{
		Repos: map[string]RepoConfig{
			"owner/repo": {
				Since: "2026-01-01T00:00:00Z",
				Query: "",
			},
		},
	}

	cfg.DeleteRepo("owner/repo")

	_, ok := cfg.GetRepo("owner/repo")
	assert.False(t, ok)
}

func TestDeleteRepo_NonExistent(t *testing.T) {
	// Test 1.4.3: Delete non-existent repo → no error
	cfg := &Config{
		Repos: make(map[string]RepoConfig),
	}

	// Should not panic or error
	cfg.DeleteRepo("nonexistent/repo")

	_, ok := cfg.GetRepo("nonexistent/repo")
	assert.False(t, ok)
}

func TestUpdateRepoSince(t *testing.T) {
	// Test 4.3.1: Update repo since timestamp
	cfg := &Config{
		Repos: map[string]RepoConfig{
			"owner/repo": {
				Since: "2026-01-01T00:00:00Z",
				Query: "",
			},
		},
	}

	newTime, _ := time.Parse(time.RFC3339, "2026-03-15T10:30:00Z")
	cfg.UpdateRepoSince("owner/repo", newTime)

	repoCfg, ok := cfg.GetRepo("owner/repo")
	require.True(t, ok)
	assert.Equal(t, "2026-03-15T10:30:00Z", repoCfg.Since)
}