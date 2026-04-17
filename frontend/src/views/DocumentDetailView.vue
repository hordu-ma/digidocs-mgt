<script setup lang="ts">
import {
  ArrowLeft,
  Clock,
  Delete,
  Document,
  Download,
  EditPen,
  Memo,
  Upload,
  UserFilled,
} from "@element-plus/icons-vue";
import { ElMessage } from "element-plus";
import type { UploadRawFile } from "element-plus";
import { computed, onBeforeUnmount, onMounted, reactive, ref } from "vue";
import { useRoute, useRouter } from "vue-router";

import AppLayout from "@/components/AppLayout.vue";
import api from "@/api";

type UserOption = {
  id: string;
  display_name: string;
  role: string;
};

const route = useRoute();
const router = useRouter();
const documentID = route.params.id as string;

const doc = ref<any>(null);
const versions = ref<any[]>([]);
const flows = ref<any[]>([]);
const suggestions = ref<any[]>([]);
const actionLoading = ref(false);
const summaryLoading = ref(false);
const summaryPolling = ref(false);
let summaryPollTimer: number | null = null;
const users = ref<UserOption[]>([]);

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

const flowActionLabel: Record<string, string> = {
  transfer: "转交",
  accept_transfer: "接受转交",
  finalize: "定稿",
  archive: "归档",
  unarchive: "取消归档",
  mark_in_progress: "开始处理",
  create: "创建",
};

