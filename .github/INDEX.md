# .github 协作资产索引

本文件是仓库 `.github/` 目录的唯一入口，用于 VS Code、Copilot 与自动化代理按需读取 GitHub 侧协作资产。

## 使用约定

- 先读本文件，再决定是否打开 `.github/` 下的具体文件。
- 不默认扫描整个 `.github/` 目录。
- 若当前任务与 CI、门禁、PR 流程或仓库自动化无关，通常无需继续深入 `.github/`。

## 当前资产

| 文件 | 用途 | 何时需要读取 |
| --- | --- | --- |
| `workflows/verify.yml` | 仓库统一验证门禁，执行 `make verify` | 当任务涉及 CI、提交门禁、验证流程或需要解释 GitHub Actions 行为时 |

## 关联入口

- `AGENTS.md`
  - 仓库级协作协议、状态账本同步、验证要求。
- `docs/INDEX.md`
  - 业务设计、数据库契约、API 契约入口。
- `ops/codex/skills/index.yaml`
  - 项目级 Codex skills 入口。

## 对代理的要求

1. 先完成 `AGENTS.md` 与 `docs/INDEX.md` 约定的读取顺序。
2. 只有在任务确实涉及 GitHub 协作资产时，才继续读取 `.github/workflows/verify.yml`。
3. 需要验证仓库健康状态时，优先使用 `./scripts/codex/doctor.sh`、`./scripts/codex/check-doc-sync.sh`、`make verify`。
