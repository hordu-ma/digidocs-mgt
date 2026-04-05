<script setup lang="ts">
import {
  ElCard,
  ElDescriptions,
  ElDescriptionsItem,
  ElTable,
  ElTableColumn,
  ElTag,
} from "element-plus";
import { onMounted, ref } from "vue";
import { useRoute } from "vue-router";

import AppLayout from "@/components/AppLayout.vue";
import api from "@/api";

const route = useRoute();
const documentID = route.params.id as string;

const doc = ref<any>(null);
const versions = ref<any[]>([]);
const flows = ref<any[]>([]);

const statusLabel: Record<string, string> = {
  draft: "草稿",
  in_progress: "处理中",
  pending_handover: "待交接",
  handed_over: "已交接",
  finalized: "定稿",
  archived: "已归档",
};

onMounted(async () => {
  const [docRes, versionsRes, flowsRes] = await Promise.all([
    api.get(`/documents/${documentID}`),
    api.get(`/documents/${documentID}/versions`),
    api.get(`/documents/${documentID}/flows`),
  ]);
  doc.value = docRes.data?.data ?? null;
  versions.value = versionsRes.data?.data ?? [];
  flows.value = flowsRes.data?.data ?? [];
});
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
