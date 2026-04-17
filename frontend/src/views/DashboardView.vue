<script setup lang="ts">
import {
  ArrowRight,
  Connection,
  DocumentChecked,
  Finished,
  Timer,
  WarningFilled,
} from "@element-plus/icons-vue";
import { computed, onMounted, ref } from "vue";
import { useRouter } from "vue-router";

import AppLayout from "@/components/AppLayout.vue";
import api from "@/api";

const router = useRouter();

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

const statusClass: Record<string, string> = {
  draft: "status-draft",
  in_progress: "status-in-progress",
  pending_handover: "status-pending-handover",
  handed_over: "status-handed-over",
  finalized: "status-finalized",
  archived: "status-archived",
};

const summaryItems = computed(() => [
  {
    label: "文档总量",
    value: overview.value.document_total,
    hint: "纳入团队空间的文档资产",
    icon: DocumentChecked,
    tone: "primary",
  },
  {
    label: "处理中",
    value: overview.value.status_counts?.in_progress ?? 0,
    hint: "仍在责任人处理中的文档",
    icon: Timer,
    tone: "blue",
  },
  {
    label: "待交接",
    value: overview.value.handover_pending_count,
    hint: "需要负责人继续推进",
    icon: Connection,
    tone: "amber",
  },
  {
    label: "风险文档",
    value: overview.value.risk_document_count,
    hint: "长期未更新或交接异常",
    icon: WarningFilled,
    tone: "red",
  },
]);

const statusEntries = computed(() =>
  Object.entries(statusLabel).map(([key, label]) => ({
    key,
    label,
    value: overview.value.status_counts?.[key] ?? 0,
    className: statusClass[key],
  })),
);

const activeDocumentCount = computed(
  () =>
    (overview.value.status_counts?.draft ?? 0) +
    (overview.value.status_counts?.in_progress ?? 0) +
    (overview.value.status_counts?.pending_handover ?? 0),
);

const focusCards = computed(() => [
  {
    title: "待负责人推进",
    count: overview.value.handover_pending_count,
    caption: "需要继续确认接收与交接闭环",
    actionLabel: "打开交接工作台",
    action: () => router.push("/handovers"),
  },
  {
    title: "风险文档",
    count: overview.value.risk_document_count,
    caption: "优先处理长期未更新或交接异常文档",
    actionLabel: "查看文档资产库",
    action: () => router.push("/documents"),
  },
  {
    title: "处理中资产",
    count: activeDocumentCount.value,
    caption: "仍处于草稿、处理中和待交接状态",
    actionLabel: "进入文档工作台",
    action: () => router.push("/documents"),
  },
]);

