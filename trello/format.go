package trello

import (
	"encoding/json"
	"fmt"

	"github.com/google/go-github/v69/github"
)

// FormatCardName returns the card name from a GitHub issue/PR title.
func FormatCardName(issue *github.Issue) string {
	return issue.GetTitle()
}

// FormatCardDesc returns the card description from a GitHub issue/PR.
// Format: {html_url}\n\n{body}\n\n{raw_json}
func FormatCardDesc(issue *github.Issue) string {
	// Build JSON representation
	jsonData := map[string]interface{}{
		"number":    issue.GetNumber(),
		"title":     issue.GetTitle(),
		"state":     issue.GetState(),
		"author":    issue.GetUser().GetLogin(),
		"labels":    getLabelNames(issue),
		"created_at": issue.GetCreatedAt().String(),
		"updated_at": issue.GetUpdatedAt().String(),
		"html_url":  issue.GetHTMLURL(),
	}

	// Add PR-specific fields if this is a PR
	if issue.PullRequestLinks != nil {
		// For PR-specific info, we'd need to fetch the PR separately
		// For now, indicate it's a PR
		jsonData["is_pr"] = true
	}

	jsonBytes, _ := json.MarshalIndent(jsonData, "", "  ")

	return fmt.Sprintf("%s\n\n%s\n\n---\nRaw JSON:\n%s",
		issue.GetHTMLURL(),
		issue.GetBody(),
		string(jsonBytes))
}

// FormatCardDescFromPR returns the card description from a GitHub PR.
// This provides more PR-specific details.
func FormatCardDescFromPR(pr *github.PullRequest) string {
	jsonData := map[string]interface{}{
		"number":         pr.GetNumber(),
		"title":          pr.GetTitle(),
		"state":          pr.GetState(),
		"author":         pr.GetUser().GetLogin(),
		"labels":         getPRLabelNames(pr),
		"created_at":     pr.GetCreatedAt().String(),
		"updated_at":     pr.GetUpdatedAt().String(),
		"html_url":       pr.GetHTMLURL(),
		"head_branch":    pr.GetHead().GetRef(),
		"base_branch":    pr.GetBase().GetRef(),
		"additions":      pr.GetAdditions(),
		"deletions":      pr.GetDeletions(),
		"changed_files":  pr.GetChangedFiles(),
		"is_pr":          true,
	}

	jsonBytes, _ := json.MarshalIndent(jsonData, "", "  ")

	return fmt.Sprintf("%s\n\n%s\n\n---\nRaw JSON:\n%s",
		pr.GetHTMLURL(),
		pr.GetBody(),
		string(jsonBytes))
}

// getLabelNames extracts label names from an issue.
func getLabelNames(issue *github.Issue) []string {
	names := make([]string, 0, len(issue.Labels))
	for _, label := range issue.Labels {
		names = append(names, label.GetName())
	}
	return names
}

// getPRLabelNames extracts label names from a PR.
func getPRLabelNames(pr *github.PullRequest) []string {
	names := make([]string, 0, len(pr.Labels))
	for _, label := range pr.Labels {
		names = append(names, label.GetName())
	}
	return names
}