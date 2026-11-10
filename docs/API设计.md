# 面向课题组的文档资产管理与智能助理平台

## REST API 设计草案

## 1. 设计约定

- 基础路径：`/api/v1`
- 认证方式：`Bearer JWT`
- 返回格式统一为 JSON
- 时间字段统一使用 ISO 8601
- 列表接口统一支持 `page`, `page_size`
- 失败响应统一返回：

```json
{
  "code": "forbidden",
  "message": "You do not have permission."
}
```

- 成功响应统一优先使用：

```json
{
  "data": {},
  "meta": {}
}
```

---

## 2. 认证接口

### 2.1 登录

`POST /api/v1/auth/login`（公开路由，独立登录限流）

同一入口支持用户名、手机号或邮箱三选一登录。老测试账号只有用户名、自助注册账号只有手机号/邮箱，均可使用本接口。

请求体：

```json
{
  "identifier": "zhangsan",
  "password": "******"
}
```

`identifier` 会按 `username = $1 OR email = $1 OR phone = $1` 精确匹配，命中任一即可。

响应体：

```json
{
  "data": {
    "access_token": "jwt-token",
    "token_type": "Bearer",
    "expires_in": 7200,
    "user": {
      "id": "uuid",
      "username": "zhangsan",
      "display_name": "张三",
      "role": "member"
    }
  }
}
```

JWT 中的 `username` 取数据库真实值。自助注册用户没有用户名时，该字段为空字符串，不会把手机号或邮箱误写入 claims。

### 2.1.1 自助注册

`POST /api/v1/auth/register`（公开路由，独立注册限流，默认 5 次/分钟/IP，`REGISTER_RATE_LIMIT_PER_MIN` 可配）

完全开放注册，不做短信/邮箱验证码；注册即激活，角色由服务端强制写为 `member`。请求体不接受 `role`。

请求体：

```json
{
  "email": "user@example.com",
  "phone": "13800138000",
  "password": "至少8位",
  "display_name": "显示姓名"
}
```

- `email`、`phone` 至少填一项，可都填；
- 手机号仅支持中国大陆号段 `^1[3-9]\\d{9}$`；
- 密码 8–72 字节（与 bcrypt 上限对齐）；
- `display_name` 必填，最长 64 字符；
- 校验失败返回 `400 validation_error`；
- `email` / `phone` 已被占用返回 `409 conflict`，错误信息区分「该邮箱已被注册」/「该手机号已被注册」。

成功响应与登录接口同构，注册成功即签发 JWT 并自动登录：

```json
{
  "data": {
    "access_token": "jwt-token",
    "token_type": "Bearer",
    "expires_in": 7200,
    "user": {
      "id": "uuid",
      "username": "",
      "display_name": "显示姓名",
      "role": "member"
    }
  }
}
```

### 2.2 当前用户

`GET /api/v1/auth/me`

响应体：

```json
{
  "data": {
    "id": "uuid",
    "username": "zhangsan",
    "display_name": "张三",
    "role": "member",
    "email": "zhangsan@example.com",
    "phone": "13800000000",
    "wechat": "zhangsan_wechat",
    "status": "active",
    "last_login_at": "2026-04-03T10:00:00Z"
  }
}
```

### 2.3 更新当前用户资料

`PATCH /api/v1/auth/me`

仅允许当前登录用户更新个人显示名称与联系方式，不允许通过该接口修改 `username`、`role`、`status` 或项目级权限。

请求体：

```json
{
  "display_name": "张三",
  "email": "zhangsan@example.com",
  "phone": "13800000000",
  "wechat": "zhangsan_wechat"
}
```

响应体：

```json
{
  "data": {
    "id": "uuid",
    "username": "zhangsan",
    "display_name": "张三",
    "role": "member",
    "email": "zhangsan@example.com",
    "phone": "13800000000",
    "wechat": "zhangsan_wechat",
    "status": "active",
    "last_login_at": "2026-04-03T10:00:00Z"
  }
}
```

校验规则：

- `display_name` 必填，最长 64 字符；
- `email`、`phone`、`wechat` 可为空，长度分别不超过 128 / 32 / 64；
- 非空 `email` 需包含 `@`；
- 权限、角色和账号状态仍由管理员侧用户管理能力维护。

### 2.4 修改当前用户密码

`POST /api/v1/auth/change-password`

自助注册账号此前无法自行改密，只能由管理员通过 `PATCH /api/v1/admin/users/{user_id}` 重置。本接口补齐自助改密闭环。

请求体：

```json
{
  "current_password": "old-password",
  "new_password": "new-password"
}
```

响应体：

```json
{
  "data": { "success": true }
}
```

校验规则：

- `current_password` 必填，且必须与当前密码一致，否则返回 `401 unauthorized`（`message` 为「当前密码不正确」）；仅凭有效 JWT 不足以改密，避免令牌泄露后账号被直接锁死；
- `new_password` 长度 8–72 字符，且不得与当前密码相同，否则返回 `400 validation_error`。

已知限制：本系统签发的是无状态 JWT，没有吊销名单，因此**改密不会使已签发的令牌失效**，旧令牌仍可用至自然过期（2 小时）。前端「修改密码」面板已明确提示这一点。若要做到改密即登出，需要引入 token 版本号或黑名单，属于后续项。

### 2.5 退出登录

`POST /api/v1/auth/logout`

响应体：

```json
{
  "data": {
    "success": true
  }
}
```

---

## 3. 组织结构接口

### 3.1 用户列表

`GET /api/v1/users`

响应体：

```json
{
  "data": [
    {
      "id": "uuid",
      "username": "zhangsan",
      "display_name": "张三",
      "role": "member",
      "email": "zhangsan@example.com",
      "phone": "13800000000",
      "wechat": "zhangsan_wechat",
      "status": "active"
    }
  ]
}
```

### 3.2 团队空间列表

`GET /api/v1/team-spaces`

响应体：

```json
{
  "data": [
    {
      "id": "uuid",
      "name": "随机控制实验室",
      "code": "lab-rc"
    }
  ]
}
```

### 3.3 项目列表

`GET /api/v1/projects?team_space_id={id}`

响应体：

```json
{
  "data": [
    {
      "id": "uuid",
      "team_space_id": "uuid",
      "name": "课题A",
      "code": "proj-a",
      "owner": {
        "id": "uuid",
        "display_name": "李老师"
      },
      "my_role": "contributor"
    }
  ]
}
```

`my_role` 是当前登录用户在该课题中的 `project_members.project_role`。`admin` 查看全部课题时该字段为空字符串。非 admin 且不属于任何课题时返回空列表。

### 3.4 目录树

`GET /api/v1/projects/{project_id}/folders/tree`

响应体：

```json
{
  "data": [
    {
      "id": "uuid",
      "name": "申报材料",
      "path": "/申报材料",
      "children": []
    }
  ]
}
```

### 3.4.1 课题成员

`GET /api/v1/projects/{project_id}/members`

