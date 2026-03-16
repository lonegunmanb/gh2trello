# gh2trello 设计文档

## 1. 项目概述

**项目名称**: gh2trello  
**项目类型**: Go CLI 工具  
**核心功能**: 同步 GitHub 仓库的 Issue 和 PR 到 Trello 看板  
**目标用户**: 需要将 GitHub 工作项同步到 Trello 进行任务管理的开发者

## 2. 运行模式

### 2.1 仓库模式 (Repo Mode)

- **输入**: JSON 配置文件 (`gh2trello.json`)，包含 `owner/repo` -> 时间戳的映射
- **运行方式**: 持续轮询，检测新 Issue/PR
- **轮询间隔**: 默认 5 分钟，可通过 `poll_interval` 配置
- **判断条件**: Issue/PR 的创建时间或最后回复时间（取更接近现在的），如果晚于配置中的时间戳，则同步
- **时间戳更新**: 同步成功后，将该 Issue/PR 的最后回复时间（无回复则用创建时间）写回配置文件
- **去重**: 同步前检查目标列表中是否已存在相同 URL 的卡片

### 2.2 Workitem 模式

- **输入**: `owner/repo` + 编号
- **自动判断**: 先尝试获取 Issue（GitHub Issues API 也会返回 PR，需检查 `pull_request` 字段）
- **运行方式**: 单次同步
- **行为**:
  1. 尝试获取 Issue (`GET /repos/{owner}/{repo}/issues/{number}`)
  2. 检查 `pull_request` 字段确定类型
  3. 在对应列表中创建 Trello 卡片
  4. 退出

## 3. 子命令

| 子命令 | 功能 |
|--------|------|
| `repo add <owner/repo> [--since <time>] [--query <search-query>]` | 添加仓库监控 |
| `repo delete <owner/repo>` | 删除仓库监控 |
| `repo run` | 启动持续轮询同步 |
| `workitem <owner/repo> <number>` | 单次同步指定 Issue/PR |

## 4. 配置

### 4.1 配置文件 (gh2trello.json)

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
    "owner/repo": {
      "since": "2026-03-16T10:00:00Z",
      "query": "label:bug"
    }
  }
}
```

- `repos` 是 map：key = `owner/repo`，value = 包含 `since`（水位线）和 `query`（可选搜索过滤）的对象
- `since` 空字符串表示"从现在开始"（添加时设为当前时间）
- `poll_interval` 是轮询间隔，默认 `5m`

### 4.2 环境变量

| 变量 | 描述 |
|------|------|
| `GITHUB_TOKEN` | GitHub 个人访问令牌 |
| `TRELLO_API_KEY` | Trello API Key |
| `TRELLO_API_TOKEN` | Trello API Token |
| `TRELLO_BOARD_ID` | 目标 Trello 看板 ID |
| `TRELLO_ISSUE_LIST_ID` | Issue 列表 ID |
| `TRELLO_PR_LIST_ID` | PR 列表 ID |

### 4.3 Flag 优先级

Flag > 配置文件 > 环境变量

#### CLI 全局 Flag

| Flag | 描述 | 默认值 |
|------|------|--------|
| `--config` | JSON 配置文件路径 | `gh2trello.json` |
| `--github-token` | GitHub token | 环境变量 |
| `--trello-api-key` | Trello API key | 环境变量 |
| `--trello-api-token` | Trello API token | 环境变量 |
| `--trello-board-id` | Trello 看板 ID | 环境变量 |
| `--trello-issue-list-id` | Issue 列表 ID | 环境变量 |
| `--trello-pr-list-id` | PR 列表 ID | 环境变量 |

> **注意**: `poll_interval` 可通过配置文件或 `--poll-interval` Flag 设置。

#### `repo add` Flag

| Flag | 描述 | 默认值 |
|------|------|--------|
| `--since` | 水位线时间 (RFC3339)。空 = 当前时间 | 当前时间 |
| `--query` | GitHub Search API 查询过滤 | 空（无额外过滤） |

#### `repo run` Flag

| Flag | 描述 | 默认值 |
|------|------|--------|
| `--poll-interval` | 轮询间隔 (如 5m, 30s) | 配置文件中的 poll_interval 或 5m |

## 5. GitHub API 设计

### 5.1 SDK

使用 `github.com/google/go-github/v69/github` SDK。

### 5.2 Search API

使用 `SearchService.Issues()` 方法，支持完整 GitHub Search 查询语法：

```go
client.Search.Issues(ctx, query, &github.SearchOptions{...})
```

查询支持：`repo:`、`is:issue`、`is:pr`、`is:open`、`is:closed`、`author:`、`assignee:`、`label:`、`milestone:`、`created:`、`updated:`、`sort:` 等。

### 5.3 Issue/PR 区分

搜索结果同时包含 Issue 和 PR。通过检查 `issue.PullRequestLinks != nil` 判断：
- 非 nil → PR
- nil → Issue

### 5.4 搜索语法参数

`repo add` 时用户传入搜索 query 字符串，例如：
- `is:issue state:open author:john`
- `is:pr state:closed review:approved`
- `created:>=2026-01-01 updated:>=2026-01-15`

### 5.5 搜索语法支持

- `author:` / `assignee:` / `mentions:` / `commenter:`
- `label:` / `milestone:` / `type:`
- `state:` / `is:issue` / `is:pr` / `is:open` / `is:closed`
- `created:` / `updated:` / `closed:` (支持范围如 `>=2026-01-01`)
- `no:assignee` / `no:label` / `no:milestone`
- `review:` (PR only) — `none` / `required` / `approved` / `changes_requested`
- `team:` / `involves:`
- `sort:` — `created` / `updated` / `comments` / `reactions`
- 支持 `AND` / `OR` / 括号嵌套

## 6. Trello 同步格式

### 6.1 卡片名称
- Issue: GitHub Issue Title
- PR: GitHub PR Title

### 6.2 卡片描述

```
{html_url}

