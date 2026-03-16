# gh2trello 任务清单

## 阶段 1: 配置管理

### 1.1 Config 结构体 & JSON 序列化
- [x] **1.1.1** 定义 `Config` 结构体（包含所有字段）
- [x] **1.1.2** 定义 `RepoConfig` 结构体（repos 中的每个仓库配置）
- [x] **1.1.3** 实现 JSON marshal/unmarshal
- [x] **1.1.4** 测试：Config 序列化/反序列化
- [x] **1.1.5** 测试：默认值处理（poll_interval 默认 5m）

### 1.2 配置文件加载
- [x] **1.2.1** 实现 `LoadConfig(path string) (*Config, error)`
- [x] **1.2.2** 实现 `SaveConfig(path string, config *Config) error`
- [x] **1.2.3** 测试：加载存在的配置文件
- [x] **1.2.4** 测试：加载不存在的文件返回空配置

### 1.3 repo add 逻辑
- [x] **1.3.1** 实现 `AddRepo(repo string, since string, query string)` 函数
- [x] **1.3.2** 测试：添加新仓库（since 为空 → 当前时间）
- [x] **1.3.3** 测试：添加已存在的仓库 → 更新
- [x] **1.3.4** 测试：添加新仓库带 query

### 1.4 repo delete 逻辑
- [x] **1.4.1** 实现 `DeleteRepo(repo string)` 函数
- [x] **1.4.2** 测试：删除存在的仓库
- [x] **1.4.3** 测试：删除不存在的仓库 → 无错误

---

## 阶段 2: GitHub 客户端

### 2.1 GitHub 客户端初始化
- [x] **2.1.1** 实现 `NewGitHubClient(token string) *github.Client`
- [x] **2.1.2** 测试：客户端正确初始化

### 2.2 搜索查询构建器
- [x] **2.2.1** 实现 `BuildSearchQuery(repo string, since string, query string) string`
- [x] **2.2.2** 测试：构建 query（repo + since + 自定义 query）
- [x] **2.2.3** 测试：构建 query（仅 repo + since）
- [x] **2.2.4** 测试：构建 query（since 为空）

### 2.3 GitHub 搜索结果分类器
- [x] **2.3.1** 实现 `IsPR(issue *github.Issue) bool` 函数
- [x] **2.3.2** 测试：issue 无 PullRequestLinks → false (Issue)
- [x] **2.3.3** 测试：issue 有 PullRequestLinks → true (PR)

### 2.4 确定最后活动时间
- [x] **2.4.1** 实现 `GetLastActivityTime(issue *github.Issue) time.Time` 函数
- [x] **2.4.2** 测试：无评论 → 使用 created_at
- [x] **2.4.3** 测试：有评论 → 使用最新评论时间
- [x] **2.4.4** 测试：比较 created_at vs last_comment_time，取较晚的

### 2.5 GitHub Issues API (workitem 模式)
- [x] **2.5.1** 实现 `GetIssue(owner, repo string, number int) (*github.Issue, error)`
- [x] **2.5.2** 测试：获取存在的 issue
- [x] **2.5.3** 测试：获取不存在的 issue → 404 错误

---

## 阶段 3: Trello 客户端

### 3.1 Trello 客户端初始化
- [x] **3.1.1** 实现 `NewTrelloClient(apiKey, token string) *TrelloClient`
- [x] **3.1.2** 测试：客户端正确初始化

### 3.2 卡片格式构建器
- [x] **3.2.1** 实现 `FormatCardName(issue *github.Issue) string`
- [x] **3.2.2** 实现 `FormatCardDesc(issue *github.Issue) string`
- [x] **3.2.3** 测试：Issue 卡片格式
- [x] **3.2.4** 测试：PR 卡片格式

### 3.3 Trello 创建卡片
- [x] **3.3.1** 实现 `CreateCard(listId, name, desc string) (*Card, error)`
- [x] **3.3.2** 测试：mock HTTP，验证 POST 请求参数
- [x] **3.3.3** 测试：Issue 发送到 issue list（`CreateCardForIssue` 分发）
- [x] **3.3.4** 测试：PR 发送到 PR list（`CreateCardForIssue` 分发）

### 3.4 Trello 去重检查
- [x] **3.4.1** 实现 `CardExists(listId, url string) (bool, error)`
- [x] **3.4.2** 测试：相同 URL 存在 → true
- [x] **3.4.3** 测试：不同 URL → false

---

## 阶段 4: 同步引擎

### 4.1 单项同步（workitem 模式核心）
- [x] **4.1.1** 实现 `SyncWorkitem(owner, repo string, number int) error`
- [x] **4.1.2** 测试：获取 issue → 创建卡片
- [x] **4.1.3** 测试：获取 PR → 在 PR 列表创建卡片
- [x] **4.1.4** 测试：item 不存在 → 错误

