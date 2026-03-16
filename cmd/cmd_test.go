package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/lonegunmanb/gh2trello/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =========================================================================
// Root command / global flags tests
// =========================================================================

func TestRootCmd_HasSubcommands(t *testing.T) {
	names := make([]string, 0)
	for _, c := range rootCmd.Commands() {
		names = append(names, c.Name())
	}
	assert.Contains(t, names, "repo")
	assert.Contains(t, names, "workitem")
}

func TestRootCmd_GlobalFlags(t *testing.T) {
	flags := rootCmd.PersistentFlags()

	f := flags.Lookup("config")
	require.NotNil(t, f)
	assert.Equal(t, "gh2trello.json", f.DefValue)

	for _, name := range []string{"github-token", "trello-api-key", "trello-api-token", "trello-board-id", "trello-issue-list-id", "trello-pr-list-id"} {
		f := flags.Lookup(name)
		require.NotNil(t, f, "flag %s should exist", name)
	}
}

func TestRepoCmd_HasSubcommands(t *testing.T) {
	names := make([]string, 0)
	for _, c := range repoCmd.Commands() {
		names = append(names, c.Name())
	}
	assert.Contains(t, names, "add")
	assert.Contains(t, names, "delete")
	assert.Contains(t, names, "run")
}

// =========================================================================
// applyOverride tests
// =========================================================================

func TestApplyOverride_FlagWins(t *testing.T) {
	target := "original"
	applyOverride(&target, "from_flag", "NONEXISTENT_ENV_VAR_TEST")
	assert.Equal(t, "from_flag", target)
}

func TestApplyOverride_EnvFallback(t *testing.T) {
	t.Setenv("TEST_APPLY_OVERRIDE_ENV", "from_env")
	target := ""
	applyOverride(&target, "", "TEST_APPLY_OVERRIDE_ENV")
	assert.Equal(t, "from_env", target)
}

func TestApplyOverride_ConfigKept(t *testing.T) {
	target := "from_config"
	applyOverride(&target, "", "NONEXISTENT_ENV_VAR_TEST")
	assert.Equal(t, "from_config", target)
}

// =========================================================================
// repo add tests
// =========================================================================

func TestRepoAdd_CreatesConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test.json")

	// Reset global state for test
	oldCfgPath := cfgPath
	defer func() { cfgPath = oldCfgPath }()
	cfgPath = configFile

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"repo", "add", "owner/myrepo", "--config", configFile, "--since", "2026-01-01T00:00:00Z", "--query", "label:bug"})

	err := rootCmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "Added repository: owner/myrepo")

	// Verify config file
	cfg, err := config.Load(configFile)
	require.NoError(t, err)
	repoCfg, ok := cfg.GetRepo("owner/myrepo")
	require.True(t, ok)
	assert.Equal(t, "2026-01-01T00:00:00Z", repoCfg.Since)
	assert.Equal(t, "label:bug", repoCfg.Query)
}

func TestRepoAdd_DefaultSince(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test.json")

	oldCfgPath := cfgPath
	defer func() { cfgPath = oldCfgPath }()
	cfgPath = configFile

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"repo", "add", "owner/repo2", "--config", configFile})

	err := rootCmd.Execute()
	require.NoError(t, err)

	cfg, err := config.Load(configFile)
	require.NoError(t, err)
	repoCfg, ok := cfg.GetRepo("owner/repo2")
	require.True(t, ok)
	assert.NotEmpty(t, repoCfg.Since, "Since should default to current time")
}

func TestRepoAdd_MissingArg(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"repo", "add"})

	err := rootCmd.Execute()
	assert.Error(t, err)
}

// =========================================================================
// repo delete tests
// =========================================================================

func TestRepoDelete_RemovesRepo(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test.json")

	// Pre-create config with a repo
	cfg := &config.Config{
		PollInterval: "5m",
		Repos: map[string]config.RepoConfig{
			"owner/repo": {Since: "2026-01-01T00:00:00Z"},
		},
	}
	require.NoError(t, config.Save(configFile, cfg))

	oldCfgPath := cfgPath
	defer func() { cfgPath = oldCfgPath }()
	cfgPath = configFile

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"repo", "delete", "owner/repo", "--config", configFile})

	err := rootCmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "Deleted repository: owner/repo")

	// Verify repo removed from config
	loaded, err := config.Load(configFile)
	require.NoError(t, err)
	_, ok := loaded.GetRepo("owner/repo")
	assert.False(t, ok, "repo should be deleted")
}

