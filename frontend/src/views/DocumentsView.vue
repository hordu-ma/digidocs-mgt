<script setup lang="ts">
import DocumentCreateDialog from "@/components/DocumentCreateDialog.vue";
import {
  CopyDocument,
  DocumentAdd,
  FolderOpened,
  MoreFilled,
  RefreshLeft,
  Search,
} from "@element-plus/icons-vue";
import { ElMessage, ElMessageBox } from "element-plus";
import type { UploadRawFile } from "element-plus";
import { computed, onMounted, reactive, ref, watch } from "vue";
import { useRouter } from "vue-router";

import AppLayout from "@/components/AppLayout.vue";
import api from "@/api";
import { documentStatusClass as statusClass, documentStatusLabel as statusLabel } from "@/utils/labels";
import { extractError } from "@/utils/error";
import { useAuthStore } from "@/stores/auth";
import type { ProjectOption, UserOption } from "@/types";

type TeamSpaceOption = {
  id: string;
  name: string;
  code: string;
};

type FolderNode = {
  id: string;
  path: string;
  children?: FolderNode[];
};

type FolderOption = {
  id: string;
  path: string;
};

const router = useRouter();
const auth = useAuthStore();
const rows = ref<any[]>([]);
const total = ref(0);
const keyword = ref("");
const showArchived = ref(false);
const quickFilter = ref("all");
const sortBy = ref("updated_desc");
const teamSpaces = ref<TeamSpaceOption[]>([]);
const users = ref<UserOption[]>([]);
const projects = ref<ProjectOption[]>([]);
const folderOptions = ref<FolderOption[]>([]);
const referenceLoading = ref(false);
const collapsedGroups = ref<Set<string>>(new Set());
const allProjects = ref<{ id: string; name: string }[]>([]);
// Server-side paging (2026-11-10): the list used to load the first 200 rows
// and filter/sort them in the browser, so filters silently missed anything
// beyond row 200 and the rail counts only covered the loaded page.
const selectedProjectId = ref("");
const page = ref(1);
const PAGE_SIZE = 30;
const projectCounts = ref<{ project_id: string; project_name: string; count: number }[]>([]);
const listLoading = ref(false);

interface ContentHit {
  document_id: string;
  title: string;
  project_name?: string;
  version_no: number;
  snippet: string;
  matched_chunks?: number;
}
const contentHits = ref<ContentHit[]>([]);
const searchedKeyword = ref("");



function quickFilterParams(): Record<string, string | number | undefined> {
  switch (quickFilter.value) {
    case "mine":
      return { mine: "true" };
    case "pending_handover":
    case "in_progress":
      return { status: quickFilter.value };
    case "recent":
      return { updated_within_days: 7 };
    case "deleted":
      return { deleted_only: "true" };
    default:
      return {};
  }
}

let fetchSeq = 0;
async function fetchDocuments() {
  const seq = ++fetchSeq;
  listLoading.value = true;
  try {
    const res = await api.get("/documents", {
      params: {
        page: page.value,
        page_size: PAGE_SIZE,
        keyword: keyword.value || undefined,
        include_archived: showArchived.value || undefined,
        sort: sortBy.value,
        project_id: selectedProjectId.value || undefined,
        with_counts: "true",
        ...quickFilterParams(),
      },
    });
    if (seq !== fetchSeq) return; // a newer filter change superseded this request
    rows.value = res.data?.data ?? [];
    total.value = res.data?.meta?.total ?? 0;
    projectCounts.value = res.data?.meta?.project_counts ?? [];
  } catch (err) {
    if (seq === fetchSeq) ElMessage.error(extractError(err, "文档列表加载失败"));
  } finally {
    if (seq === fetchSeq) listLoading.value = false;
  }
}

// Full-text hits: documents whose current-version text mentions the keyword.
async function fetchContentHits() {
  const q = keyword.value.trim();
  searchedKeyword.value = q;
  if (!q) {
    contentHits.value = [];
    return;
  }
  try {
    const res = await api.get("/search", {
      params: { q, project_id: selectedProjectId.value || undefined, limit: 10 },
    });
    contentHits.value = res.data?.data ?? [];
  } catch {
    contentHits.value = [];
  }
}

function refetchFromFirstPage() {
  page.value = 1;
  void fetchDocuments();
}

watch([quickFilter, sortBy, selectedProjectId, showArchived], refetchFromFirstPage);

