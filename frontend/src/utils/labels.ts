// Single source of truth for status wording. Views used to keep their own
// copies, which drifted (the same enum read "待交接" in one place and meant a
// different thing in another). Change a label here and every page follows.

/**
 * Document status. `pending_handover` is shown as "待接收": it is the state a
 * document is in after a *transfer*, waiting for the new owner to accept. It is
 * unrelated to graduation handovers, which have their own statuses below.
 */
export const documentStatusLabel: Record<string, string> = {
  draft: "草稿",
  in_progress: "处理中",
  pending_handover: "待接收",
  handed_over: "已交接",
  finalized: "定稿",
  archived: "已归档",
};

export const documentStatusClass: Record<string, string> = {
  draft: "status-draft",
  in_progress: "status-in-progress",
  pending_handover: "status-pending-handover",
  handed_over: "status-handed-over",
  finalized: "status-finalized",
  archived: "status-archived",
};

/** Graduation handover status, named after who has to act next. */
export const handoverStatusLabel: Record<string, string> = {
  generated: "待接收人确认",
  pending_confirm: "待负责人完成",
  completed: "已完成",
  cancelled: "已取消",
};

export const handoverStatusClass: Record<string, string> = {
  generated: "handover-generated",
  pending_confirm: "handover-pending",
  completed: "handover-completed",
  cancelled: "handover-cancelled",
};

export const flowActionLabel: Record<string, string> = {
  mark_in_progress: "开始处理",
  transfer: "转交",
  accept_transfer: "接收转交",
  finalize: "定稿",
  archive: "归档",
  unarchive: "取消归档",
  create: "创建",
};