func TestRepoDelete_MissingArg(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"repo", "delete"})

	err := rootCmd.Execute()
	assert.Error(t, err)
}

// =========================================================================
// repo run tests
// =========================================================================

func TestRepoRun_MissingGitHubToken(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test.json")
	require.NoError(t, config.Save(configFile, &config.Config{
		PollInterval: "5m",
		Repos:        map[string]config.RepoConfig{},
	}))

	// Clear env to ensure no token
	t.Setenv("GITHUB_TOKEN", "")

	oldCfgPath := cfgPath
	oldGhToken := githubToken
	defer func() { cfgPath = oldCfgPath; githubToken = oldGhToken }()
	cfgPath = configFile
	githubToken = ""

	buf := new(bytes.Buffer)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"repo", "run", "--config", configFile})

	err := rootCmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "github token is required")
}

func TestRepoRun_MissingTrelloKeys(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test.json")
	require.NoError(t, config.Save(configFile, &config.Config{
		GitHubToken:  "gh_token",
		PollInterval: "5m",
		Repos:        map[string]config.RepoConfig{},
	}))

	t.Setenv("TRELLO_API_KEY", "")
	t.Setenv("TRELLO_API_TOKEN", "")

	oldCfgPath := cfgPath
	oldGhToken := githubToken
	oldTrelloKey := trelloAPIKey
	oldTrelloToken := trelloAPIToken
	defer func() {
		cfgPath = oldCfgPath
		githubToken = oldGhToken
		trelloAPIKey = oldTrelloKey
		trelloAPIToken = oldTrelloToken
	}()
	cfgPath = configFile
	githubToken = ""
	trelloAPIKey = ""
	trelloAPIToken = ""

	buf := new(bytes.Buffer)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"repo", "run", "--config", configFile})

	err := rootCmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "trello API key and token are required")
}

func TestRepoRun_PollIntervalFlag(t *testing.T) {
	f := repoRunCmd.Flags().Lookup("poll-interval")
	require.NotNil(t, f)
}

// =========================================================================
// workitem tests
// =========================================================================

func TestWorkitem_InvalidRepoFormat(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test.json")
	require.NoError(t, config.Save(configFile, &config.Config{
		GitHubToken:       "gh_token",
		TrelloAPIKey:      "tk",
		TrelloAPIToken:    "tt",
		TrelloIssueListID: "il",
		TrelloPRListID:    "pl",
		PollInterval:      "5m",
		Repos:             map[string]config.RepoConfig{},
	}))

	oldCfgPath := cfgPath
	oldGhToken := githubToken
	oldTrelloKey := trelloAPIKey
	oldTrelloToken := trelloAPIToken
	oldIssueList := trelloIssueList
	oldPRList := trelloPRList
	defer func() {
		cfgPath = oldCfgPath
		githubToken = oldGhToken
		trelloAPIKey = oldTrelloKey
		trelloAPIToken = oldTrelloToken
		trelloIssueList = oldIssueList
		trelloPRList = oldPRList
	}()
	cfgPath = configFile
	githubToken = ""
	trelloAPIKey = ""
	trelloAPIToken = ""
	trelloIssueList = ""
	trelloPRList = ""

	buf := new(bytes.Buffer)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"workitem", "badformat", "42", "--config", configFile})

	err := rootCmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid repository format")
}

