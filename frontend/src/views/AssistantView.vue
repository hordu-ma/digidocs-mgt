<script setup lang="ts">
import { ElMessage } from "element-plus";
import { onBeforeUnmount, onMounted, ref } from "vue";

import AppLayout from "@/components/AppLayout.vue";
import api from "@/api";

type TimelineItem = {
  title: string;
  content: string;
  html: string;
};

type AssistantRequestItem = {
  id: string;
  request_type: string;
  related_type?: string;
  related_id?: string;
  status: string;
  question?: string;
  source_scope?: Record<string, unknown>;
  error_message?: string;
  output?: Record<string, any>;
  model?: string;
  upstream_request_id?: string;
  usage?: Record<string, unknown>;
  created_at: string;
  completed_at?: string;
  processing_duration_ms?: number;
};

const requestTypeOptions = [
  { label: "全部类型", value: "" },
  { label: "问答", value: "assistant.ask" },
  { label: "文档摘要", value: "document.summarize" },
  { label: "交接摘要", value: "handover.summarize" },
  { label: "正文抽取", value: "document.extract_text" },
];

const statusOptions = [
  { label: "全部状态", value: "" },
  { label: "待处理", value: "pending" },
  { label: "运行中", value: "running" },
  { label: "已完成", value: "completed" },
  { label: "失败", value: "failed" },
];

const question = ref("课题A 最近一个月有哪些文档在流转？");
const loading = ref(false);
const timeline = ref<TimelineItem[]>([]);
const activeRequestID = ref("");
const submittedQuestion = ref("");
const historyLoading = ref(false);
const historyItems = ref<AssistantRequestItem[]>([]);
const historyFilters = ref({
  requestType: "assistant.ask",
  status: "",
  keyword: "",
});
const historyPage = ref(1);
const historyPageSize = ref(10);
const historyTotal = ref(0);
let pollTimer: number | null = null;

function escapeHtml(value: string) {
  return value
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;");
}

