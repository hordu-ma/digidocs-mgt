<script setup lang="ts">
import { ElMessage } from "element-plus";
import type { UploadRawFile } from "element-plus";
import { onMounted, reactive, ref } from "vue";
import { useRouter } from "vue-router";

import AppLayout from "@/components/AppLayout.vue";
import api from "@/api";

const router = useRouter();
const rows = ref<any[]>([]);
const total = ref(0);
const page = ref(1);
const pageSize = ref(20);
const keyword = ref("");

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
      page: page.value,
      page_size: pageSize.value,
      keyword: keyword.value,
    },
  });
  rows.value = res.data?.data ?? [];
  total.value = res.data?.meta?.total ?? 0;
}

function handleSearch() {
  page.value = 1;
  fetchDocuments();
}

function handlePageChange(p: number) {
  page.value = p;
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
  title: "",
  description: "",
  commit_message: "",
});
const createFile = ref<UploadRawFile | null>(null);

function openCreate() {
  Object.assign(createForm, {
    team_space_id: "",
    project_id: "",
    folder_id: "",
    title: "",
    description: "",
    commit_message: "",
  });
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
    !createForm.project_id
  ) {
    ElMessage.warning("请填写标题、团队空间 ID 和课题 ID");
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

onMounted(fetchDocuments);
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
            @keyup.enter="handleSearch"
          />
        </div>
        <ElTable :data="rows" style="width: 100%" @row-click="goDetail">
          <ElTableColumn prop="title" label="文档标题" />
          <ElTableColumn label="当前责任人">
            <template #default="{ row }">{{
              row.current_owner?.display_name ?? "-"
            }}</template>
          </ElTableColumn>
          <ElTableColumn label="当前版本">
            <template #default="{ row }">{{
              row.current_version_no ?? "-"
            }}</template>
          </ElTableColumn>
          <ElTableColumn label="状态">
            <template #default="{ row }">
              <ElTag>{{
                statusLabel[row.current_status] ?? row.current_status
              }}</ElTag>
            </template>
          </ElTableColumn>
          <ElTableColumn prop="updated_at" label="更新时间" />
        </ElTable>
        <ElPagination
          v-if="total > pageSize"
          :current-page="page"
          :page-size="pageSize"
          :total="total"
          layout="prev, pager, next"
          style="margin-top: 16px; justify-content: flex-end"
          @current-change="handlePageChange"
        />
      </ElCard>

      <ElDialog v-model="showCreateDialog" title="新建文档" width="520px">
        <ElForm label-position="top">
          <ElFormItem label="标题" required>
            <ElInput v-model="createForm.title" placeholder="文档标题" />
          </ElFormItem>
          <ElFormItem label="团队空间 ID" required>
            <ElInput
              v-model="createForm.team_space_id"
              placeholder="团队空间 UUID"
            />
          </ElFormItem>
          <ElFormItem label="课题 ID" required>
            <ElInput v-model="createForm.project_id" placeholder="课题 UUID" />
          </ElFormItem>
          <ElFormItem label="目录 ID">
            <ElInput
              v-model="createForm.folder_id"
              placeholder="目录 UUID（可选）"
            />
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
  margin-bottom: 16px;
}
</style>
