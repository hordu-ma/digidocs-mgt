<script setup lang="ts">
import {
  Delete,
  Download,
  FolderAdd,
  FolderOpened,
  Search,
  Upload,
} from "@element-plus/icons-vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { computed, onMounted, reactive, ref } from "vue";

import AppLayout from "@/components/AppLayout.vue";
import api from "@/api";
import { useAuthStore } from "@/stores/auth";

// ─────────────────────────── types ───────────────────────────────

type ProjectOption = { id: string; name: string };
type TeamSpaceOption = { id: string; name: string };
type FolderItem = {
  id: string;
  project_id: string;
  parent_id: string;
  depth: number;
  name: string;
  created_at: string;
  children?: FolderItem[];
};

type DataAsset = {
  id: string;
  project_id: string;
  project_name: string;
  folder_id: string;
  folder_name: string;
  display_name: string;
  file_name: string;
  mime_type: string;
  file_size: number;
  created_by_name: string;
  created_at: string;
};

// ─────────────────────────── state ───────────────────────────────

const auth = useAuthStore();

const allProjects = ref<ProjectOption[]>([]);
const teamSpaces = ref<TeamSpaceOption[]>([]);
const projects = ref<ProjectOption[]>([]);

// list state
const assets = ref<DataAsset[]>([]);
const total = ref(0);
const keyword = ref("");
const filterProjectID = ref("");
const filterFolderID = ref("");

// folder sidebar
const folders = ref<FolderItem[]>([]);
const folderTree = computed<FolderItem[]>(() => buildTree(folders.value));
const selectedFolderID = ref("");

// ─────────────────────────── fetch list ──────────────────────────

async function fetchAssets() {
  const res = await api.get("/data-assets", {
    params: {
      keyword: keyword.value || undefined,
      project_id: filterProjectID.value || undefined,
      folder_id: selectedFolderID.value || undefined,
      page: 1,
      page_size: 200,
    },
  });
  assets.value = res.data?.data?.items ?? [];
  total.value = res.data?.data?.total ?? 0;
}

async function fetchFolders(projectID: string) {
  if (!projectID) {
    folders.value = [];
    return;
  }
  const res = await api.get(`/projects/${projectID}/data-folders`);
  folders.value = res.data?.data ?? [];
}

function buildTree(flat: FolderItem[]): FolderItem[] {
  const map = new Map<string, FolderItem>();
  flat.forEach((f) => map.set(f.id, { ...f, children: [] }));
  const roots: FolderItem[] = [];
  flat.forEach((f) => {
    const node = map.get(f.id)!;
    if (!f.parent_id) {
      roots.push(node);
    } else {
      map.get(f.parent_id)?.children?.push(node);
    }
  });
  return roots;
}

function onFolderSelect(id: string) {
  selectedFolderID.value = id === selectedFolderID.value ? "" : id;
  fetchAssets();
}

async function onProjectFilterChange(projectID: string) {
  selectedFolderID.value = "";
  filterFolderID.value = "";
  await fetchFolders(projectID);
  fetchAssets();
}

function handleSearch() {
  fetchAssets();
}

// ─────────────────────────── reference data ──────────────────────

async function fetchReference() {
  const [tsRes] = await Promise.all([api.get("/team-spaces")]);
  teamSpaces.value = tsRes.data?.data ?? [];
  if (teamSpaces.value.length) {
    await loadProjects(teamSpaces.value[0].id);
  }
}

async function loadProjects(teamSpaceID: string) {
  if (!teamSpaceID) return;
  const res = await api.get("/projects", { params: { team_space_id: teamSpaceID } });
  projects.value = res.data?.data ?? [];
}

async function loadAllProjects() {
  let all: ProjectOption[] = [];
  for (const ts of teamSpaces.value) {
    const res = await api.get("/projects", { params: { team_space_id: ts.id } });
    all = all.concat(res.data?.data ?? []);
  }
  allProjects.value = all;
}

// ─────────────────────────── upload ──────────────────────────────