课题成员或 `admin` 可查看。响应为 `project_members` 列表，字段与管理端成员列表一致。

`POST /api/v1/projects/{project_id}/members`

课题 `owner` / `manager` 或 `admin` 可将已有平台用户加入本课题。默认 `project_role` 为 `contributor`。负责人侧不可通过本接口指定 `owner`。

```json
{
  "user_id": "uuid",
  "project_role": "contributor"
}
```

`PATCH /api/v1/projects/{project_id}/members/{member_id}`

更新课题角色。负责人侧不可把成员改成 `owner`。

`DELETE /api/v1/projects/{project_id}/members/{member_id}`

移出课题。不可移除课题的最后一位 `owner`。

### 3.4.2 课题邀请

`POST /api/v1/projects/{project_id}/invites`

生成可复用邀请链接，默认角色 `contributor`，默认 7 天有效。仅 `owner` / `manager` / `admin`。

响应：

```json
{
  "data": {
    "id": "uuid",
    "project_id": "uuid",
    "token": "hex",
    "project_role": "contributor",
    "status": "active",
    "expires_at": "2026-08-24T00:00:00Z",
    "created_at": "2026-08-17T00:00:00Z"
  }
}
```

`GET /api/v1/projects/{project_id}/invites` 列出本课题邀请（含已撤销/过期，便于管理）。

`POST /api/v1/projects/{project_id}/invites/{invite_id}/revoke` 撤销邀请。

`GET /api/v1/invites/{token}` 公开预览（无需登录），只返回课题名称、团队空间名称、将获得的角色、状态和过期时间，不返回成员名单。

`POST /api/v1/invites/{token}/accept` 需登录。校验邀请有效后写入 `project_members`；已是成员则幂等返回当前成员关系。

### 3.5 首期权限矩阵

首期权限由后端固定矩阵判断，不提供可配置 RBAC 后台。判断依据为：

- JWT 中的全局角色：`admin` / `project_lead` / `member`
- `project_members.project_role`：`owner` / `manager` / `contributor` / `viewer`
- 文档当前责任人：`documents.current_owner_id`

写操作权限：

| 接口/动作 | 授权规则 |
| --- | --- |
| 创建文档 | `admin`、项目 `owner/manager/contributor` |
| 修改文档元数据 | `admin`、项目 `owner/manager`、文档当前责任人 |
| 删除 / 恢复文档 | `admin`、项目 `owner/manager` |
| 上传新版本 | `admin`、项目 `owner/manager`、文档当前责任人 |
| 普通流转动作 | `admin`、项目 `owner/manager`、文档当前责任人 |
| 定稿 / 归档 / 取消归档 | `admin`、项目 `owner/manager` |
| 发起交接 | `admin`、项目 `owner/manager` |
| 编辑交接清单 | `admin`、项目 `owner/manager` |
| 确认交接 | `admin`、项目 `owner/manager`、交接接收人 |
| 完成交接 / 取消交接 | `admin`、项目 `owner/manager` |
| 查看课题成员 | `admin`、该课题任意成员 |
| 添加/移除课题成员、生成/撤销邀请 | `admin`、项目 `owner/manager` |

读操作权限（2026-08-17 补齐）：

早期实现只在少数详情接口上做了成员校验，其余读接口完全没有取调用者身份，导致任意登录用户可以凭 `GET /audit-events` 里返回的 `document_id` / `version_id` 跨课题读取甚至下载他人文件。现口径如下，非 `admin` 违反时统一返回 403：

| 接口/动作 | 授权规则 |
| --- | --- |
| 查看文档详情 | `admin`、文档所属项目成员、文档当前责任人 |
| 查看版本列表 `GET /documents/{id}/versions` | 同「查看文档详情」 |
| 查看流转记录 `GET /documents/{id}/flows` | 同「查看文档详情」 |
| 版本详情 / 下载 / 预览 `GET /versions/{id}[/download|/preview]` | 由版本反查所属文档后套用「查看文档详情」规则 |
| 查看交接单详情 `GET /handovers/{id}` | `admin`、交接单所属项目成员、被交接人、接收人 |
| 查看代码推送记录 `GET /code-repositories/{id}/push-events` | `admin`、仓库所属项目成员 |

聚合类读接口没有单一资源 ID 可校验，因此不返回 403，而是在 SQL 层按调用者可见的项目集合收敛结果（`admin` 不收敛）：

| 接口 | 收敛口径 |
| --- | --- |
| `GET /dashboard/overview`、`/recent-flows`、`/risk-documents` | 仅统计调用者拥有或所属项目下的文档与交接单；省略 `project_id` 时不再返回全库数据 |
| `GET /audit-events`、`GET /audit-events/summary` | 仅返回可见项目下的事件；项目先按文档归属判定，无文档时按 `extra_data.project_id`；两者都没有的事件对非 `admin` 不可见 |
| `GET /handovers` | 仅返回调用者可见项目的交接单，外加本人作为被交接人/接收人的交接单 |

已知例外：`GET /users` 仍对全部登录用户返回精简用户名录（`id` / `username` / `display_name` / `role` / `status`，不含联系方式）。课题负责人的「添加成员」需要按姓名检索尚未加入本课题的账号，收敛该接口会直接破坏该功能，因此保持现状。

读操作权限（2026-08-19 补齐，AI 助手子系统）：

2026-08-17 那轮读权限收口覆盖了 document/version/handover/code-repository 等常规资源，但完全遗漏了 AI 助手（Assistant）子系统——`GET /assistant/requests[/{id}]`、`GET /assistant/suggestions`、`POST /assistant/suggestions/{id}/confirm|dismiss`、`GET /assistant/conversations/{id}[/messages]`、`PATCH /assistant/conversations/{id}/archive`、`POST /assistant/{documents,handovers}/{id}/summarize`、`POST /assistant/ask`、`POST /assistant/conversations` 均未取调用者身份，任意登录用户可跨课题读取 AI 生成的问答/摘要正文，甚至确认或忽略别人课题的 AI 建议、对无权查看的文档触发摘要任务。现口径如下，非 `admin` 违反时统一返回 403：

| 接口/动作 | 授权规则 |
| --- | --- |
| 发起问答 `POST /assistant/ask` | scope 为 `document` 时按「查看文档详情」规则；为 `project` 时要求调用者是该项目成员 |
| 创建会话 `POST /assistant/conversations` | 同「发起问答」的 scope 规则 |
| 查看会话详情 / 消息 `GET /assistant/conversations/{id}[/messages]` | `admin`、会话创建者本人（与 `GET /assistant/conversations` 列表口径一致） |
| 归档 / 恢复会话 `PATCH /assistant/conversations/{id}/archive` | 同上 |
| 生成文档摘要 `POST /assistant/documents/{id}/summarize` | 同「查看文档详情」 |
| 生成交接摘要 `POST /assistant/handovers/{id}/summarize` | 同「查看交接单详情」 |
| 查询问答任务详情 `GET /assistant/requests/{id}` | `admin`、请求创建者本人、关联资源（文档/交接单/项目）所属项目成员 |
| 确认 / 忽略建议 `POST /assistant/suggestions/{id}/confirm\|dismiss` | `admin`、建议关联资源所属项目成员 |

