package github

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/v69/github"
)

// Client wraps the GitHub client.
type Client struct {
	*github.Client
}

// NewClient creates a new GitHub client with the given token.
func NewClient(token string) *Client {
	client := github.NewClient(nil).WithAuthToken(token)
	return &Client{
		Client: client,
	}
}

// BuildSearchQuery builds a GitHub Search API query string.
func BuildSearchQuery(repo string, since string, customQuery string) string {
	// Base query: repo:owner/repo updated:>=timestamp
	query := fmt.Sprintf("repo:%s", repo)

	if since != "" {
		// Format: updated:>=2026-01-01T00:00:00Z
		query += fmt.Sprintf(" updated:>=%s", since)
	}

	// Append custom query if provided
	if customQuery != "" {
		query += " " + customQuery
	}

	return query
}

// IsPR determines if an issue is actually a pull request.
// An issue is a PR if PullRequestLinks is not nil.
func IsPR(issue *github.Issue) bool {
	return issue.PullRequestLinks != nil
}

// GetLastActivityTime returns the later of created_at or updated_at.
// For items with comments, we would need to fetch comments separately.
// This implementation uses updated_at as a proxy for last activity.
func GetLastActivityTime(issue *github.Issue) time.Time {
	created := issue.GetCreatedAt().Time
	updated := issue.GetUpdatedAt().Time

	if updated.After(created) {
		return updated
	}
	return created
}

// GetIssue fetches a single issue by number.
// Returns the issue, or an error if not found.
func (c *Client) GetIssue(ctx context.Context, owner, repo string, number int) (*github.Issue, error) {
	issue, _, err := c.Client.Issues.Get(ctx, owner, repo, number)
	if err != nil {
		return nil, err
	}
	return issue, nil
}

// SearchIssues searches for issues matching the query.
func (c *Client) SearchIssues(ctx context.Context, query string) ([]*github.Issue, error) {
	result, _, err := c.Client.Search.Issues(ctx, query, nil)
	if err != nil {
		return nil, err
	}
	return result.Issues, nil
}

// GetComments fetches comments for an issue.
func (c *Client) GetComments(ctx context.Context, owner, repo string, number int) ([]*github.IssueComment, error) {
	comments, _, err := c.Client.Issues.ListComments(ctx, owner, repo, number, nil)
	if err != nil {
		return nil, err
	}
	return comments, nil
}

// GetLatestCommentTime gets the time of the latest comment on an issue.
// Returns the comment time, or (zero time, false) if no comments exist.
func (c *Client) GetLatestCommentTime(ctx context.Context, owner, repo string, number int) (time.Time, bool) {
	comments, err := c.GetComments(ctx, owner, repo, number)
	if err != nil || len(comments) == 0 {
		return time.Time{}, false
	}

	// Find the latest comment
	var latest time.Time
	for _, comment := range comments {
		commentTime := comment.GetCreatedAt().Time
		if latest.IsZero() || commentTime.After(latest) {
			latest = commentTime
		}
	}

	return latest, !latest.IsZero()
}

// GetEffectiveTime returns the later of created_at or last comment time.
// This is the "effective" time used for watermark comparison.
func (c *Client) GetEffectiveTime(ctx context.Context, issue *github.Issue) time.Time {
	created := issue.GetCreatedAt().Time

	// Try to get latest comment time
	// Extract owner/repo from HTML URL
	owner, repo, number, err := parseHTMLURL(issue.GetHTMLURL())
	if err != nil {
		return created
	}

	commentTime, hasComments := c.GetLatestCommentTime(ctx, owner, repo, number)
	if !hasComments {
		return created
	}

	// Return the later of created or comment time
	if commentTime.After(created) {
		return commentTime
	}
	return created
}

// parseHTMLURL extracts owner, repo, and issue number from HTML URL.
// URL format: https://github.com/owner/repo/issues/123 or https://github.com/owner/repo/pull/123
func parseHTMLURL(htmlURL string) (string, string, int, error) {
	// Trim the prefix
	const prefix = "https://github.com/"
	if !strings.HasPrefix(htmlURL, prefix) {
		return "", "", 0, fmt.Errorf("invalid GitHub URL: %s", htmlURL)
	}
	path := strings.TrimPrefix(htmlURL, prefix)
	// Expected: owner/repo/issues/123 or owner/repo/pull/123
	parts := strings.Split(path, "/")
	if len(parts) < 4 {
		return "", "", 0, fmt.Errorf("invalid GitHub URL format: %s", htmlURL)
	}
	number, err := strconv.Atoi(parts[3])
	if err != nil {
		return "", "", 0, fmt.Errorf("invalid issue/PR number in URL: %s", htmlURL)
	}
	return parts[0], parts[1], number, nil
}