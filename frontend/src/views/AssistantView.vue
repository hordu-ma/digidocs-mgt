<script setup lang="ts">
import { ElMessage } from "element-plus";
import { ChatDotRound } from "@element-plus/icons-vue";
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";

import api from "@/api";
import AppLayout from "@/components/AppLayout.vue";

/* ---------- types ---------- */

type ProjectOption = { id: string; name: string };
type DocumentOption = { id: string; title: string };

type AssistantConversationItem = {
  id: string;
  scope_type: string;
  scope_id: string;
  source_scope?: Record<string, unknown>;
  scope_display_name?: string;
  title?: string;
  created_by?: string;
  created_at: string;
  last_message_at?: string;
  archived_at?: string;
};

type AssistantConversationMessageItem = {
  id: string;
  conversation_id: string;
  role: "user" | "assistant";
  content: string;
  request_id?: string;
  metadata?: Record<string, any>;
  created_by?: string;
  created_at: string;
};

type AssistantRequestItem = {
  id: string;
  status: string;
  error_message?: string;
  output?: Record<string, any>;
};

/* ---------- state ---------- */

const question = ref("");
const loading = ref(false);
const thinking = ref(false);
const conversationsLoading = ref(false);
const messagesLoading = ref(false);
const showArchived = ref(false);

// scope selectors (browsing / composing)
const projects = ref<ProjectOption[]>([]);
const documents = ref<DocumentOption[]>([]);
const selectedProjectID = ref("");
const selectedDocumentID = ref("");

// conversations & messages
const conversations = ref<AssistantConversationItem[]>([]);
const messages = ref<AssistantConversationMessageItem[]>([]);
const activeConversationID = ref("");
const activeRequestID = ref("");
let pollTimer: number | null = null;

/* ---------- computed ---------- */

const visibleConversations = computed(() => {
  if (showArchived.value) {
    return conversations.value.filter((c) => !!c.archived_at);
  }
  return conversations.value.filter((c) => !c.archived_at);
});

const composerScopeLabel = computed(() => {
  if (selectedDocumentID.value) {
    const doc = documents.value.find((d) => d.id === selectedDocumentID.value);
    return doc ? `📄 ${doc.title}` : "📄 文档";
  }
  if (selectedProjectID.value) {
    const proj = projects.value.find((p) => p.id === selectedProjectID.value);
    return proj ? `📁 ${proj.name}` : "📁 项目";
  }
  return "未选择范围";
});

/* ---------- helpers ---------- */