聚合类读接口同样按项目集合在 SQL 层收敛（`admin` 不收敛）：

| 接口 | 收敛口径 |
| --- | --- |
| `GET /assistant/requests` | 请求创建者本人的记录，外加调用者可见项目下的记录（按 `related_type`/`related_id` 反查所属项目） |
| `GET /assistant/suggestions` | 仅返回调用者可见项目下的建议 |

权限不足时统一返回：

```json
{
  "code": "forbidden",
  "message": "permission denied"
}
```

---

## 4. 文档接口

### 4.1 创建文档并上传首个版本

`POST /api/v1/documents`

请求类型：`multipart/form-data`

表单字段：

- `team_space_id`
- `project_id`
- `folder_id`
- `title`
- `description`
- `current_owner_id`
- `commit_message`
- `file`

响应体：

```json
{
  "data": {
    "id": "uuid",
    "title": "课题申报书",
    "current_status": "draft",
    "current_owner": {
      "id": "uuid",
      "display_name": "张三"
    },
    "current_version": {
      "id": "uuid",
      "version_no": 1
    }
  }
}
```

### 4.2 文档列表

`GET /api/v1/documents`

权限范围：非 `admin` 角色只能看到自己拥有或所属（`project_members`）项目下的文档，与 `project_id` 查询参数取值无关——即使省略 `project_id` 或传入一个自己不属于的项目 ID，也只会返回有权限的项目内数据，不会像早期实现那样返回全库结果。`admin` 不受此限制。

查询参数：

- `team_space_id`
- `project_id`
- `folder_id`
- `owner_id`
- `status`
- `keyword`（标题模糊匹配，`%`、`_` 按字面处理）
- `include_archived`
- `sort`（2026-11-10）：`updated_desc`（默认）、`updated_asc`、`title_asc`、`owner_asc`
- `mine=true`：等同 `owner_id=<调用者>`
- `updated_within_days`：只看最近 N 天更新的文档
- `deleted_only=true`：回收站，只返回已删除文档，且仅限调用者为 owner/manager 的课题（`admin` 不限）
- `with_counts=true`：在 `meta.project_counts` 返回按课题的计数（同样的筛选条件，但不含 `project_id`），供左侧课题栏使用
- `page`
- `page_size`（上限 200）

说明：此前前端一次取 200 条后在浏览器里筛选与排序，超过 200 条的文档会被漏掉，课题计数也只统计已加载部分。现在所有筛选、排序、计数都在服务端完成，前端按页加载。

响应体：

```json
{
  "data": [
    {
      "id": "uuid",
      "title": "课题申报书",
      "project_name": "课题A",
      "folder_path": "/申报材料",
      "current_status": "in_progress",
      "current_owner": {
        "id": "uuid",
        "display_name": "张三"
      },
      "current_version_no": 3,
      "updated_at": "2026-04-03T10:00:00Z",
      "project_id": "uuid",
      "is_archived": false,
      "is_deleted": false
    }
  ],
  "meta": {
    "page": 1,
    "page_size": 20,
    "total": 128,
    "project_counts": [
      { "project_id": "uuid", "project_name": "课题A", "count": 42 }
    ]
  }
}
```

### 4.2.1 全文检索（2026-11-10）

`GET /api/v1/search?q=&project_id=&limit=20`

在调用者所属课题的文档**当前版本正文**中检索，每篇文档返回一条，代表片段为命中查询词最多的一段。查询词规则与 AI 检索相同（中文两字一组、英文按单词），正文来自 `document_text_chunks`，尚未抽取正文的文档搜不到。

响应：`[{document_id, title, project_id, project_name, version_id, version_no, chunk_no, snippet, score, matched_chunks}]`，按命中词数、命中片段数、更新时间排序，`limit` 上限 50。

### 4.3 文档详情

`GET /api/v1/documents/{document_id}`

权限范围：非 `admin` 角色必须是文档所属项目的成员（或该文档当前责任人）才能访问，否则返回 403；`admin` 不受此限制。

响应体：

```json
{
  "data": {
    "id": "uuid",
    "title": "课题申报书",
    "description": "2026年度申报",
    "team_space": {
      "id": "uuid",
      "name": "随机控制实验室"
    },
    "project": {
      "id": "uuid",
      "name": "课题A"
    },
    "folder": {
      "id": "uuid",
      "path": "/申报材料"
    },
    "current_status": "in_progress",
    "current_owner": {
      "id": "uuid",
      "display_name": "张三"
    },
    "current_version_id": "uuid",
    "is_archived": false,
    "created_at": "2026-04-03T10:00:00Z",
    "updated_at": "2026-04-03T12:00:00Z"
  }
}
```

### 4.4 更新文档基础信息

`PATCH /api/v1/documents/{document_id}`

请求体：

```json
{
  "title": "课题申报书-修订",
  "description": "更新说明",
  "folder_id": "uuid"
}
```

响应体：返回最新文档详情。

### 4.5 删除文档

`POST /api/v1/documents/{document_id}/delete`

请求体：

```json
{
  "reason": "误建文档"
}
```

响应体：

```json
{
  "data": {
    "id": "uuid",
    "is_deleted": true
  }
}
```

### 4.6 恢复文档

`POST /api/v1/documents/{document_id}/restore`

响应体：

```json
{
  "data": {
    "id": "uuid",
    "is_deleted": false
  }
}
```

---

## 5. 版本接口

### 5.1 提交新版本

`POST /api/v1/documents/{document_id}/versions`

请求类型：`multipart/form-data`

表单字段：

- `commit_message`
- `file`

响应体：

```json
{
  "data": {
    "id": "uuid",
    "document_id": "uuid",
    "version_no": 4,
    "commit_message": "补充实验数据",
    "created_by": {
      "id": "uuid",
      "display_name": "张三"
    },
    "created_at": "2026-04-03T13:00:00Z"
  }
}
```

### 5.2 版本列表

`GET /api/v1/documents/{document_id}/versions`

响应体：

```json
{
  "data": [
    {
      "id": "uuid",
      "version_no": 4,
      "file_name": "申报书-v4.docx",
      "file_size": 102400,
      "commit_message": "补充实验数据",
      "summary_status": "completed",
      "created_by": {
        "id": "uuid",
        "display_name": "张三"
      },
      "created_at": "2026-04-03T13:00:00Z"
    }
  ]
}
```

### 5.3 版本详情

`GET /api/v1/versions/{version_id}`

响应体：

```json
{
  "data": {
    "id": "uuid",
    "document_id": "uuid",
    "version_no": 4,
    "file_name": "申报书-v4.docx",
    "mime_type": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
    "file_size": 102400,
    "commit_message": "补充实验数据",
    "summary_status": "completed",
    "summary_text": "本版本补充了实验数据和结论说明。",
    "created_at": "2026-04-03T13:00:00Z"
  }
}
```

