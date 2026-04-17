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
const assetTypeFilter = ref("all");

// folder sidebar
const folders = ref<FolderItem[]>([]);
const folderTree = computed<FolderItem[]>(() => buildTree(folders.value));
const selectedFolderID = ref("");
const filteredAssets = computed(() => {
  if (assetTypeFilter.value === "all") {
    return assets.value;
  }
  return assets.value.filter((item) => inferAssetFileType(item.file_name) === assetTypeFilter.value);
});

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
  filterProjectID.value = projectID;
  selectedFolderID.value = "";
  filterFolderID.value = "";
  await fetchFolders(projectID);
  fetchAssets();
}

function clearProjectFilter() {
  filterProjectID.value = "";
  selectedFolderID.value = "";
  folders.value = [];
  fetchAssets();
}

function clearFolderSelect() {
  selectedFolderID.value = "";
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
  uploadForm.project_id = "";
  uploadForm.folder_id = "";
  uploadFolderOptions.value = [];
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
const fileInputRef = ref<HTMLInputElement | null>(null);
const isDragging = ref(false);
const dragCounter = ref(0);

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

function onDropZoneClick() {
  fileInputRef.value?.click();
}

function onDragEnter(e: DragEvent) {
  e.preventDefault();
  dragCounter.value++;
  isDragging.value = true;
}

function onDragLeave() {
  dragCounter.value--;
  if (dragCounter.value <= 0) {
    dragCounter.value = 0;
    isDragging.value = false;
  }
}

function onDrop(e: DragEvent) {
  dragCounter.value = 0;
  isDragging.value = false;
  const file = e.dataTransfer?.files?.[0];
  if (file) {
    uploadFile.value = file;
    if (!uploadForm.display_name) uploadForm.display_name = file.name;
  }
}

function clearUploadFile() {
  uploadFile.value = null;
  uploadForm.display_name = "";
  if (fileInputRef.value) fileInputRef.value.value = "";
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
  isDragging.value = false;
  dragCounter.value = 0;
  if (fileInputRef.value) fileInputRef.value.value = "";
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

function inferAssetFileType(fileName: string): string {
  const ext = fileName?.split(".").pop()?.toLowerCase() ?? "";
  if (["jpg", "jpeg", "png", "gif", "svg", "webp", "tiff", "bmp", "heic", "raw"].includes(ext)) return "img";
  if (["mp4", "mov", "avi", "mkv", "webm", "flv", "m4v", "wmv"].includes(ext)) return "vid";
  if (["zip", "tar", "gz", "rar", "7z", "bz2", "xz", "tgz"].includes(ext)) return "zip";
  if (["pt", "pth", "pkl", "onnx", "h5", "pb", "safetensors", "bin", "npz", "npy", "ckpt"].includes(ext)) return "mdl";
  if (["csv", "json", "jsonl", "parquet", "hdf5", "tsv"].includes(ext)) return "dat";
  if (["py", "js", "ts", "sh", "ipynb", "r", "m", "cpp", "c", "java", "go"].includes(ext)) return "code";
  if (["pdf", "docx", "doc", "pptx", "ppt", "txt", "md"].includes(ext)) return "doc";
  return "file";
}

function getFileExt(fileName: string): string {
  const ext = fileName?.split(".").pop()?.toUpperCase() ?? "FILE";
  return ext.length > 5 ? ext.slice(0, 4) + "…" : ext;
}

const assetTypeOptions = [
  { key: "all", label: "全部文件" },
  { key: "img", label: "图片" },
  { key: "vid", label: "视频" },
  { key: "zip", label: "压缩包" },
  { key: "mdl", label: "模型" },
  { key: "dat", label: "数据集" },
  { key: "code", label: "代码" },
  { key: "doc", label: "文档" },
];

// ─────────────────────────── lifecycle ───────────────────────────

onMounted(async () => {
  await fetchReference();
  await loadAllProjects();
  fetchAssets();
});
</script>

<template>
  <AppLayout>
    <div class="page-shell">
      <div class="page-header">
        <div>
          <div class="page-eyebrow">数据资产库</div>
          <h1>数据管理</h1>
          <p>按课题统一存储图片、视频、压缩文件、模型等数据文件，快速检索，不依赖复杂目录体系。</p>
        </div>
        <div class="header-actions">
          <ElButton @click="openNewFolder">
            <ElIcon><FolderAdd /></ElIcon>
            新建文件夹
          </ElButton>
          <ElButton type="primary" @click="openUpload">
            <ElIcon><Upload /></ElIcon>
            上传数据文件
          </ElButton>
        </div>
      </div>

      <div class="asset-workspace">
        <!-- project rail -->
        <aside class="project-rail page-card">
          <div class="rail-title">
            <ElIcon><FolderOpened /></ElIcon>
            课题
          </div>
          <button
            class="project-filter"
            :class="{ active: filterProjectID === '' }"
            type="button"
            @click="clearProjectFilter"
          >
            <span>全部文件</span>
            <strong v-if="filterProjectID === ''">{{ total }}</strong>
          </button>
          <button
            v-for="p in allProjects"
            :key="p.id"
            class="project-filter"
            :class="{ active: filterProjectID === p.id }"
            type="button"
            @click="onProjectFilterChange(p.id)"
          >
            <span>{{ p.name }}</span>
          </button>

          <!-- folder tree (shown when a project is selected and has folders) -->
          <template v-if="filterProjectID && folderTree.length">
            <div class="rail-divider" />
            <div class="rail-title rail-subtitle">
              <ElIcon><FolderOpened /></ElIcon>
              文件夹
            </div>
            <div class="folder-filter-row">
              <button
                class="folder-filter"
                :class="{ active: selectedFolderID === '' }"
                type="button"
                @click="clearFolderSelect"
              >
                <span>全部文件</span>
              </button>
            </div>
            <template v-for="node in folderTree" :key="node.id">
              <div class="folder-filter-row">
                <button
                  class="folder-filter"
                  :class="{ active: selectedFolderID === node.id }"
                  type="button"
                  @click="onFolderSelect(node.id)"
                >
                  <ElIcon><FolderOpened /></ElIcon>
                  <span>{{ node.name }}</span>
                </button>
                <button
                  class="folder-del-btn"
                  type="button"
                  title="删除文件夹"
                  @click.stop="deleteFolder(node)"
                >
                  <ElIcon><Delete /></ElIcon>
                </button>
              </div>
              <template v-for="child in node.children" :key="child.id">
                <div class="folder-filter-row sub">
                  <button
                    class="folder-filter"
                    :class="{ active: selectedFolderID === child.id }"
                    type="button"
                    @click="onFolderSelect(child.id)"
                  >
                    <ElIcon><FolderOpened /></ElIcon>
                    <span>{{ child.name }}</span>
                  </button>
                  <button
                    class="folder-del-btn"
                    type="button"
                    title="删除文件夹"
                    @click.stop="deleteFolder(child)"
                  >
                    <ElIcon><Delete /></ElIcon>
                  </button>
                </div>
              </template>
            </template>
          </template>
        </aside>

        <div class="asset-column">
          <!-- asset panel -->
          <section class="page-card asset-panel">
            <div class="control-bar asset-control-bar">
              <div class="segmented-filters">
                <button
                  v-for="item in assetTypeOptions"
                  :key="item.key"
                  class="segment-chip"
                  :class="{ active: assetTypeFilter === item.key }"
                  type="button"
                  @click="assetTypeFilter = item.key"
                >
                  {{ item.label }}
                </button>
              </div>
            </div>
            <div class="toolbar">
              <ElInput
                v-model="keyword"
                placeholder="搜索文件名称…"
                clearable
                @keyup.enter="handleSearch"
                @clear="handleSearch"
              >
                <template #prefix><ElIcon><Search /></ElIcon></template>
              </ElInput>
              <span class="file-count">当前显示 {{ filteredAssets.length }} / {{ total }} 个文件</span>
            </div>

            <div v-if="filteredAssets.length === 0" class="empty-state">
              <p class="empty-title">暂无数据文件</p>
              <p class="empty-hint">上传后，文件会按课题自动归入数据资产库</p>
            </div>

            <div v-else class="asset-list">
              <div v-for="row in filteredAssets" :key="row.id" class="asset-row">
                <span class="file-badge" :class="inferAssetFileType(row.file_name)">
                  {{ getFileExt(row.file_name) }}
                </span>
                <div class="asset-main">
                  <strong>{{ row.display_name }}</strong>
                  <span>
                    {{ row.file_name }}
                    <template v-if="row.project_name"> · {{ row.project_name }}</template>
                    <template v-if="row.folder_name"> / {{ row.folder_name }}</template>
                  </span>
                </div>
                <span class="asset-meta size">{{ formatSize(row.file_size) }}</span>
                <span class="asset-meta owner">{{ row.created_by_name }}</span>
                <span class="asset-meta date">{{ formatDate(row.created_at) }}</span>
                <div class="asset-actions">
                  <ElButton size="small" text title="下载" @click="downloadAsset(row)">
                    <ElIcon><Download /></ElIcon>
                  </ElButton>
                  <ElButton size="small" text type="danger" title="删除" @click="deleteAsset(row)">
                    <ElIcon><Delete /></ElIcon>
                  </ElButton>
                </div>
              </div>
            </div>
          </section>
        </div>
      </div>

      <!-- Upload Dialog -->
      <ElDialog v-model="showUploadDialog" title="上传数据文件" width="520px" :close-on-click-modal="false">
        <ElForm label-position="top">
          <ElFormItem label="团队空间" required>
            <ElSelect v-model="uploadForm.team_space_id" @change="loadProjects" style="width:100%">
              <ElOption v-for="ts in teamSpaces" :key="ts.id" :label="ts.name" :value="ts.id" />
            </ElSelect>
          </ElFormItem>
          <ElFormItem label="关联课题" required>
            <ElSelect v-model="uploadForm.project_id" @change="onUploadProjectChange" style="width:100%">
              <ElOption v-for="p in projects" :key="p.id" :label="p.name" :value="p.id" />
            </ElSelect>
          </ElFormItem>
          <ElFormItem label="文件夹">
            <ElSelect v-model="uploadForm.folder_id" clearable style="width:100%">
              <ElOption
                v-for="f in uploadFolderOptions"
                :key="f.id"
                :label="'　'.repeat(f.depth) + f.name"
                :value="f.id"
              />
            </ElSelect>
          </ElFormItem>
          <ElFormItem label="文件名称" required>
            <ElInput v-model="uploadForm.display_name" placeholder="为文件取一个有意义的名称" />
            <p class="field-hint">{{ FILE_NAME_HINT }}</p>
          </ElFormItem>
          <ElFormItem label="说明">
            <ElInput v-model="uploadForm.description" type="textarea" :rows="2" placeholder="可选" />
          </ElFormItem>
          <ElFormItem label="选择文件" required>
            <input ref="fileInputRef" type="file" style="display:none" @change="onFileChange" />
            <div
              class="drop-zone"
              :class="{ dragging: isDragging, 'has-file': !!uploadFile }"
              @click="onDropZoneClick"
              @dragenter="onDragEnter"
              @dragover.prevent
              @dragleave="onDragLeave"
              @drop.prevent="onDrop"
            >
              <template v-if="uploadFile">
                <div class="dz-file-info">
                  <span class="dz-name">{{ uploadFile.name }}</span>
                  <span class="dz-size">{{ formatSize(uploadFile.size) }}</span>
                </div>
                <button class="dz-clear" type="button" @click.stop="clearUploadFile">✕ 重新选择</button>
              </template>
              <template v-else>
                <div class="dz-body">
                  <div class="dz-icon-wrap"><ElIcon size="28"><Upload /></ElIcon></div>
                  <p class="dz-hint">将文件拖拽到此处，或 <span class="dz-link">点击选择文件</span></p>
                  <p class="dz-sub">支持所有格式，单文件最大 10 GB</p>
                </div>
              </template>
            </div>
          </ElFormItem>
          <ElFormItem v-if="uploadLoading" label="上传进度">
            <ElProgress :percentage="uploadProgress" style="width:100%" />
          </ElFormItem>
        </ElForm>
        <template #footer>
          <ElButton @click="showUploadDialog = false" :disabled="uploadLoading">取消</ElButton>
          <ElButton type="primary" :loading="uploadLoading" @click="submitUpload">上传</ElButton>
        </template>
      </ElDialog>

      <!-- New Folder Dialog -->
      <ElDialog v-model="showFolderDialog" title="新建文件夹" width="420px">
        <ElForm label-position="top">
          <ElFormItem label="课题" required>
            <ElSelect v-model="newFolderForm.project_id" @change="onFolderProjectChange" style="width:100%">
              <ElOption v-for="p in folderProjects" :key="p.id" :label="p.name" :value="p.id" />
            </ElSelect>
          </ElFormItem>
          <ElFormItem label="父文件夹">
            <ElSelect v-model="newFolderForm.parent_id" clearable style="width:100%" placeholder="顶层文件夹">
              <ElOption v-for="f in folderParentOptions" :key="f.id" :label="f.name" :value="f.id" />
            </ElSelect>
            <p class="field-hint">最多 2 层，选择父文件夹后可创建子文件夹</p>
          </ElFormItem>
          <ElFormItem label="名称" required>
            <ElInput v-model="newFolderForm.name" placeholder="文件夹名称" maxlength="128" />
          </ElFormItem>
        </ElForm>
        <template #footer>
          <ElButton @click="showFolderDialog = false">取消</ElButton>
          <ElButton type="primary" @click="submitNewFolder">创建</ElButton>
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
  overflow-y: auto;
  max-height: calc(100vh - 140px);
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
  flex-shrink: 0;
}

.project-filter:hover,
.project-filter.active {
  border-color: #c8ddf2;
  background: var(--dd-primary-soft);
  color: var(--dd-primary-strong);
}

.asset-column {
  display: flex;
  flex-direction: column;
  gap: 14px;
  min-width: 0;
}

/* Folder tree in Rail */
.rail-divider {
  height: 1px;
  background: var(--dd-line);
  margin: 6px 0;
}

.rail-subtitle {
  font-size: 12px;
  font-weight: 600;
  color: var(--dd-muted);
}

.folder-filter-row {
  display: flex;
  align-items: center;
  gap: 2px;
}

.folder-filter-row.sub {
  padding-left: 20px;
}

.folder-filter {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 7px;
  min-height: 36px;
  padding: 0 10px;
  border: 1px solid transparent;
  border-radius: 8px;
  background: transparent;
  color: var(--dd-ink-2);
  font-size: 13px;
  text-align: left;
  cursor: pointer;
  overflow: hidden;
  min-width: 0;
}

.folder-filter span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.folder-filter:hover,
.folder-filter.active {
  border-color: #c8ddf2;
  background: var(--dd-primary-soft);
  color: var(--dd-primary-strong);
}

.folder-del-btn {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--dd-muted);
  cursor: pointer;
  opacity: 0;
  transition: opacity 0.15s;
}