function formatTime(value?: string) {
  if (!value) return "-";
  return new Date(value).toLocaleString("zh-CN", {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
}

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
          <div class="page-eyebrow">负责人工作台</div>
          <h1>负责人总览</h1>
          <p>优先呈现需要负责人关注的文档状态、近期流转和交接风险。</p>
        </div>
      </div>

      <section class="action-card-grid focus-grid">
        <button
          v-for="item in focusCards"
          :key="item.title"
          class="action-card focus-card"
          type="button"
          @click="item.action()"
        >
          <div class="action-card-head">
            <div>
              <div class="focus-label">{{ item.title }}</div>
              <div class="focus-value">{{ item.count }}</div>
            </div>
            <span class="action-card-icon">
              <ElIcon :size="18"><ArrowRight /></ElIcon>
            </span>
          </div>
          <p class="focus-note">{{ item.caption }}</p>
          <span class="focus-action">{{ item.actionLabel }}</span>
        </button>
      </section>

      <section class="command-strip page-card">
        <div class="command-copy">
          <div class="section-title">今日关注</div>
          <p class="section-note">
            当前有 {{ activeDocumentCount }} 份文档仍处于草稿、处理中或待交接状态。
          </p>
        </div>
        <div class="status-distribution">
          <div
            v-for="item in statusEntries"
            :key="item.key"
            class="status-segment"
          >
            <span class="status-pill" :class="item.className">{{ item.label }}</span>
            <strong>{{ item.value }}</strong>
          </div>
        </div>
      </section>

      <section class="kpi-grid dashboard-kpis">
        <div
          v-for="item in summaryItems"
          :key="item.label"
          class="metric-tile page-card"
          :class="`metric-${item.tone}`"
        >
          <div class="metric-icon">
            <ElIcon :size="20"><component :is="item.icon" /></ElIcon>
          </div>
          <div>
            <div class="kpi-label">{{ item.label }}</div>
            <div class="kpi-value">{{ item.value }}</div>
            <div class="metric-hint">{{ item.hint }}</div>
          </div>
        </div>
      </section>

      <div class="dashboard-grid">
        <section class="page-card activity-panel">
          <div class="panel-head">
            <div>
              <h2 class="section-title">近期流转</h2>
              <p class="section-note">跟踪文档责任和状态变化</p>
            </div>
            <ElIcon :size="20"><Finished /></ElIcon>
          </div>
          <div v-if="recentFlows.length === 0" class="empty-state compact">
            <p class="empty-title">暂无流转记录</p>
            <p class="empty-hint">文档转交、定稿或归档后会出现在这里</p>
          </div>
          <div v-else class="flow-timeline">
            <div v-for="row in recentFlows" :key="row.id || `${row.title}-${row.created_at}`" class="flow-item">
              <div class="flow-dot"></div>
              <div class="flow-body">
                <div class="flow-title">{{ row.title }}</div>
                <div class="flow-meta">
                  <span>{{ actionLabel[row.action] ?? row.action }}</span>
                  <span>{{ formatTime(row.created_at) }}</span>
                </div>
                <div class="flow-status">
                  <span class="status-pill" :class="statusClass[row.from_status]">
                    {{ statusLabel[row.from_status] ?? row.from_status ?? "原状态" }}
                  </span>
                  <span class="flow-arrow">→</span>
                  <span class="status-pill" :class="statusClass[row.to_status]">
                    {{ statusLabel[row.to_status] ?? row.to_status }}
                  </span>
                </div>
              </div>
            </div>
          </div>
        </section>

        <section class="page-card risk-panel">
          <div class="panel-head">
            <div>
              <h2 class="section-title">风险提示</h2>
              <p class="section-note">优先处理会影响交接和归档的文档</p>
            </div>
            <span class="status-pill status-risk">{{ riskDocs.length }}</span>
          </div>
          <div class="risk-list">
            <div
              v-for="item in riskDocs"
              :key="item.document_id"
              class="risk-item"
            >
              <span class="status-pill status-risk">需关注</span>
              <div class="risk-title">{{ item.title }}</div>
              <div class="risk-message">{{ item.risk_message }}</div>
            </div>
            <div v-if="riskDocs.length === 0" class="empty-state compact">
              <p class="empty-title">暂无风险文档</p>
              <p class="empty-hint">当前没有需要负责人立即处理的文档风险</p>
            </div>
          </div>
        </section>
      </div>
    </div>
  </AppLayout>
</template>

<style scoped>
.command-strip {
  display: grid;
  grid-template-columns: minmax(240px, 0.8fr) minmax(0, 1.2fr);
  gap: 20px;
  align-items: center;
  padding: 20px;
  margin-bottom: 16px;
}

.focus-grid {
  margin-bottom: 18px;
}

.focus-card {
  cursor: pointer;
  text-align: left;
  transition:
    border-color 0.16s ease,
    box-shadow 0.16s ease,
    transform 0.16s ease;
}

.focus-card:hover {
  border-color: #bfd5ed;
  transform: translateY(-1px);
  box-shadow: var(--dd-shadow-sm);
}

.focus-label {
  color: var(--dd-muted);
  font-size: 13px;
  font-weight: 700;
}

.focus-value {
  margin-top: 6px;
  color: var(--dd-ink);
  font-size: 32px;
  font-weight: 800;
}

.focus-note {
  margin: 0;
  color: var(--dd-muted);
  font-size: 13px;
  line-height: 1.6;
}

.focus-action {
  color: var(--dd-primary);
  font-size: 13px;
  font-weight: 700;
}

.status-distribution {
  display: grid;
  grid-template-columns: repeat(6, minmax(82px, 1fr));
  gap: 10px;
}

.status-segment {
  display: grid;
  gap: 8px;
  min-height: 74px;
  padding: 12px;
  border: 1px solid var(--dd-line-soft);
  border-radius: 8px;
  background: var(--dd-surface-soft);
}

.status-segment strong {
  color: var(--dd-ink);
  font-size: 22px;
}

.dashboard-kpis {
  margin-bottom: 18px;
}

.metric-tile {
  display: flex;
  gap: 14px;
  min-height: 128px;
  padding: 18px;
  border-left: 4px solid var(--dd-primary);
}

.metric-icon {
  display: grid;
  place-items: center;
  width: 42px;
  height: 42px;
  border-radius: 10px;
  background: var(--dd-primary-soft);
  color: var(--dd-primary);
}

.metric-hint {
  margin-top: 6px;
  color: var(--dd-muted);
  font-size: 13px;
}

.metric-amber {
  border-left-color: var(--dd-warning);
}

.metric-amber .metric-icon {
  background: var(--dd-warning-soft);
  color: var(--dd-warning);
}

.metric-red {
  border-left-color: var(--dd-danger);
}

.metric-red .metric-icon {
  background: var(--dd-danger-soft);
  color: var(--dd-danger);
}

.dashboard-grid {
  display: grid;
  grid-template-columns: 1.4fr 1fr;
  gap: 18px;
}

.activity-panel,
.risk-panel {
  padding: 20px;
}

.panel-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 18px;
}

.flow-timeline {
  display: grid;
  gap: 0;
}

.flow-item {
  position: relative;
  display: grid;
  grid-template-columns: 18px minmax(0, 1fr);
  gap: 14px;
  padding-bottom: 18px;
}

.flow-item:not(:last-child)::before {
  position: absolute;
  top: 16px;
  bottom: 0;
  left: 7px;
  width: 1px;
  background: var(--dd-line);
  content: "";
}

.flow-dot {
  position: relative;
  z-index: 1;
  width: 15px;
  height: 15px;
  margin-top: 4px;
  border: 3px solid #fff;
  border-radius: 50%;
  background: var(--dd-primary);
  box-shadow: 0 0 0 1px var(--dd-primary);
}

.flow-body {
  padding: 14px;
  border: 1px solid var(--dd-line-soft);
  border-radius: 8px;
  background: #fff;
}

.flow-title {
  color: var(--dd-ink);
  font-weight: 750;
}

.flow-meta {
  display: flex;
  justify-content: space-between;
  gap: 10px;
  margin-top: 4px;
  color: var(--dd-muted);
  font-size: 12px;
}

.flow-status {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  margin-top: 10px;
}

.flow-arrow {
  color: var(--dd-subtle);
}

.risk-list {
  display: grid;
  gap: 12px;
}

.risk-item {
  display: grid;
  gap: 8px;
  padding: 14px;
  border: 1px solid #f1c3bd;
  border-radius: 8px;
  background: #fff8f7;
}

.risk-title {
  color: var(--dd-ink);
  font-weight: 750;
}

.risk-message {
  color: var(--dd-muted);
  font-size: 13px;
}

.compact {
  padding: 34px 12px;
}

@media (max-width: 900px) {
  .command-strip,
  .dashboard-grid {
    grid-template-columns: 1fr;
  }

  .status-distribution {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