### 5.4 下载版本文件

`GET /api/v1/versions/{version_id}/download`

响应：文件流。

审计（2026-11-10）：每次下载在返回文件前写入一条 `download` 审计事件。审计写入失败时接口返回 `500 audit_failed`，不输出文件（fail closed），保证「谁下载了什么」可查。

### 5.5 版本预览信息

`GET /api/v1/versions/{version_id}/preview`

当前实现：响应文件流，行为与下载接口类似，但以浏览器内联打开为目标。返回文件前写入 `preview` 审计事件，失败同样返回 `500 audit_failed`。

- `Content-Disposition: inline`
- `Content-Type: 根据文件名或存储元数据推断`

说明：

- 当前接口不承诺返回独立的预览地址；
- `preview_type`、`preview_url`、`watermark_enabled` 相关 richer preview 契约保留为未来可选增强；
- 若后续启动 PDF 预览 / 水印能力，建议新增或扩展单独的预览编排返回结构，而不是改变现有文件流语义。

响应：文件流。

---

## 6. 流转接口

### 6.1 标记处理中

`POST /api/v1/documents/{document_id}/flow/mark-in-progress`

请求体：

```json
{
  "note": "开始继续处理"
}
```

### 6.2 发起转交

`POST /api/v1/documents/{document_id}/flow/transfer`

请求体：

```json
{
  "to_user_id": "uuid",
  "note": "请继续完善第四章"
}
```

响应体：

```json
{
  "data": {
    "document_id": "uuid",
    "from_status": "in_progress",
    "to_status": "pending_handover",
    "to_user": {
      "id": "uuid",
      "display_name": "李四"
    }
  }
}
```

### 6.3 接收交接

`POST /api/v1/documents/{document_id}/flow/accept-transfer`

请求体：

```json
{
  "note": "已接收"
}
```

### 6.4 标记定稿

`POST /api/v1/documents/{document_id}/flow/finalize`

请求体：

```json
{
  "note": "内容已确认定稿"
}
```

### 6.5 标记归档

`POST /api/v1/documents/{document_id}/flow/archive`

请求体：

```json
{
  "note": "项目阶段结束，执行归档"
}
```

### 6.6 恢复归档

`POST /api/v1/documents/{document_id}/flow/unarchive`

请求体：

```json
{
  "note": "需要继续修订"
}
```

### 6.7 流转历史

`GET /api/v1/documents/{document_id}/flows`

响应体：

```json
{
  "data": [
    {
      "id": "uuid",
      "action": "transfer",
      "from_status": "in_progress",
      "to_status": "pending_handover",
      "from_user": {
        "id": "uuid",
        "display_name": "张三"
      },
      "to_user": {
        "id": "uuid",
        "display_name": "李四"
      },
      "note": "请继续完善第四章",
      "created_at": "2026-04-03T14:00:00Z"
    }
  ]
}
```

---

## 7. 毕业交接接口

### 7.1 生成交接单

`POST /api/v1/handovers`

请求体：

```json
{
  "target_user_id": "uuid",
  "receiver_user_id": "uuid",
  "project_id": "uuid",
  "remark": "毕业交接"
}
```

响应体：

```json
{
  "data": {
    "id": "uuid",
    "status": "generated",
    "document_item_count": 12,
    "data_item_count": 3
  }
}
```

候选自动归集（2026-11-10）：创建交接单的同一事务内，系统把被交接人名下的候选写入清单并默认勾选，负责人只需取消不该交接的项。

- 文档：被交接人是当前责任人、未删除、未归档；指定 `project_id` 时只取该课题，不指定（仅 `admin`）时跨课题。备注标出 `未挂接目录` 或 `超过 30 天未更新`，提示负责人优先核对。
- 数据资产：被交接人上传、未删除，课题范围同上。
- `document_item_count` / `data_item_count` 为自动带入的数量；为 0 时清单为空，「我的待办」会提示负责人补全。

### 7.2 交接列表

`GET /api/v1/handovers`

查询参数：

- `status`
- `target_user_id`
- `receiver_user_id`
- `project_id`

### 7.3 交接详情

`GET /api/v1/handovers/{handover_id}`

响应体：

```json
{
  "data": {
    "id": "uuid",
    "status": "pending_confirm",
    "target_user_id": "uuid",
    "receiver_user_id": "uuid",
    "project_id": "uuid",
    "remark": "备注",
    "items": [
      {
        "document_id": "uuid",
        "title": "实验记录",
        "current_status": "in_progress",
        "selected": true,
        "note": ""
      }
    ]
  }
}
```

`items[].title` / `items[].current_status`由后端 JOIN `documents` 表解析，不依赖前端已加载的文档列表，因此不受课题范围、归档状态或分页大小影响。当前响应不包含 `ai_summary` 字段（AI 摘要写入 `graduation_handovers.ai_summary`，尚未在本接口透出，如需展示需前端另行查询）。

### 7.4 更新交接项

`PATCH /api/v1/handovers/{handover_id}/items`

请求体：

```json
{
  "items": [
    {
      "document_id": "uuid",
      "selected": false,
      "note": "无需移交"
    }
  ]
}
```

### 7.5 接收人确认交接

`POST /api/v1/handovers/{handover_id}/confirm`

请求体：

```json
{
  "note": "确认接收"
}
```

### 7.6 完成交接

`POST /api/v1/handovers/{handover_id}/complete`

请求体：

```json
{
  "note": "已完成归属更新"
}
```

### 7.7 取消交接

`POST /api/v1/handovers/{handover_id}/cancel`

请求体：

```json
{
  "reason": "信息填写有误"
}
```

---

## 8. 审计与总览接口

### 8.1 负责人总览

`GET /api/v1/dashboard/overview?project_id={id}`

响应体：

```json
{
  "data": {
    "document_total": 120,
    "status_counts": {
      "draft": 5,
      "in_progress": 32,
      "pending_handover": 6,
      "handed_over": 8,
      "finalized": 28,
      "archived": 41
    },
    "handover_pending_count": 2,
    "risk_document_count": 7
  }
}
```

口径（2026-11-10）：

- `handover_pending_count`：尚未完成的交接单数，即状态为 `generated` 或 `pending_confirm`。此前只统计 `pending_confirm`，会漏掉刚创建、等接收人确认的交接单。
- `risk_document_count`：与 8.3 风险列表使用同一条 SQL 计数，两者在列表截断前完全一致。

### 8.2 近期流转

`GET /api/v1/dashboard/recent-flows?project_id={id}`

### 8.3 风险文档

`GET /api/v1/dashboard/risk-documents?project_id={id}`

响应体：

```json
{
  "data": [
    {
      "document_id": "uuid",
      "title": "阶段报告",
      "risk_type": "stale",
      "risk_message": "41 天未更新",
      "related_type": "document",
      "related_id": "uuid",
      "link": "/documents/uuid"
    }
  ]
}
```

