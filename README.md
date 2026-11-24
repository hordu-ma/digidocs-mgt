# DigiDocs Mgt

面向课题组的文档资产管理平台，支持文件级版本、课题权限、流转、毕业交接、审计、数据与代码资产，以及 DocsAgent 摘要和问答。

## 部署与使用

- **首次客户部署**从[现场部署手册](docs/现场部署逐条清单.md)开始。
- **运维与验收**见[运行参考](docs/部署准备与运行说明.md)和[优化清单](docs/客户交付优化清单.md)。
- **用户操作**见[用户使用说明](docs/用户使用说明.md)。
- **设计、开发与历史资料**见[文档索引](docs/INDEX.md)。

标准交付在 DGX Spark 用 Compose 运行 Vue/Nginx、Go API、Python Worker 和 PostgreSQL 17 + pgvector。群晖负责 File Station 文件存储与异机备份。DocsAgent 通过客户批准的 OpenAI 兼容模型端点运行。

## 开发与验证

`backend-go/` 维护主业务、权限、审计和迁移。 `backend-py-worker/` 处理正文抽取、模型请求和附属结果。 `frontend/` 维护用户界面。协作和契约更新遵循 [AGENTS.md](AGENTS.md)，项目技能维护在 `ops/codex/skills/`。

```bash
./scripts/codex/install-project-skills.sh
./scripts/codex/install-hooks.sh
./scripts/codex/doctor.sh
(cd backend-py-worker && uv sync --group dev)
(cd frontend && npm ci)
make verify
make integration-postgres
make smoke-e2e
```

`make verify` 验证文档、Go、备份、Worker 和前端。端到端烟测使用独立容器与数据库。开发工具版本见各模块锁文件；CI 和门禁见 [GitHub 协作索引](.github/INDEX.md)。

## 开发进度

当前阶段已进入 交付闭环收口与部署验收准备阶段：

- 已修复 GitHub Actions 隔离烟测的容器名误判，本地 `make verify` 与 `make smoke-e2e` 通过。

- 现有 DGX 环境已完成首批交付代码部署。验证证据和现场签收未结项见[客户交付优化清单](docs/客户交付优化清单.md)。
- 已按标准拓扑整理[现场部署手册](docs/现场部署逐条清单.md)，包括空库初始化、HTTPS、账号权限和恢复验收。

详细任务与历史进度见 [TASKS.md](TASKS.md)。