const showUploadDialog = ref(false);
const uploadLoading = ref(false);
const uploadProgress = ref(0);
const uploadForm = reactive({
  team_space_id: "",
  project_id: "",
  folder_id: "",
  display_name: "",
  description: "",
});
const uploadFile = ref<File | null>(null);
const uploadFolderOptions = ref<FolderItem[]>([]);

// Name hint pattern: suggest alphanumeric + hyphen + Chinese
const FILE_NAME_HINT = "建议使用：字母、数字、中文、下划线、连字符，避免空格和特殊字符";

function onFileChange(e: Event) {
  const input = e.target as HTMLInputElement;
  if (input.files?.[0]) {
    uploadFile.value = input.files[0];
    if (!uploadForm.display_name) {
      uploadForm.display_name = input.files[0].name;
    }
  }
}

async function openUpload() {
  if (!teamSpaces.value.length) {
    await fetchReference();
  }
  Object.assign(uploadForm, {
    team_space_id: teamSpaces.value[0]?.id ?? "",
    project_id: "",
    folder_id: "",
    display_name: "",
    description: "",
  });
  uploadFile.value = null;
  uploadProgress.value = 0;
  if (uploadForm.team_space_id) {
    await loadProjects(uploadForm.team_space_id);
  }
  showUploadDialog.value = true;
}

async function onUploadProjectChange(projectID: string) {
  uploadFolderOptions.value = [];
  uploadForm.folder_id = "";
  if (!projectID) return;
  const res = await api.get(`/projects/${projectID}/data-folders`);
  uploadFolderOptions.value = res.data?.data ?? [];
}

async function submitUpload() {
  if (!uploadForm.project_id) {
    ElMessage.warning("请选择关联课题");
    return;
  }
  if (!uploadForm.display_name.trim()) {
    ElMessage.warning("请填写文件名称");
    return;
  }
  if (!uploadFile.value) {
    ElMessage.warning("请选择要上传的文件");
    return;
  }

  const token = localStorage.getItem("access_token");
  const formData = new FormData();
  formData.append("file", uploadFile.value);
  formData.append("team_space_id", uploadForm.team_space_id);
  formData.append("project_id", uploadForm.project_id);
  formData.append("folder_id", uploadForm.folder_id);
  formData.append("display_name", uploadForm.display_name.trim());
  formData.append("description", uploadForm.description);

  uploadLoading.value = true;
  uploadProgress.value = 0;

  await new Promise<void>((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    xhr.open("POST", "/api/v1/data-assets");
    if (token) xhr.setRequestHeader("Authorization", `Bearer ${token}`);

    xhr.upload.onprogress = (e) => {
      if (e.lengthComputable) {
        uploadProgress.value = Math.round((e.loaded / e.total) * 100);
      }
    };

    xhr.onload = () => {
      if (xhr.status >= 200 && xhr.status < 300) {
        resolve();
      } else {
        try {
          const body = JSON.parse(xhr.responseText);
          reject(new Error(body?.error?.message ?? `HTTP ${xhr.status}`));
        } catch {
          reject(new Error(`HTTP ${xhr.status}`));
        }
      }
    };

    xhr.onerror = () => reject(new Error("网络错误"));
    xhr.send(formData);
  })
    .then(() => {
      ElMessage.success("上传成功");
      showUploadDialog.value = false;
      fetchAssets();
    })
    .catch((err: Error) => {
      ElMessage.error(err.message || "上传失败");
    })
    .finally(() => {
      uploadLoading.value = false;
    });
}

// ─────────────────────────── download ────────────────────────────

function downloadAsset(row: DataAsset) {
  const token = localStorage.getItem("access_token");
  const a = document.createElement("a");
  a.href = `/api/v1/data-assets/${row.id}/download`;
  // Attach token via query only if needed — prefer Authorization header via fetch
  fetch(a.href, { headers: { Authorization: `Bearer ${token}` } })
    .then((res) => {
      if (!res.ok) throw new Error("下载失败");
      return res.blob();
    })
    .then((blob) => {
      const url = URL.createObjectURL(blob);
      a.href = url;
      a.download = row.file_name;
      a.click();
      URL.revokeObjectURL(url);
    })
    .catch((e) => ElMessage.error(e.message ?? "下载失败"));
}