{body}

---
Raw JSON:
{number, title, state, author, labels, created_at, updated_at, html_url, PR 特有: head branch, base branch, additions, deletions, changed_files}
```

### 6.3 去重检查

同步前检查目标列表中是否已存在描述以相同 `html_url` 开头的卡片。如果存在则跳过，避免重复创建。

## 7. 数据流

### 7.1 仓库模式流程

```
1. 读取配置文件
2. 轮询每个仓库:
   a. 使用 Search API 查询符合 query 条件的 Issue/PR
   b. 对每个结果:
      - 计算 "有效时间" = max(createdAt, lastCommentAt)
      - 如果有效时间 > 配置的时间戳 → 同步到 Trello
      - 同步成功后更新配置文件的时间戳为该 Issue/PR 的有效时间
3. 等待间隔后重复
```

### 7.2 Workitem 模式流程

```
1. 读取配置
2. 调用 GitHub API 获取 Issue (GET /repos/{owner}/{repo}/issues/{number})
3. 如果 404，再调用 PR API (GET /repos/{owner}/{repo}/pulls/{number})
4. 同步到 Trello
```

## 8. 目录结构

```
gh2trello/
├── main.go              # CLI 入口 (cobra)
├── cmd/
│   ├── root.go          # 根命令，全局 flags
│   ├── repo_add.go      # repo add 子命令
│   ├── repo_delete.go   # repo delete 子命令
│   ├── repo_run.go      # repo run 子命令
│   └── workitem.go     # workitem 子命令
├── config/
│   ├── config.go        # 配置结构体，加载/保存 JSON
│   └── config_test.go
├── github/
│   ├── client.go        # GitHub API 客户端封装
│   ├── search.go        # 搜索 issues/PRs
│   └── search_test.go
├── trello/
│   ├── client.go        # Trello API 客户端
│   ├── card.go          # 卡片创建，去重
│   └── card_test.go
├── sync/
│   ├── sync.go          # 核心同步逻辑 (GitHub → Trello)
│   ├── sync_test.go
│   └── format.go        # 卡片格式化
├── integration_test/
│   └── integration_test.go  # 集成测试 (build tag: integration)
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

## 9. 验收标准

### 9.1 功能验收

- [ ] `repo add` 能添加仓库监控并保存到配置文件
- [ ] `repo delete` 能删除仓库监控
- [ ] `repo run` 能持续轮询并同步新 Issue/PR
- [ ] `workitem` 能单次同步指定 Issue/PR
- [ ] 能自动判断编号是 Issue 还是 PR
- [ ] 同步后时间戳能正确更新

### 9.2 测试验收

- [ ] 单元测试覆盖核心逻辑
- [ ] 集成测试验证过滤和同步功能
- [ ] GitHub Action 在 PR 时运行测试

## 10. Trello REST API 参考

### 10.1 认证方式

所有 Trello API 请求需要以下认证参数：
- `key`: API Key (通过 `trello_api_key` 配置)
- `token`: Token (通过 `trello_api_token` 配置)

### 10.2 常用端点

