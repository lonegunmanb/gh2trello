# gh2trello

Sync GitHub issues and PRs to Trello boards.

## Installation

```bash
go install github.com/lonegunmanb/gh2trello@latest
```

Or build from source:

```bash
git clone https://github.com/lonegunmanb/gh2trello.git
cd gh2trello
go build -o gh2trello .
```

## Quick Start

1. Set up credentials via environment variables or config file:

```bash
export GITHUB_TOKEN="your_github_token"
export TRELLO_API_KEY="your_trello_api_key"
export TRELLO_API_TOKEN="your_trello_api_token"
export TRELLO_ISSUE_LIST_ID="your_issue_list_id"
export TRELLO_PR_LIST_ID="your_pr_list_id"
```

2. Add a repository to monitor:

```bash
gh2trello repo add owner/repo --query "is:issue label:bug"
```

3. Start polling:

```bash
gh2trello repo run
```

Or sync a single work item:

```bash
gh2trello workitem owner/repo 42
```

## Configuration

gh2trello uses a JSON config file (default: `gh2trello.json`).

```json
{
  "github_token": "ghp_...",
  "trello_api_key": "your_key",
  "trello_api_token": "your_token",
  "trello_board_id": "board_id",
  "trello_issue_list_id": "list_id_for_issues",
  "trello_pr_list_id": "list_id_for_prs",
  "poll_interval": "5m",
  "repos": {
    "owner/repo": {
      "since": "2026-03-16T10:00:00Z",
      "query": "label:bug created:>=2026-01-01"
    }
  }
}
```

### Priority

Flag > Config file > Environment variable

## Environment Variables

| Variable | Description |
|----------|-------------|
| `GITHUB_TOKEN` | GitHub personal access token |
| `TRELLO_API_KEY` | Trello API key |
| `TRELLO_API_TOKEN` | Trello API token |
| `TRELLO_BOARD_ID` | Target Trello board ID |
| `TRELLO_ISSUE_LIST_ID` | Trello list ID for issues |
| `TRELLO_PR_LIST_ID` | Trello list ID for PRs |

## Commands

### Global Flags

```
--config string                 Config file path (default "gh2trello.json")
--github-token string           GitHub personal access token
--trello-api-key string         Trello API key
--trello-api-token string       Trello API token
--trello-board-id string        Trello board ID
--trello-issue-list-id string   Trello issue list ID
--trello-pr-list-id string      Trello PR list ID
```

### `repo add`

Add a repository to monitor.

```bash
gh2trello repo add <owner/repo> [--since <RFC3339>] [--query <search-query>]
```

**Examples:**

```bash
# Monitor all new issues and PRs from now
gh2trello repo add lonegunmanb/gh2trello

# Monitor only bugs since a specific date
gh2trello repo add lonegunmanb/gh2trello --since "2026-01-01T00:00:00Z" --query "label:bug is:issue"

# Monitor only approved PRs
gh2trello repo add owner/repo --query "is:pr review:approved"

# Only sync issues created after June 2025
gh2trello repo add owner/repo --query "created:>=2025-06-01"
```

### `repo delete`

Remove a repository from monitoring.

```bash
gh2trello repo delete <owner/repo>
```

### `repo run`

Start continuous polling for all configured repositories.

```bash
gh2trello repo run [--poll-interval <duration>]
```

**Examples:**

```bash
# Poll every 5 minutes (default)
gh2trello repo run

# Poll every 30 seconds
gh2trello repo run --poll-interval 30s
```

### `workitem`

Sync a single GitHub issue or PR to Trello (one-shot).

```bash
gh2trello workitem <owner/repo> <number>
```

**Examples:**

```bash
gh2trello workitem lonegunmanb/gh2trello 42
```

The command automatically detects whether the number refers to an issue or a PR and creates the Trello card in the corresponding list.

## How It Works

### Repo Mode

1. Searches GitHub using the configured query for each repository
2. For each result, computes the "effective time" = max(created_at, latest_comment_time)
3. If effective time > watermark → creates a Trello card
4. Updates the watermark in the config file
5. Repeats after the poll interval

### Workitem Mode

1. Fetches the issue/PR by number from GitHub
2. Checks for duplicates (by URL) in the target Trello list
3. Creates a Trello card in the issue or PR list

### Deduplication

Before creating a card, gh2trello checks if a card with the same GitHub URL already exists in the target list. Duplicates are skipped.

## License

See [LICENSE](LICENSE) for details.
