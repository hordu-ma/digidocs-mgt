# Go 主业务 API

维护账户与权限、文档和独立版本、数据与代码资产、流转、交接、审计及 AI 附属结果。生产使用 PostgreSQL 和群晖。Go 在启动时应用 `migrations/`，Worker 通过内部 API 领取任务并回写。

在本目录运行 `go test ./...`。使用真实 PostgreSQL 的回归从仓库根目录执行 `make integration-postgres`，完整开发联调执行 `make smoke-e2e`。

阅读 [API 设计](../docs/API设计.md)、[数据库设计](../docs/数据库设计.md)和[源码说明](../docs/backend-go代码说明.md)。客户安装以[现场部署手册](../docs/现场部署逐条清单.md)为准。