const railTotal = computed(() => projectCounts.value.reduce((sum, c) => sum + c.count, 0));

function parseDateValue(value?: string) {
  return value ? new Date(value).getTime() : 0;
}

function formatShortTime(value?: string) {
  if (!value) return "暂无更新时间";
  return new Date(value).toLocaleDateString("zh-CN", {
    month: "2-digit",
    day: "2-digit",
  });
}

function isRecent(row: any) {
  const updatedAt = parseDateValue(row.updated_at);
  return updatedAt > 0 && Date.now() - updatedAt <= 7 * 24 * 60 * 60 * 1000;
}

const quickFilters = [
  { key: "all", label: "全部文档" },
  { key: "mine", label: "我负责的" },
  { key: "pending_handover", label: "待接收" },
  { key: "in_progress", label: "处理中" },
  { key: "recent", label: "近 7 天更新" },
  { key: "deleted", label: "回收站" },
];

type DocumentGroup = {
  projectName: string;
  documents: any[];
};

const processedRows = computed(() => rows.value);

const groupedDocuments = computed<DocumentGroup[]>(() => {
  const groups = new Map<string, any[]>();
  for (const doc of processedRows.value) {
    const key = doc.project_name || "未分类";
    if (!groups.has(key)) groups.set(key, []);
    groups.get(key)!.push(doc);
  }
  return Array.from(groups.entries())
    .sort((a, b) => a[0].localeCompare(b[0], "zh-Hans"))
    .map(([name, docs]) => ({ projectName: name, documents: docs }));
});

const displayedGroupedDocuments = groupedDocuments;

const visibleDocumentTotal = computed(() =>
  displayedGroupedDocuments.value.reduce((sum, group) => sum + group.documents.length, 0),
);

function toggleGroup(name: string) {
  if (collapsedGroups.value.has(name)) {
    collapsedGroups.value.delete(name);
  } else {
    collapsedGroups.value.add(name);
  }
}

function handleSearch() {
  refetchFromFirstPage();
  void fetchContentHits();
}

function goDetail(row: any) {
  router.push(`/documents/${row.id}`);
}

function ownerInitial(row: any) {
  return (row.current_owner?.display_name || "责").slice(0, 1);
}

// Row menu: delete is a soft delete that only project owners/managers can undo
// from the 回收站 filter. The menu used to say "移出工作台" and decide on
// is_archived (never returned by the list), so it always soft-deleted.
async function deleteRow(row: any) {
  try {
    await ElMessageBox.confirm(
      `删除《${row.title}》？删除后课题负责人可在「回收站」中恢复。`,
      "删除文档",
      { confirmButtonText: "删除", cancelButtonText: "取消", type: "warning" },
    );
  } catch {
    return;
  }
  try {
    await api.post(`/documents/${row.id}/delete`, { reason: "文档列表删除" });
    ElMessage.success("文档已删除，可在回收站恢复");
    await fetchDocuments();
  } catch (err) {
    ElMessage.error(extractError(err, "删除失败"));
  }
}

async function restoreRow(row: any) {
  try {
    await api.post(`/documents/${row.id}/restore`);
    ElMessage.success("文档已恢复");
    await fetchDocuments();
  } catch (err) {
    ElMessage.error(extractError(err, "恢复失败"));
  }
}

async function copyTitle(row: any) {
  try {
    await navigator.clipboard.writeText(row.title || "");
    ElMessage.success("标题已复制");
  } catch {
    ElMessage.error("复制失败");
  }
}

function inferFileType(row: any) {
  const raw = `${row.file_type || row.title || ""}`.toLowerCase();
  const match = raw.match(/\.(docx|xlsx|pptx|pdf|txt|md)$/);
  if (match) return match[1];
  if (raw.includes("pdf")) return "pdf";
  if (raw.includes("xlsx") || raw.includes("表")) return "xlsx";
  if (raw.includes("pptx") || raw.includes("汇报")) return "pptx";
  if (raw.includes("docx") || raw.includes("文档")) return "docx";
  return "doc";
}

// --- Create document dialog ---
const showCreateDialog = ref(false);
const createLoading = ref(false);
const createForm = reactive({
  team_space_id: "",
  project_id: "",
  folder_id: "",
  current_owner_id: "",
  title: "",
  description: "",
  commit_message: "",
});
const createFile = ref<UploadRawFile | null>(null);

