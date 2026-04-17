<script setup lang="ts">
import {
  ChatDotRound,
  Connection,
  DataBoard,
  Document,
  Folder,
  Operation,
  Search,
  Setting,
  User,
} from "@element-plus/icons-vue";
import { computed, ref, watch } from "vue";
import { useRouter } from "vue-router";

import api from "@/api";

const props = defineProps<{
  modelValue: boolean;
  isAdmin?: boolean;
}>();

const emit = defineEmits<{
  "update:modelValue": [value: boolean];
}>();

const router = useRouter();
const keyword = ref("");
const loading = ref(false);
const documents = ref<any[]>([]);
let searchTimer: number | null = null;

const quickActions = computed(() => {
  const base = [
    {
      title: "负责人总览",
      caption: "查看今日关注与风险提示",
      route: "/dashboard",
      icon: DataBoard,
    },
    {
      title: "文档资产库",
      caption: "进入文档工作台",
      route: "/documents",
      icon: Document,
    },
    {
      title: "数据资产库",
      caption: "查看图片、模型和压缩包",
      route: "/data",
      icon: Folder,
    },
    {
      title: "工作交接",
      caption: "推进交接清单和确认流程",
      route: "/handovers",
      icon: Connection,
    },
    {
      title: "OpenClaw 助手",
      caption: "进入可信问答工作区",
      route: "/assistant",
      icon: ChatDotRound,
    },
    {
      title: "个人信息",
      caption: "维护显示名和联系方式",
      route: "/profile",
      icon: User,
    },
  ];

  if (props.isAdmin) {
    base.push({
      title: "系统管理",
      caption: "管理用户、空间和成员关系",
      route: "/admin",
      icon: Setting,
    });
  }

  return base;
});

watch(
  () => props.modelValue,
  (visible) => {
    if (visible) {
      keyword.value = "";
      documents.value = [];
    } else if (searchTimer !== null) {
      window.clearTimeout(searchTimer);
      searchTimer = null;
    }
  },
);

watch(keyword, (value) => {
  if (searchTimer !== null) {
    window.clearTimeout(searchTimer);
  }
  const trimmed = value.trim();
  if (!trimmed) {
    documents.value = [];
    return;
  }
  searchTimer = window.setTimeout(() => {
    void searchDocuments(trimmed);
  }, 220);
});

async function searchDocuments(value: string) {
  loading.value = true;
  try {
    const [documentRes] = await Promise.all([
      api.get("/documents", {
        params: {
          keyword: value,
          page: 1,
          page_size: 8,
        },
      }),
    ]);
    documents.value = documentRes.data?.data ?? [];
  } finally {
    loading.value = false;
  }
}

function closeDialog() {
  emit("update:modelValue", false);
}

function goTo(route: string) {
  closeDialog();
  void router.push(route);
}

function openDocument(id: string) {
  closeDialog();
  void router.push(`/documents/${id}`);
}
</script>

