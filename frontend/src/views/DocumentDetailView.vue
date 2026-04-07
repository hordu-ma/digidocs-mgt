<script setup lang="ts">
import { ElMessage, ElMessageBox } from "element-plus";
import type { UploadRawFile } from "element-plus";
import { computed, onMounted, reactive, ref } from "vue";
import { useRoute, useRouter } from "vue-router";

import AppLayout from "@/components/AppLayout.vue";
import api from "@/api";

const route = useRoute();
const router = useRouter();
const documentID = route.params.id as string;

const doc = ref<any>(null);
const versions = ref<any[]>([]);
const flows = ref<any[]>([]);
const suggestions = ref<any[]>([]);
const actionLoading = ref(false);
const summaryLoading = ref(false);

const statusLabel: Record<string, string> = {
  draft: "草稿",
  in_progress: "处理中",
  pending_handover: "待交接",
  handed_over: "已交接",
  finalized: "定稿",
  archived: "已归档",
};

// Map current status → available flow actions
const flowActions: Record<
  string,
  { action: string; label: string; endpoint: string }[]
> = {
  draft: [
    {
      action: "mark_in_progress",
      label: "开始处理",
      endpoint: "mark-in-progress",
    },
  ],
  in_progress: [
    { action: "transfer", label: "转交", endpoint: "transfer" },
    { action: "finalize", label: "定稿", endpoint: "finalize" },
  ],
  pending_handover: [
    {
      action: "accept_transfer",
      label: "接受转交",
      endpoint: "accept-transfer",
    },
  ],
  finalized: [{ action: "archive", label: "归档", endpoint: "archive" }],
  archived: [{ action: "unarchive", label: "取消归档", endpoint: "unarchive" }],
};

const availableActions = computed(() => {
  const status = doc.value?.current_status;
  return status ? (flowActions[status] ?? []) : [];
});

async function loadData() {
  const [docRes, versionsRes, flowsRes, suggestionsRes] = await Promise.all([
    api.get(`/documents/${documentID}`),
    api.get(`/documents/${documentID}/versions`),
    api.get(`/documents/${documentID}/flows`),
    api.get("/assistant/suggestions", {
      params: {
        related_type: "document",
        related_id: documentID,
        status: "pending",
      },
    }),
  ]);
  doc.value = docRes.data?.data ?? null;
  versions.value = versionsRes.data?.data ?? [];
  flows.value = flowsRes.data?.data ?? [];
  suggestions.value = suggestionsRes.data?.data ?? [];
}

async function applyFlowAction(endpoint: string, label: string) {
  actionLoading.value = true;
  try {
    const payload: Record<string, string> = {};
    if (endpoint === "transfer") {
      const { value } = await ElMessageBox.prompt(
        "请输入接收人的用户 ID",
        "转交文档",
        {
          confirmButtonText: "确认转交",
          cancelButtonText: "取消",
          inputPlaceholder: "目标用户 UUID",
          inputValidator: (input) => {
            if (!input?.trim()) {
              return "接收人用户 ID 不能为空";
            }
            return true;
          },
        },
      );
      payload.to_user_id = value.trim();
    }

    await api.post(`/documents/${documentID}/flow/${endpoint}`, payload);
    ElMessage.success(`${label}成功`);
    await loadData();
  } catch (err: any) {
    if (err === "cancel") {
      return;
    }
    const msg = err.response?.data?.message ?? `${label}失败`;
    ElMessage.error(msg);
  } finally {
    actionLoading.value = false;
  }
}

// --- Edit document info ---
const showEditDialog = ref(false);
const editLoading = ref(false);
const editForm = reactive({ title: "", description: "", folder_id: "" });

function openEdit() {
  editForm.title = doc.value?.title ?? "";
  editForm.description = doc.value?.description ?? "";
  editForm.folder_id = "";
  showEditDialog.value = true;
}

async function submitEdit() {
  editLoading.value = true;
  try {
    await api.patch(`/documents/${documentID}`, editForm);
    ElMessage.success("文档信息已更新");
    showEditDialog.value = false;
    await loadData();
  } catch (err: any) {
    ElMessage.error(err.response?.data?.message ?? "更新失败");
  } finally {
    editLoading.value = false;
  }
}

// --- Delete / Restore ---
async function deleteDocument() {
  actionLoading.value = true;
  try {
    await api.post(`/documents/${documentID}/delete`, { reason: "" });
    ElMessage.success("文档已删除");
    router.push("/documents");
  } catch (err: any) {
    ElMessage.error(err.response?.data?.message ?? "删除失败");
  } finally {
    actionLoading.value = false;
  }
}

