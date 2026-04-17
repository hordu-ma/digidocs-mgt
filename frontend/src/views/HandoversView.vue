<script setup lang="ts">
import { ElMessage, ElMessageBox } from "element-plus";
import {
  ArrowRight,
  CircleCheckFilled,
  CircleCloseFilled,
  Connection,
  DocumentAdd,
  DocumentChecked,
  FolderOpened,
  UserFilled,
} from "@element-plus/icons-vue";
import { computed, onMounted, reactive, ref } from "vue";

import AppLayout from "@/components/AppLayout.vue";
import api from "@/api";

type UserOption = {
  id: string;
  display_name: string;
  role: string;
};

type ProjectOption = {
  id: string;
  name: string;
};

type DocumentOption = {
  id: string;
  title: string;
  current_status: string;
};

type HandoverLine = {
  document_id: string;
  selected: boolean;
  note: string;
};

type HandoverDetail = {
  id: string;
  target_user_id: string;
  receiver_user_id: string;
  project_id?: string;
  status: string;
  remark?: string;
  items?: HandoverLine[];
};

type EditableHandoverLine = HandoverLine & {
  title: string;
  current_status: string;
};

type HandoverDataLine = {
  data_asset_id: string;
  display_name: string;
  file_name: string;
  selected: boolean;
  note: string;
};

type EditableDataLine = HandoverDataLine & {
  mime_type?: string;
  file_size?: number;
};

type DataAssetItem = {
  id: string;
  display_name: string;
  file_name: string;
  mime_type?: string;
  file_size?: number;
};

const handovers = ref<HandoverDetail[]>([]);
const users = ref<UserOption[]>([]);
const projects = ref<ProjectOption[]>([]);
const projectDocuments = ref<DocumentOption[]>([]);
const projectDataAssets = ref<DataAssetItem[]>([]);
const showDialog = ref(false);
const formLoading = ref(false);
const referenceLoading = ref(false);
const detailLoading = ref(false);
const actionLoading = ref(false);
const showDetailDialog = ref(false);
const activeHandover = ref<HandoverDetail | null>(null);
const editableItems = ref<EditableHandoverLine[]>([]);
const editableDataItems = ref<EditableDataLine[]>([]);
const activeTab = ref<"documents" | "data-assets">("documents");

const form = reactive({
  target_user_id: "",
  receiver_user_id: "",
  project_id: "",
  remark: "",
});

const statusLabel: Record<string, string> = {
  generated: "已生成",
  pending_confirm: "待确认",
  completed: "已完成",
  cancelled: "已取消",
};

const statusClass: Record<string, string> = {
  generated: "handover-generated",
  pending_confirm: "handover-pending",
  completed: "handover-completed",
  cancelled: "handover-cancelled",
};

const documentStatusLabel: Record<string, string> = {
  draft: "草稿",
  in_progress: "处理中",
  pending_handover: "待交接",
  handed_over: "已交接",
  finalized: "定稿",
  archived: "已归档",
};

const documentStatusClass: Record<string, string> = {
  draft: "status-draft",
  in_progress: "status-in-progress",
  pending_handover: "status-pending-handover",
  handed_over: "status-handed-over",
  finalized: "status-finalized",
  archived: "status-archived",
};

const canEditItems = computed(() => activeHandover.value?.status === "generated");
const selectedItemCount = computed(
  () => editableItems.value.filter((item) => item.selected).length,
);
const unselectedItemCount = computed(() => editableItems.value.length - selectedItemCount.value);
const selectedDataItemCount = computed(
  () => editableDataItems.value.filter((item) => item.selected).length,
);
const unselectedDataItemCount = computed(
  () => editableDataItems.value.length - selectedDataItemCount.value,
);

const handoverMetrics = computed(() => ({
  total: handovers.value.length,
  pending: handovers.value.filter((item) => item.status === "pending_confirm").length,
  generated: handovers.value.filter((item) => item.status === "generated").length,
  completed: handovers.value.filter((item) => item.status === "completed").length,
}));

function userLabel(userID?: string) {
  const user = users.value.find((item) => item.id === userID);
  return user ? user.display_name : userID || "-";
}

function documentTitle(documentID: string) {
  const document = projectDocuments.value.find((item) => item.id === documentID);
  return document ? document.title : documentID;
}

function projectLabel(projectID?: string) {
  const project = projects.value.find((item) => item.id === projectID);
  return project?.name ?? "全部课题";
}