const suggestionStatusLabel: Record<string, string> = {
  pending: "待确认",
  confirmed: "已确认",
  dismissed: "已忽略",
  expired: "已过期",
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

const selectableUsers = computed(() =>
  users.value.filter((item) => item.id !== doc.value?.current_owner?.id),
);

const currentVersionNo = computed(
  () => doc.value?.current_version_no ?? versions.value[0]?.version_no ?? "-",
);

function ownerInitial() {
  return (doc.value?.current_owner?.display_name || "责").slice(0, 1);
}

function inferFileType(value?: string) {
  const raw = `${value || versions.value[0]?.file_name || doc.value?.file_type || doc.value?.title || ""}`.toLowerCase();
  const match = raw.match(/\.(docx|xlsx|pptx|pdf|txt|md)$/);
  if (match) return match[1];
  if (raw.includes("pdf")) return "pdf";
  if (raw.includes("xlsx") || raw.includes("表")) return "xlsx";
  if (raw.includes("pptx") || raw.includes("汇报")) return "pptx";
  if (raw.includes("docx") || raw.includes("文档")) return "docx";
  return "doc";
}

function formatTime(value?: string) {
  if (!value) return "-";
  return new Date(value).toLocaleString("zh-CN", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
}

async function downloadVersion(versionID: string, fileName: string) {
  try {
    const res = await api.get(`/versions/${versionID}/download`, { responseType: "blob" });
    const url = URL.createObjectURL(new Blob([res.data]));
    const a = document.createElement("a");
    a.href = url;
    a.download = fileName || "download";
    a.click();
    URL.revokeObjectURL(url);
  } catch {
    ElMessage.error("下载失败，请稍后再试");
  }
}

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

async function loadUsers() {
  const res = await api.get("/users");
  users.value = res.data?.data ?? [];
}

const showTransferDialog = ref(false);
const transferLoading = ref(false);
const transferForm = reactive({
  to_user_id: "",
});

function openTransferDialog() {
  transferForm.to_user_id = "";
  showTransferDialog.value = true;
}

async function submitTransfer() {
  if (!transferForm.to_user_id) {
    ElMessage.warning("请选择接收人");
    return;
  }

  transferLoading.value = true;
  try {
    await api.post(`/documents/${documentID}/flow/transfer`, {
      to_user_id: transferForm.to_user_id,
    });
    ElMessage.success("转交成功");
    showTransferDialog.value = false;
    await loadData();
  } catch (err: any) {
    ElMessage.error(err.response?.data?.message ?? "转交失败");
  } finally {
    transferLoading.value = false;
  }
}

async function applyFlowAction(endpoint: string, label: string) {
  if (endpoint === "transfer") {
    openTransferDialog();
    return;
  }

  actionLoading.value = true;
  try {
    await api.post(`/documents/${documentID}/flow/${endpoint}`, {});
    ElMessage.success(`${label}成功`);
    await loadData();
  } catch (err: any) {
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

function stopSummaryPolling() {
  if (summaryPollTimer !== null) {
    window.clearTimeout(summaryPollTimer);
    summaryPollTimer = null;
  }
  summaryPolling.value = false;
}

async function pollSummaryRequest(requestID: string) {
  try {
    const res = await api.get(`/assistant/requests/${requestID}`);
    const data = res.data?.data;
    if (data?.status === "completed" || data?.status === "failed") {
      stopSummaryPolling();
      await loadData();
      if (data.status === "failed") {
        ElMessage.error(data.error_message ?? "摘要生成失败");
      } else {
        ElMessage.success("摘要已生成");
      }
      return;
    }
    summaryPollTimer = window.setTimeout(() => {
      void pollSummaryRequest(requestID);
    }, 2500);
  } catch {
    stopSummaryPolling();
    ElMessage.error("查询摘要状态失败");
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
    const res = await api.post(`/assistant/documents/${documentID}/summarize`, {
      version_id: versionID,
    });
    const requestID = res.data?.data?.request_id;
    ElMessage.info("摘要任务已提交，AI 正在处理…");
    if (requestID) {
      summaryPolling.value = true;
      void pollSummaryRequest(requestID);
    }
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

onMounted(async () => {
  try {
    await Promise.all([loadData(), loadUsers()]);
  } catch (err: any) {
    ElMessage.error(err.response?.data?.message ?? "加载文档详情失败");
  }
});

onBeforeUnmount(() => {
  stopSummaryPolling();
});
</script>

<template>
  <AppLayout>
    <div class="page-shell document-detail-shell">
      <button class="back-link" type="button" @click="router.push('/documents')">
        <ElIcon><ArrowLeft /></ElIcon>
        返回文档资产库
      </button>

      <section v-if="doc" class="document-hero page-card">
        <div class="document-identity">
          <span class="file-badge file-badge-large" :class="inferFileType()">
            {{ inferFileType().toUpperCase() }}
          </span>
          <div class="identity-copy">
            <div class="page-eyebrow">文档档案</div>
            <h1>{{ doc.title }}</h1>
            <p>{{ doc.description || "暂无文档描述" }}</p>
            <div class="identity-meta">
              <span class="status-pill" :class="statusClass[doc.current_status]">
                {{ statusLabel[doc.current_status] ?? doc.current_status }}
              </span>
              <span class="person-chip">
                <span class="person-avatar">{{ ownerInitial() }}</span>
                {{ doc.current_owner?.display_name ?? "-" }}
              </span>
              <span class="version-chip">当前版本 v{{ currentVersionNo }}</span>
            </div>
          </div>
        </div>
        <div class="hero-actions">
          <ElButton
            v-for="act in availableActions"
            :key="act.action"
            type="primary"
            :loading="actionLoading"
            @click="applyFlowAction(act.endpoint, act.label)"
          >
            {{ act.label }}
          </ElButton>
          <ElButton @click="openEdit">
            <ElIcon><EditPen /></ElIcon>
            档案编辑
          </ElButton>
          <ElButton
            v-if="doc.current_version_id"
            @click="downloadVersion(doc.current_version_id, versions[0]?.file_name)"
          >
            <ElIcon><Download /></ElIcon>
            下载当前版本
          </ElButton>
          <ElButton @click="openUpload">
            <ElIcon><Upload /></ElIcon>
            上传新版本
          </ElButton>
        </div>
      </section>

      <div class="detail-layout">
        <main class="detail-main">
          <section class="page-card info-panel">
            <div class="panel-head">
              <div>
                <h2 class="section-title">正式事实</h2>
                <p class="section-note">来自主业务账本，不混入 AI 建议</p>
              </div>
              <ElIcon :size="20"><Memo /></ElIcon>
            </div>
            <div v-if="doc" class="fact-grid">
              <div class="fact-item">
                <span>文档标题</span>
                <strong>{{ doc.title }}</strong>
              </div>
              <div class="fact-item">
                <span>当前责任人</span>
                <strong>{{ doc.current_owner?.display_name ?? "-" }}</strong>
              </div>
              <div class="fact-item">
                <span>当前状态</span>
                <strong>{{ statusLabel[doc.current_status] ?? doc.current_status }}</strong>
              </div>
              <div class="fact-item">
                <span>当前版本</span>
                <strong>v{{ currentVersionNo }}</strong>
              </div>
            </div>
          </section>

          <section class="page-card timeline-panel">
            <div class="panel-head">
              <div>
                <h2 class="section-title">版本历史</h2>
                <p class="section-note">每次文件更新都会形成独立版本记录</p>
              </div>
              <ElIcon :size="20"><Document /></ElIcon>
            </div>
            <div v-if="versions.length === 0" class="empty-state compact">
              <p class="empty-title">暂无版本记录</p>
              <p class="empty-hint">上传首个文件版本后会出现在这里</p>
            </div>
            <div v-else class="version-timeline">
              <div v-for="item in versions" :key="item.id || item.version_no" class="version-item">
                <span class="version-chip">v{{ item.version_no }}</span>
                <div class="version-body">
                  <strong>{{ item.file_name || "未命名文件" }}</strong>
                  <span>{{ item.summary_status || "未生成摘要" }} · {{ formatTime(item.created_at) }}</span>
                </div>
                <ElButton
                  v-if="item.id"
                  size="small"
                  text
                  title="下载此版本"
                  @click="downloadVersion(item.id, item.file_name)"
                >
                  <ElIcon><Download /></ElIcon>
                </ElButton>
              </div>
            </div>
          </section>

          <section class="page-card timeline-panel">
            <div class="panel-head">
              <div>
                <h2 class="section-title">流转历史</h2>
                <p class="section-note">记录责任人处理、转交、定稿和归档过程</p>
              </div>
              <ElIcon :size="20"><Clock /></ElIcon>
            </div>
            <div v-if="flows.length === 0" class="empty-state compact">
              <p class="empty-title">暂无流转记录</p>
              <p class="empty-hint">开始处理、转交或归档后会形成记录</p>
            </div>
            <div v-else class="flow-history">
              <div v-for="item in flows" :key="item.id || `${item.action}-${item.created_at}`" class="flow-history-item">
                <div class="flow-icon">
                  <ElIcon><UserFilled /></ElIcon>
                </div>
                <div class="flow-history-body">
                  <strong>{{ flowActionLabel[item.action] ?? item.action }}</strong>
                  <div class="flow-status-row">
                    <span class="status-pill" :class="statusClass[item.from_status]">
                      {{ statusLabel[item.from_status] ?? (item.from_status || "原状态") }}
                    </span>
                    <span>→</span>
                    <span class="status-pill" :class="statusClass[item.to_status]">
                      {{ statusLabel[item.to_status] ?? item.to_status }}
                    </span>
                  </div>
                  <span>{{ formatTime(item.created_at) }}</span>
                </div>
              </div>
            </div>
          </section>
        </main>

        <aside class="detail-side">
          <section class="page-card ai-panel">
            <div class="panel-head">
              <div>
                <h2 class="section-title">AI 摘要与建议</h2>
                <p class="section-note">仅作为辅助建议，正式动作需人工确认</p>
              </div>
            </div>
            <ElButton type="primary" :loading="summaryLoading || summaryPolling" @click="requestSummary">
              {{ summaryPolling ? "正在生成摘要" : "生成摘要" }}
            </ElButton>
            <div v-if="summaryPolling" class="summary-polling-hint">
              <div class="thinking-dots">
                <span class="dot"></span>
                <span class="dot"></span>
                <span class="dot"></span>
              </div>
              <span>OpenClaw 正在分析当前版本</span>
            </div>
            <div v-if="suggestions.length === 0" class="empty-state compact">
              <p class="empty-title">暂无 AI 建议</p>
              <p class="empty-hint">生成摘要后，建议会作为辅助信息显示</p>
            </div>
            <div v-else class="suggestion-list">
              <div v-for="item in suggestions" :key="item.id" class="suggestion-item">
                <div class="suggestion-head">
                  <div class="suggestion-title">
                    {{ item.title || item.suggestion_type }}
                  </div>
                  <span class="status-pill" :class="item.status === 'pending' ? 'status-pending-handover' : 'status-finalized'">
                    {{ suggestionStatusLabel[item.status] ?? item.status }}
                  </span>
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
          </section>

          <section class="page-card danger-panel">
            <div>
              <h2 class="section-title">危险操作</h2>
              <p class="section-note">删除为业务软删除，可按权限恢复。</p>
            </div>
            <ElButton type="danger" plain :loading="actionLoading" @click="deleteDocument">
              <ElIcon><Delete /></ElIcon>
              删除文档
            </ElButton>
          </section>
        </aside>
      </div>
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

    <ElDialog v-model="showTransferDialog" title="转交文档" width="420px">
      <ElForm label-position="top">
        <ElFormItem label="接收人" required>
          <ElSelect
            v-model="transferForm.to_user_id"
            filterable
            placeholder="选择接收人"
          >
            <ElOption
              v-for="item in selectableUsers"
              :key="item.id"
              :label="`${item.display_name} (${item.role})`"
              :value="item.id"
            />
          </ElSelect>
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton @click="showTransferDialog = false">取消</ElButton>
        <ElButton
          type="primary"
          :loading="transferLoading"
          @click="submitTransfer"
        >
          确认转交
        </ElButton>
      </template>
    </ElDialog>
  </AppLayout>
</template>

<style scoped>
.document-detail-shell {
  display: grid;
  gap: 18px;
}

.back-link {
  display: inline-flex;
  align-items: center;
  width: fit-content;
  gap: 8px;
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--dd-primary);
  font-weight: 700;
  cursor: pointer;
}

.document-hero {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 24px;
  padding: 24px;
}

.document-identity {
  display: flex;
  gap: 18px;
  min-width: 0;
}

.file-badge-large {
  width: 72px;
  height: 72px;
  flex: 0 0 auto;
  border-radius: 14px;
  font-size: 14px;
}

.identity-copy {
  min-width: 0;
}

.identity-copy h1 {
  margin: 0;
  color: var(--dd-ink);
  font-size: 28px;
  font-weight: 780;
  letter-spacing: 0;
}

.identity-copy p {
  max-width: 720px;
  margin: 8px 0 0;
  color: var(--dd-muted);
}

.identity-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 14px;
}

.version-chip {
  display: inline-flex;
  align-items: center;
  width: fit-content;
  min-height: 28px;
  padding: 0 10px;
  border-radius: 999px;
  background: #f1f5f9;
  color: var(--dd-ink-2);
  font-size: 12px;
  font-weight: 750;
}

.hero-actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 10px;
  max-width: 440px;
}

.detail-layout {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 360px;
  gap: 18px;
}

.detail-main,
.detail-side {
  display: grid;
  gap: 18px;
  align-content: start;
}

.info-panel,
.timeline-panel,
.ai-panel,
.danger-panel {
  padding: 20px;
}

.panel-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 18px;
}

.fact-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}

.fact-item {
  display: grid;
  gap: 8px;
  min-height: 86px;
  padding: 14px;
  border: 1px solid var(--dd-line-soft);
  border-radius: 8px;
  background: var(--dd-surface-soft);
}

.fact-item span {
  color: var(--dd-muted);
  font-size: 12px;
}

.fact-item strong {
  overflow-wrap: anywhere;
  color: var(--dd-ink);
  font-size: 15px;
}

.summary-text {
  color: var(--dd-ink-2);
  line-height: 1.7;
}

.version-timeline,
.flow-history {
  display: grid;
  gap: 12px;
}

.version-item {
  display: grid;
  grid-template-columns: 58px minmax(0, 1fr) auto;
  gap: 12px;
  align-items: center;
  padding: 14px;
  border: 1px solid var(--dd-line-soft);
  border-radius: 8px;
  background: #fff;
}

.version-body {
  display: grid;
  gap: 4px;
  min-width: 0;
}

.version-body strong {
  overflow: hidden;
  color: var(--dd-ink);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.version-body span {
  color: var(--dd-muted);
  font-size: 12px;
}

.flow-history-item {
  display: grid;
  grid-template-columns: 38px minmax(0, 1fr);
  gap: 12px;
  padding: 14px;
  border: 1px solid var(--dd-line-soft);
  border-radius: 8px;
  background: #fff;
}

.flow-icon {
  display: grid;
  place-items: center;
  width: 34px;
  height: 34px;
  border-radius: 50%;
  background: var(--dd-primary-soft);
  color: var(--dd-primary);
}

.flow-history-body {
  display: grid;
  gap: 8px;
}

.flow-history-body strong {
  color: var(--dd-ink);
}

.flow-history-body > span {
  color: var(--dd-muted);
  font-size: 12px;
}

.flow-status-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  color: var(--dd-subtle);
}

.ai-panel {
  border-top: 4px solid var(--dd-ai);
}

.ai-panel > .el-button {
  width: 100%;
  margin-bottom: 14px;
}

.suggestion-list {
  display: grid;
  gap: 12px;
}

.suggestion-item {
  padding: 14px;
  border: 1px solid #c5e6eb;
  border-radius: 8px;
  background: var(--dd-ai-soft);
}

.suggestion-head {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  align-items: center;
  margin-bottom: 8px;
}

.suggestion-title {
  color: var(--dd-ink);
  font-weight: 750;
}

.suggestion-actions {
  display: flex;
  gap: 8px;
  margin-top: 12px;
}

.danger-panel {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  border-color: #f3c3bd;
  background: #fffafa;
}

.summary-polling-hint {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 16px;
  background: var(--dd-ai-soft);
  border-radius: 8px;
  color: var(--dd-ai);
  font-size: 13px;
  margin-bottom: 12px;
}

.thinking-dots {
  display: flex;
  gap: 4px;
}

.thinking-dots .dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--dd-ai);
  opacity: 0.4;
  animation: thinking-bounce 1.4s infinite ease-in-out both;
}

.thinking-dots .dot:nth-child(1) { animation-delay: 0s; }
.thinking-dots .dot:nth-child(2) { animation-delay: 0.2s; }
.thinking-dots .dot:nth-child(3) { animation-delay: 0.4s; }

@keyframes thinking-bounce {
  0%, 80%, 100% { opacity: 0.25; transform: scale(0.8); }
  40% { opacity: 1; transform: scale(1); }
}

.compact {
  padding: 34px 12px;
}

@media (max-width: 1100px) {
  .document-hero,
  .detail-layout {
    grid-template-columns: 1fr;
  }

  .document-hero {
    display: grid;
  }

  .hero-actions {
    justify-content: flex-start;
    max-width: none;
  }

  .fact-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 720px) {
  .document-identity {
    display: grid;
  }

  .fact-grid {
    grid-template-columns: 1fr;
  }

  .danger-panel {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
