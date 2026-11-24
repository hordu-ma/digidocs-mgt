# DigiDocs 文档索引

从下表按工作目标进入。部署值以批准版本的 Compose 和环境模板为准，业务约束由设计契约维护，已完成的工程工作与待办由 [TASKS](../TASKS.md)维护。

## 交付与使用

| 文档 | 用途 |
| --- | --- |
| [现场部署手册](现场部署逐条清单.md) | 首次安装、管理员初始化、客户验收和交接 |
| [部署与运行参考](部署准备与运行说明.md) | HTTPS、备份恢复、升级回退、故障定位 |
| [客户交付优化清单](客户交付优化清单.md) | 18 项整改、验证证据和签收未结项 |
| [用户使用说明](用户使用说明.md) | 管理员、课题负责人和成员的功能说明 |
| [产品介绍](digidocs-mgt-customer-intro.md) · [产品手册](digidocs-mgt-product-brochure.html) | 对外产品定位和演示材料，现场配置以部署手册为准 |

## 设计与实现依据

| 文档 | 维护内容 |
| --- | --- |
| [项目定义与技术架构](项目定义与技术架构.md) | 产品边界、角色、状态机、服务职责与部署边界 |
| [数据库设计](数据库设计.md) | 数据模型、索引与约束，实际迁移见 [Go migrations](../backend-go/migrations/) |
| [API 设计](API设计.md) | HTTP 请求、响应、错误和访问权限 |
| [异步任务消息契约](异步任务消息契约.md) | Go 与 Worker 的任务生命周期、上下文和结果 |
| [项目端到端总览](项目端到端总览.md) | 从登录、上传、交接到 AI 的调用路径 |

## 开发与维护

| 文档 | 用途 |
| --- | --- |
| [Go 开发入口](../backend-go/README.md) · [代码说明](backend-go代码说明.md) · [源码导读](backend-go核心源码学习导读.md) | 启动验证、定位源码、学习主业务写入链 |
| [Worker 开发入口](../backend-py-worker/README.md) · [代码说明](backend-py-worker代码说明.md) | 本地运行、抽取和模型客户端 |
| [前端代码说明](frontend代码说明.md) | 页面、路由、状态与 API 客户端 |
| [测试覆盖率说明](测试覆盖率说明.md) · [CI 索引](../.github/INDEX.md) | 验证命令与历史覆盖率口径 |
| [协作约束](../AGENTS.md) · [项目技能索引](../ops/codex/skills/index.yaml) | 契约更新顺序、协作和质量门禁 |

## 历史资料

- [Go + Python 混合架构 ADR](adr/001-go-python-hybrid.md)与[注册登录改造决策](用户注册登录改造设计.md)保留当时的决策背景。
- [2026-11-10 优化记录](optimization-20261110.md)、[2026-10-22 整改记录](review-fixes-20261022.md)和[2026-10-08 备份修复记录](backup-repair-20261008.md)保留当时的证据与兼容边界。
- 早期客户介绍的 [HTML](../output/pdf/digidocs-mgt-customer-intro.html)与 [PDF](../output/pdf/digidocs-mgt-customer-intro.pdf)保留当时的发布内容。