function flattenFolders(nodes: FolderNode[]): FolderOption[] {
  const items: FolderOption[] = [];
  for (const node of nodes) {
    items.push({
      id: node.id,
      path: node.path,
    });
    if (Array.isArray(node.children) && node.children.length > 0) {
      items.push(...flattenFolders(node.children));
    }
  }
  return items;
}

async function fetchReferenceData() {
  referenceLoading.value = true;
  try {
    const [teamRes, userRes] = await Promise.all([
      api.get("/team-spaces"),
      api.get("/users"),
    ]);
    teamSpaces.value = teamRes.data?.data ?? [];
    users.value = userRes.data?.data ?? [];
  } catch (err) {
    ElMessage.error(extractError(err, "加载基础选项失败"));
  } finally {
    referenceLoading.value = false;
  }
}

async function loadProjects(teamSpaceID: string) {
  projects.value = [];
  folderOptions.value = [];
  createForm.project_id = "";
  createForm.folder_id = "";
  if (!teamSpaceID) {
    return;
  }
  const res = await api.get("/projects", {
    params: { team_space_id: teamSpaceID },
  });
  projects.value = res.data?.data ?? [];
}

async function loadFolders(projectID: string) {
  folderOptions.value = [];
  createForm.folder_id = "";
  if (!projectID) {
    return;
  }
  const res = await api.get(`/projects/${projectID}/folders/tree`);
  folderOptions.value = flattenFolders(res.data?.data ?? []);
}

function openCreate() {
  Object.assign(createForm, {
    team_space_id: teamSpaces.value[0]?.id ?? "",
    project_id: "",
    folder_id: "",
    current_owner_id: auth.userId || users.value[0]?.id || "",
    title: "",
    description: "",
    commit_message: "",
  });
  void loadProjects(createForm.team_space_id);
  createFile.value = null;
  showCreateDialog.value = true;
}

function handleFileChange(file: { raw?: UploadRawFile }) {
  createFile.value = file.raw ?? null;
}

async function submitCreate() {
  if (
    !createForm.title ||
    !createForm.team_space_id ||
    !createForm.project_id ||
    !createForm.current_owner_id
  ) {
    ElMessage.warning("请填写标题、团队空间、课题和当前责任人");
    return;
  }
  if (!createFile.value) {
    ElMessage.warning("请选择文件");
    return;
  }
  createLoading.value = true;
  try {
    const fd = new FormData();
    fd.append("team_space_id", createForm.team_space_id);
    fd.append("project_id", createForm.project_id);
    fd.append("folder_id", createForm.folder_id);
    fd.append("current_owner_id", createForm.current_owner_id);
    fd.append("title", createForm.title);
    fd.append("description", createForm.description);
    fd.append("commit_message", createForm.commit_message);
    fd.append("file", createFile.value);
    await api.post("/documents", fd, {
      headers: { "Content-Type": "multipart/form-data" },
    });
    ElMessage.success("文档创建成功");
    showCreateDialog.value = false;
    await fetchDocuments();
  } catch (err) {
    ElMessage.error(extractError(err, "创建失败"));
  } finally {
    createLoading.value = false;
  }
}

async function handleTeamSpaceChange(teamSpaceID: string) {
  try {
    await loadProjects(teamSpaceID);
  } catch (err) {
    ElMessage.error(extractError(err, "加载课题列表失败"));
  }
}

async function handleProjectChange(projectID: string) {
  try {
    await loadFolders(projectID);
  } catch (err) {
    ElMessage.error(extractError(err, "加载目录树失败"));
  }
}

onMounted(async () => {
  const [, , projRes] = await Promise.all([
    fetchDocuments(),
    fetchReferenceData(),
    api.get("/projects"),
  ]);
  allProjects.value = (projRes.data?.data ?? []).map((p: any) => ({ id: p.id, name: p.name }));
});
</script>