### 4.2 仓库同步（单次轮询周期）
- [x] **4.2.1** 实现 `SyncRepo(repo string, config *RepoConfig) (int, error)`
- [x] **4.2.2** 测试：新 items 找到 → 卡片创建，水位线更新
- [x] **4.2.3** 测试：无新 items → 水位线不变
- [x] **4.2.4** 测试：重复 items → 跳过

### 4.3 水位线更新
- [x] **4.3.1** 实现 `UpdateRepoSince(repo string, timestamp time.Time)` 函数
- [x] **4.3.2** 测试：同步后配置文件的时间戳更新

### 4.4 轮询循环
- [x] **4.4.1** 实现 `RunPolling(interval time.Duration)` 函数
- [x] **4.4.2** 测试：轮询启动，处理一个周期
- [x] **4.4.3** 测试：间隔后重复

---

## 阶段 5: CLI 集成

### 5.1 项目初始化
- [x] **5.1.1** 初始化 Go module: `go mod init github.com/lonegunmanb/gh2trello`
- [x] **5.1.2** 添加依赖: `go-github`, `cobra`

### 5.2 Root 命令
- [x] **5.2.1** 实现 `root.go` - 根命令，全局 flags，配置加载 (Flag > config > env)
- [x] **5.2.2** 测试：全局 flags 正确解析，applyOverride 优先级

### 5.3 repo add 命令
- [x] **5.3.1** 实现 `repo_add.go` - `gh2trello repo add <owner/repo>`
- [x] **5.3.2** 测试：运行 CLI，验证配置文件更新

### 5.4 repo delete 命令
- [x] **5.4.1** 实现 `repo_delete.go` - `gh2trello repo delete <owner/repo>`
- [x] **5.4.2** 测试：运行 CLI，验证配置文件更新

### 5.5 repo run 命令
- [x] **5.5.1** 实现 `repo_run.go` - `gh2trello repo run [--poll-interval]`
- [x] **5.5.2** 测试：验证缺少 token 报错，poll-interval flag 存在

### 5.6 workitem 命令
- [x] **5.6.1** 实现 `workitem.go` - `gh2trello workitem <owner/repo> <number>`
- [x] **5.6.2** 测试：参数校验、缺少 token 报错

---

## 阶段 6: README

### 6.1 编写 README.md
- [x] **6.1.1** 安装说明
- [x] **6.1.2** 快速开始
- [x] **6.1.3** 配置参考
- [x] **6.1.4** 子命令使用示例
- [x] **6.1.5** 环境变量说明

---

## 阶段 7: 集成测试

### 7.1 集成测试准备
- [ ] **7.1.1** 创建测试数据：issues 和 PRs（title 以 `[Please ignore: acceptance test]` 开头）
- [ ] **7.1.2** 配置测试环境变量

### 7.2 集成测试实现
- [ ] **7.2.1** 测试：Search with `is:issue` returns only issues
- [ ] **7.2.2** 测试：Search with `is:pr` returns only PRs
- [ ] **7.2.3** 测试：Search with `label:bug` filters correctly
- [ ] **7.2.4** 测试：Search with `author:` filters correctly
- [ ] **7.2.5** 测试：Search with `state:open` / `state:closed` filters correctly
- [ ] **7.2.6** 测试：Search with combined filters works
- [ ] **7.2.7** 测试：Search with `created:>=` time filter works
- [ ] **7.2.8** 测试：Watermark logic - only items newer than watermark are returned

---

## 阶段 8: GitHub Actions CI

### 8.1 CI Workflow
- [ ] **8.1.1** 创建 `.github/workflows/ci.yml`
- [ ] **8.1.2** 配置：PR 时运行单元测试
- [ ] **8.1.3** 配置：PR 时运行集成测试（可选，需要环境变量）

---

## 任务统计

| 阶段 | 任务数 | 已完成 | 状态 |
|------|--------|--------|------|
| 阶段 1: 配置管理 | 11 | 11 | ✅ 全部完成 |
| 阶段 2: GitHub 客户端 | 11 | 11 | ✅ 全部完成 (100% coverage) |
| 阶段 3: Trello 客户端 | 8 | 8 | ✅ 全部完成 (88.1% coverage) |
| 阶段 4: 同步引擎 | 7 | 7 | ✅ 全部完成 (85.5% coverage) |
| 阶段 5: CLI 集成 | 6 | 6 | ✅ 全部完成 (68.5% coverage) |
| 阶段 6: README | 1 | 1 | ✅ 全部完成 |
| 阶段 7: 集成测试 | 8 | 0 | ❌ 未开始 |
| 阶段 8: CI | 1 | 0 | ❌ 未开始 |
| **总计** | **53** | **42** | |