// ─────────────────────────── delete ──────────────────────────────

async function deleteAsset(row: DataAsset) {
  await ElMessageBox.confirm(
    `确认删除文件 "${row.display_name}"？此操作不可撤销。`,
    "删除数据文件",
    { confirmButtonText: "删除", cancelButtonText: "取消", type: "warning" },
  );
  await api.delete(`/data-assets/${row.id}`);
  ElMessage.success("已删除");
  fetchAssets();
}

// ─────────────────────────── folder management ───────────────────

const showFolderDialog = ref(false);
const newFolderForm = reactive({ project_id: "", parent_id: "", name: "" });
const folderProjects = ref<ProjectOption[]>([]);
const folderParentOptions = ref<FolderItem[]>([]);

async function openNewFolder() {
  Object.assign(newFolderForm, { project_id: "", parent_id: "", name: "" });
  if (!allProjects.value.length) await loadAllProjects();
  folderProjects.value = allProjects.value;
  folderParentOptions.value = [];
  showFolderDialog.value = true;
}

async function onFolderProjectChange(pid: string) {
  folderParentOptions.value = [];
  newFolderForm.parent_id = "";
  if (!pid) return;
  const res = await api.get(`/projects/${pid}/data-folders`);
  folderParentOptions.value = (res.data?.data ?? []).filter(
    (f: FolderItem) => f.depth < 2,
  );
}

async function submitNewFolder() {
  if (!newFolderForm.project_id) {
    ElMessage.warning("请选择课题");
    return;
  }
  if (!newFolderForm.name.trim()) {
    ElMessage.warning("请输入文件夹名称");
    return;
  }
  await api.post("/data-folders", {
    project_id: newFolderForm.project_id,
    parent_id: newFolderForm.parent_id || undefined,
    name: newFolderForm.name.trim(),
  });
  ElMessage.success("文件夹创建成功");
  showFolderDialog.value = false;
  if (filterProjectID.value === newFolderForm.project_id) {
    await fetchFolders(filterProjectID.value);
  }
}

async function deleteFolder(node: FolderItem) {
  await ElMessageBox.confirm(
    `确认删除文件夹 "${node.name}"？文件夹必须为空才可删除。`,
    "删除文件夹",
    { confirmButtonText: "删除", cancelButtonText: "取消", type: "warning" },
  );
  await api.delete(`/data-folders/${node.id}`);
  ElMessage.success("文件夹已删除");
  await fetchFolders(filterProjectID.value);
}

// ─────────────────────────── utils ───────────────────────────────

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
  return `${(bytes / 1024 / 1024 / 1024).toFixed(2)} GB`;
}

function formatDate(s: string): string {
  if (!s) return "";
  return s.slice(0, 10);
}

// ─────────────────────────── lifecycle ───────────────────────────

onMounted(async () => {
  await fetchReference();
  await loadAllProjects();
  fetchAssets();
});
</script>