function compactHandoverID(id: string) {
  return id ? `#${id.slice(0, 8)}` : "-";
}

function userInitial(userID?: string) {
  return userLabel(userID).slice(0, 1) || "人";
}

function statusStepState(status: string, step: "generated" | "pending_confirm" | "completed") {
  const order = ["generated", "pending_confirm", "completed"];
  if (status === "cancelled") return "cancelled";
  const currentIndex = order.indexOf(status);
  const stepIndex = order.indexOf(step);
  if (currentIndex > stepIndex) return "done";
  if (currentIndex === stepIndex) return "active";
  return "todo";
}

async function fetchHandovers() {
  const res = await api.get("/handovers");
  handovers.value = res.data?.data ?? [];
}

async function fetchReferenceData() {
  referenceLoading.value = true;
  try {
    const [userRes, teamRes] = await Promise.all([
      api.get("/users"),
      api.get("/team-spaces"),
    ]);
    users.value = userRes.data?.data ?? [];

    const teamSpaces = teamRes.data?.data ?? [];
    const projectResponses = await Promise.all(
      teamSpaces.map((item: { id: string }) =>
        api.get("/projects", { params: { team_space_id: item.id } }),
      ),
    );
    projects.value = projectResponses.flatMap((item) => item.data?.data ?? []);
  } catch (err: any) {
    ElMessage.error(err.response?.data?.message ?? "加载基础选项失败");
  } finally {
    referenceLoading.value = false;
  }
}

async function loadProjectDocuments(projectID?: string) {
  const res = await api.get("/documents", {
    params: {
      page: 1,
      page_size: 100,
      project_id: projectID || undefined,
    },
  });
  projectDocuments.value = res.data?.data ?? [];
}

async function loadProjectDataAssets(projectID?: string) {
  if (!projectID) {
    projectDataAssets.value = [];
    return;
  }
  const res = await api.get("/data-assets", {
    params: { project_id: projectID, page: 1, page_size: 200 },
  });
  projectDataAssets.value = res.data?.data ?? [];
}

async function fetchHandoverDataItems(handoverID: string): Promise<HandoverDataLine[]> {
  try {
    const res = await api.get(`/handovers/${handoverID}/data-items`);
    return res.data?.data ?? [];
  } catch {
    return [];
  }
}

function buildEditableDataItems(dataItems: HandoverDataLine[], assets: DataAssetItem[]) {
  const itemMap = new Map(dataItems.map((i) => [i.data_asset_id, i]));
  const result: EditableDataLine[] = assets.map((a) => ({
    data_asset_id: a.id,
    display_name: a.display_name,
    file_name: a.file_name,
    mime_type: a.mime_type,
    file_size: a.file_size,
    selected: itemMap.get(a.id)?.selected ?? false,
    note: itemMap.get(a.id)?.note ?? "",
  }));
  for (const item of dataItems) {
    if (!result.some((r) => r.data_asset_id === item.data_asset_id)) {
      result.push({
        data_asset_id: item.data_asset_id,
        display_name: item.display_name,
        file_name: item.file_name,
        selected: item.selected,
        note: item.note,
      });
    }
  }
  editableDataItems.value = result;
}

function formatFileSize(bytes?: number): string {
  if (!bytes) return "-";
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
  return `${(bytes / 1024 / 1024 / 1024).toFixed(2)} GB`;
}

function buildEditableItems(detail: HandoverDetail, documents: DocumentOption[]) {
  const itemMap = new Map(detail.items?.map((item) => [item.document_id, item]) ?? []);
  const baseDocuments = documents.map((item) => ({
    document_id: item.id,
    selected: itemMap.get(item.id)?.selected ?? false,
    note: itemMap.get(item.id)?.note ?? "",
    title: item.title,
    current_status: item.current_status,
  }));

  for (const item of detail.items ?? []) {
    if (!baseDocuments.some((doc) => doc.document_id === item.document_id)) {
      baseDocuments.push({
        document_id: item.document_id,
        selected: item.selected,
        note: item.note,
        title: documentTitle(item.document_id),
        current_status: "",
      });
    }
  }

  editableItems.value = baseDocuments;
}

function openCreate() {
  form.target_user_id = "";
  form.receiver_user_id = "";
  form.project_id = "";
  form.remark = "";
  showDialog.value = true;
}

