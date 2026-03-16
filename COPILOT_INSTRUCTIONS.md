# gh2trello — Requirements & Implementation Guide

## Overview

`gh2trello` is a Go CLI tool that syncs GitHub issues and pull requests to Trello boards. It has two modes: **repo mode** (continuous polling) and **workitem mode** (one-shot sync).

## Architecture

### Subcommands

The CLI has exactly 4 subcommands:

```
gh2trello repo add <owner/repo> [flags]     # Add a repo to the watch config
gh2trello repo delete <owner/repo>          # Remove a repo from the watch config
gh2trello repo run                          # Start continuous polling and syncing
gh2trello workitem <owner/repo> <number>    # One-shot sync of a single issue or PR
```

### Configuration

All configuration can be provided via:
1. **Environment variables** (default, highest priority for secrets)
2. **JSON config file** (specified via `--config` flag, default `gh2trello.json`)
3. **CLI flags** (override config file values)

#### Environment Variables

| Variable | Description |
|----------|-------------|
| `GITHUB_TOKEN` | GitHub personal access token |
| `TRELLO_API_KEY` | Trello API key |
| `TRELLO_API_TOKEN` | Trello API token |
| `TRELLO_BOARD_ID` | Target Trello board ID |
| `TRELLO_ISSUE_LIST_ID` | Trello list ID for issues |
| `TRELLO_PR_LIST_ID` | Trello list ID for PRs |

#### CLI Flags (global)

| Flag | Description | Default |
|------|-------------|---------|
| `--config` | Path to JSON config file | `gh2trello.json` |
| `--github-token` | GitHub token | from env |
| `--trello-api-key` | Trello API key | from env |
| `--trello-api-token` | Trello API token | from env |
| `--trello-board-id` | Trello board ID | from env |
| `--trello-issue-list-id` | Trello list ID for issues | from env |
| `--trello-pr-list-id` | Trello list ID for PRs | from env |

#### `repo add` Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--since` | Watermark time (RFC3339). Empty = now | current time |
| `--query` | GitHub Search API query filter (e.g. `label:bug author:octocat`) | empty (no extra filter) |

### JSON Config File Format

```json
{
  "github_token": "",
  "trello_api_key": "",
  "trello_api_token": "",
  "trello_board_id": "",
  "trello_issue_list_id": "",
  "trello_pr_list_id": "",
  "poll_interval": "5m",
  "repos": {
    "Azure/terraform-azurerm-aks": {
      "since": "2026-03-16T10:00:00Z",
      "query": "label:bug"
    },
    "Azure/terraform-azurerm-avm-res-app-containerapp": {
      "since": "2026-03-15T00:00:00Z",
      "query": ""
    }
  }
}
```

- `repos` is a map: key = `owner/repo`, value = object with `since` (watermark) and `query` (optional search filter)
- `since` empty string means "from now" (set to current time when added)
- `poll_interval` is the polling interval for `repo run`, default `5m`

## Mode Details

### 1. Repo Mode — `repo add`

Add a repository to the config file. Only modifies the JSON, does not start polling.

```
gh2trello repo add Azure/terraform-azurerm-aks --since "2026-01-01T00:00:00Z" --query "label:bug"
gh2trello repo add Azure/terraform-azurerm-aks  # since = now, no extra filter
```

**Behavior:**
- Read config file → add/update the repo entry → write config file
- If `--since` is empty or omitted, set to current time
- If repo already exists, update its `since` and `query`

### 2. Repo Mode — `repo delete`

Remove a repository from the config file.

```
gh2trello repo delete Azure/terraform-azurerm-aks
```

**Behavior:**
- Read config file → remove the repo entry → write config file
- If repo doesn't exist, print warning and exit 0

### 3. Repo Mode — `repo run`

Start continuous polling. For each configured repo:

1. Build a GitHub Search API query:
   - Base: `repo:<owner/repo> updated:>={since}`
   - Append user's custom query if present
   - Search with `is:issue` and `is:pr` separately (or search all and split by `pull_request` field presence)
2. For each result, check: `max(created_at, last_comment_time) > since`
   - `last_comment_time`: fetch via the issue/PR timeline or comments API
   - If no comments, use `created_at`
3. If the condition is met, create a Trello card:
   - **List**: issues go to `trello_issue_list_id`, PRs go to `trello_pr_list_id`
   - **Card name**: GitHub title
   - **Card description**: `{html_url}\n\n{body}\n\n{raw_json}`
   - **Position**: bottom