<template>
  <AppLayout>
    <!-- toolbar -->
    <div class="toolbar">
      <el-input
        v-model="keyword"
        placeholder="搜索文件名称…"
        clearable
        style="width: 260px"
        @keyup.enter="handleSearch"
        @clear="handleSearch"
      >
        <template #prefix><el-icon><Search /></el-icon></template>
      </el-input>

      <el-select
        v-model="filterProjectID"
        placeholder="筛选课题"
        clearable
        style="width: 200px"
        @change="onProjectFilterChange"
      >
        <el-option
          v-for="p in allProjects"
          :key="p.id"
          :label="p.name"
          :value="p.id"
        />
      </el-select>

      <div style="flex:1" />

      <el-button type="primary" @click="openUpload">
        <el-icon><Upload /></el-icon>&nbsp;上传文件
      </el-button>
      <el-button @click="openNewFolder">
        <el-icon><FolderAdd /></el-icon>&nbsp;新建文件夹
      </el-button>
    </div>

    <!-- main content -->
    <div class="content-area">
      <!-- folder sidebar -->
      <aside v-if="filterProjectID && folderTree.length" class="folder-sidebar">
        <div class="sidebar-title"><el-icon><FolderOpened /></el-icon> 文件夹</div>
        <div
          class="folder-item"
          :class="{ active: !selectedFolderID }"
          @click="onFolderSelect('')"
        >全部</div>
        <template v-for="node in folderTree" :key="node.id">
          <div
            class="folder-item depth-0"
            :class="{ active: selectedFolderID === node.id }"
            @click="onFolderSelect(node.id)"
          >
            <el-icon><FolderOpened /></el-icon>
            {{ node.name }}
            <el-icon class="folder-del" @click.stop="deleteFolder(node)"><Delete /></el-icon>
          </div>
          <template v-for="child in node.children" :key="child.id">
            <div
              class="folder-item depth-1"
              :class="{ active: selectedFolderID === child.id }"
              @click="onFolderSelect(child.id)"
            >
              <el-icon><FolderOpened /></el-icon>
              {{ child.name }}
              <el-icon class="folder-del" @click.stop="deleteFolder(child)"><Delete /></el-icon>
            </div>
            <template v-for="leaf in child.children" :key="leaf.id">
              <div
                class="folder-item depth-2"
                :class="{ active: selectedFolderID === leaf.id }"
                @click="onFolderSelect(leaf.id)"
              >
                <el-icon><FolderOpened /></el-icon>
                {{ leaf.name }}
                <el-icon class="folder-del" @click.stop="deleteFolder(leaf)"><Delete /></el-icon>
              </div>
            </template>
          </template>
        </template>
      </aside>

      <!-- asset list -->
      <div class="asset-list">
        <div class="list-header">
          <span class="count">共 {{ total }} 个文件</span>
        </div>

        <el-empty v-if="!assets.length" description="暂无数据文件" />

        <el-table v-else :data="assets" style="width:100%" stripe>
          <el-table-column label="文件名称" min-width="200">
            <template #default="{ row }">
              <div class="asset-name">{{ row.display_name }}</div>
              <div class="asset-filename">{{ row.file_name }}</div>
            </template>
          </el-table-column>
          <el-table-column label="课题" prop="project_name" min-width="140" />
          <el-table-column label="文件夹" prop="folder_name" min-width="120" />
          <el-table-column label="大小" min-width="100">
            <template #default="{ row }">{{ formatSize(row.file_size) }}</template>
          </el-table-column>
          <el-table-column label="上传者" prop="created_by_name" min-width="100" />
          <el-table-column label="上传时间" min-width="110">
            <template #default="{ row }">{{ formatDate(row.created_at) }}</template>
          </el-table-column>
          <el-table-column label="操作" width="120" fixed="right">
            <template #default="{ row }">
              <el-button type="primary" link @click="downloadAsset(row)">
                <el-icon><Download /></el-icon>
              </el-button>
              <el-button type="danger" link @click="deleteAsset(row)">
                <el-icon><Delete /></el-icon>
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </div>

    <!-- Upload Dialog -->
    <el-dialog v-model="showUploadDialog" title="上传数据文件" width="520px" :close-on-click-modal="false">
      <el-form label-width="90px">
        <el-form-item label="团队空间" required>
          <el-select v-model="uploadForm.team_space_id" @change="loadProjects" style="width:100%">
            <el-option v-for="ts in teamSpaces" :key="ts.id" :label="ts.name" :value="ts.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="关联课题" required>
          <el-select v-model="uploadForm.project_id" @change="onUploadProjectChange" style="width:100%">
            <el-option v-for="p in projects" :key="p.id" :label="p.name" :value="p.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="文件夹">
          <el-select v-model="uploadForm.folder_id" clearable style="width:100%">
            <el-option
              v-for="f in uploadFolderOptions"
              :key="f.id"
              :label="'  '.repeat(f.depth) + f.name"
              :value="f.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="文件名称" required>
          <el-input v-model="uploadForm.display_name" placeholder="为文件取一个有意义的名称" />
          <div class="field-hint">{{ FILE_NAME_HINT }}</div>
        </el-form-item>
        <el-form-item label="说明">
          <el-input v-model="uploadForm.description" type="textarea" :rows="2" placeholder="可选" />
        </el-form-item>
        <el-form-item label="选择文件" required>
          <input type="file" @change="onFileChange" style="width:100%" />
          <div v-if="uploadFile" class="file-info">
            {{ uploadFile.name }} ({{ formatSize(uploadFile.size) }})
          </div>
        </el-form-item>
        <el-form-item v-if="uploadLoading" label="上传进度">
          <el-progress :percentage="uploadProgress" style="width:100%" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showUploadDialog = false" :disabled="uploadLoading">取消</el-button>
        <el-button type="primary" :loading="uploadLoading" @click="submitUpload">上传</el-button>
      </template>
    </el-dialog>

    <!-- New Folder Dialog -->
    <el-dialog v-model="showFolderDialog" title="新建文件夹" width="420px">
      <el-form label-width="90px">
        <el-form-item label="课题" required>
          <el-select v-model="newFolderForm.project_id" @change="onFolderProjectChange" style="width:100%">
            <el-option v-for="p in folderProjects" :key="p.id" :label="p.name" :value="p.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="父文件夹">
          <el-select v-model="newFolderForm.parent_id" clearable style="width:100%" placeholder="顶层文件夹">
            <el-option v-for="f in folderParentOptions" :key="f.id" :label="f.name" :value="f.id" />
          </el-select>
          <div class="field-hint">最多 2 层，选择父文件夹后可创建子文件夹</div>
        </el-form-item>
        <el-form-item label="名称" required>
          <el-input v-model="newFolderForm.name" placeholder="文件夹名称" maxlength="128" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showFolderDialog = false">取消</el-button>
        <el-button type="primary" @click="submitNewFolder">创建</el-button>
      </template>
    </el-dialog>
  </AppLayout>