function renderInlineMarkdown(value: string) {
  const escaped = escapeHtml(value);
  return escaped
    .replace(/`([^`]+)`/g, "<code>$1</code>")
    .replace(/\*\*([^*]+)\*\*/g, "<strong>$1</strong>")
    .replace(/\*([^*]+)\*/g, "<em>$1</em>");
}

function renderMarkdown(value: string) {
  const normalized = value.replace(/\r\n/g, "\n").trim();
  if (!normalized) {
    return "<p>暂无内容</p>";
  }

  const codeBlocks: string[] = [];
  const withPlaceholders = normalized.replace(
    /```([\s\S]*?)```/g,
    (_, code: string) => {
      const index = codeBlocks.push(code.trim()) - 1;
      return `@@CODE_BLOCK_${index}@@`;
    },
  );

  const lines = withPlaceholders.split("\n");
  const html: string[] = [];
  let paragraph: string[] = [];
  let inUl = false;
  let inOl = false;
  let inBlockquote = false;

  const closeLists = () => {
    if (inUl) {
      html.push("</ul>");
      inUl = false;
    }
    if (inOl) {
      html.push("</ol>");
      inOl = false;
    }
  };

  function flushParagraph() {
    if (paragraph.length === 0) {
      return;
    }
    html.push(`<p>${paragraph.join("<br>")}</p>`);
    paragraph = [];
  }

  const closeBlockquote = () => {
    if (inBlockquote) {
      flushParagraph();
      closeLists();
      html.push("</blockquote>");
      inBlockquote = false;
    }
  };

  for (const rawLine of lines) {
    const line = rawLine.trim();
    if (!line) {
      flushParagraph();
      closeLists();
      closeBlockquote();
      continue;
    }

    const codeMatch = line.match(/^@@CODE_BLOCK_(\d+)@@$/);
    if (codeMatch) {
      flushParagraph();
      closeLists();
      closeBlockquote();
      const code = codeBlocks[Number(codeMatch[1])] ?? "";
      html.push(`<pre><code>${escapeHtml(code)}</code></pre>`);
      continue;
    }

    const headingMatch = line.match(/^(#{1,6})\s+(.+)$/);
    if (headingMatch) {
      flushParagraph();
      closeLists();
      closeBlockquote();
      const level = headingMatch[1].length;
      html.push(
        `<h${level}>${renderInlineMarkdown(headingMatch[2])}</h${level}>`,
      );
      continue;
    }

    const quoteMatch = line.match(/^>\s?(.*)$/);
    if (quoteMatch) {
      flushParagraph();
      closeLists();
      if (!inBlockquote) {
        html.push("<blockquote>");
        inBlockquote = true;
      }
      paragraph.push(renderInlineMarkdown(quoteMatch[1]));
      continue;
    }

    const unorderedMatch = line.match(/^[-*+]\s+(.+)$/);
    if (unorderedMatch) {
      flushParagraph();
      if (inOl) {
        html.push("</ol>");
        inOl = false;
      }
      if (!inUl) {
        html.push("<ul>");
        inUl = true;
      }
      html.push(`<li>${renderInlineMarkdown(unorderedMatch[1])}</li>`);
      continue;
    }

    const orderedMatch = line.match(/^\d+\.\s+(.+)$/);
    if (orderedMatch) {
      flushParagraph();
      if (inUl) {
        html.push("</ul>");
        inUl = false;
      }
      if (!inOl) {
        html.push("<ol>");
        inOl = true;
      }
      html.push(`<li>${renderInlineMarkdown(orderedMatch[1])}</li>`);
      continue;
    }

    paragraph.push(renderInlineMarkdown(line));
  }

  flushParagraph();
  closeLists();
  closeBlockquote();
  return html.join("");
}

function createTimelineItem(title: string, content: string): TimelineItem {
  return {
    title,
    content,
    html: renderMarkdown(content),
  };
}

function formatScope(scope?: Record<string, unknown>) {
  if (!scope || Object.keys(scope).length === 0) {
    return "- 当前范围：未指定 project_id / document_id";
  }
  const parts: string[] = [];
  if (typeof scope.project_id === "string" && scope.project_id) {
    parts.push(`project_id = ${scope.project_id}`);
  }
  if (typeof scope.document_id === "string" && scope.document_id) {
    parts.push(`document_id = ${scope.document_id}`);
  }
  if (parts.length === 0) {
    return "- 当前范围：未指定 project_id / document_id";
  }
  return `- 当前范围：${parts.join("，")}`;
}

function buildTimeline(request: AssistantRequestItem) {
  const answer =
    typeof request.output?.answer === "string" ? request.output.answer : "";
  const infoLines = [
    `- 当前状态：**${request.status || "unknown"}**`,
    request.model ? `- 使用模型：\`${request.model}\`` : "- 使用模型：-",
    request.upstream_request_id
      ? `- OpenClaw 响应 ID：\`${request.upstream_request_id}\``
      : "- OpenClaw 响应 ID：-",
    request.processing_duration_ms
      ? `- 处理耗时：${request.processing_duration_ms} ms`
      : "- 处理耗时：处理中或未记录",
    formatScope(request.source_scope),
  ];

  timeline.value = [
    createTimelineItem(
      "已提交",
      `问题「${request.question || submittedQuestion.value || "-"}」已提交至 AI 助手（request_id: ${request.id}）`,
    ),
    createTimelineItem("执行信息", infoLines.join("\n")),
    createTimelineItem(
      "结果",
      request.status === "completed"
        ? answer || "任务已完成，但未返回回答内容。"
        : request.status === "failed"
          ? request.error_message || "任务执行失败。"
          : "任务仍在处理中，正在轮询最新状态。",
    ),
  ];
}

function statusTagType(status: string) {
  if (status === "completed") {
    return "success";
  }
  if (status === "failed") {
    return "danger";
  }
  if (status === "running") {
    return "warning";
  }
  return "info";
}

function stopPolling() {
  if (pollTimer !== null) {
    window.clearTimeout(pollTimer);
    pollTimer = null;
  }
}

