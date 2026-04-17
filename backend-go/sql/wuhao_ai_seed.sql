-- DigiDocs Mgt wuhao-ai 测试项目数据
-- 前提：已执行 backend-go/migrations/001_initial_schema.sql，且基础 seed 用户/团队已存在。
-- 用法：psql -h localhost -p 15432 -U postgres -d digidocs_mgt -f backend-go/sql/wuhao_ai_seed.sql

BEGIN;

-- ========== 项目 ==========
INSERT INTO projects (id, team_space_id, name, code, description, owner_id, status)
VALUES (
  '20000000-0000-0000-0000-000000000004',
  '10000000-0000-0000-0000-000000000002',
  '五好爱学 AI 教育产品资料库',
  'wuhao-ai',
  '五好爱学产品、规划、用户手册、生涯规划、资质与素材测试项目',
  '00000000-0000-0000-0000-000000000010',
  'active'
)
ON CONFLICT ON CONSTRAINT uq_projects_team_space_code DO UPDATE
SET name = EXCLUDED.name,
    description = EXCLUDED.description,
    owner_id = EXCLUDED.owner_id,
    status = EXCLUDED.status,
    updated_at = NOW();

-- ========== 项目成员 ==========
INSERT INTO project_members (project_id, user_id, project_role)
VALUES
  ('20000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000010', 'owner'),
  ('20000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000011', 'contributor'),
  ('20000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000012', 'contributor'),
  ('20000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000013', 'contributor')
ON CONFLICT ON CONSTRAINT uq_project_members_project_user DO UPDATE
SET project_role = EXCLUDED.project_role,
    updated_at = NOW();

-- ========== 目录 ==========
INSERT INTO folders (id, project_id, parent_id, name, path, sort_order)
VALUES
  ('30000000-0000-0000-0000-000000000006', '20000000-0000-0000-0000-000000000004', NULL, '产品规划', '/产品规划', 10),
  ('30000000-0000-0000-0000-000000000007', '20000000-0000-0000-0000-000000000004', NULL, '用户手册', '/用户手册', 20),
  ('30000000-0000-0000-0000-000000000008', '20000000-0000-0000-0000-000000000004', NULL, '生涯规划', '/生涯规划', 30),
  ('30000000-0000-0000-0000-000000000009', '20000000-0000-0000-0000-000000000004', NULL, '资质证明', '/资质证明', 40),
  ('30000000-0000-0000-0000-000000000010', '20000000-0000-0000-0000-000000000004', NULL, '素材图片', '/素材图片', 50)
ON CONFLICT ON CONSTRAINT uq_folders_project_path DO UPDATE
SET name = EXCLUDED.name,
    sort_order = EXCLUDED.sort_order,
    updated_at = NOW();