async function submitCreate() {
  if (!form.target_user_id || !form.receiver_user_id) {
    ElMessage.warning("请选择交接人和接收人");
    return;
  }
  if (form.target_user_id === form.receiver_user_id) {
    ElMessage.warning("交接人和接收人不能相同");
    return;
  }

  formLoading.value = true;
  try {
    await api.post("/handovers", form);
    ElMessage.success("交接单已创建");
    showDialog.value = false;
    await fetchHandovers();
  } catch (err: any) {
    const msg = err.response?.data?.message ?? "创建失败";
    ElMessage.error(msg);
  } finally {
    formLoading.value = false;
  }
}

async function openDetail(handoverID: string) {
  detailLoading.value = true;
  activeTab.value = "documents";
  try {
    const res = await api.get(`/handovers/${handoverID}`);
    const detail = (res.data?.data ?? null) as HandoverDetail | null;
    if (!detail) {
      ElMessage.error("交接单详情不存在");
      return;
    }
    activeHandover.value = detail;
    const [, dataItems] = await Promise.all([
      loadProjectDocuments(detail.project_id),
      fetchHandoverDataItems(handoverID),
      loadProjectDataAssets(detail.project_id),
    ]);
    buildEditableItems(detail, projectDocuments.value);
    buildEditableDataItems(dataItems, projectDataAssets.value);
    showDetailDialog.value = true;
  } catch (err: any) {
    ElMessage.error(err.response?.data?.message ?? "加载交接单详情失败");
  } finally {
    detailLoading.value = false;
  }
}

async function saveItems() {
  if (!activeHandover.value) {
    return;
  }
  actionLoading.value = true;
  try {
    await api.patch(`/handovers/${activeHandover.value.id}/items`, {
      items: editableItems.value.map((item) => ({
        document_id: item.document_id,
        selected: item.selected,
        note: item.note,
      })),
    });
    ElMessage.success("交接清单已更新");
    await openDetail(activeHandover.value.id);
    await fetchHandovers();
  } catch (err: any) {
    ElMessage.error(err.response?.data?.message ?? "更新交接清单失败");
  } finally {
    actionLoading.value = false;
  }
}

async function saveDataItems() {
  if (!activeHandover.value) {
    return;
  }
  actionLoading.value = true;
  try {
    await api.put(`/handovers/${activeHandover.value.id}/data-items`, {
      items: editableDataItems.value.map((item) => ({
        data_asset_id: item.data_asset_id,
        selected: item.selected,
        note: item.note,
      })),
    });
    ElMessage.success("数据资产清单已更新");
  } catch (err: any) {
    ElMessage.error(err.response?.data?.message ?? "更新数据资产清单失败");
  } finally {
    actionLoading.value = false;
  }
}

async function applyAction(action: "confirm" | "complete" | "cancel") {
  if (!activeHandover.value) {
    return;
  }
  if (action === "cancel") {
    try {
      await ElMessageBox.confirm(
        "取消后该交接单将不再继续推进，确认取消吗？",
        "取消交接",
        {
          confirmButtonText: "确认取消",
          cancelButtonText: "返回",
          type: "warning",
        },
      );
    } catch {
      return;
    }
  }
  actionLoading.value = true;
  try {
    await api.post(`/handovers/${activeHandover.value.id}/${action}`, {});
    ElMessage.success(
      action === "confirm"
        ? "交接单已确认"
        : action === "complete"
          ? "交接单已完成"
          : "交接单已取消",
    );
    await openDetail(activeHandover.value.id);
    await fetchHandovers();
  } catch (err: any) {
    ElMessage.error(err.response?.data?.message ?? "交接动作执行失败");
  } finally {
    actionLoading.value = false;
  }
}

onMounted(async () => {
  await Promise.all([fetchHandovers(), fetchReferenceData()]);
});
</script>

