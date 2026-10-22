# 2026-10-22 审查整改交付记录

范围：依据 `/home/pgx/ops/digidocs-reviews/2026-10-22-code-review.md` 修复八类问题，并落实报告中的首轮优化；不包含 TASKS 中与本报告无关的历史产品需求和凭据轮换。

## 实现

| 审查项 | 修复 |
| --- | --- |
| AI 资源越权 | 摘要字段白名单，版本所属文档校验；Worker 从任务资源及后端文档归属构造上下文 |
| 成员跨项目操作 | 成员查询/修改/移除带 projectID；事务锁定项目并保护最后 owner |
| 交接清单 | GET/PUT 权限检查；事务内锁定、generated 状态、未删除资产及所属范围校验、审计 |
| 数据目录权限 | 列表校验项目可见，删除先查目录所属项目再检查写权限 |
| 他人会话追问 | 读取会话历史之前验证会话所有权 |
| 版本缓存 | 正文输出携带 version_id；按文档及输出版本匹配缓存，旧无版本缓存失效 |
| 最新版本选择 | 按 version_no 最大值选择，不依赖列表顺序 |
| 上传承诺 | 单文件 511 MiB；保留后端请求体 512 MiB，选择/拖入/提交均检查 |

优化：五个视图各提取独立表单或输入组件；增加前端交互测试；数据页并行加载项目；移除固定“在线”提示；Element Plus 按需样式及自动分包；更新过时 CI/集成测试待办，CI 补真实 PostgreSQL 检查。不增加应用依赖。

## 验证记录

- Go 单元测试、PostgreSQL 临时库真实集成测试执行，包含跨项目成员、目录、交接数据项权限和写入拒绝后的数据不变性、最后 owner、完成后编辑、资产范围、审计、会话所有权、缓存版本隔离。
- Worker：59 项测试通过，ruff 通过。
- 前端：16 项测试通过；类型检查、生产构建通过。
- JS 最大分块从原 Element Plus 896 kB 改为当前约 108 kB；这是构建产物指标，不等同于线上实测加载提速。
- `make verify`、`go vet ./...`、`make integration-postgres` 全部通过；本机使用临时 Go 1.26 工具链，与应用 Dockerfile 一致。
- 隔离栈严格烟测通过：注册、邀请、文档、审计、交接、版本上传/下载/预览。通用脚本关闭 AI 检查，另用真实 Worker + OpenClaw 完成两次正文抽取及摘要，确认 v1/v2 缓存隔离、v2 正文/版本号与非空模型摘要。
- 浏览器使用临时无头 Chrome 会话验证四个主要页面、文档详情、四个弹窗打开/取消以及移动端助手，无 pageerror；组件测试覆盖提交和防重复交互。内置浏览器无可用实例，采用已有本机 Chrome/Playwright，无新增依赖。
- 烟测脚本附加修复：NAS 预检改为仅在 `RUN_SYNOLOGY_PREFLIGHT=1` 时执行。首轮旧脚本曾执行临时 NAS 预检并完成清理；修复后严格烟测没有触发 NAS 预检。
- 2026-10-22 20:48（UTC+8）三个应用容器部署完成，均 healthy、运行镜像与本次构建一致；`/healthz`、系统信息、本地前端及公网入口 HTTP 200，Worker 心跳正常，启动后日志未发现 error/panic/traceback。
- 上线验证为只读健康和发布一致性检查；写入业务及 AI 验收在隔离数据库/内存存储完成，不向生产业务库写测试数据。未执行 511 MiB 大文件实际传输，已验证大小边界逻辑。
- 验证证据保存在 `/home/pgx/ops/digidocs-reviews/verification-20261022/`。

## 影响与回滚

没有数据库结构迁移、角色或状态枚举变更。旧无版本正文缓存不再复用，新任务重新抽取。保留历史 AI 输出，不追溯删除。新增查询参数和响应字段为兼容扩展；后端与 Worker 同批部署，避免旧 Worker 继续产生无版本输出。

部署只重建 DigiDocs 三个应用容器，不重建数据库、不改网络、防火墙或路由。部署前备份位于 `/home/pgx/ops/digidocs-reviews/rollback-20261022/`，三份旧镜像保留 `before-review-20261022` 标签。

应用回滚：将三个旧标签分别恢复为 `digidocs-mgt-backend-go:latest`、`digidocs-mgt-backend-py-worker:latest`、`digidocs-mgt-frontend:latest`，执行 `docker compose --profile app up -d --no-deps --no-build backend-go backend-py-worker frontend`。因未修改 schema，应用回滚不需要恢复数据库；数据库备份仅作为额外恢复点，勿覆盖上线后的业务数据。
