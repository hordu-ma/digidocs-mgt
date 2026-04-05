<script setup lang="ts">
import { ElMessage } from "element-plus";
import { computed, onMounted, ref } from "vue";
import { useRoute } from "vue-router";

import AppLayout from "@/components/AppLayout.vue";
import api from "@/api";

const route = useRoute();
const documentID = route.params.id as string;

const doc = ref<any>(null);
const versions = ref<any[]>([]);
const flows = ref<any[]>([]);
const actionLoading = ref(false);

const statusLabel: Record<string, string> = {
  draft: "草稿",
  in_progress: "处理中",
  pending_handover: "待交接",
  handed_over: "已交接",
  finalized: "定稿",
  archived: "已归档",
};

// Map current status → available flow actions
const flowActions: Record<string, { action: string; label: string; endpoint: string }[]> = {
  draft: [{ action: "mark_in_progress", label: "开始处理", endpoint: "mark-in-progress" }],
  in_progress: [
    { action: "transfer", label: "转交", endpoint: "transfer" },
    { action: "finalize", label: "定稿", endpoint: "finalize" },
  ],
  pending_handover: [
    { action: "accept_transfer", label: "接受转交", endpoint: "accept-transfer" },
  ],
  finalized: [{ action: "archive", label: "归档", endpoint: "archive" }],
  archived: [{ action: "unarchive", label: "取消归档", endpoint: "unarchive" }],
};

const availableActions = computed(() => {
  const status = doc.value?.current_status;
  return status ? (flowActions[status] ?? []) : [];
});

async function loadData() {
  const [docRes, versionsRes, flowsRes] = await Promise.all([
    api.get(`/documents/${documentID}`),
    api.get(`/documents/${documentID}/versions`),
    api.get(`/documents/${documentID}/flows`),
  ]);
  doc.value = docRes.data?.data ?? null;
  versions.value = versionsRes.data?.data ?? [];
  flows.value = flowsRes.data?.data ?? [];
}

async function applyFlowAction(endpoint: string, label: string) {
  actionLoading.value = true;
  try {
    await api.post(`/documents/${documentID}/flow/${endpoint}`);
    ElMessage.success(`${label}成功`);
    await loadData();
  } catch (err: any) {
    const msg = err.response?.data?.error?.message ?? `${label}失败`;
    ElMessage.error(msg);
  } finally {
    actionLoading.value = false;
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
          >{{ act.label }}</ElButton>
        </div>
      </ElCard>

      <ElCard class="page-card">
        <template #header>AI 摘要与建议</template>
        <div class="summary-text">暂无 AI 摘要（OpenClaw 待接入）</div>
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