async function fetchHistory() {
  historyLoading.value = true;
  try {
    const res = await api.get("/assistant/requests", {
      params: {
        request_type: historyFilters.value.requestType,
        status: historyFilters.value.status,
        keyword: historyFilters.value.keyword.trim() || undefined,
        page: historyPage.value,
        page_size: historyPageSize.value,
      },
    });
    historyItems.value = (res.data?.data ?? []) as AssistantRequestItem[];
    historyTotal.value = Number(res.data?.meta?.total ?? 0);
  } catch (err: any) {
    ElMessage.error(err.response?.data?.message ?? "加载问答历史失败");
  } finally {
    historyLoading.value = false;
  }
}

async function openHistory(item: AssistantRequestItem) {
  activeRequestID.value = item.id;
  submittedQuestion.value = item.question ?? "";
  question.value = item.question ?? question.value;
  await loadRequest(item.id);
}

async function loadRequest(requestID: string) {
  const res = await api.get(`/assistant/requests/${requestID}`);
  const data = res.data?.data as AssistantRequestItem;
  buildTimeline(data);
  return data;
}

async function pollRequest(requestID: string) {
  try {
    const data = await loadRequest(requestID);
    if (data?.status === "completed" || data?.status === "failed") {
      stopPolling();
      await fetchHistory();
      return;
    }

    pollTimer = window.setTimeout(() => {
      void pollRequest(requestID);
    }, 2000);
  } catch (err: any) {
    stopPolling();
    ElMessage.error(err.response?.data?.message ?? "查询 AI 任务状态失败");
  }
}

async function submitQuestion() {
  if (!question.value.trim()) {
    ElMessage.warning("请输入问题");
    return;
  }

  loading.value = true;
  try {
    stopPolling();
    const res = await api.post("/assistant/ask", {
      question: question.value,
      scope: {
        project_id: null,
        document_id: null,
      },
    });
    const data = res.data?.data;
    activeRequestID.value = data?.request_id ?? "";
    submittedQuestion.value = data?.question ?? question.value;
    buildTimeline({
      id: activeRequestID.value,
      request_type: "assistant.ask",
      status: data?.status ?? "queued",
      question: submittedQuestion.value,
      source_scope: data?.source_scope,
      created_at: data?.generated_at ?? "",
      output: {},
    });
    historyPage.value = 1;
    await fetchHistory();
    if (activeRequestID.value) {
      await pollRequest(activeRequestID.value);
    }
    ElMessage.success("问题已提交");
  } catch (err: any) {
    const msg = err.response?.data?.message ?? "提交失败";
    ElMessage.error(msg);
  } finally {
    loading.value = false;
  }
}

async function applyHistoryFilters() {
  historyPage.value = 1;
  await fetchHistory();
}

async function handleHistoryPageChange(page: number) {
  historyPage.value = page;
  await fetchHistory();
}

async function resetHistoryFilters() {
  historyFilters.value = {
    requestType: "assistant.ask",
    status: "",
    keyword: "",
  };
  historyPage.value = 1;
  await fetchHistory();
}

onMounted(() => {
  void fetchHistory();
});

onBeforeUnmount(() => {
  stopPolling();
});
</script>

