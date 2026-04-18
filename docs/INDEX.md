# docs 文档索引

本目录存放项目所有设计文档、接口契约和架构决策记录。

## 核心设计文档

| 文件 | 说明 |
|------|------|
| [项目定义与技术架构.md](项目定义与技术架构.md) | 项目目标、技术选型、系统边界、模块职责 |
| [数据库设计.md](数据库设计.md) | 表结构、字段定义、枚举值、索引设计 |
| [API设计.md](API设计.md) | REST 接口契约、请求/响应结构、错误码 |
| [异步任务消息契约.md](异步任务消息契约.md) | Go → Python Worker 任务消息与结果回写结构 |
| [部署准备与运行说明.md](部署准备与运行说明.md) | 仓库内部署资产、环境模板与启动顺序 |
| [现场部署逐条清单.md](现场部署逐条清单.md) | 给现场同事逐条执行的部署清单 |
| [backend-go代码说明.md](backend-go代码说明.md) | backend-go 目录逐文件职责说明与主业务链路说明 |
| [backend-go核心源码学习导读.md](backend-go核心源码学习导读.md) | 面向 Go 初学者的 backend-go 核心文件函数级与代码块级学习导读 |
| [backend-py-worker代码说明.md](backend-py-worker代码说明.md) | backend-py-worker 目录逐文件职责说明与 Worker 链路说明 |
| [frontend代码说明.md](frontend代码说明.md) | frontend 目录逐文件职责说明与前端页面协作链路说明 |
| [项目端到端总览.md](项目端到端总览.md) | 从前端、Go、Worker、群晖、数据库到 OpenClaw 的整体业务与实现链路总览 |
| [digidocs-mgt-customer-intro.md](digidocs-mgt-customer-intro.md) | 面向课题组负责人的产品介绍、部署清单与基础预算口径 |
| [用户使用说明.md](用户使用说明.md) | 面向团队负责人和成员的平台功能使用说明（按模块逐一说明） |

## 架构决策记录（ADR）

| 文件 | 说明 |
|------|------|
| [adr/001-go-python-hybrid.md](adr/001-go-python-hybrid.md) | Go + Python 混合架构迁移方案与决策背景 |


## 约定

- 实现与文档冲突时，先改文档再改代码。
- 新增接口、表结构、跨系统集成，必须先在对应文档中确认后再落代码。
- ADR 记录"当时为什么这么决定"，不记录"怎么实现"。
