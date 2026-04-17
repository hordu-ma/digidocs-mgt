<script setup lang="ts">
import { onMounted, ref } from "vue";

import AppLayout from "@/components/AppLayout.vue";
import api from "@/api";

const overview = ref({
  document_total: 0,
  status_counts: {} as Record<string, number>,
  handover_pending_count: 0,
  risk_document_count: 0,
});
const recentFlows = ref<any[]>([]);
const riskDocs = ref<any[]>([]);

const statusLabel: Record<string, string> = {
  draft: "草稿",
  in_progress: "处理中",
  pending_handover: "待交接",
  handed_over: "已交接",
  finalized: "定稿",
  archived: "已归档",
};

const actionLabel: Record<string, string> = {
  transfer: "移交",
  accept_transfer: "接收移交",
  finalize: "定稿",
  archive: "归档",
  unarchive: "取消归档",
  mark_in_progress: "标记处理中",
  create: "创建",
};

onMounted(async () => {
  const [overviewRes, flowsRes, riskRes] = await Promise.all([
    api.get("/dashboard/overview"),
    api.get("/dashboard/recent-flows"),
    api.get("/dashboard/risk-documents"),
  ]);
  overview.value = overviewRes.data?.data ?? overview.value;
  recentFlows.value = flowsRes.data?.data ?? [];
  riskDocs.value = riskRes.data?.data ?? [];
});
</script>

<template>
  <AppLayout>
    <div class="page-shell">
      <div class="page-header">
        <div>
          <h1>负责人总览</h1>
          <p>面向课题进度、文档状态、交接风险和 AI 建议的管理面板。</p>
        </div>
      </div>

      <section class="kpi-grid">
        <ElCard class="page-card kpi-card">
          <div class="kpi-label">文档总量</div>
          <div class="kpi-value">{{ overview.document_total }}</div>
        </ElCard>
        <ElCard class="page-card kpi-card">
          <div class="kpi-label">处理中</div>
          <div class="kpi-value">
            {{ overview.status_counts?.in_progress ?? 0 }}
          </div>
        </ElCard>
        <ElCard class="page-card kpi-card">
          <div class="kpi-label">待交接</div>
          <div class="kpi-value">{{ overview.handover_pending_count }}</div>
        </ElCard>
        <ElCard class="page-card kpi-card">
          <div class="kpi-label">风险文档</div>
          <div class="kpi-value">{{ overview.risk_document_count }}</div>
        </ElCard>
      </section>

      <div class="dashboard-grid">
        <ElCard class="page-card">
          <template #header>近期流转</template>
          <ElTable :data="recentFlows" style="width: 100%">
            <template #empty>
              <div style="padding: 32px 0; color: #909399; font-size: 13px">
                暂无流转记录
              </div>
            </template>
            <ElTableColumn prop="title" label="文档" />
            <ElTableColumn label="操作">
              <template #default="{ row }">
                {{ actionLabel[row.action] ?? row.action }}
              </template>
            </ElTableColumn>
            <ElTableColumn label="状态变更">
              <template #default="{ row }">
                <ElTag
                  >{{ statusLabel[row.from_status] ?? row.from_status }} →
                  {{ statusLabel[row.to_status] ?? row.to_status }}</ElTag
                >
              </template>
            </ElTableColumn>
            <ElTableColumn prop="created_at" label="时间" />
          </ElTable>
        </ElCard>

        <ElCard class="page-card">
          <template #header>风险提示</template>
          <div class="risk-list">
            <div
              v-for="item in riskDocs"
              :key="item.document_id"
              class="risk-item"
            >
              <div class="risk-title">{{ item.title }}</div>
              <div class="risk-message">{{ item.risk_message }}</div>
            </div>
            <div v-if="riskDocs.length === 0" class="risk-item">
              <div class="risk-message">暂无风险文档</div>
            </div>
          </div>
        </ElCard>
      </div>
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

.dashboard-grid {
  display: grid;
  grid-template-columns: 1.4fr 1fr;
  gap: 18px;
  margin-top: 18px;
}

.risk-list {
  display: grid;
  gap: 12px;
}

.risk-item {
  padding: 14px;
  border-radius: 14px;
  background: #f7f9fc;
}

.risk-title {
  font-weight: 600;
}

.risk-message {
  color: #61748d;
  margin-top: 6px;
}

@media (max-width: 900px) {
  .dashboard-grid {
    grid-template-columns: 1fr;
  }
}
</style>