#### 获取用户的看板列表
```
GET https://api.trello.com/1/members/me/boards
```
参数: `key`, `token`  
返回: 用户所有看板的列表

#### 获取看板信息
```
GET https://api.trello.com/1/boards/{board_id}
```
参数: `key`, `token`  
返回: 看板详细信息

#### 获取看板下的列表
```
GET https://api.trello.com/1/boards/{board_id}/lists
```
参数: `key`, `token`  
返回: 看板中所有列表

#### 创建卡片
```
POST https://api.trello.com/1/cards
```
参数:
- `key` - API Key
- `token` - Token
- `idList` - 目标列表 ID (必需)
- `name` - 卡片名称 (必需)
- `desc` - 卡片描述
- `pos` - 位置，可选值: `top`, `bottom`, 或数字

#### 获取卡片
```
GET https://api.trello.com/1/cards/{card_id}
```
参数: `key`, `token`

#### 更新卡片
```
PUT https://api.trello.com/1/cards/{card_id}
```
参数: `key`, `token`, 可选: `name`, `desc`, `idList`, `pos`, `closed`

#### 删除卡片
```
DELETE https://api.trello.com/1/cards/{card_id}
```
参数: `key`, `token`

#### 获取列表
```
GET https://api.trello.com/1/lists/{list_id}
```
参数: `key`, `token`

#### 获取列表中的卡片
```
GET https://api.trello.com/1/lists/{list_id}/cards
```
参数: `key`, `token`

### 10.3 请求示例

#### 创建卡片
```bash
curl -X POST "https://api.trello.com/1/cards?key=YOUR_KEY&token=YOUR_TOKEN" \
  -d "idList=LIST_ID" \
  -d "name=Card Title" \
  -d "desc=Card description" \
  -d "pos=bottom"
```

#### 获取看板列表
```bash
curl "https://api.trello.com/1/members/me/boards?key=YOUR_KEY&token=YOUR_TOKEN"
```

### 10.4 错误处理

- 401: 无效或过期的 token
- 400: 请求参数错误
- 404: 资源不存在
- 429: 请求频率超限

### 10.5 速率限制

Trello API 没有严格的速率限制，但建议：
- 每秒不超过 10 个请求
- 批量操作时使用批量 API
- 缓存看板和列表 ID 避免重复请求

## 11. Trello API 设计

### 11.1 SDK

直接使用 Trello REST API（无需 SDK，简单的 HTTP 调用）。

### 11.2 常用端点

#### 创建卡片
```
POST https://api.trello.com/1/cards
```
参数: `idList`, `name`, `desc`, `pos`, `key`, `token`

#### 列表中搜索卡片（去重检查）
```
GET https://api.trello.com/1/lists/{listId}/cards?key={key}&token={token}&fields=desc
```
返回列表中所有卡片的描述，用于检查是否存在相同 URL 的卡片。

### 11.3 请求示例

#### 搜索卡片
```bash
curl "https://api.trello.com/1/lists/{listId}/cards?key=...&token=...&fields=desc"
```

## 12. 开发流程 — 严格 TDD

**每个功能必须遵循测试驱动开发：**

1. **先写失败的测试**
2. **运行测试确认失败 (RED)**
3. **编写最少量代码使其通过 (GREEN)**
4. **如有需要则重构 (REFACTOR)**
5. **提交并附清晰信息**

### 12.1 实现顺序

#### 阶段 1: 配置管理

1. Config 结构体 & JSON 序列化
2. 从文件加载配置
3. 保存配置到文件
4. repo add 逻辑
5. repo delete 逻辑

#### 阶段 2: GitHub 客户端

6. GitHub 搜索查询构建器
7. GitHub 搜索结果分类器（Issue vs PR）
8. 确定最后活动时间

#### 阶段 3: Trello 客户端

9. 卡片格式构建器
10. Trello 创建卡片
11. Trello 去重检查

#### 阶段 4: 同步引擎

12. 单项同步（workitem 模式核心）
13. 仓库同步（单次轮询周期）
14. 水位线更新

#### 阶段 5: CLI 集成

15. Wire `repo add` 命令
16. Wire `repo delete` 命令
17. Wire `workitem` 命令
18. Wire `repo run` 命令

#### 阶段 6: README

19. 编写完整的 README.md

#### 阶段 7: 集成测试

20. 集成测试 (build tag: `//go:build integration`)
    - 针对真实 GitHub API 测试
    - 测试搜索过滤功能
    - 测试水位线逻辑

#### 阶段 8: GitHub Actions CI

21. 创建 CI workflow