</template>

<style scoped>
.toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}

.content-area {
  display: flex;
  gap: 16px;
  min-height: 400px;
}

.folder-sidebar {
  width: 200px;
  flex-shrink: 0;
  border: 1px solid var(--el-border-color);
  border-radius: 6px;
  padding: 12px 0;
  background: var(--el-bg-color);
}

.sidebar-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-secondary);
  padding: 0 14px 8px;
  display: flex;
  align-items: center;
  gap: 4px;
}

.folder-item {
  padding: 6px 14px;
  font-size: 13px;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--el-text-color-primary);
  position: relative;
}

.folder-item:hover {
  background: var(--el-fill-color-light);
}

.folder-item.active {
  background: var(--el-color-primary-light-9);
  color: var(--el-color-primary);
}

.folder-item.depth-1 { padding-left: 28px; }
.folder-item.depth-2 { padding-left: 42px; }

.folder-del {
  margin-left: auto;
  color: var(--el-text-color-placeholder);
  font-size: 12px;
  opacity: 0;
}

.folder-item:hover .folder-del {
  opacity: 1;
}

.folder-del:hover {
  color: var(--el-color-danger);
}

.asset-list {
  flex: 1;
  min-width: 0;
}

.list-header {
  display: flex;
  align-items: center;
  margin-bottom: 12px;
}

.count {
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.asset-name {
  font-weight: 500;
  font-size: 14px;
}

.asset-filename {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-top: 2px;
}

.field-hint {
  font-size: 11px;
  color: var(--el-text-color-placeholder);
  margin-top: 4px;
  line-height: 1.4;
}

.file-info {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-top: 4px;
}
</style>