<template>
  <AppLayout>
    <div class="page-shell">
      <div class="page-header">
        <div>
          <div class="page-eyebrow">文档资产库</div>
          <h1>文档管理</h1>
          <p>按课题沉淀文档资产，快速查看责任人、版本和当前流转状态。</p>
        </div>
        <ElButton type="primary" @click="openCreate">
          <ElIcon><DocumentAdd /></ElIcon>
          新建文档
        </ElButton>
      </div>

      <div class="asset-workspace">
        <aside class="project-rail page-card">
          <div class="rail-title">
            <ElIcon><FolderOpened /></ElIcon>
            文档
          </div>
          <button
            class="project-filter"
            :class="{ active: selectedProjectId === '' }"
            type="button"
            @click="selectedProjectId = ''"
          >
            <span>全部课题</span>
            <strong>{{ railTotal }}</strong>
          </button>
          <button
            v-for="c in projectCounts"
            :key="c.project_id"
            class="project-filter"
            :class="{ active: selectedProjectId === c.project_id }"
            type="button"
            @click="selectedProjectId = c.project_id"
          >
            <span>{{ c.project_name || "未分类" }}</span>
            <strong>{{ c.count }}</strong>
          </button>
        </aside>

        <section class="page-card asset-panel">
          <div class="control-bar asset-control-bar">
            <div class="segmented-filters">
              <button
                v-for="item in quickFilters"
                :key="item.key"
                class="segment-chip"
                :class="{ active: quickFilter === item.key }"
                type="button"
                @click="quickFilter = item.key"
              >
                {{ item.label }}
              </button>
            </div>
            <ElSelect v-model="sortBy" class="sort-select">
              <ElOption label="按最近更新" value="updated_desc" />
              <ElOption label="按最早更新" value="updated_asc" />
              <ElOption label="按标题排序" value="title_asc" />
              <ElOption label="按责任人排序" value="owner_asc" />
            </ElSelect>
          </div>
          <div class="toolbar">
            <ElInput
              v-model="keyword"
              placeholder="搜索标题或正文，回车检索"
              clearable
              @keyup.enter="handleSearch"
              @clear="handleSearch"
            >
              <template #prefix>
                <ElIcon><Search /></ElIcon>
              </template>
            </ElInput>
            <label class="archived-toggle">
              <ElSwitch v-model="showArchived" size="small" />
              <span>显示已归档</span>
            </label>
            <span class="doc-count">
              共 {{ total }} 篇，第 {{ page }} 页显示 {{ rows.length }} 篇
            </span>
          </div>

          <div v-if="searchedKeyword && contentHits.length" class="content-hits">
            <div class="content-hits-head">正文中提到「{{ searchedKeyword }}」的文档</div>
            <button
              v-for="hit in contentHits"
              :key="hit.document_id"
              type="button"
              class="content-hit"
              @click="router.push(`/documents/${hit.document_id}`)"
            >
              <strong>{{ hit.title }}</strong>
              <span class="content-hit-meta">
                {{ hit.project_name }} · v{{ hit.version_no }}<template v-if="hit.matched_chunks"> · {{ hit.matched_chunks }} 处相关</template>
              </span>
              <span class="content-hit-snippet">{{ hit.snippet }}</span>
            </button>
          </div>

          <div v-if="!listLoading && displayedGroupedDocuments.length === 0" class="empty-state">
            <p class="empty-title">{{ quickFilter === "deleted" ? "回收站为空" : "暂无符合条件的文档" }}</p>
            <p class="empty-hint">
              {{ quickFilter === "deleted" ? "只有课题负责人和协管人能看到并恢复已删除的文档" : "调整筛选条件，或新建文档后会按课题自动归入资产库" }}
            </p>
          </div>

          <div v-else class="project-groups">
            <div
              v-for="group in displayedGroupedDocuments"
              :key="group.projectName"
              class="project-group"
            >
              <button
                class="group-header"
                type="button"
                @click="toggleGroup(group.projectName)"
              >
                <span class="group-toggle">{{ collapsedGroups.has(group.projectName) ? "›" : "⌄" }}</span>
                <span class="group-name">{{ group.projectName }}</span>
                <span class="group-count">{{ group.documents.length }} 篇</span>
              </button>
              <div
                v-if="group.documents.length === 0 && !collapsedGroups.has(group.projectName)"
                class="empty-project-hint"
              >
                该课题尚未沉淀文档资产
              </div>
              <div
                v-show="!collapsedGroups.has(group.projectName) && group.documents.length > 0"
                class="document-list"
              >
                <div class="document-list-head">
                  <span>类型</span>
                  <span>文档信息</span>
                  <span>责任人</span>
                  <span>版本</span>
                  <span>状态</span>
                  <span>操作</span>
                </div>
                <div
                  v-for="row in group.documents"
                  :key="row.id"
                  class="document-row"
                  :class="{ 'is-archived': row.is_archived }"
                >
                  <button class="document-row-main" type="button" @click="goDetail(row)">
                    <span class="file-badge" :class="inferFileType(row)">
                      {{ inferFileType(row).toUpperCase() }}
                    </span>
                    <span class="document-main">
                      <strong>{{ row.title }}</strong>
                      <span>
                        {{ row.project_name || group.projectName }} · {{ formatShortTime(row.updated_at) }}
                        <template v-if="isRecent(row)"> · 最近 7 天更新</template>
                      </span>
                    </span>
                    <span class="person-chip">
                      <span class="person-avatar">{{ ownerInitial(row) }}</span>
                      {{ row.current_owner?.display_name ?? "-" }}
                    </span>
                    <span class="version-chip">v{{ row.current_version_no ?? "-" }}</span>
                    <span class="status-pill" :class="statusClass[row.current_status]">
                      {{ statusLabel[row.current_status] ?? row.current_status }}
                    </span>
                  </button>
                  <div class="document-row-actions">
                    <ElDropdown trigger="click">
                      <button class="row-action-trigger" type="button" @click.stop>
                        <ElIcon><MoreFilled /></ElIcon>
                      </button>
                      <template #dropdown>
                        <ElDropdownMenu>
                          <ElDropdownItem @click="goDetail(row)">查看档案</ElDropdownItem>
                          <ElDropdownItem @click="copyTitle(row)">
                            <ElIcon><CopyDocument /></ElIcon>
                            复制标题
                          </ElDropdownItem>
                          <ElDropdownItem v-if="row.is_deleted" @click="restoreRow(row)">
                            <ElIcon><RefreshLeft /></ElIcon>
                            恢复文档
                          </ElDropdownItem>
                          <ElDropdownItem v-else @click="deleteRow(row)">
                            <ElIcon><RefreshLeft /></ElIcon>
                            删除文档
                          </ElDropdownItem>
                        </ElDropdownMenu>
                      </template>
                    </ElDropdown>
                  </div>
                </div>
              </div>
            </div>
          </div>
          <ElPagination
            v-if="total > PAGE_SIZE"
            v-model:current-page="page"
            class="list-pagination"
            layout="prev, pager, next, total"
            :page-size="PAGE_SIZE"
            :total="total"
            @current-change="fetchDocuments"
          />
        </section>
      </div>

      <DocumentCreateDialog v-model="showCreateDialog" v-model:form="createForm" :users="users" :projects="projects" :team-spaces="teamSpaces" :folder-options="folderOptions" :reference-loading="referenceLoading" :create-loading="createLoading" @team-space-change="handleTeamSpaceChange" @project-change="handleProjectChange" @file-change="handleFileChange" @submit="submitCreate" />
    </div>
  </AppLayout>