-- ========== 文档 ==========
INSERT INTO documents (
  id, team_space_id, project_id, folder_id, title, description, file_type,
  current_owner_id, current_status, current_version_id, is_archived, is_deleted,
  created_by, created_at, updated_at
)
VALUES
  ('40000000-0000-0000-0000-000000000101', '10000000-0000-0000-0000-000000000002', '20000000-0000-0000-0000-000000000004', '30000000-0000-0000-0000-000000000006',
   '五好教育发展及规划', '五好教育发展方向、产品规划与资料沉淀。', 'docx',
   '00000000-0000-0000-0000-000000000010', 'in_progress', '50000000-0000-0000-0000-000000000101', false, false,
   '00000000-0000-0000-0000-000000000010', NOW() - INTERVAL '12 days', NOW() - INTERVAL '2 days'),
  ('40000000-0000-0000-0000-000000000102', '10000000-0000-0000-0000-000000000002', '20000000-0000-0000-0000-000000000004', '30000000-0000-0000-0000-000000000006',
   '学情雷达系统建设路径', '学情雷达系统建设方案与路径说明。', 'pdf',
   '00000000-0000-0000-0000-000000000010', 'in_progress', '50000000-0000-0000-0000-000000000102', false, false,
   '00000000-0000-0000-0000-000000000010', NOW() - INTERVAL '10 days', NOW() - INTERVAL '1 day'),
  ('40000000-0000-0000-0000-000000000103', '10000000-0000-0000-0000-000000000002', '20000000-0000-0000-0000-000000000004', '30000000-0000-0000-0000-000000000007',
   'Wuhao Tutor 用户手册', 'Wuhao Tutor 产品用户手册。', 'pdf',
   '00000000-0000-0000-0000-000000000011', 'finalized', '50000000-0000-0000-0000-000000000103', false, false,
   '00000000-0000-0000-0000-000000000011', NOW() - INTERVAL '30 days', NOW() - INTERVAL '8 days'),
  ('40000000-0000-0000-0000-000000000104', '10000000-0000-0000-0000-000000000002', '20000000-0000-0000-0000-000000000004', '30000000-0000-0000-0000-000000000007',
   '寒假特训营日志', '寒假特训营运营与执行日志。', 'pdf',
   '00000000-0000-0000-0000-000000000013', 'pending_handover', '50000000-0000-0000-0000-000000000104', false, false,
   '00000000-0000-0000-0000-000000000012', NOW() - INTERVAL '20 days', NOW() - INTERVAL '3 days'),
  ('40000000-0000-0000-0000-000000000105', '10000000-0000-0000-0000-000000000002', '20000000-0000-0000-0000-000000000004', '30000000-0000-0000-0000-000000000008',
   '五好生涯规划介绍', '生涯规划产品介绍材料。', 'pdf',
   '00000000-0000-0000-0000-000000000011', 'in_progress', '50000000-0000-0000-0000-000000000105', false, false,
   '00000000-0000-0000-0000-000000000011', NOW() - INTERVAL '18 days', NOW() - INTERVAL '4 days'),
  ('40000000-0000-0000-0000-000000000106', '10000000-0000-0000-0000-000000000002', '20000000-0000-0000-0000-000000000004', '30000000-0000-0000-0000-000000000008',
   'MBTI 职业性格测评解析', 'MBTI 职业性格测评解析资料。', 'docx',
   '00000000-0000-0000-0000-000000000011', 'draft', '50000000-0000-0000-0000-000000000106', false, false,
   '00000000-0000-0000-0000-000000000011', NOW() - INTERVAL '7 days', NOW() - INTERVAL '7 days'),
  ('40000000-0000-0000-0000-000000000107', '10000000-0000-0000-0000-000000000002', '20000000-0000-0000-0000-000000000004', '30000000-0000-0000-0000-000000000008',
   '职业生涯规划分享课件', '职业生涯规划分享课件。', 'pptx',
   '00000000-0000-0000-0000-000000000012', 'in_progress', '50000000-0000-0000-0000-000000000107', false, false,
   '00000000-0000-0000-0000-000000000012', NOW() - INTERVAL '9 days', NOW() - INTERVAL '2 days'),
  ('40000000-0000-0000-0000-000000000108', '10000000-0000-0000-0000-000000000002', '20000000-0000-0000-0000-000000000004', '30000000-0000-0000-0000-000000000008',
   '五好生涯报价体系', '五好生涯服务报价体系。', 'xlsx',
   '00000000-0000-0000-0000-000000000012', 'draft', '50000000-0000-0000-0000-000000000108', false, false,
   '00000000-0000-0000-0000-000000000012', NOW() - INTERVAL '6 days', NOW() - INTERVAL '6 days'),
  ('40000000-0000-0000-0000-000000000109', '10000000-0000-0000-0000-000000000002', '20000000-0000-0000-0000-000000000004', '30000000-0000-0000-0000-000000000009',
   '网信算法备案公示内容', '网信算法备案公示材料。', 'pdf',
   '00000000-0000-0000-0000-000000000010', 'finalized', '50000000-0000-0000-0000-000000000109', false, false,
   '00000000-0000-0000-0000-000000000010', NOW() - INTERVAL '40 days', NOW() - INTERVAL '15 days'),
  ('40000000-0000-0000-0000-000000000110', '10000000-0000-0000-0000-000000000002', '20000000-0000-0000-0000-000000000004', '30000000-0000-0000-0000-000000000009',
   '通义千问大模型 API 接口合作证明', '通义千问大模型 API 接口合作证明。', 'pdf',
   '00000000-0000-0000-0000-000000000010', 'finalized', '50000000-0000-0000-0000-000000000110', false, false,
   '00000000-0000-0000-0000-000000000010', NOW() - INTERVAL '32 days', NOW() - INTERVAL '12 days'),
  ('40000000-0000-0000-0000-000000000111', '10000000-0000-0000-0000-000000000002', '20000000-0000-0000-0000-000000000004', '30000000-0000-0000-0000-000000000010',
   '寒假特训营海报', '寒假特训营宣传素材。', 'jpeg',
   '00000000-0000-0000-0000-000000000013', 'archived', '50000000-0000-0000-0000-000000000111', true, false,
   '00000000-0000-0000-0000-000000000013', NOW() - INTERVAL '16 days', NOW() - INTERVAL '10 days')
