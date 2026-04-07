# frontend 代码说明

本文档用于解释 `frontend/` 目录中每个文件承担的职责，以及这些文件如何组合成 DigiDocs Mgt 的前端界面。

## 1. frontend 最终解决什么问题

`frontend/` 是面向用户的 Web 界面，负责把 `backend-go` 提供的 API 能力组织成可操作的管理界面。

它最终支撑的用户功能包括：

- 登录并保存登录态；
- 浏览仪表盘总览、近期流转和风险文档；
- 浏览文档列表、进入文档详情；
- 创建文档、编辑文档、上传新版本、执行流转动作；
- 查看和创建交接单；
- 发起 AI 问答、轮询 AI 请求结果、查看建议；
- 在同一套布局中切换各业务模块。

前端不直接处理业务规则，只负责调用后端 API、展示状态，并把用户操作组织成表单和页面交互。

## 2. 整体协作链路

典型前端交互链路如下：

1. `main.ts` 启动 Vue 应用，挂载 Pinia 和 Vue Router。
2. 路由进入某个页面组件，例如 `DocumentsView.vue` 或 `AssistantView.vue`。
3. 页面通过 `api.ts` 调用后端 `/api/v1/*` 接口。
4. `api.ts` 从 `auth` store 中读取 token，自动带上 `Authorization` 请求头。
5. 页面根据接口返回结果更新列表、卡片、详情、时间线等 UI。
6. 用户继续触发按钮、弹窗、表单提交流程，形成完整业务闭环。

因此 `frontend/` 可以理解为“把后端业务能力组装成用户可操作界面”的一层。

## 3. 根目录与构建文件

| 文件 | 作用 |
| --- | --- |
| `frontend/package.json` | 前端依赖与脚本入口，定义 `dev`、`build`、`preview`。 |
| `frontend/package-lock.json` | npm 锁文件，固定依赖版本。 |
| `frontend/vite.config.ts` | Vite 配置，包含 Vue 插件、Element Plus 自动导入、`@` 路径别名和 `/api` 代理。 |
| `frontend/tsconfig.json` | TypeScript 主配置。 |
| `frontend/tsconfig.node.json` | Node/Vite 配置文件所需的 TypeScript 配置。 |
| `frontend/index.html` | Vite 应用入口 HTML。 |
| `frontend/Dockerfile` | 前端生产镜像构建文件。 |
| `frontend/nginx/default.conf` | 生产环境 Nginx 配置，把 `/api/` 转发到 `backend-go`，其余请求回退到 `index.html`。 |

## 4. src 目录文件说明

### 4.1 应用入口

| 文件 | 作用 |
| --- | --- |
| `frontend/src/main.ts` | 创建 Vue 应用，注册 Pinia 和 Router，并挂载全局样式。 |
| `frontend/src/App.vue` | 应用根组件，仅负责渲染 `RouterView`。 |
| `frontend/src/env.d.ts` | Vite/TypeScript 环境声明文件。 |
| `frontend/src/styles.css` | 全局样式，定义页面底色、卡片、布局、KPI 样式等通用视觉规则。 |

### 4.2 API 与状态管理

| 文件 | 作用 |
| --- | --- |
| `frontend/src/api.ts` | Axios 实例封装，统一设置 `baseURL` 和 `Authorization` 头。 |
| `frontend/src/stores/auth.ts` | Pinia 登录态仓库，负责保存 token、用户 ID、显示名、角色，并同步到 `localStorage`。 |

### 4.3 路由

| 文件 | 作用 |
| --- | --- |
| `frontend/src/router/index.ts` | 定义所有页面路由，并在 `beforeEach` 中根据本地 token 做登录拦截。 |

当前页面路由包括：

- `/login` 登录页
- `/dashboard` 仪表盘
- `/documents` 文档列表
- `/documents/:id` 文档详情
- `/handovers` 交接页
- `/assistant` AI 助手页

### 4.4 通用布局组件

| 文件 | 作用 |
| --- | --- |
| `frontend/src/components/AppLayout.vue` | 后台通用布局壳，提供左侧导航栏和右侧内容区。所有业务页面都复用这套布局。 |

### 4.5 视图页面