<template>
  <AppLayout>
    <div class="page-shell">
      <div class="page-header">
        <div>
          <div class="page-eyebrow">交接工作台</div>
          <h1>工作交接</h1>
          <p>围绕成员离组、资料清单和接收确认推进交接闭环。</p>
        </div>
        <ElButton type="primary" @click="openCreate">
          <ElIcon><DocumentAdd /></ElIcon>
          创建交接
        </ElButton>
      </div>

      <section class="handover-summary page-card">
        <div class="handover-summary-copy">
          <h2 class="section-title">交接任务概览</h2>
          <p class="section-note">重点关注待确认和刚生成的交接单，避免资料交接断档。</p>
        </div>
        <div class="handover-metrics">
          <div class="handover-metric">
            <span>交接单</span>
            <strong>{{ handoverMetrics.total }}</strong>
          </div>
          <div class="handover-metric warning">
            <span>待确认</span>
            <strong>{{ handoverMetrics.pending }}</strong>
          </div>
          <div class="handover-metric primary">
            <span>已生成</span>
            <strong>{{ handoverMetrics.generated }}</strong>
          </div>
          <div class="handover-metric success">
            <span>已完成</span>
            <strong>{{ handoverMetrics.completed }}</strong>
          </div>
        </div>
      </section>

      <section class="handover-board" v-loading="detailLoading">
        <div v-if="handovers.length === 0" class="page-card empty-state">
          <ElIcon :size="36"><Connection /></ElIcon>
          <p class="empty-title">暂无交接任务</p>
          <p class="empty-hint">创建交接单后，可在这里确认资料范围并推进接收。</p>
        </div>
        <button
          v-for="item in handovers"
          v-else
          :key="item.id"
          class="handover-card page-card"
          type="button"
          @click="openDetail(item.id)"
        >
          <span class="handover-status" :class="statusClass[item.status]">
            {{ statusLabel[item.status] ?? item.status }}
          </span>
          <div class="handover-people">
            <span class="handover-person">
              <span class="person-avatar">{{ userInitial(item.target_user_id) }}</span>
              {{ userLabel(item.target_user_id) }}
            </span>
            <ElIcon class="handover-arrow"><ArrowRight /></ElIcon>
            <span class="handover-person">
              <span class="person-avatar">{{ userInitial(item.receiver_user_id) }}</span>
              {{ userLabel(item.receiver_user_id) }}
            </span>
          </div>
          <div class="handover-meta">
            <span>
              <ElIcon><FolderOpened /></ElIcon>
              {{ projectLabel(item.project_id) }}
            </span>
            <span>{{ compactHandoverID(item.id) }}</span>
          </div>
          <p class="handover-remark">{{ item.remark || "暂无备注" }}</p>
          <span class="handover-action">管理交接</span>
        </button>
      </section>

      <ElDialog v-model="showDialog" title="创建交接单" width="520px">
        <ElForm label-position="top" v-loading="referenceLoading">
          <ElFormItem label="交接人" required>
            <ElSelect
              v-model="form.target_user_id"
              filterable
              placeholder="选择即将离开的成员"
            >
              <ElOption
                v-for="item in users"
                :key="item.id"
                :label="`${item.display_name} (${item.role})`"
                :value="item.id"
              />
            </ElSelect>
          </ElFormItem>
          <ElFormItem label="接收人" required>
            <ElSelect
              v-model="form.receiver_user_id"
              filterable
              placeholder="选择接手成员"
            >
              <ElOption
                v-for="item in users"
                :key="item.id"
                :label="`${item.display_name} (${item.role})`"
                :value="item.id"
              />
            </ElSelect>
          </ElFormItem>
          <ElFormItem label="课题">
            <ElSelect
              v-model="form.project_id"
              clearable
              filterable
              placeholder="限定交接范围（可选）"
            >
              <ElOption
                v-for="item in projects"
                :key="item.id"
                :label="item.name"
                :value="item.id"
              />
            </ElSelect>
          </ElFormItem>
          <ElFormItem label="备注">
            <ElInput v-model="form.remark" type="textarea" :rows="2" />
          </ElFormItem>
        </ElForm>
        <template #footer>
          <ElButton @click="showDialog = false">取消</ElButton>
          <ElButton type="primary" :loading="formLoading" @click="submitCreate">
            确认创建
          </ElButton>
        </template>
      </ElDialog>

      <ElDialog
        v-model="showDetailDialog"
        title="交接工作台"
        width="1040px"
        destroy-on-close
      >
        <div v-if="activeHandover" class="handover-detail">
          <div class="detail-hero">
            <div class="detail-people-flow">
              <span class="detail-person">
                <span class="person-avatar large">{{ userInitial(activeHandover.target_user_id) }}</span>
                <span>
                  <small>交接人</small>
                  <strong>{{ userLabel(activeHandover.target_user_id) }}</strong>
                </span>
              </span>
              <ElIcon class="detail-arrow"><ArrowRight /></ElIcon>
              <span class="detail-person">
                <span class="person-avatar large">{{ userInitial(activeHandover.receiver_user_id) }}</span>
                <span>
                  <small>接收人</small>
                  <strong>{{ userLabel(activeHandover.receiver_user_id) }}</strong>
                </span>
              </span>
            </div>
            <span class="handover-status" :class="statusClass[activeHandover.status]">
              {{ statusLabel[activeHandover.status] ?? activeHandover.status }}
            </span>
          </div>

          <div class="handover-steps" :class="{ cancelled: activeHandover.status === 'cancelled' }">
            <div class="handover-step" :class="statusStepState(activeHandover.status, 'generated')">
              <ElIcon><DocumentChecked /></ElIcon>
              <span>生成清单</span>
            </div>
            <div class="handover-step" :class="statusStepState(activeHandover.status, 'pending_confirm')">
              <ElIcon><UserFilled /></ElIcon>
              <span>接收确认</span>
            </div>
            <div class="handover-step" :class="statusStepState(activeHandover.status, 'completed')">
              <ElIcon><CircleCheckFilled /></ElIcon>
              <span>完成交接</span>
            </div>
            <div v-if="activeHandover.status === 'cancelled'" class="handover-step cancelled active">
              <ElIcon><CircleCloseFilled /></ElIcon>
              <span>已取消</span>
            </div>
          </div>

          <div class="detail-toolbar">
            <div class="detail-counts">
              <span v-if="activeTab === 'documents'">文档纳入 <strong>{{ selectedItemCount }}</strong> 项</span>
              <span v-if="activeTab === 'documents'">暂不纳入 <strong>{{ unselectedItemCount }}</strong> 项</span>
              <span v-if="activeTab === 'data-assets'">资产纳入 <strong>{{ selectedDataItemCount }}</strong> 项</span>
              <span v-if="activeTab === 'data-assets'">暂不纳入 <strong>{{ unselectedDataItemCount }}</strong> 项</span>
            </div>
            <div class="detail-actions">
              <ElButton
                v-if="canEditItems && activeTab === 'documents'"
                type="primary"
                :loading="actionLoading"
                @click="saveItems"
              >
                保存文档清单
              </ElButton>
              <ElButton
                v-if="canEditItems && activeTab === 'data-assets'"
                type="primary"
                :loading="actionLoading"
                @click="saveDataItems"
              >
                保存数据资产
              </ElButton>
              <ElButton
                v-if="activeHandover.status === 'generated'"
                :loading="actionLoading"
                @click="applyAction('confirm')"
              >
                确认交接
              </ElButton>
              <ElButton
                v-if="activeHandover.status === 'pending_confirm'"
                type="success"
                :loading="actionLoading"
                @click="applyAction('complete')"
              >
                完成交接
              </ElButton>
              <ElButton
                v-if="activeHandover.status !== 'completed' && activeHandover.status !== 'cancelled'"
                type="danger"
                plain
                :loading="actionLoading"
                @click="applyAction('cancel')"
              >
                取消交接
              </ElButton>
            </div>
          </div>

          <div class="detail-tabs">
            <button
              class="detail-tab"
              :class="{ active: activeTab === 'documents' }"
              @click="activeTab = 'documents'"
            >
              文档清单
              <span class="tab-count">{{ selectedItemCount }}/{{ editableItems.length }}</span>
            </button>
            <button
              class="detail-tab"
              :class="{ active: activeTab === 'data-assets' }"
              @click="activeTab = 'data-assets'"
            >
              数据资产
              <span class="tab-count">{{ selectedDataItemCount }}/{{ editableDataItems.length }}</span>
            </button>
          </div>

          <div v-show="activeTab === 'documents'" class="handover-items">
            <div v-if="editableItems.length === 0" class="items-empty">
              <p>该课题下暂无文档，创建文档后可添加到交接清单。</p>
            </div>
            <div
              v-for="row in editableItems"
              :key="row.document_id"
              class="handover-item"
              :class="{ selected: row.selected }"
            >
              <div class="item-main">
                <ElSwitch v-model="row.selected" :disabled="!canEditItems" />
                <div class="item-copy">
                  <strong>{{ row.title || row.document_id }}</strong>
                  <span class="status-pill" :class="documentStatusClass[row.current_status]">
                    {{ documentStatusLabel[row.current_status] ?? (row.current_status || "未知状态") }}
                  </span>
                </div>
              </div>
              <ElInput
                v-model="row.note"
                :disabled="!canEditItems"
                placeholder="补充交接说明"
              />
            </div>
          </div>

          <div v-show="activeTab === 'data-assets'" class="handover-items">
            <div v-if="editableDataItems.length === 0" class="items-empty">
              <p>该课题下暂无数据资产，上传数据文件后可添加到交接清单。</p>
            </div>
            <div
              v-for="row in editableDataItems"
              :key="row.data_asset_id"
              class="handover-item"
              :class="{ selected: row.selected }"
            >
              <div class="item-main">
                <ElSwitch v-model="row.selected" :disabled="!canEditItems" />
                <div class="item-copy">
                  <strong>{{ row.display_name || row.file_name }}</strong>
                  <span class="item-meta-row">
                    <span class="item-file-name">{{ row.file_name }}</span>
                    <span class="item-size">{{ formatFileSize(row.file_size) }}</span>
                  </span>
                </div>
              </div>
              <ElInput
                v-model="row.note"
                :disabled="!canEditItems"
                placeholder="补充交接说明"
              />
            </div>
          </div>
        </div>
      </ElDialog>
    </div>
  </AppLayout>