ON CONFLICT (id) DO UPDATE
SET folder_id = EXCLUDED.folder_id,
    title = EXCLUDED.title,
    description = EXCLUDED.description,
    file_type = EXCLUDED.file_type,
    current_owner_id = EXCLUDED.current_owner_id,
    current_status = EXCLUDED.current_status,
    current_version_id = EXCLUDED.current_version_id,
    is_archived = EXCLUDED.is_archived,
    is_deleted = false,
    updated_at = EXCLUDED.updated_at;

-- ========== 文档版本 ==========
INSERT INTO document_versions (
  id, document_id, version_no, file_name, mime_type, file_size,
  storage_provider, storage_bucket_or_share, storage_object_key, external_path,
  commit_message, extracted_text_status, summary_status, created_by, created_at
)
VALUES
  ('50000000-0000-0000-0000-000000000101', '40000000-0000-0000-0000-000000000101', 1, '“五好教育”发展及规划.docx', 'application/vnd.openxmlformats-officedocument.wordprocessingml.document', 95631, 'fixture', 'local-dev-assets', 'wuhao-ai/产品规划/“五好教育”发展及规划.docx', '~/workspace/asset-base/internal/五好爱学/“五好教育”发展及规划.docx', '导入 wuhao-ai 测试资源', 'pending', 'pending', '00000000-0000-0000-0000-000000000010', NOW() - INTERVAL '12 days'),
  ('50000000-0000-0000-0000-000000000102', '40000000-0000-0000-0000-000000000102', 1, '学情雷达系统建设路径.pdf', 'application/pdf', 231208, 'fixture', 'local-dev-assets', 'wuhao-ai/产品规划/学情雷达系统建设路径.pdf', '~/workspace/asset-base/internal/五好爱学/学情雷达系统建设路径.pdf', '导入 wuhao-ai 测试资源', 'pending', 'pending', '00000000-0000-0000-0000-000000000010', NOW() - INTERVAL '10 days'),
  ('50000000-0000-0000-0000-000000000103', '40000000-0000-0000-0000-000000000103', 1, 'Wuhao-tutor_User_Manual.pdf', 'application/pdf', 737378, 'fixture', 'local-dev-assets', 'wuhao-ai/用户手册/Wuhao-tutor_User_Manual.pdf', '~/workspace/asset-base/internal/五好爱学/Wuhao-tutor_User_Manual.pdf', '导入 wuhao-ai 测试资源', 'pending', 'pending', '00000000-0000-0000-0000-000000000011', NOW() - INTERVAL '30 days'),
  ('50000000-0000-0000-0000-000000000104', '40000000-0000-0000-0000-000000000104', 1, '寒假特训营日志.pdf', 'application/pdf', 409142, 'fixture', 'local-dev-assets', 'wuhao-ai/用户手册/寒假特训营日志.pdf', '~/workspace/asset-base/internal/五好爱学/寒假特训营日志.pdf', '导入 wuhao-ai 测试资源', 'pending', 'pending', '00000000-0000-0000-0000-000000000012', NOW() - INTERVAL '20 days'),
  ('50000000-0000-0000-0000-000000000105', '40000000-0000-0000-0000-000000000105', 1, '五好教育.pdf', 'application/pdf', 453202, 'fixture', 'local-dev-assets', 'wuhao-ai/生涯规划/五好教育.pdf', '~/workspace/asset-base/internal/五好爱学/生涯规划/五好教育.pdf', '导入 wuhao-ai 测试资源', 'pending', 'pending', '00000000-0000-0000-0000-000000000011', NOW() - INTERVAL '18 days'),
  ('50000000-0000-0000-0000-000000000106', '40000000-0000-0000-0000-000000000106', 1, 'MBTI职业性格测评解析.docx', 'application/vnd.openxmlformats-officedocument.wordprocessingml.document', 45954, 'fixture', 'local-dev-assets', 'wuhao-ai/生涯规划/MBTI职业性格测评解析.docx', '~/workspace/asset-base/internal/五好爱学/生涯规划/MBTI职业性格测评解析.docx', '导入 wuhao-ai 测试资源', 'pending', 'pending', '00000000-0000-0000-0000-000000000011', NOW() - INTERVAL '7 days'),
  ('50000000-0000-0000-0000-000000000107', '40000000-0000-0000-0000-000000000107', 1, '314 职业生涯规划分享1110.pptx', 'application/vnd.openxmlformats-officedocument.presentationml.presentation', 10146110, 'fixture', 'local-dev-assets', 'wuhao-ai/生涯规划/314 职业生涯规划分享1110.pptx', '~/workspace/asset-base/internal/五好爱学/生涯规划/314 职业生涯规划分享1110.pptx', '导入 wuhao-ai 测试资源', 'pending', 'pending', '00000000-0000-0000-0000-000000000012', NOW() - INTERVAL '9 days'),
  ('50000000-0000-0000-0000-000000000108', '40000000-0000-0000-0000-000000000108', 1, '五好生涯报价体系.xlsx', 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet', 14736, 'fixture', 'local-dev-assets', 'wuhao-ai/生涯规划/五好生涯报价体系.xlsx', '~/workspace/asset-base/internal/五好爱学/生涯规划/五好生涯报价体系.xlsx', '导入 wuhao-ai 测试资源', 'pending', 'pending', '00000000-0000-0000-0000-000000000012', NOW() - INTERVAL '6 days'),
  ('50000000-0000-0000-0000-000000000109', '40000000-0000-0000-0000-000000000109', 1, '公示内容_网信算备330110507206401240101号.pdf', 'application/pdf', 159065, 'fixture', 'local-dev-assets', 'wuhao-ai/资质证明/公示内容_网信算备330110507206401240101号.pdf', '~/workspace/asset-base/internal/五好爱学/老马识学/公示内容_网信算备330110507206401240101号.pdf', '导入 wuhao-ai 测试资源', 'pending', 'pending', '00000000-0000-0000-0000-000000000010', NOW() - INTERVAL '40 days'),
  ('50000000-0000-0000-0000-000000000110', '40000000-0000-0000-0000-000000000110', 1, '通义千问大模型API接口合作证明.pdf', 'application/pdf', 319684, 'fixture', 'local-dev-assets', 'wuhao-ai/资质证明/通义千问大模型API接口合作证明.pdf', '~/workspace/asset-base/internal/五好爱学/老马识学/通义千问大模型API接口合作证明.pdf', '导入 wuhao-ai 测试资源', 'pending', 'pending', '00000000-0000-0000-0000-000000000010', NOW() - INTERVAL '32 days'),
  ('50000000-0000-0000-0000-000000000111', '40000000-0000-0000-0000-000000000111', 1, '五好爱学（寒假特训）1.jpeg', 'image/jpeg', 981763, 'fixture', 'local-dev-assets', 'wuhao-ai/素材图片/五好爱学（寒假特训）1.jpeg', '~/workspace/asset-base/internal/五好爱学/五好爱学（寒假特训）1.jpeg', '导入 wuhao-ai 测试资源', 'pending', 'pending', '00000000-0000-0000-0000-000000000013', NOW() - INTERVAL '16 days')
ON CONFLICT (id) DO UPDATE
SET file_name = EXCLUDED.file_name,
    mime_type = EXCLUDED.mime_type,
    file_size = EXCLUDED.file_size,
    storage_provider = EXCLUDED.storage_provider,
    storage_bucket_or_share = EXCLUDED.storage_bucket_or_share,
    storage_object_key = EXCLUDED.storage_object_key,
    external_path = EXCLUDED.external_path,
    commit_message = EXCLUDED.commit_message;

-- ========== 流转记录 ==========
INSERT INTO flow_records (id, document_id, version_id, from_user_id, to_user_id, from_status, to_status, action, note, created_by, created_at)
VALUES
  ('60000000-0000-0000-0000-000000000101', '40000000-0000-0000-0000-000000000104', '50000000-0000-0000-0000-000000000104', '00000000-0000-0000-0000-000000000012', '00000000-0000-0000-0000-000000000013', 'in_progress', 'pending_handover', 'transfer', '寒假特训营日志转交运营同学补充材料。', '00000000-0000-0000-0000-000000000012', NOW() - INTERVAL '3 days'),
  ('60000000-0000-0000-0000-000000000102', '40000000-0000-0000-0000-000000000103', '50000000-0000-0000-0000-000000000103', '00000000-0000-0000-0000-000000000011', '00000000-0000-0000-0000-000000000011', 'in_progress', 'finalized', 'finalize', '用户手册已确认定稿。', '00000000-0000-0000-0000-000000000011', NOW() - INTERVAL '8 days'),
  ('60000000-0000-0000-0000-000000000103', '40000000-0000-0000-0000-000000000111', '50000000-0000-0000-0000-000000000111', '00000000-0000-0000-0000-000000000013', '00000000-0000-0000-0000-000000000013', 'finalized', 'archived', 'archive', '寒假特训营海报归档。', '00000000-0000-0000-0000-000000000013', NOW() - INTERVAL '10 days')
ON CONFLICT (id) DO NOTHING;

-- ========== 审计事件 ==========
INSERT INTO audit_events (id, document_id, version_id, user_id, action_type, terminal_info, extra_data, created_at)
VALUES
  ('70000000-0000-0000-0000-000000000101', '40000000-0000-0000-0000-000000000101', '50000000-0000-0000-0000-000000000101', '00000000-0000-0000-0000-000000000010', 'upload', 'fixture', '{"project_code":"wuhao-ai"}', NOW() - INTERVAL '12 days'),
  ('70000000-0000-0000-0000-000000000102', '40000000-0000-0000-0000-000000000102', '50000000-0000-0000-0000-000000000102', '00000000-0000-0000-0000-000000000010', 'upload', 'fixture', '{"project_code":"wuhao-ai"}', NOW() - INTERVAL '10 days'),
  ('70000000-0000-0000-0000-000000000103', '40000000-0000-0000-0000-000000000103', '50000000-0000-0000-0000-000000000103', '00000000-0000-0000-0000-000000000011', 'upload', 'fixture', '{"project_code":"wuhao-ai"}', NOW() - INTERVAL '30 days'),
  ('70000000-0000-0000-0000-000000000104', '40000000-0000-0000-0000-000000000104', '50000000-0000-0000-0000-000000000104', '00000000-0000-0000-0000-000000000012', 'upload', 'fixture', '{"project_code":"wuhao-ai"}', NOW() - INTERVAL '20 days'),
  ('70000000-0000-0000-0000-000000000105', '40000000-0000-0000-0000-000000000105', '50000000-0000-0000-0000-000000000105', '00000000-0000-0000-0000-000000000011', 'upload', 'fixture', '{"project_code":"wuhao-ai"}', NOW() - INTERVAL '18 days'),
  ('70000000-0000-0000-0000-000000000106', '40000000-0000-0000-0000-000000000106', '50000000-0000-0000-0000-000000000106', '00000000-0000-0000-0000-000000000011', 'upload', 'fixture', '{"project_code":"wuhao-ai"}', NOW() - INTERVAL '7 days'),
  ('70000000-0000-0000-0000-000000000107', '40000000-0000-0000-0000-000000000107', '50000000-0000-0000-0000-000000000107', '00000000-0000-0000-0000-000000000012', 'upload', 'fixture', '{"project_code":"wuhao-ai"}', NOW() - INTERVAL '9 days'),
  ('70000000-0000-0000-0000-000000000108', '40000000-0000-0000-0000-000000000108', '50000000-0000-0000-0000-000000000108', '00000000-0000-0000-0000-000000000012', 'upload', 'fixture', '{"project_code":"wuhao-ai"}', NOW() - INTERVAL '6 days'),
  ('70000000-0000-0000-0000-000000000109', '40000000-0000-0000-0000-000000000109', '50000000-0000-0000-0000-000000000109', '00000000-0000-0000-0000-000000000010', 'upload', 'fixture', '{"project_code":"wuhao-ai"}', NOW() - INTERVAL '40 days'),
  ('70000000-0000-0000-0000-000000000110', '40000000-0000-0000-0000-000000000110', '50000000-0000-0000-0000-000000000110', '00000000-0000-0000-0000-000000000010', 'upload', 'fixture', '{"project_code":"wuhao-ai"}', NOW() - INTERVAL '32 days'),
  ('70000000-0000-0000-0000-000000000111', '40000000-0000-0000-0000-000000000111', '50000000-0000-0000-0000-000000000111', '00000000-0000-0000-0000-000000000013', 'upload', 'fixture', '{"project_code":"wuhao-ai"}', NOW() - INTERVAL '16 days'),
  ('70000000-0000-0000-0000-000000000112', '40000000-0000-0000-0000-000000000104', NULL, '00000000-0000-0000-0000-000000000012', 'transfer', 'fixture', '{"project_code":"wuhao-ai"}', NOW() - INTERVAL '3 days'),
  ('70000000-0000-0000-0000-000000000113', '40000000-0000-0000-0000-000000000103', NULL, '00000000-0000-0000-0000-000000000011', 'finalize', 'fixture', '{"project_code":"wuhao-ai"}', NOW() - INTERVAL '8 days'),
  ('70000000-0000-0000-0000-000000000114', '40000000-0000-0000-0000-000000000111', NULL, '00000000-0000-0000-0000-000000000013', 'archive', 'fixture', '{"project_code":"wuhao-ai"}', NOW() - INTERVAL '10 days')
ON CONFLICT (id) DO NOTHING;

COMMIT;
