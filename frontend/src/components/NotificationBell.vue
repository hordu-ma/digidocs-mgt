<script setup lang="ts">
import { Bell } from "@element-plus/icons-vue";
import { ElMessage } from "element-plus";
import { onBeforeUnmount, onMounted, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";

import api from "@/api";
import { extractError } from "@/utils/error";
import { formatDateTime } from "@/utils/format";

interface NotificationItem {
  id: string;
  event_type: string;
  title: string;
  body?: string;
  link?: string;
  actor_name?: string;
  read: boolean;
  created_at: string;
}

const POLL_MS = 60_000;

const route = useRoute();
const router = useRouter();
const unread = ref(0);
const visible = ref(false);
const loading = ref(false);
const items = ref<NotificationItem[]>([]);
const onlyUnread = ref(false);
let timer: number | null = null;

async function refreshCount() {
  try {
    const res = await api.get("/notifications/unread-count");
    unread.value = res.data?.data?.unread ?? 0;
  } catch {
    // The badge is a convenience; a failed poll keeps the last value.
  }
}

async function loadList() {
  loading.value = true;
  try {
    const res = await api.get("/notifications", {
      params: { page_size: 50, unread_only: onlyUnread.value ? "true" : undefined },
    });
    items.value = res.data?.data ?? [];
    unread.value = res.data?.meta?.unread ?? unread.value;
  } catch (err) {
    ElMessage.error(extractError(err, "通知加载失败"));
  } finally {
    loading.value = false;
  }
}

async function open() {
  visible.value = true;
  await loadList();
}

async function openItem(item: NotificationItem) {
  if (!item.read) {
    try {
      await api.post(`/notifications/${item.id}/read`);
      item.read = true;
      unread.value = Math.max(0, unread.value - 1);
    } catch {
      // Navigation still works even if marking read failed.
    }
  }
  if (item.link) {
    visible.value = false;
    void router.push(item.link);
  }
}

async function markAll() {
  try {
    await api.post("/notifications/read-all");
    items.value.forEach((item) => (item.read = true));
    unread.value = 0;
  } catch (err) {
    ElMessage.error(extractError(err, "操作失败"));
  }
}

watch(onlyUnread, () => void loadList());
watch(() => route.fullPath, () => void refreshCount());

onMounted(() => {
  void refreshCount();
  timer = window.setInterval(() => {
    if (document.visibilityState === "visible") void refreshCount();
  }, POLL_MS);
});
onBeforeUnmount(() => {
  if (timer !== null) window.clearInterval(timer);
});
</script>

<template>
  <button class="bell-trigger" type="button" :aria-label="`通知，${unread} 条未读`" @click="open">
    <ElBadge :value="unread" :hidden="unread === 0" :max="99">
      <ElIcon :size="18"><Bell /></ElIcon>
    </ElBadge>
  </button>
  <ElDrawer v-model="visible" title="通知" size="min(92vw, 400px)" append-to-body>
    <div class="bell-toolbar">
      <ElSwitch v-model="onlyUnread" active-text="只看未读" />
      <ElButton link type="primary" :disabled="unread === 0" @click="markAll">全部标为已读</ElButton>
    </div>
    <div v-loading="loading" class="bell-list">
      <div v-if="!loading && items.length === 0" class="empty-state compact">
        <p class="empty-title">{{ onlyUnread ? "没有未读通知" : "暂无通知" }}</p>
        <p class="empty-hint">文档转交、交接进展和 AI 摘要完成时会在这里提醒你</p>
      </div>
      <button
        v-for="item in items"
        :key="item.id"
        type="button"
        class="bell-item"
        :class="{ unread: !item.read }"
        @click="openItem(item)"
      >
        <span class="bell-title">{{ item.title }}</span>
        <span v-if="item.body" class="bell-body">{{ item.body }}</span>
        <span class="bell-meta">{{ formatDateTime(item.created_at) }}</span>
      </button>
    </div>
  </ElDrawer>
</template>

<style scoped>
.bell-trigger {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 38px;
  height: 38px;
  border: 1px solid var(--dd-line);
  border-radius: 8px;
  background: #fff;
  cursor: pointer;
}

.bell-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.bell-list {
  display: grid;
  gap: 8px;
  min-height: 120px;
}

.bell-item {
  display: grid;
  gap: 4px;
  width: 100%;
  padding: 12px;
  border: 1px solid var(--dd-line);
  border-radius: 8px;
  background: #fff;
  text-align: left;
  font: inherit;
  cursor: pointer;
}

.bell-item.unread {
  border-color: #bfd5ed;
  background: #f4f8fd;
}

.bell-item.unread .bell-title::before {
  content: "";
  display: inline-block;
  width: 7px;
  height: 7px;
  margin-right: 6px;
  border-radius: 50%;
  background: #2f6fb6;
  vertical-align: middle;
}

.bell-title {
  font-weight: 600;
}

.bell-body {
  color: var(--dd-muted, #5b6573);
  font-size: 13px;
  white-space: pre-line;
}

.bell-meta {
  color: var(--dd-subtle);
  font-size: 12px;
}
</style>
