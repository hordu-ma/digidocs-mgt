---
name: digidocs-frontend-vue
description: 约束 frontend/ 中的 Vue 页面实现，确保界面围绕固定角色、固定业务对象和明确 API 契约展开。
---

# Skill: digidocs-frontend-vue

## 目的

约束 `frontend/` 中的 Vue 页面实现，确保界面围绕固定角色、固定业务对象和明确 API 契约展开。

## 触发条件

- 需求涉及 `frontend/`
- 需求涉及 dashboard、文档详情、交接页面、负责人视图

## 必读信息源

1. `docs/项目定义与技术架构.md`
2. `docs/API设计.md`
3. `AGENTS.md`
4. `TASKS.md`

## 强约束

- 页面先围绕角色职责和固定流程建模，不做无边界通用后台。
- 文档详情必须区分“正式事实”和“AI 建议”。
- 接口字段命名与响应结构必须对齐 `docs/API设计.md`。

## 最小验证

```bash
cd frontend
npm run build
```

## 完成检查

- 页面语义没有越界
- API 契约没有漂移
- 构建已执行或已说明阻塞原因
- `README.md` 与 `TASKS.md` 已同步
