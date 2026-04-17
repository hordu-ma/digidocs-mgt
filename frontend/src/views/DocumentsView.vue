<script setup lang="ts">
import {
  DocumentAdd,
  FolderOpened,
  Search,
} from "@element-plus/icons-vue";
import { ElMessage } from "element-plus";
import type { UploadRawFile } from "element-plus";
import { computed, onMounted, reactive, ref } from "vue";
import { useRouter } from "vue-router";

import AppLayout from "@/components/AppLayout.vue";
import api from "@/api";
import { useAuthStore } from "@/stores/auth";

type TeamSpaceOption = {
  id: string;
  name: string;
  code: string;
};

type UserOption = {
  id: string;
  display_name: string;
  role: string;
};

type ProjectOption = {
  id: string;
  name: string;
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
const teamSpaces = ref<TeamSpaceOption[]>([]);
const users = ref<UserOption[]>([]);
const projects = ref<ProjectOption[]>([]);
const folderOptions = ref<FolderOption[]>([]);
const referenceLoading = ref(false);
const collapsedGroups = ref<Set<string>>(new Set());
const allProjects = ref<{ id: string; name: string }[]>([]);
const selectedProjectName = ref("");

const statusLabel: Record<string, string> = {
  draft: "草稿",
  in_progress: "处理中",
  pending_handover: "待交接",
  handed_over: "已交接",
  finalized: "定稿",
  archived: "已归档",
};

const statusClass: Record<string, string> = {
  draft: "status-draft",
  in_progress: "status-in-progress",
  pending_handover: "status-pending-handover",
  handed_over: "status-handed-over",
  finalized: "status-finalized",
  archived: "status-archived",
};

async function fetchDocuments() {
  const res = await api.get("/documents", {
    params: {
      page: 1,
      page_size: 500,
      keyword: keyword.value,
    },
  });
  rows.value = res.data?.data ?? [];
  total.value = res.data?.meta?.total ?? 0;
}

type DocumentGroup = {
  projectName: string;
  documents: any[];
};

const groupedDocuments = computed<DocumentGroup[]>(() => {
  const groups = new Map<string, any[]>();
  // Seed with all projects the user has access to
  for (const p of allProjects.value) {
    if (!groups.has(p.name)) groups.set(p.name, []);
  }
  for (const doc of rows.value) {
    const key = doc.project_name || "未分类";
    if (!groups.has(key)) groups.set(key, []);
    groups.get(key)!.push(doc);
  }
  return Array.from(groups.entries())
    .sort((a, b) => a[0].localeCompare(b[0], "zh-Hans"))
    .map(([name, docs]) => ({ projectName: name, documents: docs }));
});

const displayedGroupedDocuments = computed(() => {
  if (!selectedProjectName.value) {
    return groupedDocuments.value;
  }
  return groupedDocuments.value.filter(
    (group) => group.projectName === selectedProjectName.value,
  );
});

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
  fetchDocuments();
}

function goDetail(row: any) {
  router.push(`/documents/${row.id}`);
}

function ownerInitial(row: any) {
  return (row.current_owner?.display_name || "责").slice(0, 1);
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
  } catch (err: any) {
    ElMessage.error(err.response?.data?.message ?? "加载基础选项失败");
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

function handleFileChange(file: { raw: UploadRawFile }) {
  createFile.value = file.raw;
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
  } catch (err: any) {
    ElMessage.error(err.response?.data?.message ?? "创建失败");
  } finally {
    createLoading.value = false;
  }
}

async function handleTeamSpaceChange(teamSpaceID: string) {
  try {
    await loadProjects(teamSpaceID);
  } catch (err: any) {
    ElMessage.error(err.response?.data?.message ?? "加载课题列表失败");
  }
}

