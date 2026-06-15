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