func TestWorkitem_InvalidNumber(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test.json")
	require.NoError(t, config.Save(configFile, &config.Config{
		GitHubToken:       "gh_token",
		TrelloAPIKey:      "tk",
		TrelloAPIToken:    "tt",
		TrelloIssueListID: "il",
		TrelloPRListID:    "pl",
		PollInterval:      "5m",
		Repos:             map[string]config.RepoConfig{},
	}))

	oldCfgPath := cfgPath
	oldGhToken := githubToken
	oldTrelloKey := trelloAPIKey
	oldTrelloToken := trelloAPIToken
	oldIssueList := trelloIssueList
	oldPRList := trelloPRList
	defer func() {
		cfgPath = oldCfgPath
		githubToken = oldGhToken
		trelloAPIKey = oldTrelloKey
		trelloAPIToken = oldTrelloToken
		trelloIssueList = oldIssueList
		trelloPRList = oldPRList
	}()
	cfgPath = configFile
	githubToken = ""
	trelloAPIKey = ""
	trelloAPIToken = ""
	trelloIssueList = ""
	trelloPRList = ""

	buf := new(bytes.Buffer)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"workitem", "owner/repo", "abc", "--config", configFile})

	err := rootCmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid issue/PR number")
}

func TestWorkitem_MissingArgs(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"workitem"})

	err := rootCmd.Execute()
	assert.Error(t, err)
}

func TestWorkitem_MissingTokens(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test.json")
	require.NoError(t, config.Save(configFile, &config.Config{
		PollInterval: "5m",
		Repos:        map[string]config.RepoConfig{},
	}))

	t.Setenv("GITHUB_TOKEN", "")

	oldCfgPath := cfgPath
	oldGhToken := githubToken
	defer func() { cfgPath = oldCfgPath; githubToken = oldGhToken }()
	cfgPath = configFile
	githubToken = ""

	buf := new(bytes.Buffer)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"workitem", "owner/repo", "42", "--config", configFile})

	err := rootCmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "github token is required")
}

// =========================================================================
// loadConfig with env vars
// =========================================================================

func TestLoadConfig_EnvOverrides(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test.json")

	// Write an empty config
	require.NoError(t, os.WriteFile(configFile, []byte(`{"poll_interval": "5m"}`), 0644))

	t.Setenv("GITHUB_TOKEN", "env_gh_token")
	t.Setenv("TRELLO_API_KEY", "env_trello_key")
	t.Setenv("TRELLO_API_TOKEN", "env_trello_token")
	t.Setenv("TRELLO_BOARD_ID", "env_board")
	t.Setenv("TRELLO_ISSUE_LIST_ID", "env_issue_list")
	t.Setenv("TRELLO_PR_LIST_ID", "env_pr_list")

	oldCfgPath := cfgPath
	oldGhToken := githubToken
	oldTrelloKey := trelloAPIKey
	oldTrelloToken := trelloAPIToken
	oldBoardID := trelloBoardID
	oldIssueList := trelloIssueList
	oldPRList := trelloPRList
	defer func() {
		cfgPath = oldCfgPath
		githubToken = oldGhToken
		trelloAPIKey = oldTrelloKey
		trelloAPIToken = oldTrelloToken
		trelloBoardID = oldBoardID
		trelloIssueList = oldIssueList
		trelloPRList = oldPRList
	}()
	cfgPath = configFile
	githubToken = ""
	trelloAPIKey = ""
	trelloAPIToken = ""
	trelloBoardID = ""
	trelloIssueList = ""
	trelloPRList = ""

	cfg, err := loadConfig()
	require.NoError(t, err)
	assert.Equal(t, "env_gh_token", cfg.GitHubToken)
	assert.Equal(t, "env_trello_key", cfg.TrelloAPIKey)
	assert.Equal(t, "env_trello_token", cfg.TrelloAPIToken)
	assert.Equal(t, "env_board", cfg.TrelloBoardID)
	assert.Equal(t, "env_issue_list", cfg.TrelloIssueListID)
	assert.Equal(t, "env_pr_list", cfg.TrelloPRListID)
}

func TestLoadConfig_FlagOverridesAll(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test.json")
	require.NoError(t, config.Save(configFile, &config.Config{
		GitHubToken: "config_token",
		PollInterval: "5m",
		Repos: map[string]config.RepoConfig{},
	}))

	t.Setenv("GITHUB_TOKEN", "env_token")

	oldCfgPath := cfgPath
	oldGhToken := githubToken
	defer func() { cfgPath = oldCfgPath; githubToken = oldGhToken }()
	cfgPath = configFile
	githubToken = "flag_token"

	cfg, err := loadConfig()
	require.NoError(t, err)
	assert.Equal(t, "flag_token", cfg.GitHubToken, "flag should override config and env")
}
