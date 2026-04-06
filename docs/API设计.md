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
    "last_login_at": "2026-04-03T10:00:00Z"
  }
}
```

### 2.3 退出登录

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

### 3.1 团队空间列表

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

### 3.2 项目列表

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

### 3.3 目录树

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

响应体：

```json
{
  "data": {
    "preview_type": "pdf",
    "preview_url": "https://example.com/preview/xxx",
    "watermark_enabled": true
  }
}
```

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
- 当前摘要能力优先基于结构化业务上下文；若尚未提供文档正文，则结果属于“元数据级摘要”。

### 9.1 发起问答

`POST /api/v1/assistant/ask`

请求体：

```json
{
  "scope": {
    "project_id": "uuid",
    "document_id": null
  },
  "question": "课题A 最近一个月有哪些文档在流转？"
}
```

响应体：

```json
{
  "data": {
    "request_id": "uuid",
    "question": "课题A 最近一个月有哪些文档在流转？",
    "status": "queued",
    "answer": "",
    "source_scope": {
      "project_id": "uuid",
      "document_id": null
    },
    "generated_at": "2026-04-03T16:00:00Z"
  }
}
```

### 9.2 生成文档摘要

`POST /api/v1/assistant/documents/{document_id}/summarize`

请求体：

```json
{
  "version_id": "uuid"
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

### 9.3 生成交接摘要

`POST /api/v1/assistant/handovers/{handover_id}/summarize`

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
    "flows": []
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
- 版本预览信息
- 审计事件明细列表
- 建议确认与忽略
- 交接项编辑