</template>

<style scoped>
.asset-workspace {
  display: grid;
  grid-template-columns: 260px minmax(0, 1fr);
  gap: 18px;
  align-items: start;
}

.project-rail {
  position: sticky;
  top: 92px;
  display: grid;
  gap: 8px;
  padding: 16px;
}

.rail-title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 6px;
  color: var(--dd-ink);
  font-weight: 750;
}

.project-filter {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  min-height: 42px;
  padding: 0 12px;
  border: 1px solid transparent;
  border-radius: 8px;
  background: transparent;
  color: var(--dd-ink-2);
  text-align: left;
  cursor: pointer;
}

.project-filter span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.project-filter strong {
  color: var(--dd-muted);
  font-size: 12px;
}

.project-filter:hover,
.project-filter.active {
  border-color: #c8ddf2;
  background: var(--dd-primary-soft);
  color: var(--dd-primary-strong);
}

.asset-panel {
  padding: 20px;
}

.asset-control-bar {
  justify-content: space-between;
  margin-bottom: 14px;
}

.list-pagination {
  justify-content: center;
  margin-top: 16px;
}

.content-hits {
  display: grid;
  gap: 8px;
  margin-bottom: 16px;
  padding: 12px;
  border: 1px solid #bfd5ed;
  border-radius: 8px;
  background: #f6f9fd;
}

.content-hits-head {
  font-weight: 600;
}

.content-hit {
  display: grid;
  gap: 2px;
  padding: 8px 0;
  border: 0;
  border-top: 1px solid var(--dd-line);
  background: transparent;
  text-align: left;
  font: inherit;
  cursor: pointer;
}

.content-hit-meta,
.content-hit-snippet {
  color: var(--dd-subtle);
  font-size: 13px;
}

.sort-select {
  width: 180px;
}