<template>
  <ElDialog
    :model-value="modelValue"
    class="command-dialog"
    width="720px"
    top="10vh"
    :show-close="false"
    append-to-body
    @update:model-value="emit('update:modelValue', $event)"
  >
    <div class="command-panel">
      <div class="command-search">
        <ElIcon><Search /></ElIcon>
        <ElInput
          v-model="keyword"
          placeholder="搜索页面、命令或文档标题"
          autofocus
          clearable
        />
        <span class="app-kbd">ESC</span>
      </div>

      <div class="command-section">
        <div class="command-section-head">
          <span>快捷入口</span>
          <span class="command-section-note">按工作目标进入，而不只是按菜单寻找</span>
        </div>
        <div class="command-grid">
          <button
            v-for="item in quickActions"
            :key="item.route"
            class="command-item"
            type="button"
            @click="goTo(item.route)"
          >
            <span class="command-item-icon">
              <ElIcon><component :is="item.icon" /></ElIcon>
            </span>
            <span class="command-item-copy">
              <strong>{{ item.title }}</strong>
              <small>{{ item.caption }}</small>
            </span>
            <span class="app-kbd">↵</span>
          </button>
        </div>
      </div>

      <div class="command-section">
        <div class="command-section-head">
          <span>文档直达</span>
          <span class="command-section-note">输入关键字后直接进入文档档案页</span>
        </div>

        <div v-if="!keyword.trim()" class="command-empty">
          <ElIcon><Operation /></ElIcon>
          <span>输入文档标题关键词后，这里会显示直达结果。</span>
        </div>

        <div v-else-if="loading" class="command-empty">
          <ElIcon class="is-loading"><Search /></ElIcon>
          <span>正在搜索文档…</span>
        </div>

        <div v-else-if="documents.length === 0" class="command-empty">
          <ElIcon><Document /></ElIcon>
          <span>没有找到匹配文档。</span>
        </div>

        <div v-else class="command-results">
          <button
            v-for="item in documents"
            :key="item.id"
            class="command-result"
            type="button"
            @click="openDocument(item.id)"
          >
            <div class="command-result-copy">
              <strong>{{ item.title }}</strong>
              <small>
                {{ item.project_name || "未分类课题" }} ·
                {{ item.current_owner?.display_name || "未分配责任人" }}
              </small>
            </div>
            <span class="status-pill" :class="`status-${item.current_status?.replaceAll('_', '-')}`">
              {{ item.current_status }}
            </span>
          </button>
        </div>
      </div>
    </div>
  </ElDialog>
</template>

<style scoped>
.command-panel {
  display: grid;
  gap: 18px;
}

.command-search {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  gap: 12px;
  align-items: center;
  padding: 14px 16px;
  border: 1px solid var(--dd-line);
  border-radius: var(--dd-radius-xl);
  background: linear-gradient(180deg, #fff, var(--dd-surface-soft));
  box-shadow: var(--dd-shadow-xs);
  color: var(--dd-muted);
}

.command-search :deep(.el-input__wrapper) {
  box-shadow: none;
  background: transparent;
}

.command-section {
  display: grid;
  gap: 12px;
}

.command-section-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 12px;
  color: var(--dd-ink);
  font-size: 13px;
  font-weight: 700;
}

.command-section-note {
  color: var(--dd-muted);
  font-weight: 500;
}

.command-grid,
.command-results {
  display: grid;
  gap: 10px;
}

.command-grid {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.command-item,
.command-result {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  gap: 12px;
  align-items: center;
  padding: 14px 16px;
  border: 1px solid var(--dd-line);
  border-radius: var(--dd-radius-xl);
  background: #fff;
  color: var(--dd-ink);
  text-align: left;
  cursor: pointer;
  transition:
    border-color 0.16s ease,
    transform 0.16s ease,
    box-shadow 0.16s ease;
}

.command-item:hover,
.command-result:hover {
  border-color: #bfd5ed;
  transform: translateY(-1px);
  box-shadow: var(--dd-shadow-sm);
}

.command-item-icon {
  display: grid;
  place-items: center;
  width: 36px;
  height: 36px;
  border-radius: 10px;
  background: var(--dd-primary-soft);
  color: var(--dd-primary);
}

.command-item-copy,
.command-result-copy {
  display: grid;
  gap: 4px;
  min-width: 0;
}

.command-item-copy strong,
.command-result-copy strong {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 14px;
}

.command-item-copy small,
.command-result-copy small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--dd-muted);
  font-size: 12px;
}

.command-empty {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 18px 16px;
  border: 1px dashed var(--dd-line);
  border-radius: var(--dd-radius-xl);
  color: var(--dd-muted);
  background: var(--dd-surface-soft);
}

@media (max-width: 720px) {
  .command-grid {
    grid-template-columns: 1fr;
  }

  .command-item,
  .command-result {
    grid-template-columns: auto minmax(0, 1fr);
  }

  .command-item > .app-kbd,
  .command-result > .status-pill {
    justify-self: start;
    grid-column: 2;
  }
}
</style>
