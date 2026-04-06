# AGENTS

本文件定义本仓库的多轮协作与代理执行约束。目标是让不同开发者、不同会话、不同自动化代理在本项目中保持一致的工程行为。

## 1. 工作原则

遵循面向 Harness Engineering 风格的工程协作原则：

- 明确目标：每次改动都要能对齐到产品目标、设计文档和当前任务。
- 明确边界：业务层、存储层、AI 能力层、外部集成层必须职责清楚。
- 明确契约：表结构、API、外部系统接口先定义，再实现。
- 可追踪：每次阶段性工作都应更新 `README.md` 末尾的“当前开发进度”和 `TASKS.md`。
- 可恢复：优先做可重复执行的初始化、迁移和编排，不依赖手工记忆步骤。

## 2. 仓库内信息源优先级

实现时按以下优先级读取信息：

1. `docs/项目定义与技术架构.md`
2. `docs/数据库设计.md`
3. `docs/API设计.md`
4. `.github/INDEX.md`
5. `README.md`
6. `TASKS.md`
7. `ops/codex/skills/` 下的项目级可执行技能

如果实现与这些文档冲突，先更新文档再改代码，禁止静默偏离设计。

## 3. 目录职责

- `frontend/`
  - Vue 3 前端实现
- `ops/codex/skills/`
  - 项目级可执行技能与约束
- `scripts/codex/`
  - 项目级 Codex 安装、体检与辅助脚本
- `docs/`
  - 设计文档、接口契约、架构决策记录（ADR）和学习材料
  - `docs/adr/` 存放架构决策记录
- `README.md`
  - 项目总入口和阶段进度
- `TASKS.md`
  - 当前开发任务账本

## 4. 强约束

- 不允许直接在业务代码中拼装群晖 DSM / File Station 请求。
  - 必须统一走 `SynologyStorageProvider` 或后续独立适配器模块。
- 不允许把 OpenClaw 当成业务主账本。
  - 所有 AI 返回都作为附属结果存储。
- 不允许在未经设计文档确认的情况下增加新状态或新角色。
- 不允许跳过仓库内迁移机制直接手工改数据库结构。
  - 当前以后端 Go 侧 `backend-go/migrations/` 为准。
- 不允许把“建议”和“确认后的正式动作”混在同一数据表字段里。

## 5. 代理执行顺序

涉及功能开发时，建议按以下顺序工作：

1. 确认需求影响到哪份设计文档
2. 如有必要，先改文档
3. 改数据模型或 API 契约
4. 落后端实现
5. 落前端联动
6. 更新任务账本和 README 开发进度

## 5.1 会话启动清单

每次进入本仓库开始开发前，代理或开发者应先完成以下动作：

1. 读取 `docs/项目定义与技术架构.md`、`docs/数据库设计.md`、`docs/API设计.md`
2. 读取 `.github/INDEX.md`，仅按索引按需进入 `.github/` 资产
3. 读取 `README.md` 和 `TASKS.md`，确认当前阶段与进行中事项
4. 判断本次任务是否需要启用 `ops/codex/skills/` 下的专项 skill
5. 明确本次会影响的目录、契约和验证命令
6. 如发现 `README.md` 与 `TASKS.md` 阶段不一致，先修正文档再进入实现

## 5.2 项目技能使用约定

- 仓库内项目级 Codex 技能统一放在 `ops/codex/skills/`
- 技能索引维护在 `ops/codex/skills/index.yaml`
- 如需让 Codex 在运行时发现这些项目技能，应执行 `./scripts/codex/install-project-skills.sh`
- 项目技能统一以 `ops/codex/skills/` 为准

## 5.3 状态账本同步规则

- `TASKS.md` 是执行账本，记录“已完成 / 进行中 / 待办”
- `README.md` 是阶段快照，记录当前阶段与对外可读进展
- 两者允许细节粒度不同，但不允许阶段判断冲突
- 任何会改变项目阶段判断的工作，必须同时检查并按需更新 `TASKS.md` 与 `README.md`

## 5.4 验证与交付证据

- 任何功能或环境改造完成后，至少给出一种基础验证方式
- 优先使用可重复执行的命令，而不是口头说明
- 仓库级环境检查统一优先使用 `./scripts/codex/doctor.sh`
- 涉及多子系统联动时，优先使用 `make verify`
- 本地提交与推送前，优先启用 `./scripts/codex/install-hooks.sh`
- 需要说明当前仓库健康状态时，优先使用 `make status`
- 需要做本地容器联调烟测时，优先使用 `make smoke`
- 子系统验证优先使用：
  - `cd backend-go && go test ./...`
  - `cd backend-py-worker && uv run pytest -q`
  - `cd frontend && npm run build`

## 5.5 冲突处理规则

- 若实现与设计文档冲突：先改文档，再改代码
- 若 `README.md` 与 `TASKS.md` 冲突：以 `TASKS.md` 为执行账本，并在本次交付中同步 `README.md`
- 若项目技能与 `AGENTS.md` 规则冲突：以 `AGENTS.md` 为准，并及时修正 skill

## 6. 完成定义

一项功能只有同时满足以下条件才算完成：

- 设计文档与实现一致
- 对应数据模型或 API 已落地
- 至少有基础验证方式
- `TASKS.md` 已更新状态
- `README.md` 的开发进度已按需更新
- 如涉及协作约束或环境能力，`ops/codex/skills/`、相关脚本与说明文档也已同步更新
