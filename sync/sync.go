package sync

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/lonegunmanb/gh2trello/config"
	gh "github.com/lonegunmanb/gh2trello/github"
	"github.com/lonegunmanb/gh2trello/trello"

	"github.com/google/go-github/v69/github"
)

// GitHubClient defines the GitHub operations needed by the sync engine.
type GitHubClient interface {
	GetIssue(ctx context.Context, owner, repo string, number int) (*github.Issue, error)
	SearchIssues(ctx context.Context, query string) ([]*github.Issue, error)
	GetEffectiveTime(ctx context.Context, issue *github.Issue) time.Time
}

// TrelloClient defines the Trello operations needed by the sync engine.
type TrelloClient interface {
	CreateCardForIssue(issue *github.Issue, issueListID, prListID string) (*trello.Card, error)
	CardExists(listID, url string) (bool, error)
}

// Engine orchestrates syncing GitHub issues/PRs to Trello.
type Engine struct {
	GitHub      GitHubClient
	Trello      TrelloClient
	Config      *config.Config
	ConfigPath  string
}

// NewEngine creates a new sync engine.
func NewEngine(ghClient GitHubClient, trelloClient TrelloClient, cfg *config.Config, configPath string) *Engine {
	return &Engine{
		GitHub:     ghClient,
		Trello:     trelloClient,
		Config:     cfg,
		ConfigPath: configPath,
	}
}

// SyncWorkitem syncs a single GitHub issue/PR to Trello.
func (e *Engine) SyncWorkitem(ctx context.Context, owner, repo string, number int) error {
	issue, err := e.GitHub.GetIssue(ctx, owner, repo, number)
	if err != nil {
		return fmt.Errorf("failed to get issue %s/%s#%d: %w", owner, repo, number, err)
	}

	// Determine target list
	listID := e.Config.TrelloIssueListID
	if gh.IsPR(issue) {
		listID = e.Config.TrelloPRListID
	}

	// Check for duplicates
	exists, err := e.Trello.CardExists(listID, issue.GetHTMLURL())
	if err != nil {
		return fmt.Errorf("failed to check card existence: %w", err)
	}
	if exists {
		return nil
	}

	_, err = e.Trello.CreateCardForIssue(issue, e.Config.TrelloIssueListID, e.Config.TrelloPRListID)
	if err != nil {
		return fmt.Errorf("failed to create Trello card: %w", err)
	}

	return nil
}

// SyncRepo performs a single sync cycle for a repository.
// Returns the number of cards created.
func (e *Engine) SyncRepo(ctx context.Context, repoFullName string) (int, error) {
	repoCfg, ok := e.Config.GetRepo(repoFullName)
	if !ok {
		return 0, fmt.Errorf("repository %s not found in config", repoFullName)
	}

	query := gh.BuildSearchQuery(repoFullName, repoCfg.Since, repoCfg.Query)
	issues, err := e.GitHub.SearchIssues(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to search issues for %s: %w", repoFullName, err)
	}

	var sinceTime time.Time
	if repoCfg.Since != "" {
		sinceTime, _ = time.Parse(time.RFC3339, repoCfg.Since)
	}

	created := 0
	var latestTime time.Time

	for _, issue := range issues {
		effectiveTime := e.GitHub.GetEffectiveTime(ctx, issue)

		// Skip items not newer than watermark
		if !sinceTime.IsZero() && !effectiveTime.After(sinceTime) {
			continue
		}

		// Determine target list
		listID := e.Config.TrelloIssueListID
		if gh.IsPR(issue) {
			listID = e.Config.TrelloPRListID
		}

		// Check dedup
		exists, err := e.Trello.CardExists(listID, issue.GetHTMLURL())
		if err != nil {
			log.Printf("Warning: failed to check card existence for %s: %v", issue.GetHTMLURL(), err)
			continue
		}
		if exists {
			// Still track time for watermark even if skipped
			if effectiveTime.After(latestTime) {
				latestTime = effectiveTime
			}
			continue
		}

		_, err = e.Trello.CreateCardForIssue(issue, e.Config.TrelloIssueListID, e.Config.TrelloPRListID)
		if err != nil {
			log.Printf("Warning: failed to create card for %s: %v", issue.GetHTMLURL(), err)
			continue
		}

		created++
		if effectiveTime.After(latestTime) {
			latestTime = effectiveTime
		}
	}

	// Update watermark if we processed any items
	if !latestTime.IsZero() {
		e.Config.UpdateRepoSince(repoFullName, latestTime)
		if err := config.Save(e.ConfigPath, e.Config); err != nil {
			return created, fmt.Errorf("failed to save config after sync: %w", err)
		}
	}

	return created, nil
}

// RunPolling runs the sync loop for all configured repositories.
// It blocks until the context is cancelled.
func (e *Engine) RunPolling(ctx context.Context) error {
	interval, err := time.ParseDuration(e.Config.PollInterval)
	if err != nil {
		return fmt.Errorf("invalid poll interval %q: %w", e.Config.PollInterval, err)
	}

	// Run immediately on start, then on ticker
	e.syncAllRepos(ctx)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			e.syncAllRepos(ctx)
		}
	}
}

func (e *Engine) syncAllRepos(ctx context.Context) {
	for repoName := range e.Config.Repos {
		parts := strings.SplitN(repoName, "/", 2)
		if len(parts) != 2 {
			log.Printf("Warning: invalid repo name format: %s (expected owner/repo)", repoName)
			continue
		}
		count, err := e.SyncRepo(ctx, repoName)
		if err != nil {
			log.Printf("Error syncing %s: %v", repoName, err)
			continue
		}
		if count > 0 {
			log.Printf("Synced %d new items from %s", count, repoName)
		}
	}
}
