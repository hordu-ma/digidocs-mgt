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

`POST /api/v1/auth/login`

请求体：

```json
{
  "username": "zhangsan",
  "password": "******"
}
```

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

### 2.4 退出登录

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
      }
    }
  ]
}
```

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

查询参数：

- `team_space_id`
- `project_id`
- `folder_id`
- `owner_id`
- `status`
- `keyword`
- `include_archived`
- `page`
- `page_size`

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
      "updated_at": "2026-04-03T10:00:00Z"
    }
  ],
  "meta": {
    "page": 1,
    "page_size": 20,
    "total": 128
  }
}
```

### 4.3 文档详情

`GET /api/v1/documents/{document_id}`

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

### 5.5 版本预览信息

`GET /api/v1/versions/{version_id}/preview`

当前实现：响应文件流，行为与下载接口类似，但以浏览器内联打开为目标。

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
    "candidate_count": 12,
    "generated_at": "2026-04-03T15:00:00Z"
  }
}
```

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
    "target_user": {
      "id": "uuid",
      "display_name": "王五"
    },
    "receiver_user": {
      "id": "uuid",
      "display_name": "赵六"
    },
    "items": [
      {
        "document_id": "uuid",
        "title": "实验记录",
        "selected": true,
        "current_status": "in_progress"
      }
    ],
    "ai_summary": "该成员主要遗留实验记录、汇报PPT和数据分析表。"
  }
}
```

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
      "risk_message": "超过30天未更新"
    }
  ]
}
```

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
    "top_active_users": [
      {
        "user_id": "uuid",
        "display_name": "张三",
        "count": 12
      }
    ]
  }
}
```

---

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

### 9.6 确认建议

`POST /api/v1/assistant/suggestions/{suggestion_id}/confirm`

请求体：

```json
{
  "note": "采纳该建议"
}
```

响应体：

```json
{
  "data": {
    "id": "uuid",
    "status": "confirmed",
    "confirmed_by": "uuid",
    "note": "采纳该建议"
  }
}
```

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

数据资产模块用于存储图片、视频、压缩包、模型文件等非 Office/PDF 数据文件。以课题为核心，支持 1-10GB 大文件，简单文件夹结构（最多 2 层），无工作流与版本管理。

**权限规则：**
- 上传：项目成员即可（`CanUploadDataAsset`）
- 删除/管理：资产创建者、项目 owner、项目 manager（`CanManageDataAsset`）

### 13.2 数据资产端点

#### 列出数据资产
`GET /api/v1/data-assets`

Query: `project_id`, `folder_id`, `keyword`, `page`, `page_size`

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

流式返回文件内容，`Content-Disposition: attachment`，`Content-Length` 按文件实际大小设置。

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