</template>

<style scoped>
.handover-summary {
  display: grid;
  grid-template-columns: minmax(260px, 1fr) minmax(0, 1.6fr);
  gap: 20px;
  align-items: center;
  padding: 20px;
  margin-bottom: 18px;
}

.handover-metrics {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}

.handover-metric {
  display: grid;
  gap: 6px;
  min-height: 82px;
  padding: 14px;
  border: 1px solid var(--dd-line-soft);
  border-radius: 8px;
  background: var(--dd-surface-soft);
}

.handover-metric span {
  color: var(--dd-muted);
  font-size: 12px;
  font-weight: 700;
}

.handover-metric strong {
  color: var(--dd-ink);
  font-size: 28px;
}

.handover-metric.primary {
  border-color: #c8ddf2;
  background: var(--dd-primary-soft);
}

.handover-metric.warning {
  border-color: #f1d18b;
  background: var(--dd-warning-soft);
}

.handover-metric.success {
  border-color: #bae5d0;
  background: var(--dd-success-soft);
}

.handover-board {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 16px;
}

.handover-card {
  position: relative;
  display: grid;
  gap: 14px;
  min-height: 214px;
  padding: 18px;
  border: 1px solid var(--dd-line);
  color: var(--dd-ink-2);
  text-align: left;
  cursor: pointer;
  transition:
    border-color 0.16s ease,
    box-shadow 0.16s ease,
    transform 0.16s ease;
}

