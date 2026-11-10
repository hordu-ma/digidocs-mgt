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
import {
  documentStatusClass as statusClass,
  documentStatusLabel as statusLabel,
  flowActionLabel as actionLabel,
} from "@/utils/labels";
import { formatDateTime } from "@/utils/format";

const router = useRouter();

const overview = ref({
  document_total: 0,
  status_counts: {} as Record<string, number>,
  handover_pending_count: 0,
  risk_document_count: 0,
});
const recentFlows = ref<any[]>([]);
const riskDocs = ref<any[]>([]);

interface DownloadMetric {
  document_id: string;
  title: string;
  download_count: number;
  user_count: number;
  last_downloaded_at: string;
}
interface AuditSummary {
  view_count: number;
  preview_count: number;
  download_count: number;
  recent_download_count: number;
  top_downloads: DownloadMetric[];
}
// Download activity is a secondary panel: if the summary call fails the rest
// of the dashboard must still render, so it is loaded independently.
const auditSummary = ref<AuditSummary | null>(null);

const riskTypeLabel: Record<string, string> = {
  owner_inactive: "责任人停用",
  transfer_unaccepted: "转交未接收",
  handover_overdue: "交接超期",
  stale: "长期未更新",
};

interface InboxItem {
  kind: string;
  title: string;
  detail?: string;
  link: string;
  related_id: string;
  project_name?: string;
  since?: string;
}
const inbox = ref<InboxItem[]>([]);
const inboxLoaded = ref(false);
const INBOX_PREVIEW = 6;
const inboxExpanded = ref(false);
const visibleInbox = computed(() =>
  inboxExpanded.value ? inbox.value : inbox.value.slice(0, INBOX_PREVIEW),
);
const inboxKindLabel: Record<string, string> = {
  accept_transfer: "待接收",
  confirm_handover: "待确认交接",
  complete_handover: "待完成交接",
  fill_handover_items: "待补清单",
  review_suggestion: "待确认建议",
  stale_document: "长期未更新",
};
const inboxKindTone: Record<string, string> = {
  accept_transfer: "status-pending-handover",
  confirm_handover: "status-pending-handover",
  complete_handover: "status-in-progress",
  fill_handover_items: "status-draft",
  review_suggestion: "status-finalized",
  stale_document: "status-risk",
};




const activeDocumentCount = computed(
  () =>
    (overview.value.status_counts?.draft ?? 0) +
    (overview.value.status_counts?.in_progress ?? 0) +
    (overview.value.status_counts?.pending_handover ?? 0),
);