function stopPolling() {
  if (pollTimer !== null) {
    window.clearTimeout(pollTimer);
    pollTimer = null;
  }
}

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

  const lines = normalized.split("\n");
  const html: string[] = [];
  let inUl = false;

  function closeUl() {
    if (inUl) {
      html.push("</ul>");
      inUl = false;
    }
  }

  for (const rawLine of lines) {
    const line = rawLine.trim();
    if (!line) {
      closeUl();
      continue;
    }
    const headingMatch = line.match(/^(#{1,6})\s+(.+)$/);
    if (headingMatch) {
      closeUl();
      const level = headingMatch[1].length;
      html.push(`<h${level}>${renderInlineMarkdown(headingMatch[2])}</h${level}>`);
      continue;
    }
    const unorderedMatch = line.match(/^[-*+]\s+(.+)$/);
    if (unorderedMatch) {
      if (!inUl) {
        html.push("<ul>");
        inUl = true;
      }
      html.push(`<li>${renderInlineMarkdown(unorderedMatch[1])}</li>`);
      continue;
    }
    closeUl();
    html.push(`<p>${renderInlineMarkdown(line)}</p>`);
  }
  closeUl();
  return html.join("");
}

function relativeTime(iso: string | undefined) {
  if (!iso) return "";
  const date = new Date(iso);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMin = Math.floor(diffMs / 60000);
  if (diffMin < 1) return "刚刚";
  if (diffMin < 60) return `${diffMin} 分钟前`;
  const diffHour = Math.floor(diffMin / 60);
  if (diffHour < 24) return `${diffHour} 小时前`;
  const diffDay = Math.floor(diffHour / 24);
  if (diffDay === 1) return "昨天";
  if (diffDay < 30) return `${diffDay} 天前`;
  return date.toLocaleDateString("zh-CN");
}

function formatScopeDisplay(item: AssistantConversationItem) {
  if (item.scope_display_name) {
    const icon = item.scope_type === "document" ? "📄" : "📁";
    return `${icon} ${item.scope_display_name}`;
  }
  // fallback: show scope type
  return item.scope_type === "document" ? "📄 文档" : "📁 项目";
}

function formatMemorySourcesFriendly(metadata?: Record<string, any>) {
  const raw = metadata?.memory_sources;
  if (!Array.isArray(raw) || raw.length === 0) {
    return "未命中可复用记忆";
  }
  const typeLabels: Record<string, string> = {
    conversation_messages: "历史对话",
    confirmed_suggestions: "已确认建议",
    historical_answers: "历史问答",
  };
  return raw
    .map((item: any) => {
      const type = typeof item?.type === "string" ? item.type : "memory";
      const label = typeLabels[type] || type;
      const count = Number(item?.count ?? 0);
      return count > 0 ? `${count} 条${label}` : label;
    })
    .join("、");
}

/* ---------- data loading ---------- */

async function loadProjects() {
  try {
    const res = await api.get("/projects");
    projects.value = (res.data?.data ?? []).map((p: any) => ({
      id: p.id,
      name: p.name,
    }));
  } catch {
    projects.value = [];
  }
}

async function loadDocuments(projectID: string) {
  if (!projectID) {
    documents.value = [];
    return;
  }
  try {
    const res = await api.get("/documents", {
      params: { project_id: projectID, page_size: 200 },
    });
    documents.value = (res.data?.data ?? []).map((d: any) => ({
      id: d.id,
      title: d.title,
    }));
  } catch {
    documents.value = [];
  }
}

/* ---------- watchers ---------- */

watch(selectedProjectID, (newVal) => {
  selectedDocumentID.value = "";
  void loadDocuments(newVal);
  void fetchConversations();
});

watch(selectedDocumentID, () => {
  void fetchConversations();
});

/* ---------- conversations & messages ---------- */

async function fetchConversations() {
  conversationsLoading.value = true;
  try {
    const params: Record<string, string> = {};
    if (selectedDocumentID.value) {
      params.document_id = selectedDocumentID.value;
    } else if (selectedProjectID.value) {
      params.project_id = selectedProjectID.value;
    }
    if (showArchived.value) {
      params.include_archived = "true";
    }
    const res = await api.get("/assistant/conversations", { params });
    conversations.value = (res.data?.data ?? []) as AssistantConversationItem[];
  } catch (err: any) {
    ElMessage.error(err.response?.data?.message ?? "加载会话列表失败");
  } finally {
    conversationsLoading.value = false;
  }
}

async function fetchMessages(conversationID: string) {
  messagesLoading.value = true;
  try {
    const res = await api.get(`/assistant/conversations/${conversationID}/messages`);
    messages.value = (res.data?.data ?? []) as AssistantConversationMessageItem[];
  } catch (err: any) {
    ElMessage.error(err.response?.data?.message ?? "加载会话消息失败");
  } finally {
    messagesLoading.value = false;
  }
}

function openConversation(item: AssistantConversationItem) {
  activeConversationID.value = item.id;
  // sync selectors from conversation scope
  const projId =
    typeof item.source_scope?.project_id === "string"
      ? (item.source_scope.project_id as string)
      : "";
  const docId =
    typeof item.source_scope?.document_id === "string"
      ? (item.source_scope.document_id as string)
      : "";
  if (projId && projId !== selectedProjectID.value) {
    selectedProjectID.value = projId;
    // loadDocuments will fire via watcher, then set document
    if (docId) {
      const unwatch = watch(documents, () => {
        selectedDocumentID.value = docId;
        unwatch();
      });
    }
  } else if (docId) {
    selectedDocumentID.value = docId;
  }
  void fetchMessages(item.id);
}

/* ---------- polling ---------- */

async function pollRequest(requestID: string) {
  try {
    const res = await api.get(`/assistant/requests/${requestID}`);
    const data = res.data?.data as AssistantRequestItem;
    if (data.status === "completed" || data.status === "failed") {
      stopPolling();
      thinking.value = false;
      await fetchConversations();
      if (activeConversationID.value) {
        await fetchMessages(activeConversationID.value);
      }
      if (data.status === "failed") {
        ElMessage.error(data.error_message ?? "AI 请求执行失败");
      }
      return;
    }
    pollTimer = window.setTimeout(() => {
      void pollRequest(requestID);
    }, 2000);
  } catch (err: any) {
    stopPolling();
    thinking.value = false;
    ElMessage.error(err.response?.data?.message ?? "查询 AI 请求状态失败");
  }
}

/* ---------- submit ---------- */

async function submitQuestion() {
  const questionText = question.value.trim();
  if (!questionText) {
    ElMessage.warning("请输入问题");
    return;
  }
  if (!selectedProjectID.value && !selectedDocumentID.value) {
    ElMessage.warning("请先选择一个项目或文档作为提问范围");
    return;
  }

  loading.value = true;
  try {
    stopPolling();
    const res = await api.post("/assistant/ask", {
      question: questionText,
      conversation_id: activeConversationID.value || undefined,
      scope: {
        project_id: selectedProjectID.value || null,
        document_id: selectedDocumentID.value || null,
      },
    });
    const data = res.data?.data;
    activeRequestID.value = data?.request_id ?? "";
    if (typeof data?.conversation_id === "string" && data.conversation_id) {
      activeConversationID.value = data.conversation_id;
    }
    question.value = "";
    thinking.value = true;
    await fetchConversations();
    if (activeConversationID.value) {
      await fetchMessages(activeConversationID.value);
    }
    if (activeRequestID.value) {
      await pollRequest(activeRequestID.value);
    }
  } catch (err: any) {
    ElMessage.error(err.response?.data?.message ?? "提交失败");
  } finally {
    loading.value = false;
  }
}

function startNewConversation() {
  stopPolling();
  activeConversationID.value = "";
  activeRequestID.value = "";
  messages.value = [];
}

async function archiveConversation(conversationID: string, archive: boolean) {
  try {
    await api.patch(`/assistant/conversations/${conversationID}/archive`, { archive });
    ElMessage.success(archive ? "会话已归档" : "会话已恢复");
    if (conversationID === activeConversationID.value && archive) {
      startNewConversation();
    }
    await fetchConversations();
  } catch (err: any) {
    ElMessage.error(err.response?.data?.message ?? "操作失败");
  }
}

function toggleShowArchived() {
  showArchived.value = !showArchived.value;
  void fetchConversations();
}

/* ---------- lifecycle ---------- */

onMounted(async () => {
  await loadProjects();
  void fetchConversations();
});

onBeforeUnmount(() => {
  stopPolling();
});
</script>

<template>
  <AppLayout>
    <div class="page-shell assistant-layout">
      <!-- Left: conversation panel -->
      <ElCard class="page-card conversation-panel">
        <template #header>
          <div class="panel-header">
            <span>{{ showArchived ? '已归档会话' : '会话列表' }}</span>
            <div class="panel-header-actions">
              <ElButton link :type="showArchived ? 'warning' : 'default'" @click="toggleShowArchived">
                {{ showArchived ? '返回列表' : '📦 归档' }}
              </ElButton>
              <ElButton v-if="!showArchived" link type="primary" @click="startNewConversation">新会话</ElButton>
            </div>
          </div>
        </template>

        <div class="scope-filters">
          <ElSelect
            v-model="selectedProjectID"
            placeholder="选择项目（可选）"
            clearable
            filterable
          >
            <ElOption
              v-for="p in projects"
              :key="p.id"
              :label="p.name"
              :value="p.id"
            />
          </ElSelect>
          <ElSelect
            v-model="selectedDocumentID"
            placeholder="选择文档（可选）"
            clearable
            filterable
            :disabled="!selectedProjectID"
          >
            <ElOption
              v-for="d in documents"
              :key="d.id"
              :label="d.title"
              :value="d.id"
            />
          </ElSelect>
        </div>

        <div v-if="!conversationsLoading && visibleConversations.length === 0" class="empty-state">
          <el-icon :size="32" color="var(--el-text-color-placeholder)"><ChatDotRound /></el-icon>
          <p class="empty-title">{{ showArchived ? '暂无归档会话' : '暂无会话' }}</p>
          <p class="empty-hint">{{ showArchived ? '归档的会话将显示在这里' : '选择项目后开始您的第一次对话' }}</p>
        </div>
        <div v-else v-loading="conversationsLoading" class="conversation-list">
          <div
            v-for="item in visibleConversations"
            :key="item.id"
            class="conversation-item"
            :class="{ active: item.id === activeConversationID }"
          >
            <button
              class="conversation-body"
              type="button"
              @click="openConversation(item)"
            >
              <div class="conversation-title">{{ item.title || "未命名会话" }}</div>
              <div class="conversation-meta">
                <ElTag size="small" :type="item.scope_type === 'document' ? 'warning' : 'info'" disable-transitions>
                  {{ formatScopeDisplay(item) }}
                </ElTag>
                <span class="conversation-time" :title="item.last_message_at || item.created_at">
                  {{ relativeTime(item.last_message_at || item.created_at) }}
                </span>
              </div>
            </button>
            <div class="conversation-actions">
              <ElButton
                v-if="!item.archived_at"
                link
                size="small"
                title="归档此会话"
                @click.stop="archiveConversation(item.id, true)"
              >📥</ElButton>
              <ElButton
                v-else
                link
                size="small"
                title="恢复此会话"
                @click.stop="archiveConversation(item.id, false)"
              >↩️</ElButton>
            </div>
          </div>
        </div>
      </ElCard>

      <!-- Right: main area -->
      <div class="assistant-main">
        <ElCard class="page-card">
          <template #header>会话与追问</template>
          <div class="assistant-form">
            <ElInput
              v-model="question"
              :rows="4"
              type="textarea"
              :disabled="thinking"
              :placeholder="thinking ? 'AI 正在思考，请稍候…' : '请输入您想问的问题…'"
            />
            <div class="assistant-actions">
              <ElTag v-if="selectedProjectID || selectedDocumentID" size="default" type="info" disable-transitions>
                {{ composerScopeLabel }}
              </ElTag>
              <span v-else class="assistant-hint">请先在左侧选择项目或文档</span>
              <ElButton
                type="primary"
                :loading="loading"
                :disabled="thinking"
                @click="submitQuestion"
              >{{ thinking ? 'AI 思考中…' : '发送问题' }}</ElButton>
            </div>
          </div>
        </ElCard>

        <ElCard class="page-card">
          <template #header>消息流</template>
          <div v-if="!messagesLoading && messages.length === 0 && !thinking" class="empty-state">
            <el-icon :size="36" color="var(--el-text-color-placeholder)"><ChatDotRound /></el-icon>
            <p class="empty-title">暂无消息</p>
            <p class="empty-hint">选择左侧会话或发起新提问，消息将显示在这里</p>
          </div>
          <div v-else v-loading="messagesLoading" class="message-list">
            <div
              v-for="item in messages"
              :key="item.id"
              class="message-item"
              :class="item.role"
            >
              <div class="message-meta">
                <ElTag size="small" :type="item.role === 'assistant' ? 'success' : 'info'">
                  {{ item.role === "assistant" ? "AI" : "用户" }}
                </ElTag>
                <span :title="item.created_at">{{ relativeTime(item.created_at) }}</span>
              </div>
              <div class="assistant-markdown" v-html="renderMarkdown(item.content)"></div>
              <div v-if="item.role === 'assistant'" class="message-extra">
                <span class="memory-label">{{ formatMemorySourcesFriendly(item.metadata) }}</span>
                <details class="debug-details">
                  <summary>调试详情</summary>
                  <div class="debug-content">
                    <div>模型：{{ item.metadata?.model || "-" }}</div>
                    <div>处理耗时：{{ item.metadata?.processing_duration_ms ? `${item.metadata.processing_duration_ms}ms` : "-" }}</div>
                    <div>Request ID：{{ item.request_id || "-" }}</div>
                    <div>OpenClaw ID：{{ item.metadata?.upstream_request_id || "-" }}</div>
                  </div>
                </details>
              </div>
            </div>
            <!-- AI 思考中占位气泡 -->
            <div v-if="thinking" class="message-item assistant thinking-bubble">
              <div class="message-meta">
                <ElTag size="small" type="success">AI</ElTag>
                <span>正在思考…</span>
              </div>
              <div class="thinking-dots">
                <span class="dot"></span>
                <span class="dot"></span>
                <span class="dot"></span>
              </div>
            </div>
          </div>
        </ElCard>
      </div>
    </div>
  </AppLayout>
</template>

<style scoped>
.assistant-layout {
  display: grid;
  grid-template-columns: 320px minmax(0, 1fr);
  gap: 20px;
}

.assistant-main {
  display: grid;
  gap: 20px;
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.panel-header-actions {
  display: flex;
  gap: 8px;
  align-items: center;
}

.scope-filters {
  display: grid;
  gap: 12px;
  margin-bottom: 16px;
}

.conversation-list {
  display: grid;
  gap: 10px;
}

.conversation-item {
  border: 1px solid var(--el-border-color);
  border-radius: 12px;
  padding: 0;
  background: #fff;
  display: flex;
  align-items: stretch;
  overflow: hidden;
}

.conversation-body {
  flex: 1;
  padding: 12px;
  border: none;
  background: transparent;
  text-align: left;
  cursor: pointer;
}

.conversation-actions {
  display: flex;
  align-items: center;
  padding: 0 6px;
  opacity: 0;
  transition: opacity 0.15s;
}

.conversation-item:hover .conversation-actions {
  opacity: 1;
}

.conversation-item.active {
  border-color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}

.conversation-title {
  font-weight: 600;
  margin-bottom: 6px;
}

.conversation-meta {
  display: flex;
  align-items: center;
  gap: 8px;
}

.conversation-time,
.assistant-hint,
.message-meta,
.message-extra {
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.assistant-form {
  display: grid;
  gap: 16px;
}

.assistant-actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
}

.message-list {
  display: grid;
  gap: 16px;
}

.message-item {
  border-radius: 14px;
  padding: 16px;
  border: 1px solid var(--el-border-color-light);
  background: #fff;
}

.message-item.assistant {
  background: #f8fbff;
}

.message-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
  margin-bottom: 10px;
}

.message-extra {
  margin-top: 12px;
  display: grid;
  gap: 6px;
}

.memory-label {
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.debug-details {
  margin-top: 4px;
}

.debug-details summary {
  cursor: pointer;
  color: var(--el-text-color-placeholder);
  font-size: 12px;
  user-select: none;
}

.debug-content {
  margin-top: 6px;
  padding: 8px 12px;
  background: var(--el-fill-color-lighter);
  border-radius: 8px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  display: grid;
  gap: 4px;
}

.assistant-markdown :deep(p) {
  margin: 0 0 8px;
  line-height: 1.7;
}

.assistant-markdown :deep(ul) {
  margin: 0;
  padding-left: 20px;
}

.assistant-markdown :deep(code) {
  padding: 2px 4px;
  border-radius: 4px;
  background: #f2f4f7;
}

@media (max-width: 960px) {
  .assistant-layout {
    grid-template-columns: 1fr;
  }

  .assistant-actions {
    flex-direction: column;
    align-items: stretch;
  }
}

/* empty state */
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

/* thinking bubble animation */
.thinking-bubble {
  background: #f8fbff;
}

.thinking-dots {
  display: flex;
  gap: 6px;
  padding: 4px 0;
}

.thinking-dots .dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--el-color-primary);
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
</style>