.handover-card:hover {
  border-color: #b8d1e8;
  box-shadow: 0 14px 32px rgba(15, 23, 42, 0.08);
  transform: translateY(-1px);
}

.handover-status {
  display: inline-flex;
  align-items: center;
  width: fit-content;
  min-height: 28px;
  padding: 0 10px;
  border: 1px solid transparent;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 750;
}

.handover-generated {
  border-color: #c8ddf2;
  background: var(--dd-primary-soft);
  color: var(--dd-primary-strong);
}

.handover-pending {
  border-color: #f1d18b;
  background: var(--dd-warning-soft);
  color: var(--dd-warning);
}

.handover-completed {
  border-color: #bae5d0;
  background: var(--dd-success-soft);
  color: var(--dd-success);
}

.handover-cancelled {
  border-color: #d9e2ec;
  background: #f1f5f9;
  color: #64748b;
}

.handover-people,
.detail-people-flow {
  display: flex;
  align-items: center;
  gap: 10px;
}

.handover-person,
.detail-person {
  display: inline-flex;
  align-items: center;
  min-width: 0;
  gap: 8px;
  color: var(--dd-ink);
  font-weight: 750;
}

.handover-arrow,
.detail-arrow {
  flex: 0 0 auto;
  color: var(--dd-muted);
}

.handover-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  color: var(--dd-muted);
  font-size: 12px;
}

.handover-meta span {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}

.handover-remark {
  min-height: 40px;
  margin: 0;
  color: var(--dd-muted);
  font-size: 13px;
  line-height: 1.6;
}