async function restoreDocument() {
  actionLoading.value = true;
  try {
    await api.post(`/documents/${documentID}/restore`);
    ElMessage.success("文档已恢复");
    await loadData();
  } catch (err: any) {
    ElMessage.error(err.response?.data?.message ?? "恢复失败");
  } finally {
    actionLoading.value = false;
  }
}

// --- Upload new version ---
const showUploadDialog = ref(false);
const uploadLoading = ref(false);
const uploadMessage = ref("");
const uploadFile = ref<UploadRawFile | null>(null);

function openUpload() {
  uploadMessage.value = "";
  uploadFile.value = null;
  showUploadDialog.value = true;
}

function handleUploadFileChange(file: { raw: UploadRawFile }) {
  uploadFile.value = file.raw;
}

async function submitUpload() {
  if (!uploadFile.value) {
    ElMessage.warning("请选择文件");
    return;
  }
  uploadLoading.value = true;
  try {
    const fd = new FormData();
    fd.append("commit_message", uploadMessage.value);
    fd.append("file", uploadFile.value);
    await api.post(`/documents/${documentID}/versions`, fd, {
      headers: { "Content-Type": "multipart/form-data" },
    });
    ElMessage.success("新版本上传成功");
    showUploadDialog.value = false;
    await loadData();
  } catch (err: any) {
    ElMessage.error(err.response?.data?.message ?? "上传失败");
  } finally {
    uploadLoading.value = false;
  }
}

async function requestSummary() {
  const versionID = doc.value?.current_version_id;
  if (!versionID) {
    ElMessage.warning("当前文档没有可摘要的版本");
    return;
  }

  summaryLoading.value = true;
  try {
    await api.post(`/assistant/documents/${documentID}/summarize`, {
      version_id: versionID,
    });
    ElMessage.success("摘要任务已提交");
    await loadData();
  } catch (err: any) {
    ElMessage.error(err.response?.data?.message ?? "摘要提交失败");
  } finally {
    summaryLoading.value = false;
  }
}

async function updateSuggestionStatus(id: string, action: "confirm" | "dismiss") {
  try {
    await api.post(
      `/assistant/suggestions/${id}/${action}`,
      action === "confirm"
        ? { note: "前端确认采纳" }
        : { reason: "前端手动忽略" },
    );
    ElMessage.success(action === "confirm" ? "建议已确认" : "建议已忽略");
    await loadData();
  } catch (err: any) {
    ElMessage.error(err.response?.data?.message ?? "操作失败");
  }
}

onMounted(loadData);
</script>