4. After successful sync, update the repo's `since` in the config file to `max(created_at, last_comment_time)` of the synced item
5. Sleep for `poll_interval`, then repeat

**Important:** The `since` watermark is a **dynamic high-water mark**. After each successful sync, it advances to prevent re-syncing the same items.

**Deduplication:** Before creating a card, search existing cards in the target list for a card whose description starts with the same `html_url`. If found, skip (don't create duplicate).

### 4. Workitem Mode — `workitem`

One-shot sync of a single GitHub item.

```
gh2trello workitem Azure/terraform-azurerm-aks 722
```

**Behavior:**
1. Try to fetch as issue first (`GET /repos/{owner}/{repo}/issues/{number}`)
   - Note: GitHub Issues API returns PRs too; check `pull_request` field to determine type
2. Create Trello card in the appropriate list (issue list or PR list)
3. Same card format as repo mode
4. Exit

## Trello Card Format

For both modes, the card format is:

- **Name**: `{title}` (the GitHub issue/PR title)
- **Description**:
  ```
  {html_url}

  {body}

  {raw_json}
  ```
  Where `raw_json` is the JSON representation of key fields (number, title, state, author, labels, created_at, updated_at, html_url, and for PRs: head branch, base branch, additions, deletions, changed_files)
- **List**: Issues → `trello_issue_list_id`, PRs → `trello_pr_list_id`
- **Position**: bottom

## GitHub API

Use the `github.com/google/go-github/v69/github` SDK.

For repo mode search, use `SearchService.Issues()` which accepts the full GitHub Search query syntax:
```go
client.Search.Issues(ctx, query, &github.SearchOptions{...})
```

The query supports: `repo:`, `is:issue`, `is:pr`, `is:open`, `is:closed`, `author:`, `assignee:`, `label:`, `milestone:`, `created:`, `updated:`, `sort:`, etc.

Results contain both issues and PRs. Distinguish by checking if `issue.PullRequestLinks != nil` (it's a PR) or `nil` (it's an issue).

## Trello API

Use Trello REST API directly (no SDK needed, simple HTTP calls):

- **Create card**: `POST https://api.trello.com/1/cards` with `idList`, `name`, `desc`, `pos`, `key`, `token`
- **Search cards in list**: `GET https://api.trello.com/1/lists/{listId}/cards?key=...&token=...&fields=desc` for dedup check

## Project Structure

```
gh2trello/
├── main.go              # CLI entry point (cobra)
├── cmd/
│   ├── root.go          # Root command, global flags
│   ├── repo_add.go      # repo add subcommand
│   ├── repo_delete.go   # repo delete subcommand
│   ├── repo_run.go      # repo run subcommand
│   └── workitem.go      # workitem subcommand
├── config/
│   ├── config.go        # Config struct, load/save JSON
│   └── config_test.go
├── github/
│   ├── client.go        # GitHub API client wrapper
│   ├── search.go        # Search issues/PRs
│   └── search_test.go
├── trello/
│   ├── client.go        # Trello API client
│   ├── card.go          # Card creation, dedup
│   └── card_test.go
├── sync/
│   ├── sync.go          # Core sync logic (GitHub → Trello)
│   ├── sync_test.go
│   └── format.go        # Card formatting
├── integration_test/
│   └── integration_test.go  # Integration tests (build tag: integration)
├── .github/
│   └── workflows/
│       └── ci.yml       # GitHub Actions CI
├── go.mod
├── go.sum
├── gh2trello.json.example
├── COPILOT_INSTRUCTIONS.md
├── LICENSE
└── README.md
```

## Development Process — STRICT TDD

**You MUST follow Test-Driven Development strictly. For every feature:**

1. **Write the failing test FIRST**
2. **Run the test — confirm it fails (RED)**
3. **Write the minimum code to make it pass (GREEN)**
4. **Refactor if needed (REFACTOR)**
5. **Commit with a clear message**

### Implementation Order

Follow this order exactly. Each step = one TDD cycle = one commit.

#### Phase 1: Config Management

1. **Config struct & JSON serialization**
   - Test: marshal/unmarshal config to/from JSON
   - Test: default values when fields are missing

2. **Config load from file**
   - Test: load existing config file
   - Test: load non-existent file returns empty config

3. **Config save to file**
   - Test: save config, reload, verify round-trip

4. **repo add logic**
   - Test: add new repo with since and query
   - Test: add repo with empty since → uses current time
   - Test: add repo that already exists → updates

5. **repo delete logic**
   - Test: delete existing repo
   - Test: delete non-existent repo → no error

#### Phase 2: GitHub Client

6. **GitHub search query builder**
   - Test: build query with repo + since + custom query
   - Test: build query with repo + since only (no custom query)
   - Test: build query with empty since

7. **GitHub search result classifier**
   - Test: classify result as issue (no PullRequestLinks)
   - Test: classify result as PR (has PullRequestLinks)

8. **Determine last activity time**
   - Test: item with no comments → use created_at
   - Test: item with comments → use latest comment time
   - Test: compare created_at vs last_comment_time, pick the later one

#### Phase 3: Trello Client

9. **Card format builder**
   - Test: format issue card (name, description with url + body + json)
   - Test: format PR card (includes PR-specific fields in json)

10. **Trello create card**
    - Test: mock HTTP, verify correct POST body
    - Test: verify issues go to issue list, PRs go to PR list

11. **Trello dedup check**
    - Test: card with same URL exists → skip
    - Test: card with different URL → create

#### Phase 4: Sync Engine

12. **Single item sync (workitem mode core)**
    - Test: fetch issue by number → create card
    - Test: fetch PR by number → create card in PR list
    - Test: item doesn't exist → error

13. **Repo sync (single poll cycle)**
    - Test: new items found → cards created, watermark updated
    - Test: no new items → watermark unchanged
    - Test: duplicate items → skipped

14. **Watermark update**
    - Test: after sync, config file's since is updated to latest activity time

#### Phase 5: CLI Wiring

15. **Wire `repo add` command**
    - Test: run CLI with args, verify config file updated

16. **Wire `repo delete` command**
    - Test: run CLI with args, verify config file updated

17. **Wire `workitem` command**
    - Test: run CLI with args (mock GitHub + Trello)

18. **Wire `repo run` command**
    - Test: starts polling, processes one cycle (mock GitHub + Trello)

#### Phase 6: README

19. **Write comprehensive README.md**
    - Installation
    - Quick start
    - Configuration reference
    - Subcommand usage with examples
    - Environment variables

**After Phase 6, commit everything and push. Then STOP and report completion.**

#### Phase 7: Integration Tests (will be triggered separately after test data is created)

20. **Integration tests** (build tag: `//go:build integration`)
    - These test against real GitHub API using the repo `lonegunmanb/gh2trello`
    - Test data (issues and PRs) will be created in that repo before this phase
    - Tests verify:
      - Search with `is:issue` returns only issues
      - Search with `is:pr` returns only PRs
      - Search with `label:bug` filters correctly
      - Search with `author:` filters correctly
      - Search with `state:open` / `state:closed` filters correctly
      - Search with combined filters works
      - Search with `created:>=` time filter works
      - Watermark logic: only items newer than watermark are returned
    - Environment variables needed: `GITHUB_TOKEN`, `TRELLO_API_KEY`, `TRELLO_API_TOKEN`, `TRELLO_BOARD_ID`, `TRELLO_ISSUE_LIST_ID`, `TRELLO_PR_LIST_ID`
    - Run with: `go test -tags integration ./integration_test/...`

#### Phase 8: GitHub Actions CI

21. **CI workflow** (`.github/workflows/ci.yml`)
    - Trigger: pull_request
    - Jobs:
      - **unit-tests**: `go test ./...` (excludes integration tests)
      - **integration-tests**: `go test -tags integration ./integration_test/...`
        - Needs secrets: `GITHUB_TOKEN`, `TRELLO_API_KEY`, `TRELLO_API_TOKEN`, `TRELLO_BOARD_ID`, `TRELLO_ISSUE_LIST_ID`, `TRELLO_PR_LIST_ID`

## Key Libraries

- **CLI framework**: `github.com/spf13/cobra`
- **GitHub SDK**: `github.com/google/go-github/v69/github` (use latest stable version, check `go get`)
- **HTTP mocking for tests**: `net/http/httptest`
- **Assertions**: standard `testing` package (or `github.com/stretchr/testify` if preferred)

## Important Notes

1. **`go mod init github.com/lonegunmanb/gh2trello`** — run this first before any Go code
2. **Every test must be runnable independently** — no test ordering dependencies
3. **Unit tests must not make real API calls** — use interfaces and mocks
4. **Integration tests are gated by build tag `integration`** — they DO make real API calls
5. **Config file must be safe for concurrent read/write** — use file locking or atomic write (write to temp file, rename)
6. **All errors must be wrapped with context** — use `fmt.Errorf("...: %w", err)`
7. **Use `context.Context` throughout** — for cancellation support in `repo run`