| 文件 | 作用 |
| --- | --- |
| `frontend/src/views/LoginView.vue` | 登录页，提交用户名密码到 `/auth/login`，成功后写入 auth store 并跳转仪表盘。 |
| `frontend/src/views/DashboardView.vue` | 仪表盘页面，读取 `/dashboard/overview`、`/dashboard/recent-flows`、`/dashboard/risk-documents` 三组数据，展示 KPI、近期流转和风险提示。 |
| `frontend/src/views/DocumentsView.vue` | 文档列表页，支持分页、关键词搜索和“新建文档”弹窗。 |
| `frontend/src/views/DocumentDetailView.vue` | 文档详情页，是前端业务最复杂的页面，负责展示基本信息、版本历史、流转历史、AI 建议，并支持编辑、删除/恢复、上传新版本、执行流转动作、触发摘要。 |
| `frontend/src/views/HandoversView.vue` | 交接页，展示交接记录并支持创建交接单。 |
| `frontend/src/views/AssistantView.vue` | AI 助手页，负责提交问答请求到 `/assistant/ask`，再轮询 `/assistant/requests/{id}` 获取结果。 |

## 5. 各页面如何对应后端功能

### 5.1 登录页

`LoginView.vue` 对应后端：

- `POST /api/v1/auth/login`

作用：

- 获取 `access_token`；
- 写入 `auth` store；
- 触发后续所有受保护页面的访问能力。

### 5.2 仪表盘页

`DashboardView.vue` 对应后端：

- `GET /api/v1/dashboard/overview`
- `GET /api/v1/dashboard/recent-flows`
- `GET /api/v1/dashboard/risk-documents`

作用：

- 让负责人快速看到文档总量、处理中数量、待交接数量、风险文档数量；
- 展示近期流转与风险摘要。

### 5.3 文档列表页

`DocumentsView.vue` 对应后端：

- `GET /api/v1/documents`
- `POST /api/v1/documents`

作用：

- 展示文档目录；
- 支持搜索和分页；
- 支持通过 multipart 表单创建新文档和首版本。

### 5.4 文档详情页

`DocumentDetailView.vue` 对应后端：

- `GET /api/v1/documents/{id}`
- `GET /api/v1/documents/{id}/versions`
- `GET /api/v1/documents/{id}/flows`
- `PATCH /api/v1/documents/{id}`
- `POST /api/v1/documents/{id}/delete`
- `POST /api/v1/documents/{id}/restore`
- `POST /api/v1/documents/{id}/versions`
- `POST /api/v1/documents/{id}/flow/*`
- `POST /api/v1/assistant/documents/{id}/summarize`
- `GET /api/v1/assistant/suggestions`
- `POST /api/v1/assistant/suggestions/{id}/confirm`
- `POST /api/v1/assistant/suggestions/{id}/dismiss`

作用：

- 把“文档基本信息 + 版本管理 + 流转管理 + AI 建议”聚合到一页；
- 是最接近业务主闭环的前端页面。

### 5.5 交接页

`HandoversView.vue` 对应后端：

- `GET /api/v1/handovers`
- `POST /api/v1/handovers`

作用：

- 展示交接单列表；
- 支持创建新的交接记录。

### 5.6 助手页

`AssistantView.vue` 对应后端：

- `POST /api/v1/assistant/ask`
- `GET /api/v1/assistant/requests/{request_id}`

作用：

- 提交 AI 问答任务；
- 轮询异步结果并展示问答输出。

## 6. 文件组合后支撑的项目需求

把这些文件组合起来，前端最终实现了项目里的以下需求：

- 成员登录进入统一后台；
- 负责人查看整体文档状态和流转风险；
- 用户管理文档和版本；
- 用户执行流转、定稿、归档等动作；
- 团队执行毕业交接；
- 用户发起 AI 问答和文档摘要；
- 用户查看并确认或忽略 AI 建议。

换句话说，`frontend/` 把项目定义文档中的“文档集中管理、显式流转、毕业交接、负责人总览、AI 助手”这些需求，转成了一个可直接操作的 Web 界面。

## 7. 学习这套前端的推荐顺序

如果你是为了理解代码，建议按这个顺序阅读：

1. `src/main.ts`
2. `src/router/index.ts`
3. `src/stores/auth.ts`
4. `src/api.ts`
5. `src/components/AppLayout.vue`
6. `src/views/LoginView.vue`
7. `src/views/DashboardView.vue`
8. `src/views/DocumentsView.vue`
9. `src/views/DocumentDetailView.vue`
10. `src/views/HandoversView.vue`
11. `src/views/AssistantView.vue`
12. `vite.config.ts` 与 `nginx/default.conf`

这样可以先理解“应用怎么启动”，再理解“路由和认证怎么工作”，最后再逐页理解每个业务模块。