<template>
  <AppLayout>
    <div class="page-shell detail-grid">
      <ElCard class="page-card">
        <template #header>文档基本信息</template>
        <ElDescriptions v-if="doc" :column="2" border>
          <ElDescriptionsItem label="标题">{{ doc.title }}</ElDescriptionsItem>
          <ElDescriptionsItem label="当前责任人">{{
            doc.current_owner?.display_name ?? "-"
          }}</ElDescriptionsItem>
          <ElDescriptionsItem label="状态"
            ><ElTag>{{
              statusLabel[doc.current_status] ?? doc.current_status
            }}</ElTag></ElDescriptionsItem
          >
          <ElDescriptionsItem label="描述">{{
            doc.description || "-"
          }}</ElDescriptionsItem>
        </ElDescriptions>
        <div v-if="availableActions.length > 0" class="action-bar">
          <ElButton
            v-for="act in availableActions"
            :key="act.action"
            type="primary"
            :loading="actionLoading"
            @click="applyFlowAction(act.endpoint, act.label)"
            >{{ act.label }}</ElButton
          >
          <ElButton @click="openEdit">编辑信息</ElButton>
          <ElButton @click="openUpload">上传新版本</ElButton>
          <ElButton
            type="danger"
            :loading="actionLoading"
            @click="deleteDocument"
            >删除</ElButton
          >
        </div>
        <div v-else class="action-bar">
          <ElButton @click="openEdit">编辑信息</ElButton>
          <ElButton @click="openUpload">上传新版本</ElButton>
          <ElButton
            type="danger"
            :loading="actionLoading"
            @click="deleteDocument"
            >删除</ElButton
          >
        </div>
      </ElCard>

      <ElCard class="page-card">
        <template #header>AI 摘要与建议</template>
        <div class="assistant-toolbar">
          <ElButton type="primary" :loading="summaryLoading" @click="requestSummary">
            生成摘要
          </ElButton>
        </div>
        <div v-if="suggestions.length === 0" class="summary-text">
          暂无 AI 摘要与建议
        </div>
        <div v-else class="suggestion-list">
          <div v-for="item in suggestions" :key="item.id" class="suggestion-item">
            <div class="suggestion-head">
              <div class="suggestion-title">
                {{ item.title || item.suggestion_type }}
              </div>
              <ElTag :type="item.status === 'pending' ? 'warning' : item.status === 'confirmed' ? 'success' : 'info'">
                {{ item.status }}
              </ElTag>
            </div>
            <div class="summary-text">{{ item.content }}</div>
            <div v-if="item.status === 'pending'" class="suggestion-actions">
              <ElButton size="small" type="success" @click="updateSuggestionStatus(item.id, 'confirm')">
                确认
              </ElButton>
              <ElButton size="small" @click="updateSuggestionStatus(item.id, 'dismiss')">
                忽略
              </ElButton>
            </div>
          </div>
        </div>
      </ElCard>

      <ElCard class="page-card">
        <template #header>版本历史</template>
        <ElTable :data="versions">
          <ElTableColumn prop="version_no" label="版本号" />
          <ElTableColumn prop="file_name" label="文件名" />
          <ElTableColumn prop="summary_status" label="摘要状态" />
          <ElTableColumn prop="created_at" label="提交时间" />
        </ElTable>
      </ElCard>

      <ElCard class="page-card">
        <template #header>流转历史</template>
        <ElTable :data="flows">
          <ElTableColumn prop="action" label="操作" />
          <ElTableColumn label="来源状态">
            <template #default="{ row }">{{
              statusLabel[row.from_status] ?? (row.from_status || "-")
            }}</template>
          </ElTableColumn>
          <ElTableColumn label="目标状态">
            <template #default="{ row }">{{
              statusLabel[row.to_status] ?? row.to_status
            }}</template>
          </ElTableColumn>
          <ElTableColumn prop="created_at" label="时间" />
        </ElTable>
      </ElCard>
    </div>

    <ElDialog v-model="showEditDialog" title="编辑文档信息" width="480px">
      <ElForm label-position="top">
        <ElFormItem label="标题">
          <ElInput v-model="editForm.title" />
        </ElFormItem>
        <ElFormItem label="描述">
          <ElInput v-model="editForm.description" type="textarea" :rows="3" />
        </ElFormItem>
        <ElFormItem label="目录 ID">
          <ElInput
            v-model="editForm.folder_id"
            placeholder="移动到新目录（可选）"
          />
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton @click="showEditDialog = false">取消</ElButton>
        <ElButton type="primary" :loading="editLoading" @click="submitEdit"
          >保存</ElButton
        >
      </template>
    </ElDialog>

    <ElDialog v-model="showUploadDialog" title="上传新版本" width="480px">
      <ElForm label-position="top">
        <ElFormItem label="提交说明">
          <ElInput v-model="uploadMessage" placeholder="版本提交说明" />
        </ElFormItem>
        <ElFormItem label="文件">
          <ElUpload
            :auto-upload="false"
            :limit="1"
            :on-change="handleUploadFileChange"
          >
            <ElButton>选择文件</ElButton>
          </ElUpload>
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton @click="showUploadDialog = false">取消</ElButton>
        <ElButton type="primary" :loading="uploadLoading" @click="submitUpload"
          >上传</ElButton
        >
      </template>
    </ElDialog>
  </AppLayout>
</template>

<style scoped>
.detail-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 18px;
}

.summary-text {
  color: #31465e;
}

.assistant-toolbar {
  margin-bottom: 12px;
}

.suggestion-list {
  display: grid;
  gap: 12px;
}

.suggestion-item {
  padding: 14px;
  border-radius: 12px;
  background: #f7f9fc;
}

.suggestion-head {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  align-items: center;
  margin-bottom: 8px;
}

.suggestion-title {
  font-weight: 600;
}

.suggestion-actions {
  display: flex;
  gap: 8px;
  margin-top: 12px;
}

.action-bar {
  display: flex;
  gap: 10px;
  margin-top: 16px;
}

.tag-row {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
  margin-top: 14px;
}

@media (max-width: 900px) {
  .detail-grid {
    grid-template-columns: 1fr;
  }
}
</style>