.folder-filter-row:hover .folder-del-btn {
  opacity: 1;
}

.folder-del-btn:hover {
  background: var(--el-fill-color-light);
  color: var(--el-color-danger);
}

/* Asset panel */
.asset-panel {
  padding: 20px;
}

.asset-control-bar {
  margin-bottom: 14px;
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

.file-count {
  white-space: nowrap;
  font-size: 13px;
  color: var(--dd-muted);
}

/* Asset rows */
.asset-list {
  display: grid;
}

.asset-row {
  display: grid;
  grid-template-columns: 64px minmax(200px, 1fr) 80px 100px 90px 76px;
  gap: 12px;
  align-items: center;
  min-height: 68px;
  padding: 10px 4px;
  border-top: 1px solid var(--dd-line-soft);
}

.asset-row:first-child {
  border-top: none;
}

.asset-main {
  display: grid;
  gap: 4px;
  min-width: 0;
}

.asset-main strong {
  overflow: hidden;
  color: var(--dd-ink);
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 14px;
}

.asset-main span {
  overflow: hidden;
  color: var(--dd-muted);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.asset-meta {
  font-size: 12px;
  color: var(--dd-muted);
  white-space: nowrap;
}

.asset-actions {
  display: flex;
  gap: 2px;
  justify-content: flex-end;
}

/* File type badge colors */
.file-badge.img  { background: #fff0e6; color: #c0570a; }
.file-badge.vid  { background: #f3edff; color: #6839d4; }
.file-badge.zip  { background: #e6f5f0; color: #1a7a5e; }
.file-badge.mdl  { background: #e6eeff; color: #1d42a2; }
.file-badge.dat  { background: #e6f7ea; color: #1a7a40; }
.file-badge.code { background: #f0f4ff; color: #3d61b0; }
.file-badge.doc  { background: #f5f5f5; color: #555;    }
.file-badge.file { background: #f5f5f5; color: #888;    }

.field-hint {
  font-size: 11px;
  color: var(--dd-muted);
  margin-top: 4px;
  line-height: 1.4;
}

/* Drop zone */
.drop-zone {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  width: 100%;
  min-height: 120px;
  border: 2px dashed var(--dd-line);
  border-radius: 12px;
  background: var(--dd-surface-soft);
  cursor: pointer;
  transition: border-color 0.2s, background 0.2s;
  padding: 20px 16px;
  box-sizing: border-box;
  text-align: center;
  user-select: none;
}

.drop-zone:hover,
.drop-zone.dragging {
  border-color: var(--dd-primary-strong);
  background: var(--dd-primary-soft);
}

.drop-zone.has-file {
  flex-direction: row;
  justify-content: space-between;
  align-items: center;
  min-height: 64px;
  padding: 12px 16px;
  border-style: solid;
  border-color: #c8ddf2;
  background: var(--dd-primary-soft);
  gap: 12px;
}

.dz-body {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
}

.dz-icon-wrap {
  color: var(--dd-muted);
  display: flex;
  align-items: center;
  justify-content: center;
}

.dz-hint {
  font-size: 14px;
  color: var(--dd-ink-2);
  margin: 0;
}

.dz-link {
  color: var(--dd-primary-strong);
  font-weight: 500;
}

.dz-sub {
  font-size: 12px;
  color: var(--dd-muted);
  margin: 0;
}

.dz-file-info {
  display: flex;
  flex-direction: column;
  gap: 3px;
  min-width: 0;
  flex: 1;
  text-align: left;
}

.dz-name {
  font-size: 13px;
  font-weight: 500;
  color: var(--dd-ink);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dz-size {
  font-size: 12px;
  color: var(--dd-muted);
}

.dz-clear {
  flex-shrink: 0;
  font-size: 12px;
  color: var(--dd-muted);
  background: transparent;
  border: 1px solid var(--dd-line);
  border-radius: 6px;
  padding: 4px 10px;
  cursor: pointer;
  transition: all 0.15s;
}

.dz-clear:hover {
  color: var(--el-color-danger);
  border-color: var(--el-color-danger);
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 48px 16px;
  text-align: center;
}

.header-actions {
  display: flex;
  gap: 10px;
  align-items: center;
  flex-shrink: 0;
}

@media (max-width: 1080px) {
  .asset-workspace {
    grid-template-columns: 1fr;
  }

  .project-rail {
    position: static;
    max-height: none;
    overflow-y: visible;
  }

  .asset-row {
    grid-template-columns: 56px minmax(0, 1fr) auto;
  }

  .asset-meta.owner,
  .asset-meta.date {
    display: none;
  }

  .asset-meta.size {
    grid-column: 3;
  }

  .asset-actions {
    grid-column: 3;
    grid-row: 1;
  }
}

@media (max-width: 720px) {
  .toolbar {
    flex-direction: column;
    align-items: flex-start;
  }

  .toolbar .el-input {
    max-width: none;
    width: 100%;
  }
}
</style>
