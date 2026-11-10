// Shared domain types reused across views.

export type UserOption = {
  id: string;
  display_name: string;
  role: string;
};

export type ProjectOption = {
  id: string;
  name: string;
  team_space_id?: string;
  my_role?: string;
};

export type TeamSpaceOption = {
  id: string;
  name: string;
};

export type PaginationMeta = {
  page: number;
  page_size: number;
  total: number;
};

// ── API response shapes (2026-11-10). Mirrors backend-go/internal/domain/query.

export type UserSummary = {
  id: string;
  display_name: string;
};

export type FlowActionState = {
  action: string;
  allowed: boolean;
  reason?: string;
};

export type DocumentListItem = {
  id: string;
  title: string;
  project_id?: string;
  project_name?: string;
  folder_path?: string;
  current_status: string;
  current_owner?: UserSummary;
  current_version_no?: number;
  updated_at?: string;
  is_archived: boolean;
  is_deleted: boolean;
  file_type?: string;
};

export type DocumentDetail = {
  id: string;
  project_id: string;
  project_name?: string;
  title: string;
  description?: string;
  current_status: string;
  current_owner?: UserSummary;
  current_version_id?: string;
  current_version_no?: number;
  folder_id?: string;
  folder_path?: string;
  is_archived: boolean;
  created_at?: string;
  updated_at?: string;
  file_type?: string;
  actions: FlowActionState[];
};

export type VersionItem = {
  id: string;
  version_no: number;
  file_name?: string;
  file_size?: number;
  commit_message?: string;
  summary_status?: string;
  summary_text?: string;
  created_by?: UserSummary;
  created_at?: string;
};

export type FlowItem = {
  id?: string;
  action: string;
  from_status?: string;
  to_status: string;
  note?: string;
  created_at?: string;
};

export type SuggestionItem = {
  id: string;
  related_type: string;
  related_id: string;
  suggestion_type: string;
  status: string;
  title?: string;
  content: string;
  confidence?: number;
  generated_at: string;
  suggested_action?: string;
};

export type RecentFlowItem = {
  id?: string;
  document_id: string;
  title: string;
  action: string;
  from_status?: string;
  to_status: string;
  created_at: string;
};

export type RiskItem = {
  document_id: string;
  title: string;
  risk_type: string;
  risk_message: string;
  related_type: string;
  related_id: string;
  link: string;
};

export type TextSearchHit = {
  document_id: string;
  title: string;
  project_name?: string;
  version_no: number;
  snippet: string;
  matched_chunks?: number;
};

export type AdminUser = {
  id: string;
  username?: string;
  display_name: string;
  role: string;
  status?: string;
  email?: string;
  phone?: string;
};

export type ProjectMember = {
  id: string;
  user_id: string;
  display_name: string;
  username: string;
  account?: string;
  project_role: string;
};
