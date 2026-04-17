<script setup lang="ts">
import { ElMessage } from "element-plus";
import { Document as DocumentIcon } from "@element-plus/icons-vue";
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

const handovers = ref<HandoverDetail[]>([]);
const users = ref<UserOption[]>([]);
const projects = ref<ProjectOption[]>([]);
const projectDocuments = ref<DocumentOption[]>([]);
const showDialog = ref(false);
const formLoading = ref(false);
const referenceLoading = ref(false);
const detailLoading = ref(false);
const actionLoading = ref(false);
const showDetailDialog = ref(false);
const activeHandover = ref<HandoverDetail | null>(null);
const editableItems = ref<EditableHandoverLine[]>([]);

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

const canEditItems = computed(() => activeHandover.value?.status === "generated");
const selectedItemCount = computed(
  () => editableItems.value.filter((item) => item.selected).length,
);

function userLabel(userID?: string) {
  const user = users.value.find((item) => item.id === userID);
  return user ? user.display_name : userID || "-";
}

function documentTitle(documentID: string) {
  const document = projectDocuments.value.find((item) => item.id === documentID);
  return document ? document.title : documentID;
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
  try {
    const res = await api.get(`/handovers/${handoverID}`);
    const detail = (res.data?.data ?? null) as HandoverDetail | null;
    if (!detail) {
      ElMessage.error("交接单详情不存在");
      return;
    }
    activeHandover.value = detail;
    await loadProjectDocuments(detail.project_id);
    buildEditableItems(detail, projectDocuments.value);
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

async function applyAction(action: "confirm" | "complete" | "cancel") {
  if (!activeHandover.value) {
    return;
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
          <h1>毕业交接</h1>
          <p>管理课题组人员毕业交接流程，确保文档资产完整移交。</p>
        </div>
        <ElButton type="primary" @click="openCreate">创建交接</ElButton>
      </div>

      <ElCard class="page-card">
        <template #header>交接记录</template>
        <ElTable
          v-if="handovers.length > 0"
          :data="handovers"
          v-loading="detailLoading"
          style="width: 100%"
        >
          <ElTableColumn prop="id" label="交接单 ID" min-width="280" />
          <ElTableColumn label="交接人" min-width="140">
            <template #default="{ row }">
              {{ userLabel(row.target_user_id) }}
            </template>
          </ElTableColumn>
          <ElTableColumn label="接收人" min-width="140">
            <template #default="{ row }">
              {{ userLabel(row.receiver_user_id) }}
            </template>
          </ElTableColumn>
          <ElTableColumn label="状态" width="120">
            <template #default="{ row }">
              <ElTag>{{ statusLabel[row.status] ?? row.status }}</ElTag>
            </template>
          </ElTableColumn>
          <ElTableColumn prop="remark" label="备注" min-width="220" />
          <ElTableColumn label="操作" width="100" fixed="right">
            <template #default="{ row }">
              <ElButton link type="primary" @click="openDetail(row.id)">管理</ElButton>
            </template>
          </ElTableColumn>
        </ElTable>
        <div v-else class="empty-state">
          <el-icon :size="36" color="var(--el-text-color-placeholder)"><DocumentIcon /></el-icon>
          <p class="empty-title">暂无交接记录</p>
          <p class="empty-hint">点击上方「新建交接」创建第一条交接单</p>
        </div>
      </ElCard>

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
        title="交接单管理"
        width="920px"
        destroy-on-close
      >
        <div v-if="activeHandover" class="handover-detail">
          <div class="detail-header">
            <div class="detail-item">
              <span class="detail-label">交接人</span>
              <strong>{{ userLabel(activeHandover.target_user_id) }}</strong>
            </div>
            <div class="detail-item">
              <span class="detail-label">接收人</span>
              <strong>{{ userLabel(activeHandover.receiver_user_id) }}</strong>
            </div>
            <div class="detail-item">
              <span class="detail-label">状态</span>
              <ElTag>{{ statusLabel[activeHandover.status] ?? activeHandover.status }}</ElTag>
            </div>
            <div class="detail-item detail-remark">
              <span class="detail-label">备注</span>
              <strong>{{ activeHandover.remark || "-" }}</strong>
            </div>
          </div>

          <div class="detail-toolbar">
            <span>已选中文档 {{ selectedItemCount }} 项</span>
            <div class="detail-actions">
              <ElButton
                v-if="canEditItems"
                type="primary"
                :loading="actionLoading"
                @click="saveItems"
              >
                保存清单
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

          <ElTable :data="editableItems" style="width: 100%">
            <ElTableColumn label="纳入交接" width="100">
              <template #default="{ row }">
                <ElSwitch v-model="row.selected" :disabled="!canEditItems" />
              </template>
            </ElTableColumn>
            <ElTableColumn label="文档标题" min-width="260">
              <template #default="{ row }">
                {{ row.title || row.document_id }}
              </template>
            </ElTableColumn>
            <ElTableColumn label="当前状态" width="140">
              <template #default="{ row }">
                {{ statusLabel[row.current_status] ?? (row.current_status || "-") }}
              </template>
            </ElTableColumn>
            <ElTableColumn label="备注" min-width="220">
              <template #default="{ row }">
                <ElInput
                  v-model="row.note"
                  :disabled="!canEditItems"
                  placeholder="补充交接说明"
                />
              </template>
            </ElTableColumn>
          </ElTable>
        </div>
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

.handover-detail {
  display: grid;
  gap: 16px;
}

.detail-header {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}

.detail-item {
  padding: 12px 14px;
  border-radius: 14px;
  background: #f7f9fc;
}

.detail-remark {
  grid-column: 1 / -1;
}

.detail-label {
  display: block;
  margin-bottom: 8px;
  color: #61748d;
  font-size: 13px;
}

.detail-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
}

.detail-actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
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

@media (max-width: 900px) {
  .detail-header {
    grid-template-columns: 1fr;
  }

  .detail-toolbar {
    flex-direction: column;
    align-items: flex-start;
  }
}
</style>