<template>
  <AppLayout>
    <div class="page-shell assistant-grid">
      <ElCard class="page-card">
        <template #header>OpenClaw 助手</template>
        <div class="assistant-form">
          <ElInput v-model="question" :rows="4" type="textarea" />
          <ElButton
            type="primary"
            :loading="loading"
            style="margin-top: 16px"
            @click="submitQuestion"
            >发起问答</ElButton
          >
        </div>
      </ElCard>

      <ElCard class="page-card">
        <template #header>结果与建议</template>
        <ElEmpty
          v-if="timeline.length === 0"
          description="发起问答后将在这里展示结果"
        />
        <ElTimeline v-else>
          <ElTimelineItem
            v-for="item in timeline"
            :key="item.title"
            :timestamp="item.title"
          >
            <div class="assistant-markdown" v-html="item.html"></div>
          </ElTimelineItem>
        </ElTimeline>
      </ElCard>

      <ElCard class="page-card assistant-history-card">
        <template #header>问答历史</template>
        <div class="assistant-history-toolbar">
          <ElSelect
            v-model="historyFilters.requestType"
            class="history-filter"
            placeholder="请求类型"
          >
            <ElOption
              v-for="item in requestTypeOptions"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </ElSelect>
          <ElSelect
            v-model="historyFilters.status"
            class="history-filter"
            placeholder="状态"
          >
            <ElOption
              v-for="item in statusOptions"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </ElSelect>
          <ElInput
            v-model="historyFilters.keyword"
            class="history-keyword"
            clearable
            placeholder="按问题关键词筛选"
            @keyup.enter="applyHistoryFilters"
          />
          <ElButton @click="applyHistoryFilters">筛选</ElButton>
          <ElButton link @click="resetHistoryFilters">重置</ElButton>
        </div>

        <ElTable
          :data="historyItems"
          v-loading="historyLoading"
          empty-text="暂无问答历史"
        >
          <ElTableColumn
            prop="question"
            label="问题"
            min-width="320"
            show-overflow-tooltip
          />
          <ElTableColumn label="状态" width="120">
            <template #default="{ row }">
              <ElTag :type="statusTagType(row.status)">{{ row.status }}</ElTag>
            </template>
          </ElTableColumn>
          <ElTableColumn
            prop="model"
            label="模型"
            min-width="160"
            show-overflow-tooltip
          />
          <ElTableColumn prop="created_at" label="提交时间" width="200" />
          <ElTableColumn label="操作" width="100" fixed="right">
            <template #default="{ row }">
              <ElButton link type="primary" @click="openHistory(row)"
                >查看</ElButton
              >
            </template>
          </ElTableColumn>
        </ElTable>

        <div class="assistant-history-pagination">
          <ElPagination
            background
            layout="prev, pager, next, total"
            :current-page="historyPage"
            :page-size="historyPageSize"
            :total="historyTotal"
            @current-change="handleHistoryPageChange"
          />
        </div>
      </ElCard>
    </div>
  </AppLayout>
</template>

<style scoped>
.assistant-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 18px;
}

.assistant-form {
  display: flex;
  flex-direction: column;
  min-height: 100%;
}

.assistant-history-card {
  grid-column: 1 / -1;
}

.assistant-history-toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
  flex-wrap: wrap;
}

.history-filter {
  width: 160px;
}

.history-keyword {
  flex: 1;
  min-width: 220px;
}

.assistant-history-pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}

.assistant-markdown :deep(p) {
  margin: 0 0 12px;
  white-space: normal;
  word-break: break-word;
}

.assistant-markdown :deep(ul),
.assistant-markdown :deep(ol) {
  margin: 0 0 12px;
  padding-left: 20px;
}

.assistant-markdown :deep(li) {
  margin-bottom: 6px;
}

.assistant-markdown :deep(strong) {
  color: #10243e;
}

.assistant-markdown :deep(code) {
  padding: 2px 6px;
  border-radius: 6px;
  background: rgba(16, 36, 62, 0.08);
  font-family: "JetBrains Mono", "SFMono-Regular", monospace;
  font-size: 0.92em;
}

.assistant-markdown :deep(pre) {
  overflow-x: auto;
  margin: 0 0 12px;
  padding: 12px;
  border-radius: 12px;
  background: #0f1b2d;
  color: #f7fbff;
}

.assistant-markdown :deep(pre code) {
  padding: 0;
  background: transparent;
  color: inherit;
}

.assistant-markdown :deep(blockquote) {
  margin: 0 0 12px;
  padding: 8px 12px;
  border-left: 4px solid rgba(66, 133, 244, 0.45);
  background: rgba(66, 133, 244, 0.06);
  border-radius: 0 10px 10px 0;
}

.assistant-markdown :deep(h1),
.assistant-markdown :deep(h2),
.assistant-markdown :deep(h3),
.assistant-markdown :deep(h4),
.assistant-markdown :deep(h5),
.assistant-markdown :deep(h6) {
  margin: 0 0 12px;
  color: #10243e;
  line-height: 1.35;
}

@media (max-width: 900px) {
  .assistant-grid {
    grid-template-columns: 1fr;
  }

  .assistant-history-card {
    grid-column: auto;
  }
}
</style>