.handover-action {
  color: var(--dd-primary);
  font-size: 13px;
  font-weight: 750;
}

.person-avatar.large {
  width: 36px;
  height: 36px;
  font-size: 14px;
}

.handover-detail {
  display: grid;
  gap: 18px;
}

.detail-hero {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 16px;
  border: 1px solid var(--dd-line);
  border-radius: 10px;
  background: var(--dd-surface-soft);
}

.detail-person small {
  display: block;
  margin-bottom: 2px;
  color: var(--dd-muted);
  font-size: 12px;
  font-weight: 650;
}

.handover-steps {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
}

.handover-steps.cancelled {
  grid-template-columns: repeat(4, minmax(0, 1fr));
}

.handover-step {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  min-height: 44px;
  border: 1px solid var(--dd-line);
  border-radius: 8px;
  background: #fff;
  color: var(--dd-muted);
  font-weight: 750;
}

.handover-step.done {
  border-color: #bae5d0;
  background: var(--dd-success-soft);
  color: var(--dd-success);
}

.handover-step.active {
  border-color: #c8ddf2;
  background: var(--dd-primary-soft);
  color: var(--dd-primary-strong);
}

.handover-step.cancelled.active {
  border-color: #f3c3bd;
  background: var(--dd-danger-soft);
  color: var(--dd-danger);
}

.detail-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
}

.detail-counts {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  color: var(--dd-muted);
  font-size: 13px;
}

.detail-counts span {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  min-height: 30px;
  padding: 0 10px;
  border-radius: 999px;
  background: var(--dd-surface-soft);
}

.detail-counts strong {
  color: var(--dd-ink);
}

.detail-actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.handover-items {
  display: grid;
  gap: 10px;
}

.handover-item {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(220px, 0.65fr);
  gap: 12px;
  align-items: center;
  padding: 14px;
  border: 1px solid var(--dd-line);
  border-radius: 10px;
  background: #fff;
}

.handover-item.selected {
  border-color: #c8ddf2;
  background: #fbfdff;
}

.item-main {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

.item-copy {
  display: grid;
  gap: 7px;
  min-width: 0;
}

.item-copy strong {
  overflow: hidden;
  color: var(--dd-ink);
  text-overflow: ellipsis;
  white-space: nowrap;
}

@media (max-width: 900px) {
  .handover-summary,
  .detail-hero,
  .handover-item {
    grid-template-columns: 1fr;
  }

  .handover-summary,
  .detail-hero {
    display: grid;
  }

  .handover-metrics,
  .handover-steps,
  .handover-steps.cancelled {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .detail-toolbar {
    flex-direction: column;
    align-items: flex-start;
  }
}

@media (max-width: 640px) {
  .handover-board,
  .handover-metrics,
  .handover-steps,
  .handover-steps.cancelled {
    grid-template-columns: 1fr;
  }

  .handover-people,
  .detail-people-flow {
    align-items: flex-start;
    flex-direction: column;
  }
}

.detail-tabs {
  display: flex;
  gap: 4px;
  border-bottom: 2px solid var(--dd-line);
}

.detail-tab {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  border: none;
  border-bottom: 2px solid transparent;
  background: none;
  color: var(--dd-muted);
  font-size: 14px;
  font-weight: 700;
  cursor: pointer;
  margin-bottom: -2px;
  transition: color 0.15s ease, border-color 0.15s ease;
}

.detail-tab:hover {
  color: var(--dd-ink);
}

.detail-tab.active {
  border-bottom-color: var(--dd-primary);
  color: var(--dd-primary-strong);
}

.tab-count {
  display: inline-flex;
  align-items: center;
  min-width: 32px;
  padding: 2px 6px;
  border-radius: 999px;
  background: var(--dd-surface-soft);
  color: var(--dd-muted);
  font-size: 11px;
  font-weight: 700;
}

.detail-tab.active .tab-count {
  background: var(--dd-primary-soft);
  color: var(--dd-primary-strong);
}

.items-empty {
  padding: 24px 16px;
  color: var(--dd-muted);
  font-size: 13px;
  text-align: center;
}

.item-meta-row {
  display: flex;
  gap: 12px;
  align-items: center;
  flex-wrap: wrap;
}

.item-file-name {
  overflow: hidden;
  color: var(--dd-muted);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.item-size {
  flex-shrink: 0;
  color: var(--dd-muted);
  font-size: 11px;
}</style>