async function handleProjectChange(projectID: string) {
  try {
    await loadFolders(projectID);
  } catch (err: any) {
    ElMessage.error(err.response?.data?.message ?? "加载目录树失败");
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
            :class="{ active: selectedProjectName === '' }"
            type="button"
            @click="selectedProjectName = ''"
          >
            <span>全部文档</span>
            <strong>{{ total }}</strong>
          </button>
          <button
            v-for="group in groupedDocuments"
            :key="group.projectName"
            class="project-filter"
            :class="{ active: selectedProjectName === group.projectName }"
            type="button"
            @click="selectedProjectName = group.projectName"
          >
            <span>{{ group.projectName }}</span>
            <strong>{{ group.documents.length }}</strong>
          </button>
        </aside>

        <section class="page-card asset-panel">
          <div class="toolbar">
            <ElInput
              v-model="keyword"
              placeholder="搜索文档标题"
              clearable
              @keyup.enter="handleSearch"
              @clear="handleSearch"
            >
              <template #prefix>
                <ElIcon><Search /></ElIcon>
              </template>
            </ElInput>
            <span class="doc-count">
              当前显示 {{ visibleDocumentTotal }} 篇文档，共 {{ groupedDocuments.length }} 个课题
            </span>
          </div>

          <div v-if="displayedGroupedDocuments.length === 0" class="empty-state">
            <p class="empty-title">暂无文档资产</p>
            <p class="empty-hint">新建文档后会按课题自动归入资产库</p>
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
                <button
                  v-for="row in group.documents"
                  :key="row.id"
                  class="document-row"
                  type="button"
                  @click="goDetail(row)"
                >
                  <span class="file-badge" :class="inferFileType(row)">
                    {{ inferFileType(row).toUpperCase() }}
                  </span>
                  <span class="document-main">
                    <strong>{{ row.title }}</strong>
                    <span>{{ row.project_name || group.projectName }} · {{ row.updated_at || "暂无更新时间" }}</span>
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
              </div>
            </div>
          </div>
        </section>
      </div>

      <ElDialog v-model="showCreateDialog" title="新建文档" width="520px">
        <ElForm label-position="top" v-loading="referenceLoading">
          <ElFormItem label="标题" required>
            <ElInput v-model="createForm.title" placeholder="文档标题" />
          </ElFormItem>
          <ElFormItem label="团队空间" required>
            <ElSelect
              v-model="createForm.team_space_id"
              filterable
              placeholder="选择团队空间"
              @change="handleTeamSpaceChange"
            >
              <ElOption
                v-for="item in teamSpaces"
                :key="item.id"
                :label="`${item.name} (${item.code})`"
                :value="item.id"
              />
            </ElSelect>
          </ElFormItem>
          <ElFormItem label="课题" required>
            <ElSelect
              v-model="createForm.project_id"
              filterable
              placeholder="选择课题"
              :disabled="projects.length === 0"
              @change="handleProjectChange"
            >
              <ElOption
                v-for="item in projects"
                :key="item.id"
                :label="item.name"
                :value="item.id"
              />
            </ElSelect>
          </ElFormItem>
          <ElFormItem label="目录">
            <ElSelect
              v-model="createForm.folder_id"
              clearable
              filterable
              placeholder="选择目录（可选）"
              :disabled="folderOptions.length === 0"
            >
              <ElOption
                v-for="item in folderOptions"
                :key="item.id"
                :label="item.path"
                :value="item.id"
              />
            </ElSelect>
          </ElFormItem>
          <ElFormItem label="当前责任人" required>
            <ElSelect
              v-model="createForm.current_owner_id"
              filterable
              placeholder="选择责任人"
            >
              <ElOption
                v-for="item in users"
                :key="item.id"
                :label="`${item.display_name} (${item.role})`"
                :value="item.id"
              />
            </ElSelect>
          </ElFormItem>
          <ElFormItem label="描述">
            <ElInput
              v-model="createForm.description"
              type="textarea"
              :rows="2"
            />
          </ElFormItem>
          <ElFormItem label="提交说明">
            <ElInput
              v-model="createForm.commit_message"
              placeholder="首版本提交说明"
            />
          </ElFormItem>
          <ElFormItem label="文件" required>
            <ElUpload
              :auto-upload="false"
              :limit="1"
              :on-change="handleFileChange"
            >
              <ElButton>选择文件</ElButton>
            </ElUpload>
          </ElFormItem>
        </ElForm>
        <template #footer>
          <ElButton @click="showCreateDialog = false">取消</ElButton>
          <ElButton
            type="primary"
            :loading="createLoading"
            @click="submitCreate"
            >创建</ElButton
          >
        </template>
      </ElDialog>
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

.toolbar {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 16px;
}

.toolbar .el-input {
  max-width: 420px;
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

.document-row {
  display: grid;
  grid-template-columns: 64px minmax(220px, 1fr) minmax(120px, auto) 72px minmax(86px, auto);
  gap: 14px;
  align-items: center;
  width: 100%;
  min-height: 74px;
  padding: 12px 16px;
  border: 0;
  border-top: 1px solid var(--dd-line-soft);
  background: #fff;
  color: var(--dd-ink-2);
  text-align: left;
  cursor: pointer;
  transition:
    background 0.16s ease,
    transform 0.16s ease;
}

.document-row:hover {
  background: #f8fbff;
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

  .document-row {
    grid-template-columns: 56px minmax(0, 1fr);
  }

  .person-chip,
  .version-chip,
  .document-row > .status-pill {
    justify-self: start;
    grid-column: 2;
  }
}

@media (max-width: 720px) {
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
