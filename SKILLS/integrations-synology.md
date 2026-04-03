# Skill: integrations-synology

## 适用范围

- 群晖 DSM Web API 对接
- File Station API 封装
- 存储适配层实现

## 规则

- 不在业务服务中直接调用 `SYNO.*` API。
- 所有调用必须通过统一适配器封装。
- 接口能力优先围绕：
  - 登录
  - 列目录
  - 上传
  - 下载
  - 复制/移动
  - 删除
  - 共享链接
  - 权限检查
- 使用 `SYNO.API.Info` 先做能力探测，避免直接假设 API 版本。

## 官方资料

- File Station API Guide
  - `https://global.download.synology.com/download/Document/Software/DeveloperGuide/Package/FileStation/All/enu/Synology_File_Station_API_Guide.pdf`

