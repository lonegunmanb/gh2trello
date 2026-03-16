package config

import (
	"encoding/json"
	"os"
	"time"
)

// Config represents the application configuration.
type Config struct {
	GitHubToken        string                `json:"github_token"`
	TrelloAPIKey       string                `json:"trello_api_key"`
	TrelloAPIToken     string                `json:"trello_api_token"`
	TrelloBoardID      string                `json:"trello_board_id"`
	TrelloIssueListID  string                `json:"trello_issue_list_id"`
	TrelloPRListID     string                `json:"trello_pr_list_id"`
	PollInterval       string                `json:"poll_interval"`
	Repos              map[string]RepoConfig `json:"repos"`
}

// RepoConfig represents the configuration for a single repository.
type RepoConfig struct {
	Since string `json:"since"`
	Query string `json:"query"`
}

// DefaultPollInterval is the default polling interval.
const DefaultPollInterval = "5m"

// Load loads the configuration from the given file path.
// If the file doesn't exist, it returns an empty config.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{
				PollInterval: DefaultPollInterval,
				Repos:        make(map[string]RepoConfig),
			}, nil
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Set defaults
	if cfg.PollInterval == "" {
		cfg.PollInterval = DefaultPollInterval
	}
	if cfg.Repos == nil {
		cfg.Repos = make(map[string]RepoConfig)
	}

	return &cfg, nil
}

// Save saves the configuration to the given file path.
func Save(path string, cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0644)
}

// AddRepo adds or updates a repository in the configuration.
func (c *Config) AddRepo(repo string, since string, query string) {
	if since == "" {
		since = time.Now().Format(time.RFC3339)
	}
	c.Repos[repo] = RepoConfig{
		Since: since,
		Query: query,
	}
}

// DeleteRepo removes a repository from the configuration.
func (c *Config) DeleteRepo(repo string) {
	delete(c.Repos, repo)
}

// GetRepo returns the configuration for a specific repository.
func (c *Config) GetRepo(repo string) (RepoConfig, bool) {
	cfg, ok := c.Repos[repo]
	return cfg, ok
}

// UpdateRepoSince updates the since timestamp for a repository.
func (c *Config) UpdateRepoSince(repo string, timestamp time.Time) {
	if cfg, ok := c.Repos[repo]; ok {
		cfg.Since = timestamp.Format(time.RFC3339)
		c.Repos[repo] = cfg
	}
}