// One KPI row, not three. The page previously stacked `focusCards`, the status
// strip and these tiles, so "风险文档" and "处理中" each appeared three times with
// the same number — on mobile that pushed the actual content below three screens
// of repeated figures. The focus cards' only unique contribution was navigation,
// so each tile now carries its own target.
const summaryItems = computed(() => [
  {
    label: "文档总量",
    value: overview.value.document_total,
    hint: "纳入团队空间的文档资产",
    icon: DocumentChecked,
    tone: "primary",
    to: "/documents",
  },
  {
    label: "处理中",
    value: activeDocumentCount.value,
    hint: "仍处于草稿、处理中和待接收状态",
    icon: Timer,
    tone: "blue",
    to: "/documents",
  },
  {
    label: "进行中交接",
    value: overview.value.handover_pending_count,
    hint: "尚未完成的成员交接单",
    icon: Connection,
    tone: "amber",
    to: "/handovers",
  },
  {
    label: "风险事项",
    value: overview.value.risk_document_count,
    hint: "长期未更新或交接异常",
    icon: WarningFilled,
    tone: "red",
    to: "/documents",
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

function formatTime(value?: string) {
  return formatDateTime(value);
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

onMounted(async () => {
  try {
    const res = await api.get("/inbox");
    inbox.value = res.data?.data ?? [];
  } catch {
    inbox.value = [];
  } finally {
    inboxLoaded.value = true;
  }
});

onMounted(async () => {
  try {
    const res = await api.get("/audit-events/summary");
    auditSummary.value = res.data?.data ?? null;
  } catch {
    auditSummary.value = null;
  }
});

function openDocument(id?: string) {
  if (id) void router.push(`/documents/${id}`);
}
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

      <section v-if="inboxLoaded" class="page-card inbox-panel">
        <div class="panel-head">
          <div>
            <h2 class="section-title">我的待办</h2>
            <p class="section-note">需要你本人处理的事项，处理完成后会自动消失</p>
          </div>
          <span class="status-pill" :class="inbox.length ? 'status-pending-handover' : ''">{{ inbox.length }}</span>
        </div>
        <div v-if="inbox.length === 0" class="empty-state compact">
          <p class="empty-title">暂无待办</p>
          <p class="empty-hint">有人把文档转交给你、或交接需要你确认时，会出现在这里</p>
        </div>
        <button
          v-for="item in visibleInbox"
          :key="`${item.kind}-${item.related_id}`"
          type="button"
          class="inbox-item"
          @click="router.push(item.link)"
        >
          <span class="status-pill" :class="inboxKindTone[item.kind]">{{ inboxKindLabel[item.kind] ?? item.kind }}</span>
          <span class="inbox-title">{{ item.title }}</span>
          <span class="inbox-detail">
            {{ item.detail }}<template v-if="item.project_name"> · {{ item.project_name }}</template>
          </span>
          <span class="inbox-since">{{ formatDateTime(item.since, "") }}</span>
        </button>
        <ElButton
          v-if="inbox.length > INBOX_PREVIEW"
          link
          type="primary"
          @click="inboxExpanded = !inboxExpanded"
        >
          {{ inboxExpanded ? "收起" : `展开全部 ${inbox.length} 项` }}
        </ElButton>
      </section>

      <section class="command-strip page-card">
        <div class="command-copy">
          <div class="section-title">今日关注</div>
          <p class="section-note">
            当前有 {{ activeDocumentCount }} 份文档仍处于草稿、处理中或待接收状态。
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
        <button
          v-for="item in summaryItems"
          :key="item.label"
          class="metric-tile page-card"
          :class="`metric-${item.tone}`"
          type="button"
          @click="router.push(item.to)"
        >
          <div class="metric-icon">
            <ElIcon :size="20"><component :is="item.icon" /></ElIcon>
          </div>
          <div class="metric-body">
            <div class="kpi-label">{{ item.label }}</div>
            <div class="kpi-value">{{ item.value }}</div>
            <div class="metric-hint">{{ item.hint }}</div>
          </div>
          <ElIcon class="metric-arrow"><ArrowRight /></ElIcon>
        </button>
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
            <button
              v-for="item in riskDocs"
              :key="`${item.risk_type}-${item.related_id}`"
              class="risk-item"
              type="button"
              @click="router.push(item.link)"
            >
              <span class="status-pill status-risk">{{ riskTypeLabel[item.risk_type] ?? "需关注" }}</span>
              <div class="risk-title">{{ item.title }}</div>
              <div class="risk-message">{{ item.risk_message }}</div>
            </button>
            <div v-if="riskDocs.length === 0" class="empty-state compact">
              <p class="empty-title">暂无风险文档</p>
              <p class="empty-hint">当前没有需要负责人立即处理的文档风险</p>
            </div>
          </div>
        </section>

        <section v-if="auditSummary" class="page-card download-panel">
          <div class="panel-head">
            <div>
              <h2 class="section-title">下载活跃度</h2>
              <p class="section-note">近 30 天文档被下载的情况，来自访问审计</p>
            </div>
          </div>
          <div class="download-stats">
            <div><strong>{{ auditSummary.recent_download_count }}</strong><span>近 30 天下载</span></div>
            <div><strong>{{ auditSummary.download_count }}</strong><span>累计下载</span></div>
            <div><strong>{{ auditSummary.preview_count }}</strong><span>累计预览</span></div>
            <div><strong>{{ auditSummary.view_count }}</strong><span>累计查看</span></div>
          </div>
          <div v-if="auditSummary.top_downloads.length === 0" class="empty-state compact">
            <p class="empty-title">近 30 天暂无下载</p>
            <p class="empty-hint">成员下载文档后，这里会列出下载最多的文档</p>
          </div>
          <button
            v-for="item in auditSummary.top_downloads"
            v-else
            :key="item.document_id"
            class="download-item"
            type="button"
            @click="openDocument(item.document_id)"
          >
            <span class="download-title">{{ item.title }}</span>
            <span class="download-meta">
              {{ item.download_count }} 次 · {{ item.user_count }} 人 · 最近 {{ formatTime(item.last_downloaded_at) }}
            </span>
          </button>
        </section>
      </div>
    </div>
  </AppLayout>
</template>

<style scoped>
.inbox-panel {
  padding: 20px;
  margin-bottom: 16px;
}

.inbox-item {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  grid-template-areas:
    "tag title since"
    "tag detail since";
  gap: 2px 12px;
  align-items: center;
  width: 100%;
  padding: 10px 0;
  border: 0;
  border-top: 1px solid var(--dd-line);
  background: transparent;
  text-align: left;
  font: inherit;
  cursor: pointer;
}

.inbox-item .status-pill {
  grid-area: tag;
}

.inbox-title {
  grid-area: title;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.inbox-detail {
  grid-area: detail;
  color: var(--dd-subtle);
  font-size: 13px;
}

.inbox-since {
  grid-area: since;
  color: var(--dd-subtle);
  font-size: 12px;
}

@media (max-width: 720px) {
  .inbox-item {
    grid-template-columns: auto minmax(0, 1fr);
    grid-template-areas:
      "tag title"
      "detail detail"
      "since since";
  }
}

.risk-item,
.download-item {
  width: 100%;
  border: 0;
  background: transparent;
  text-align: left;
  font: inherit;
  cursor: pointer;
}

.download-panel {
  grid-column: 1 / -1;
  padding: 20px;
}

.download-stats {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
  margin: 12px 0 16px;
}

.download-stats div {
  display: grid;
  gap: 4px;
  padding: 10px;
  border: 1px solid var(--dd-line);
  border-radius: 8px;
}

.download-stats strong {
  font-size: 20px;
}

.download-stats span,
.download-meta {
  color: var(--dd-subtle);
  font-size: 12px;
}

.download-item {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 0;
  border-top: 1px solid var(--dd-line);
}

.download-title {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

@media (max-width: 720px) {
  .download-stats {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .download-item {
    flex-direction: column;
    gap: 4px;
  }
}

.command-strip {
  display: grid;
  grid-template-columns: minmax(240px, 0.8fr) minmax(0, 1.2fr);
  gap: 20px;
  align-items: center;
  padding: 20px;
  margin-bottom: 16px;
}

/* The KPI tiles are buttons now, so they need the affordances the removed
   focus cards used to carry. */
.metric-tile {
  width: 100%;
  border: 1px solid var(--dd-line);
  text-align: left;
  cursor: pointer;
  transition:
    border-color 0.16s ease,
    box-shadow 0.16s ease,
    transform 0.16s ease;
}

.metric-tile:hover {
  border-color: #bfd5ed;
  transform: translateY(-1px);
  box-shadow: var(--dd-shadow-sm);
}

.metric-body {
  min-width: 0;
}

/* Positioned rather than a flex item: as a third column it squeezed the hint
   text into awkward mid-word wraps at narrow tile widths. */
.metric-arrow {
  position: absolute;
  top: 18px;
  right: 14px;
  color: var(--dd-subtle);
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
  position: relative;
  display: flex;
  gap: 14px;
  min-height: 128px;
  padding: 18px 34px 18px 18px;
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