.toolbar {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 16px;
}

.toolbar .el-input {
  max-width: 420px;
}

.archived-toggle {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
  white-space: nowrap;
  font-size: 13px;
  color: var(--dd-muted);
  cursor: pointer;
  user-select: none;
}

.doc-count {
  white-space: nowrap;
  font-size: 13px;
  color: var(--dd-muted);
}

.project-groups {
  display: grid;
  gap: 14px;
}

.project-group {
  border: 1px solid var(--dd-line);
  border-radius: 10px;
  overflow: hidden;
  background: #fff;
}

.group-header {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  min-height: 48px;
  padding: 0 16px;
  border: 0;
  background: var(--dd-surface-soft);
  color: var(--dd-ink);
  cursor: pointer;
  user-select: none;
  font-weight: 750;
}

.group-toggle {
  width: 18px;
  color: var(--dd-muted);
  font-size: 20px;
  line-height: 1;
}

.group-name {
  flex: 1;
  text-align: left;
}

.group-count {
  color: var(--dd-muted);
  font-size: 12px;
}

.document-list {
  display: grid;
}

.document-list-head {
  display: grid;
  grid-template-columns: 64px minmax(220px, 1fr) minmax(120px, auto) 72px minmax(86px, auto) 56px;
  gap: 14px;
  padding: 10px 16px;
  background: var(--dd-surface-soft);
  color: var(--dd-muted);
  font-size: 12px;
  font-weight: 700;
}

.document-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 56px;
  align-items: center;
  width: 100%;
  min-height: 78px;
  border-top: 1px solid var(--dd-line-soft);
  background: #fff;
  color: var(--dd-ink-2);
  text-align: left;
}

.document-row-main {
  display: grid;
  grid-template-columns: 64px minmax(220px, 1fr) minmax(120px, auto) 72px minmax(86px, auto);
  gap: 14px;
  align-items: center;
  min-height: 78px;
  padding: 12px 16px;
  border: 0;
  background: transparent;
  color: inherit;
  text-align: left;
  cursor: pointer;
}

.document-row:hover,
.document-row-main:hover {
  background: #f8fbff;
}

.document-row-actions {
  display: flex;
  justify-content: center;
}

.row-action-trigger {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 34px;
  border: 1px solid transparent;
  border-radius: 10px;
  background: transparent;
  color: var(--dd-muted);
  cursor: pointer;
}

.row-action-trigger:hover {
  border-color: var(--dd-line);
  background: var(--dd-surface-soft);
  color: var(--dd-ink);
}

.document-row.is-archived {
  opacity: 0.6;
}

.document-row.is-archived .document-main strong {
  text-decoration: line-through;
  color: var(--dd-muted);
}

.document-main {
  display: grid;
  gap: 5px;
  min-width: 0;
}

.document-main strong {
  overflow: hidden;
  color: var(--dd-ink);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.document-main span {
  overflow: hidden;
  color: var(--dd-muted);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.version-chip {
  display: inline-flex;
  justify-content: center;
  min-width: 44px;
  padding: 6px 8px;
  border-radius: 999px;
  background: #f1f5f9;
  color: var(--dd-ink-2);
  font-size: 12px;
  font-weight: 750;
}

.empty-project-hint {
  padding: 24px 16px;
  text-align: center;
  color: var(--dd-muted);
  font-size: 13px;
}

@media (max-width: 1080px) {
  .asset-workspace {
    grid-template-columns: 1fr;
  }

  .project-rail {
    position: static;
  }

  .document-list-head {
    display: none;
  }

  .document-row {
    grid-template-columns: 1fr;
  }

  .document-row-main {
    grid-template-columns: 56px minmax(0, 1fr);
  }

  .person-chip,
  .version-chip,
  .document-row-main > .status-pill {
    justify-self: start;
    grid-column: 2;
  }

  .document-row-actions {
    justify-content: flex-start;
    padding: 0 16px 14px 72px;
  }
}

@media (max-width: 720px) {
  .asset-control-bar {
    align-items: stretch;
    flex-direction: column;
  }

  .sort-select {
    width: 100%;
  }

  .toolbar {
    align-items: flex-start;
    flex-direction: column;
  }

  .toolbar .el-input {
    max-width: none;
  }

  .doc-count {
    white-space: normal;
  }
}

/* Keep dialog empty states isolated from the global product empty state. */
.asset-panel .empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px 16px;
  text-align: center;
}
</style>