风险规则（2026-11-10）。同一文档只按最严重的一条出现，列表按严重程度、再按时间排序，最多 50 条：

| `risk_type` | 条件 | 严重度 |
| --- | --- | --- |
| `owner_inactive` | 未定稿/未归档文档的责任人账号已停用 | 1 |
| `transfer_unaccepted` | 状态为 `pending_handover`（待接收）超过 7 天 | 2 |
| `handover_overdue` | 交接单创建超过 14 天仍未完成；此时 `document_id` 为空，`related_type=handover` | 3 |
| `stale` | 未定稿/未归档文档超过 30 天未更新 | 4 |

前端统一用 `link` 跳转，不再自行拼接文档地址。

### 8.4 审计事件列表

`GET /api/v1/audit-events`

查询参数：

- `project_id`
- `document_id`
- `user_id`
- `action_type`
- `date_from`
- `date_to`
- `page`
- `page_size`

响应项字段：`id`、`document_id`、`version_id`、`user_id`、`action_type`、`request_id`、`ip_address`、`terminal_info`、`created_at`，以及展示用的 `user_display_name`、`document_title`（2026-11-10 新增）。文档详情页「访问记录」使用 `document_id` + `page_size=10` 调用本接口。带 `document_id` 查询时，只要调用者有权查看该文档（例如跨课题转交后的当前责任人），即可看到这份文档的全部审计事件；结果仍只限于这一份文档。

### 8.5 审计摘要

`GET /api/v1/audit-events/summary?project_id={id}`

响应体：

```json
{
  "data": {
    "download_count": 18,
    "upload_count": 23,
    "transfer_count": 14,
    "archive_count": 6,
    "view_count": 40,
    "preview_count": 9,
    "recent_download_count": 7,
    "top_active_users": [
      {
        "user_id": "uuid",
        "display_name": "张三",
        "action_count": 12
      }
    ],
    "top_downloads": [
      {
        "document_id": "uuid",
        "title": "开题报告",
        "download_count": 4,
        "user_count": 2,
        "last_downloaded_at": "2026-11-10T06:00:00Z"
      }
    ]
  }
}
```

说明：`recent_download_count` 与 `top_downloads` 统计近 30 天；`download_count` 含数据资产下载。总览页「下载活跃度」面板使用本接口。

---

## 8.6 待办与通知（2026-11-10）

### 8.6.1 我的待办

`GET /api/v1/inbox`

按调用者实时计算，不落库。每一类都对应调用者确实有权执行的动作，处理完成后自动消失。

| `kind` | 条件 | 跳转 |
| --- | --- | --- |
| `accept_transfer` | 文档转交给我（责任人是我且状态为 `pending_handover`） | `/documents/{id}` |
| `confirm_handover` | 我是接收人且交接单状态为 `generated` | `/handovers?id={id}` |
| `complete_handover` | 我管理的课题中，交接单状态为 `pending_confirm` | `/handovers?id={id}` |
| `fill_handover_items` | 我管理的课题中，`generated` 状态且未勾选任何文档或数据 | `/handovers?id={id}` |
| `review_suggestion` | 我负责的文档上有待确认的 AI 建议 | `/documents/{id}` |
| `stale_document` | 我负责的在办文档超过 30 天未更新 | `/documents/{id}` |

响应：`data` 为 `[{kind, title, detail, link, related_type, related_id, project_name, since}]`，最多 100 条；`meta` 为 `{total, counts: {kind: n}}`。

