<script setup lang="ts">
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

const statusLabel: Record<string, string> = {
  draft: "草稿",
  in_progress: "处理中",
  pending_handover: "待交接",
  handed_over: "已交接",
  finalized: "定稿",
  archived: "已归档",
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
  for (const doc of rows.value) {
    const key = doc.project_name || "未分类";
    if (!groups.has(key)) groups.set(key, []);
    groups.get(key)!.push(doc);
  }
  return Array.from(groups.entries())
    .sort((a, b) => a[0].localeCompare(b[0], "zh-Hans"))
    .map(([name, docs]) => ({ projectName: name, documents: docs }));
});

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
  await Promise.all([fetchDocuments(), fetchReferenceData()]);
});
</script>

<template>
  <AppLayout>
    <div class="page-shell">
      <div class="page-header">
        <div>
          <h1>文档管理</h1>
          <p>按团队空间、课题和目录组织文档，并管理责任人和版本。</p>
        </div>
        <ElButton type="primary" @click="openCreate">新建文档</ElButton>
      </div>
      <ElCard class="page-card">
        <div class="toolbar">
          <ElInput
            v-model="keyword"
            placeholder="搜索文档标题"
            clearable
            @keyup.enter="handleSearch"
            @clear="handleSearch"
          />
          <span class="doc-count">共 {{ total }} 篇文档，{{ groupedDocuments.length }} 个课题</span>
        </div>

        <div v-if="groupedDocuments.length === 0" class="empty-state">
          <p class="empty-title">暂无文档</p>
          <p class="empty-hint">点击右上角「新建文档」创建第一篇文档</p>
        </div>

        <div v-else class="project-groups">
          <div
            v-for="group in groupedDocuments"
            :key="group.projectName"
            class="project-group"
          >
            <div
              class="group-header"
              @click="toggleGroup(group.projectName)"
            >
              <span class="group-toggle">{{ collapsedGroups.has(group.projectName) ? '▶' : '▼' }}</span>
              <span class="group-name">{{ group.projectName }}</span>
              <ElTag size="small" type="info" disable-transitions>{{ group.documents.length }} 篇</ElTag>
            </div>
            <ElTable
              v-show="!collapsedGroups.has(group.projectName)"
              :data="group.documents"
              style="width: 100%"
              @row-click="goDetail"
            >
              <ElTableColumn prop="title" label="文档标题" min-width="200" />
              <ElTableColumn label="当前责任人" width="140">
                <template #default="{ row }">{{
                  row.current_owner?.display_name ?? "-"
                }}</template>
              </ElTableColumn>
              <ElTableColumn label="当前版本" width="100">
                <template #default="{ row }">{{
                  row.current_version_no ?? "-"
                }}</template>
              </ElTableColumn>
              <ElTableColumn label="状态" width="120">
                <template #default="{ row }">
                  <ElTag>{{
                    statusLabel[row.current_status] ?? row.current_status
                  }}</ElTag>
                </template>
              </ElTableColumn>
              <ElTableColumn prop="updated_at" label="更新时间" width="180" />
            </ElTable>
          </div>
        </div>
      </ElCard>

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
h1 {
  margin: 0;
  font-size: 32px;
}

p {
  color: #61748d;
}

.toolbar {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 16px;
}

.doc-count {
  white-space: nowrap;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.project-groups {
  display: grid;
  gap: 20px;
}

.project-group {
  border: 1px solid var(--el-border-color-light);
  border-radius: 12px;
  overflow: hidden;
}

.group-header {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 16px;
  background: var(--el-fill-color-light);
  cursor: pointer;
  user-select: none;
  font-weight: 600;
}

.group-toggle {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  width: 16px;
}

.group-name {
  flex: 1;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px 16px;
  text-align: center;
}

.empty-title {
  margin: 12px 0 4px;
  font-size: 14px;
  font-weight: 500;
  color: var(--el-text-color-regular);
}

.empty-hint {
  margin: 0;
  font-size: 13px;
  color: var(--el-text-color-placeholder);
}
</style>
