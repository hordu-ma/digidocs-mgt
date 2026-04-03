<script setup lang="ts">
import { ElCard, ElTable, ElTableColumn, ElTag } from "element-plus";

import AppLayout from "@/components/AppLayout.vue";

const recentFlows = [
  { title: "课题申报书", from: "张三", to: "李四", status: "待交接" },
  { title: "阶段汇报PPT", from: "王五", to: "赵六", status: "处理中" },
];

const riskDocs = [
  { title: "实验记录-2025秋", message: "超过 30 天未更新" },
  { title: "中期报告", message: "尚未归档且责任人未确认" },
];
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
          <div class="kpi-value">128</div>
        </ElCard>
        <ElCard class="page-card kpi-card">
          <div class="kpi-label">处理中</div>
          <div class="kpi-value">32</div>
        </ElCard>
        <ElCard class="page-card kpi-card">
          <div class="kpi-label">待交接</div>
          <div class="kpi-value">6</div>
        </ElCard>
        <ElCard class="page-card kpi-card">
          <div class="kpi-label">风险文档</div>
          <div class="kpi-value">7</div>
        </ElCard>
      </section>

      <div class="dashboard-grid">
        <ElCard class="page-card">
          <template #header>近期流转</template>
          <ElTable :data="recentFlows" style="width: 100%">
            <ElTableColumn prop="title" label="文档" />
            <ElTableColumn prop="from" label="来源" />
            <ElTableColumn prop="to" label="目标" />
            <ElTableColumn label="状态">
              <template #default="{ row }">
                <ElTag>{{ row.status }}</ElTag>
              </template>
            </ElTableColumn>
          </ElTable>
        </ElCard>

        <ElCard class="page-card">
          <template #header>风险提示</template>
          <div class="risk-list">
            <div v-for="item in riskDocs" :key="item.title" class="risk-item">
              <div class="risk-title">{{ item.title }}</div>
              <div class="risk-message">{{ item.message }}</div>
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