### 8.6.2 通知

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/notifications?unread_only=true&page=&page_size=` | 本人通知列表，`meta` 含 `pagination` 与 `unread` |
| GET | `/notifications/unread-count` | 未读数 `{unread}`，顶栏每 60 秒轮询 |
| POST | `/notifications/{id}/read` | 标记单条已读；不是本人的通知返回 404 |
| POST | `/notifications/read-all` | 全部标为已读，返回 `{updated}` |

通知项字段：`id`、`event_type`、`title`、`body`、`link`、`related_type`、`related_id`、`actor_name`、`read`、`created_at`。

触发规则：

| 事件 | 收件人 | 外发 |
| --- | --- | --- |
| 转交文档 | 接收人 | 是 |
| 接收转交 | 原转交人 | 否 |
| 创建交接单 | 接收人（外发）、被交接人 | 接收人是 |
| 接收人确认交接 | 课题 owner/manager 与发起人 | 是 |
| 完成 / 取消交接 | 被交接人、接收人 | 否 |
| 文档 / 交接 AI 摘要完成或失败 | 发起人 | 否 |
| 负责人添加成员 | 被添加人 | 否 |
| 通过邀请链接入组 | 课题 owner/manager | 否 |

外发通道由部署配置开启，见部署说明「通知外发」。普通问答不发通知，提问人本就在会话页等待。

## 9. OpenClaw 接口

当前实现说明：

- Python Worker 通过 OpenClaw Gateway 的 OpenAI 兼容接口 `POST /v1/chat/completions` 调用 AI 能力。
- Worker 只读取本系统暴露的受控内部上下文，不直接访问业务数据库。
- 普通问答已进入“会话 + 追问”模式，业务侧显式装配最近会话、历史回答和已确认建议，禁止依赖 OpenClaw 宿主环境隐式记忆。
- Worker 已增加 `skill_registry` / `skill_adapter`，仅允许白名单内的无状态 skill 复用，且统一只消费显式 `scope / context / memory`。
- 当前摘要能力优先基于结构化业务上下文；若尚未提供文档正文，则结果属于“元数据级摘要”。
- 当前正文抽取支持：`txt`、`md`、`csv`、`json`、`docx`、`pdf`。
- 图片与扫描 PDF OCR 依赖 Worker 主机安装 `tesseract`；若缺失则返回明确错误。

### 9.0 会话接口

#### 9.0.1 创建会话

`POST /api/v1/assistant/conversations`

请求体：

```json
{
  "title": "课题A 流转跟踪",
  "scope": {
    "project_id": "uuid",
    "document_id": null
  }
}
```

#### 9.0.2 查询会话列表

`GET /api/v1/assistant/conversations`

查询参数：

- `scope_type`
- `scope_id`
- `project_id`
- `document_id`

#### 9.0.3 查询会话消息列表

`GET /api/v1/assistant/conversations/{conversation_id}/messages`

### 9.1 发起问答

`POST /api/v1/assistant/ask`

请求体：

```json
{
  "conversation_id": "uuid",
  "scope": {
    "project_id": "uuid",
    "document_id": null
  },
  "skill_name": "answer_with_context",
  "question": "课题A 最近一个月有哪些文档在流转？"
}
```

响应体：

```json
{
  "data": {
    "request_id": "uuid",
    "conversation_id": "uuid",
    "question": "课题A 最近一个月有哪些文档在流转？",
    "status": "queued",
    "answer": "",
    "source_scope": {
      "project_id": "uuid",
      "document_id": null
    },
    "memory_sources": [
      {
        "type": "conversation_messages",
        "count": 4
      }
    ],
    "skill_name": "answer_with_context",
    "skill_version": "v1",
    "generated_at": "2026-04-03T16:00:00Z"
  }
}
```

说明：

- 首次提问可不传 `conversation_id`，后端会自动创建会话并返回；
- 继续追问时传入 `conversation_id` 即可，若未再次传 `scope`，默认沿用会话绑定范围；
- 会话必须绑定单一 `scope`，不允许跨项目自动串话。
- `skill_name` 为可选字段；若未传，Worker 会按任务类型选取白名单中的默认 skill。

### 9.1.1 查询问答任务状态

`GET /api/v1/assistant/requests/{request_id}`

响应体：

```json
{
  "data": {
    "id": "uuid",
    "request_type": "assistant.ask",
    "conversation_id": "uuid",
    "status": "completed",
    "question": "课题A 最近一个月有哪些文档在流转？",
    "source_scope": {
      "project_id": "uuid",
      "document_id": null
    },
    "memory_sources": [
      {
        "type": "confirmed_suggestions",
        "count": 2
      }
    ],
    "error_message": "",
    "skill_name": "answer_with_context",
    "skill_version": "v1",
    "output": {
      "answer": "最近一个月共有 4 份文档发生流转……"
    },
    "model": "openclaw/default",
    "upstream_request_id": "chatcmpl_xxx",
    "processing_duration_ms": 2140,
    "created_at": "2026-04-06T09:00:00Z",
    "completed_at": "2026-04-06T09:00:03Z"
  }
}
```

2026-11-10 起的输出字段：

- 运行中（`status=running`）时 `output.partial_answer` 为模型已生成的部分回答，Worker 约每秒回传一次。前端以 1 秒间隔轮询并实时渲染，完成后以 `output.answer` 为准。
- `output.citations`：回答所依据的正文片段，`[{label, document_id, title, version_no, chunk_no, snippet}]`。回答正文用 `[片段N]` 标注来源，会话消息的 `metadata.citations` 同步保存。
- 正文片段来源：文档范围从该文档正文中按问题选取；课题范围调用 9.5.5 的内部检索。总长度受 Worker `MAX_CONTEXT_CHARS`（默认 12000 字）约束，不再把整篇正文塞进提示词。

### 9.1.2 查询问答历史列表

`GET /api/v1/assistant/requests`

查询参数：

- `request_type`
- `status`
- `keyword`
- `conversation_id`
- `related_type`
- `related_id`
- `page`
- `page_size`

响应体：

```json
{
  "data": [
    {
      "id": "uuid",
      "request_type": "assistant.ask",
      "status": "completed",
      "question": "课题A 最近一个月有哪些文档在流转？",
      "model": "openclaw/default",
      "upstream_request_id": "chatcmpl_xxx",
      "processing_duration_ms": 2140,
      "created_at": "2026-04-06T09:00:00Z",
      "completed_at": "2026-04-06T09:00:03Z"
    }
  ],
  "meta": {
    "page": 1,
    "page_size": 20,
    "total": 1
  }
}
```

### 9.2 生成文档摘要

`POST /api/v1/assistant/documents/{document_id}/summarize`

请求体：

```json
{
  "version_id": "uuid",
  "skill_name": "document_summary"
}
```

响应体：

```json
{
  "data": {
    "request_id": "uuid",
    "status": "queued"
  }
}
```

说明：

- `skill_name` 为可选字段；若未传，Worker 会按 `document.summarize` 的白名单选取默认 skill。
- 2026-11-10：未传 `version_id` 时摘要写入 Worker 实际读取的最新版本。输出新增 `summary_basis`：`full_text`（全文）、`map_reduce`（长文分段提炼后汇总）、`map_reduce_sampled`（超长文均匀抽样分段）、`metadata_only`（无正文）。
- 自动摘要：上传新版本后正文抽取成功，若后端 `AUTO_SUMMARIZE_ON_UPLOAD=true`（默认），会以上传人身份自动排队一次摘要；摘要完成后上传人收到站内通知。
- 摘要建议可带可选字段 `suggested_action`，取值仅限 `mark_in_progress`、`finalize`、`archive`，其余值由 Worker 和后端双重丢弃。

### 9.3 生成交接摘要

`POST /api/v1/assistant/handovers/{handover_id}/summarize`

请求体（可选）：

```json
{
  "skill_name": "handover_summary"
}
```

### 9.4 查询建议列表

`GET /api/v1/assistant/suggestions`

当前实现说明：

- 已支持从 `assistant_suggestions` 读取结果
- 当前主要由 Worker 回调后的摘要/建议结果生成记录

查询参数：

- `related_type`
- `related_id`
- `status`
- `suggestion_type`

响应体：

```json
{
  "data": [
    {
      "id": "uuid",
      "related_type": "document",
      "related_id": "uuid",
      "suggestion_type": "document_summary",
      "status": "pending",
      "title": "文档摘要",
      "content": "本次摘要结果……",
      "source_scope": "{\"project_id\":\"uuid\"}",
      "request_id": "uuid",
      "generated_at": "2026-04-03T16:05:00Z"
    }
  ]
}
```

### 9.5 Worker 内部上下文接口

这些接口仅供 `backend-py-worker` 使用，统一使用 `Authorization: Bearer <worker-callback-token>` 鉴权。

#### 9.5.1 查询项目上下文

`GET /api/v1/internal/assistant-context/projects/{project_id}`

响应体：

```json
{
  "data": {
    "scope": {
      "project_id": "uuid"
    },
    "overview": {
      "document_total": 12,
      "status_counts": {
        "in_progress": 4
      },
      "handover_pending_count": 1,
      "risk_document_count": 2
    },
    "recent_flows": [],
    "risk_documents": []
  }
}
```

#### 9.5.2 查询文档上下文

`GET /api/v1/internal/assistant-context/documents/{document_id}`

响应体：

```json
{
  "data": {
    "scope": {
      "document_id": "uuid"
    },
    "document": {
      "id": "uuid",
      "title": "课题记录",
      "current_status": "in_progress"
    },
    "versions": [],
    "flows": [],
    "extracted_text": "最近一次已抽取的正文内容"
  }
}
```

#### 9.5.3 查询交接上下文

`GET /api/v1/internal/assistant-context/handovers/{handover_id}`

响应体：

```json
{
  "data": {
    "scope": {
      "handover_id": "uuid",
      "project_id": "uuid"
    },
    "handover": {
      "id": "uuid",
      "status": "pending"
    }
  }
}
```

### 9.5.4 下载版本原始文件

`GET /api/v1/internal/assistant-assets/versions/{version_id}/download`

说明：

- 仅供 Worker 使用；
- 使用 Worker shared token 鉴权；
- 返回版本原始文件流，供正文抽取器处理。

### 9.5.5 课题正文检索（2026-11-10）

`GET /api/v1/internal/assistant-context/projects/{project_id}/search?q=&document_id=&limit=8`

- 仅供 Worker 使用，Worker shared token 鉴权；提问人对该课题的查看权限已在 `POST /assistant/ask` 入队时校验。
- 在 `document_text_chunks` 中检索该课题文档**当前版本**的正文片段，按命中的查询词数量排序。查询词规则：英文与数字按单词，中文按两字一组，去掉「什么」「如何」等提问用语。
- 响应：`[{document_id, title, project_id, project_name, version_id, version_no, chunk_no, content, snippet, score}]`。

### 9.5.6 增量回答回传（2026-11-10）

`POST /api/v1/internal/worker-progress`

请求体 `{"request_id": "uuid", "partial_answer": "已生成的部分回答"}`。只更新状态为 `running` 的请求，请求完成后的迟到回传会被忽略。Worker 单次尝试、3 秒超时，失败不影响最终结果。

### 9.6 确认建议

`POST /api/v1/assistant/suggestions/{suggestion_id}/confirm`

请求体：

```json
{
  "note": "采纳该建议",
  "execute": false
}
```

响应体：

```json
{
  "data": {
    "id": "uuid",
    "status": "confirmed",
    "confirmed_by": "uuid",
    "note": "采纳该建议",
    "executed_action": "finalize"
  }
}
```

确认并执行（2026-11-10）：`execute=true` 且建议带 `suggested_action` 时，后端先以**确认人身份**执行该流转动作，成功后再把建议标为已确认，响应带 `executed_action`。流转动作照常经过权限与状态机校验；执行失败时返回对应错误（403 / 409），建议保持 `pending`。建议不是 `pending`、没有可执行动作或不属于文档时返回 400。AI 本身从不执行任何动作。

### 9.7 忽略建议

`POST /api/v1/assistant/suggestions/{suggestion_id}/dismiss`

请求体：

```json
{
  "reason": "当前无需处理"
}
```

响应体：

```json
{
  "data": {
    "id": "uuid",
    "status": "dismissed",
    "dismissed_by": "uuid",
    "reason": "当前无需处理"
  }
}
```

---

## 10. 群晖 NAS 适配接口

这部分是平台内部接口，不直接暴露给前端，但后端模块应按能力预留。

### 10.1 存储抽象接口

建议统一在应用层抽象以下方法：

- `put_object(file, target_path, metadata)`
- `get_object(object_ref)`
- `delete_object(object_ref)`
- `list_path(path)`
- `move_path(source, target)`
- `create_folder(path)`
- `create_share_link(path, options)`
- `check_permission(path, action)`

### 10.2 群晖映射关系

- `put_object` -> `SYNO.FileStation.Upload`
- `get_object` -> `SYNO.FileStation.Download`
- `list_path` -> `SYNO.FileStation.List`
- `move_path` -> `SYNO.FileStation.CopyMove`
- `create_folder` -> `SYNO.FileStation.CreateFolder`
- `create_share_link` -> `SYNO.FileStation.Sharing`
- `check_permission` -> `SYNO.FileStation.CheckPermission`

---

## 11. 典型错误码建议

### 11.1 通用错误

- `unauthorized`
- `forbidden`
- `not_found`
- `validation_error`
- `conflict`
- `internal_error`

### 11.2 业务错误

- `invalid_status_transition`
- `document_archived`
- `document_deleted`
- `handover_status_invalid`
- `permission_denied_on_project`
- `storage_provider_error`
- `assistant_timeout`
- `assistant_scope_forbidden`

### 11.3 请求体大小限制

| 类别 | 上限 | 超出返回 |
| --- | --- | --- |
| 普通 JSON 请求体 | **1 MB** | `413 payload_too_large` |
| 创建文档 / 提交版本（`multipart/form-data`） | **32 MB** | `413 payload_too_large`，`message` 为「文件超过 32 MB 上限」 |
| 上传数据资产（`multipart/form-data`） | **512 MB** | `413 payload_too_large`，`message` 为「文件超过 512 MB 上限」 |
| git smart-HTTP（`/api/v1/git/*`）、Worker 回调（`/api/v1/internal/*`） | 不限 | — |

说明：上传上限由 `http.MaxBytesReader` 在处理器入口强制，常量定义在 `backend-go/internal/shared/upload.go`。此前代码把上限值传给 `ParseMultipartForm`，而该参数只是内存缓冲大小、超出部分会落盘，因此文档中的 32 MB 实际从未生效；现已改为真实强制。

反向代理侧 `client_max_body_size`（`frontend/nginx/default.conf`）只作为粗粒度外层边界，应保持 **大于等于** 上表最大值（当前 512m），否则超限请求会得到 nginx 的通用错误页而不是 API 的可读 413。

---

## 12. 首期必须优先实现的 API

### 12.1 核心闭环

- 登录
- 当前用户
- 团队空间列表
- 项目列表
- 目录树
- 创建文档
- 文档列表
- 文档详情
- 提交新版本
- 版本列表
- 发起转交
- 接收交接
- 流转历史
- 生成交接单
- 交接详情
- 确认交接
- 完成交接
- 总览
- 风险文档
- 审计摘要
- 发起问答
- 查询建议

### 12.2 第二批实现

- 删除与恢复文档
- 版本预览增强信息（未来可选）
- 审计事件明细列表
- 建议确认与忽略
- 交接项编辑

### 12.3 第三批实现（管理员模块）

- 创建团队空间 `POST /api/v1/admin/team-spaces`
- 创建项目 `POST /api/v1/admin/projects`
- 创建用户 `POST /api/v1/admin/users`
- 更新用户 `PUT /api/v1/admin/users/{id}`
- 列出全部用户 `GET /api/v1/admin/users`
- 列出项目成员 `GET /api/v1/admin/projects/{id}/members`
- 添加项目成员 `POST /api/v1/admin/projects/{id}/members`
- 更新项目成员角色 `PUT /api/v1/admin/projects/{id}/members/{userId}`
- 移除项目成员 `DELETE /api/v1/admin/projects/{id}/members/{userId}`

---

## 13. 数据资产模块（第四批，已实现）

### 13.1 概述

数据资产模块用于存储图片、视频、压缩包、模型文件等非 Office/PDF 数据文件。以课题为核心，支持单文件最大 511 MiB（请求体最大 512 MiB），简单文件夹结构（最多 2 层），无工作流与版本管理。

**权限规则：**
- 上传：项目成员即可（`CanUploadDataAsset`）
- 删除/管理：资产创建者、项目 owner、项目 manager（`CanManageDataAsset`）

### 13.2 数据资产端点

#### 列出数据资产
`GET /api/v1/data-assets`

Query: `project_id`, `folder_id`, `keyword`, `page`, `page_size`

权限范围：非 `admin` 角色只能看到自己拥有或所属项目下的数据资产，省略 `project_id` 或传入无权限的项目 ID 都不会绕过该限制。`GET /{id}` 详情与 `GET /{id}/download` 下载同样要求调用者是资产所属项目的成员（或该资产创建人），否则返回 403。

```json
{
  "data": [
    {
      "id": "uuid",
      "project_id": "uuid",
      "project_name": "五好爱学 AI 平台",
      "folder_id": "uuid",
      "folder_name": "原始数据",
      "display_name": "训练集 v3",
      "file_name": "train_v3.zip",
      "mime_type": "application/zip",
      "file_size": 1073741824,
      "created_by_name": "张三",
      "created_at": "2026-04-17T12:00:00Z"
    }
  ],
  "total": 1,
  "page": 1,
  "page_size": 20
}
```

#### 上传数据资产
`POST /api/v1/data-assets`（multipart/form-data）

字段：`file`（二进制）、`project_id`（必填）、`display_name`（可选）、`folder_id`（可选）、`description`（可选）

文件名命名约束由前端执行（仅允许字母、数字、中文、下划线、连字符、点）。

#### 获取数据资产详情
`GET /api/v1/data-assets/{id}`

#### 更新数据资产
`PUT /api/v1/data-assets/{id}`

Body: `{ "display_name": "...", "description": "...", "folder_id": "..." }`

#### 删除数据资产（软删除）
`DELETE /api/v1/data-assets/{id}`

#### 下载数据资产
`GET /api/v1/data-assets/{id}/download`

流式返回文件内容，`Content-Disposition: attachment`，`Content-Length` 按文件实际大小设置。返回前写入 `download` 审计事件（`extra_data` 含 `data_asset_id`、`project_id`），写入失败返回 `500 audit_failed`。

### 13.3 文件夹端点

#### 列出课题文件夹
`GET /api/v1/projects/{id}/data-folders`

返回扁平列表，前端根据 `parent_id` 和 `depth` 自行构建树。

#### 创建文件夹
`POST /api/v1/data-folders`

Body: `{ "project_id": "uuid", "parent_id": "uuid（可选）", "name": "原始数据" }`

深度超过 2 返回 400 `validation_error`，同目录同名返回 409 `conflict`。

#### 删除文件夹
`DELETE /api/v1/data-folders/{id}`

### 13.4 交接单数据资产清单端点

#### 查看交接单数据资产清单
`GET /api/v1/handovers/{id}/data-items`

```json
{
  "data": [
    {
      "data_asset_id": "uuid",
      "display_name": "训练集 v3",
      "file_name": "train_v3.zip",
      "selected": true,
      "note": "最终版训练数据"
    }
  ]
}
```

#### 更新交接单数据资产清单
`PUT /api/v1/handovers/{id}/data-items`

Body: `{ "items": [{ "data_asset_id": "uuid", "selected": true, "note": "..." }] }`

注：使用 PUT（全量覆盖），而非 PATCH。只有 `generated` 状态的交接单可编辑。

---

## 14. 代码资产模块（第一版，已实现）

### 14.1 概述

代码资产模块用于为课题创建受控 Git remote。成员在本地完成 `git add`、`git commit` 后，可以将默认分支 push 到平台维护的 bare repository。平台记录仓库配置、目标代码文件夹、最近提交和 push 事件，并将默认分支提交内容通过统一存储适配层写入目标代码文件夹。

首期边界：

- 不提供 GitHub/GitLab 式 PR、Issue、代码评审或 CI/CD；
- Git 接收入口使用 Basic token；
- 目标代码文件夹作为受控路径，首期执行新增/覆盖同步；删除差量清理后续增强；
- 创建和配置仓库：`admin`、项目 `owner/manager`；
- push：仓库 token 校验通过后允许写入 Git 仓库，并记录事件。

### 14.2 代码仓库端点

#### 列出代码仓库

`GET /api/v1/code-repositories`

Query: `project_id`, `keyword`, `page`, `page_size`

权限范围：非 `admin` 角色只能看到自己拥有或所属项目下的代码仓库；`GET /{id}` 详情同样要求调用者是该仓库所属项目的成员，否则返回 403。`push_token` 只在创建响应（`POST`）中一次性返回，`GET`（含列表和详情）不会回显；Git smart HTTP push 入口走仓库 `push_token` 校验，属于独立信任边界，不受这里的项目成员校验约束。

#### 创建代码仓库

`POST /api/v1/code-repositories`

请求：

```json
{
  "team_space_id": "uuid",
  "project_id": "uuid",
  "name": "前端代码",
  "description": "课题前端工程",
  "default_branch": "main",
  "target_folder_path": "/projects/五好爱学 AI 平台/code/frontend"
}
```

响应会返回 `remote_url` 和第一版 `push_token`，前端用于生成推送命令。

#### 获取代码仓库详情

`GET /api/v1/code-repositories/{id}`

返回仓库详情、`remote_url`、最近提交和同步目标。

#### 更新代码仓库

`PATCH /api/v1/code-repositories/{id}`

请求：

```json
{
  "name": "前端代码",
  "description": "更新说明",
  "default_branch": "main",
  "target_folder_path": "/projects/五好爱学 AI 平台/code/frontend"
}
```

#### 查看 push 事件

`GET /api/v1/code-repositories/{id}/push-events`

响应：

```json
{
  "data": [
    {
      "id": "uuid",
      "repository_id": "uuid",
      "branch": "main",
      "after_sha": "abc123",
      "commit_message": "initial commit",
      "sync_status": "synced",
      "created_at": "2026-04-19T10:00:00Z",
      "completed_at": "2026-04-19T10:00:00Z"
    }
  ]
}
```

### 14.3 Git remote 入口

`GET/POST /api/v1/git/{slug}.git/...`

该入口服务 Git smart HTTP，供本地 Git 客户端使用：

```bash
git remote add digidocs http://git:{push_token}@host/api/v1/git/{slug}.git
git push digidocs main
```

认证失败返回 `401 unauthorized`。仓库不存在返回 `404`。


## 15. 2026-10-22 权限与版本契约补充

- 项目成员 PATCH/DELETE 的 memberID 必须属于 URL projectID；数据库更新带双重条件，最后 owner 的保护在事务中执行。
- 数据目录 GET 必须具备项目读取权限；DELETE 必须具备项目数据上传权限。
- 交接 data-items GET 复用交接读取权限；PUT 复用交接编辑权限，仅 generated 状态可编辑，事务内锁定交接单，资产必须未删除且属于交接项目（无项目交接按 target_user_id 对应资产创建人限定），写入审计。
- assistant/ask 追问必须具备会话所有权；document scope 的 project_id 由文档归属推导，客户端冲突范围拒绝。
- 摘要只接受 version_id、skill_name；交接摘要只接受 skill_name。版本必须属于 URL 文档；不接受客户端扩大上下文范围。
- Worker 文档上下文支持 version_id 查询参数，校验版本归属；响应包含 version_id、extracted_text_version_id。正文缓存仅按文档及输出 version_id 匹配，旧的无版本结果不复用。
- 数据上传请求体上限仍为 512 MiB，页面单文件限制为 511 MiB，预留 1 MiB 表单开销；选择文件时即校验，不再承诺 10 GB。
