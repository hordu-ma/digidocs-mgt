<script setup lang="ts">
import { ElMessage } from "element-plus";
import { onBeforeUnmount, onMounted, ref } from "vue";

import api from "@/api";
import AppLayout from "@/components/AppLayout.vue";

type AssistantConversationItem = {
  id: string;
  scope_type: string;
  scope_id: string;
  source_scope?: Record<string, unknown>;
  title?: string;
  created_by?: string;
  created_at: string;
  last_message_at?: string;
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

const question = ref("课题A 最近一个月有哪些文档在流转？");
const projectID = ref("");
const documentID = ref("");
const loading = ref(false);
const conversationsLoading = ref(false);
const messagesLoading = ref(false);
const conversations = ref<AssistantConversationItem[]>([]);
const messages = ref<AssistantConversationMessageItem[]>([]);
const activeConversationID = ref("");
const activeRequestID = ref("");
let pollTimer: number | null = null;

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

function formatScope(scope?: Record<string, unknown>) {
  if (!scope) {
    return "未设置范围";
  }
  const parts: string[] = [];
  if (typeof scope.project_id === "string" && scope.project_id) {
    parts.push(`project_id=${scope.project_id}`);
  }
  if (typeof scope.document_id === "string" && scope.document_id) {
    parts.push(`document_id=${scope.document_id}`);
  }
  return parts.length ? parts.join(" / ") : "未设置范围";
}

function formatMemorySources(metadata?: Record<string, any>) {
  const raw = metadata?.memory_sources;
  if (!Array.isArray(raw) || raw.length === 0) {
    return "未命中可复用记忆";
  }
  return raw
    .map((item: any) => {
      const type = typeof item?.type === "string" ? item.type : "memory";
      const count = Number(item?.count ?? 0);
      return count > 0 ? `${type}(${count})` : type;
    })
    .join("，");
}

function applyConversationScope(item: AssistantConversationItem) {
  activeConversationID.value = item.id;
  projectID.value =
    typeof item.source_scope?.project_id === "string"
      ? item.source_scope.project_id
      : "";
  documentID.value =
    typeof item.source_scope?.document_id === "string"
      ? item.source_scope.document_id
      : "";
}

async function fetchConversations() {
  conversationsLoading.value = true;
  try {
    const params: Record<string, string> = {};
    if (documentID.value.trim()) {
      params.document_id = documentID.value.trim();
    } else if (projectID.value.trim()) {
      params.project_id = projectID.value.trim();
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

async function openConversation(item: AssistantConversationItem) {
  applyConversationScope(item);
  await fetchMessages(item.id);
}

async function pollRequest(requestID: string) {
  try {
    const res = await api.get(`/assistant/requests/${requestID}`);
    const data = res.data?.data as AssistantRequestItem;
    if (data.status === "completed" || data.status === "failed") {
      stopPolling();
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
    ElMessage.error(err.response?.data?.message ?? "查询 AI 请求状态失败");
  }
}

async function submitQuestion() {
  const questionText = question.value.trim();
  if (!questionText) {
    ElMessage.warning("请输入问题");
    return;
  }
  if (!projectID.value.trim() && !documentID.value.trim()) {
    ElMessage.warning("请至少填写 project_id 或 document_id");
    return;
  }

  loading.value = true;
  try {
    stopPolling();
    const res = await api.post("/assistant/ask", {
      question: questionText,
      conversation_id: activeConversationID.value || undefined,
      scope: {
        project_id: projectID.value.trim() || null,
        document_id: documentID.value.trim() || null,
      },
    });
    const data = res.data?.data;
    activeRequestID.value = data?.request_id ?? "";
    if (typeof data?.conversation_id === "string" && data.conversation_id) {
      activeConversationID.value = data.conversation_id;
    }
    await fetchConversations();
    if (activeConversationID.value) {
      await fetchMessages(activeConversationID.value);
    }
    if (activeRequestID.value) {
      await pollRequest(activeRequestID.value);
    }
    ElMessage.success("问题已提交");
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

onMounted(() => {
  void fetchConversations();
});

onBeforeUnmount(() => {
  stopPolling();
});
</script>

<template>
  <AppLayout>
    <div class="page-shell assistant-layout">
      <ElCard class="page-card conversation-panel">
        <template #header>
          <div class="panel-header">
            <span>会话列表</span>
            <ElButton link type="primary" @click="startNewConversation">新会话</ElButton>
          </div>
        </template>
        <div class="scope-filters">
          <ElInput v-model="projectID" placeholder="project_id" clearable />
          <ElInput v-model="documentID" placeholder="document_id" clearable />
          <ElButton @click="fetchConversations">按范围筛选</ElButton>
        </div>
        <ElEmpty
          v-if="!conversationsLoading && conversations.length === 0"
          description="当前范围暂无会话"
        />
        <div v-else v-loading="conversationsLoading" class="conversation-list">
          <button
            v-for="item in conversations"
            :key="item.id"
            class="conversation-item"
            :class="{ active: item.id === activeConversationID }"
            type="button"
            @click="openConversation(item)"
          >
            <div class="conversation-title">{{ item.title || "未命名会话" }}</div>
            <div class="conversation-scope">{{ formatScope(item.source_scope) }}</div>
            <div class="conversation-time">{{ item.last_message_at || item.created_at }}</div>
          </button>
        </div>
      </ElCard>

      <div class="assistant-main">
        <ElCard class="page-card">
          <template #header>会话与追问</template>
          <div class="assistant-form">
            <ElInput v-model="question" :rows="4" type="textarea" />
            <div class="assistant-actions">
              <div class="assistant-hint">
                当前范围：{{ formatScope({ project_id: projectID || undefined, document_id: documentID || undefined }) }}
              </div>
              <ElButton type="primary" :loading="loading" @click="submitQuestion">发送问题</ElButton>
            </div>
          </div>
        </ElCard>

        <ElCard class="page-card">
          <template #header>消息流</template>
          <ElEmpty
            v-if="!messagesLoading && messages.length === 0"
            description="发起问题后会在这里显示会话消息"
          />
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
                <span>{{ item.created_at }}</span>
                <span v-if="item.request_id">request_id={{ item.request_id }}</span>
              </div>
              <div class="assistant-markdown" v-html="renderMarkdown(item.content)"></div>
              <div v-if="item.role === 'assistant'" class="message-extra">
                <div>模型：{{ item.metadata?.model || "-" }}</div>
                <div>OpenClaw 响应 ID：{{ item.metadata?.upstream_request_id || "-" }}</div>
                <div>记忆来源：{{ formatMemorySources(item.metadata) }}</div>
                <div>来源范围：{{ formatScope(item.metadata?.source_scope) }}</div>
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
  padding: 12px;
  background: #fff;
  text-align: left;
  cursor: pointer;
}

.conversation-item.active {
  border-color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}

.conversation-title {
  font-weight: 600;
  margin-bottom: 6px;
}

.conversation-scope,
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
</style>
