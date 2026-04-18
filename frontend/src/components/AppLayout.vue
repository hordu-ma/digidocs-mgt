<script setup lang="ts">
import {
  ArrowDown,
  ArrowRight,
  Bell,
  ChatDotRound,
  Connection,
  DataBoard,
  Document,
  Folder,
  Search,
  Setting,
} from "@element-plus/icons-vue";
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import { useRoute, useRouter } from "vue-router";

import GlobalCommandDialog from "@/components/GlobalCommandDialog.vue";
import { useAuthStore } from "@/stores/auth";

const route = useRoute();
const router = useRouter();
const auth = useAuthStore();
const commandVisible = ref(false);

const isAdmin = computed(() => auth.role === "admin");

const menus = [
  { label: "总览", path: "/dashboard", icon: DataBoard, caption: "负责人工作台" },
  { label: "文档", path: "/documents", icon: Document, caption: "文档资产与流转" },
  { label: "数据", path: "/data", icon: Folder, caption: "图片、模型与数据文件" },
  { label: "交接", path: "/handovers", icon: Connection, caption: "成员交接闭环" },
  { label: "助手", path: "/assistant", icon: ChatDotRound, caption: "可信 AI 工作区" },
];

const workbenchActions = [
  {
    label: "打开文档工作台",
    caption: "进入文档筛选与流转视图",
    icon: Document,
    path: "/documents",
  },
  {
    label: "推进交接任务",
    caption: "优先处理待确认交接单",
    icon: Connection,
    path: "/handovers",
  },
  {
    label: "进入 OpenClaw",
    caption: "在受控范围内提问与整理",
    icon: ChatDotRound,
    path: "/assistant",
  },
];

const routeMeta = computed(() => {
  const path = route.path;
  if (path.startsWith("/documents/")) {
    return {
      title: "文档档案",
      caption: "版本、责任人与流转记录",
    };
  }
  if (path.startsWith("/documents")) {
    return {
      title: "文档资产库",
      caption: "按课题沉淀团队文档",
    };
  }
  if (path.startsWith("/data")) {
    return {
      title: "数据资产库",
      caption: "以项目为核心的轻量文件仓库",
    };
  }
  if (path.startsWith("/handovers")) {
    return {
      title: "工作交接",
      caption: "确认资料范围与接收责任",
    };
  }
  if (path.startsWith("/assistant")) {
    return {
      title: "OpenClaw 助手",
      caption: "基于受控上下文生成建议",
    };
  }
  if (path.startsWith("/admin")) {
    return {
      title: "系统管理",
      caption: "用户、空间与项目成员",
    };
  }
  if (path.startsWith("/profile")) {
    return {
      title: "个人信息",
      caption: "联系方式与账号资料",
    };
  }
  return {
    title: "负责人总览",
    caption: "课题文档、流转和交接风险",
  };
});

const roleLabel = computed(() => {
  const labels: Record<string, string> = {
    admin: "平台管理员",
    project_lead: "课题负责人",
    member: "成员",
  };
  return labels[auth.role || ""] || auth.role || "-";
});

const breadcrumbs = computed(() => {
  const crumbs = [{ label: "工作台", path: "/dashboard" }];
  const path = route.path;
  if (path.startsWith("/documents/")) {
    crumbs.push({ label: "文档资产库", path: "/documents" });
    crumbs.push({ label: "文档档案", path });
    return crumbs;
  }
  const menu = menus.find((item) => path === item.path || path.startsWith(`${item.path}/`));
  if (menu) {
    crumbs.push({ label: menu.label, path: menu.path });
  } else if (path.startsWith("/admin")) {
    crumbs.push({ label: "系统管理", path: "/admin" });
  } else if (path.startsWith("/profile")) {
    crumbs.push({ label: "个人信息", path: "/profile" });
  }
  return crumbs;
});

function openCommandDialog() {
  commandVisible.value = true;
}

function handleGlobalKeydown(event: KeyboardEvent) {
  if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") {
    event.preventDefault();
    openCommandDialog();
  }
}

function handleUserCommand(command: string) {
  if (command === "profile") {
    void router.push("/profile");
    return;
  }
  if (command === "logout") {
    auth.logout();
    void router.push("/login");
  }
}

onMounted(() => {
  window.addEventListener("keydown", handleGlobalKeydown);
});

onBeforeUnmount(() => {
  window.removeEventListener("keydown", handleGlobalKeydown);
});
</script>

<template>
  <div class="layout">
    <aside class="sidebar">
      <div class="brand">
        <div class="brand-mark">DG</div>
        <div>
          <div class="brand-title">DigiDocs</div>
          <div class="brand-subtitle">文档资产平台</div>
        </div>
      </div>
      <nav class="nav">
        <RouterLink
          v-for="menu in menus"
          :key="menu.path"
          :to="menu.path"
          class="nav-item"
          :class="{ active: route.path === menu.path || route.path.startsWith(`${menu.path}/`) }"
        >
          <ElIcon :size="17"><component :is="menu.icon" /></ElIcon>
          {{ menu.label }}
        </RouterLink>
      </nav>
      <section class="sidebar-workbench">
        <div class="sidebar-section-label">快捷工作</div>
        <button
          v-for="action in workbenchActions"
          :key="action.path"
          class="workbench-card"
          type="button"
          @click="router.push(action.path)"
        >
          <span class="workbench-icon">
            <ElIcon :size="16"><component :is="action.icon" /></ElIcon>
          </span>
          <span class="workbench-copy">
            <strong>{{ action.label }}</strong>
            <small>{{ action.caption }}</small>
          </span>
          <ElIcon class="workbench-arrow"><ArrowRight /></ElIcon>
        </button>
      </section>
      <div v-if="isAdmin" class="nav-admin">
        <div class="nav-divider"></div>
        <RouterLink
          to="/admin"
          class="nav-item nav-item-admin"
          :class="{ active: route.path === '/admin' }"
        >
          <ElIcon :size="16"><Setting /></ElIcon>
          系统管理
        </RouterLink>
      </div>
    </aside>
    <main class="content">
      <header class="topbar">
        <div class="topbar-context">
          <div class="topbar-breadcrumbs">
            <RouterLink
              v-for="(item, index) in breadcrumbs"
              :key="item.path"
              :to="item.path"
              class="topbar-crumb"
            >
              <span>{{ item.label }}</span>
              <ElIcon v-if="index < breadcrumbs.length - 1"><ArrowRight /></ElIcon>
            </RouterLink>
          </div>
          <div class="topbar-title">{{ routeMeta.title }}</div>
          <div class="topbar-caption">{{ routeMeta.caption }}</div>
        </div>
        <div class="topbar-tools">
          <button class="search-trigger" type="button" @click="openCommandDialog">
            <ElIcon><Search /></ElIcon>
            <span>搜索 / 跳转</span>
            <span class="app-kbd">Ctrl K</span>
          </button>
          <span class="topbar-signal">
            <ElIcon><Bell /></ElIcon>
            <span>工作流在线</span>
          </span>
          <ElDropdown trigger="click" @command="handleUserCommand">
            <button class="user-menu" type="button">
              <span class="user-avatar">{{ auth.displayName.slice(0, 1) || "用" }}</span>
              <span class="user-meta">
                <span class="user-name">{{ auth.displayName || auth.username || "当前用户" }}</span>
                <span class="user-role">{{ roleLabel }}</span>
              </span>
              <ElIcon><ArrowDown /></ElIcon>
            </button>
            <template #dropdown>
              <ElDropdownMenu>
                <ElDropdownItem command="profile">个人信息</ElDropdownItem>
                <ElDropdownItem divided command="logout">退出登录</ElDropdownItem>
              </ElDropdownMenu>
            </template>
          </ElDropdown>
        </div>
      </header>
      <slot />
    </main>
  </div>
  <GlobalCommandDialog
    v-model="commandVisible"
    :is-admin="isAdmin"
  />
</template>

<style scoped>
.layout {
  display: grid;
  grid-template-columns: 280px minmax(0, 1fr);
  min-height: 100vh;
}

.sidebar {
  position: sticky;
  top: 0;
  display: flex;
  flex-direction: column;
  height: 100vh;
  gap: 20px;
  height: 100vh;
  padding: 24px 18px;
  border-right: 1px solid var(--dd-line);
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.96), rgba(245, 248, 252, 0.94)),
    #f9fbfd;
}

.brand {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 0 8px;
  margin-bottom: 8px;
}

.brand-mark {
  display: grid;
  place-items: center;
  width: 42px;
  height: 42px;
  border-radius: 10px;
  background: linear-gradient(145deg, var(--dd-primary), var(--dd-primary-strong));
  color: #fff;
  font-weight: 760;
  box-shadow: 0 10px 20px rgba(18, 75, 135, 0.18);
}

.brand-title {
  font-size: 18px;
  font-weight: 760;
  letter-spacing: 0;
}

.brand-subtitle {
  color: var(--dd-muted);
  font-size: 12px;
}

.nav {
  display: grid;
  gap: 6px;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 52px;
  padding: 0 14px;
  border-radius: 12px;
  color: var(--dd-ink-2);
  font-weight: 650;
  transition:
    background 0.16s ease,
    color 0.16s ease,
    transform 0.16s ease,
    box-shadow 0.16s ease;
}

.nav-item:hover {
  background: var(--dd-primary-soft);
  color: var(--dd-primary-strong);
  box-shadow: var(--dd-shadow-xs);
}

.nav-item.active {
  background: var(--dd-primary);
  color: #fff;
  box-shadow: 0 12px 24px rgba(18, 75, 135, 0.18);
}

.sidebar-workbench {
  display: grid;
  gap: 10px;
}

.sidebar-section-label {
  padding: 0 8px;
  color: var(--dd-subtle);
  font-size: 12px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.workbench-card {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  gap: 10px;
  align-items: center;
  padding: 12px;
  border: 1px solid var(--dd-line);
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.78);
  color: var(--dd-ink);
  text-align: left;
  cursor: pointer;
  transition:
    border-color 0.16s ease,
    transform 0.16s ease,
    box-shadow 0.16s ease;
}

.workbench-card:hover {
  border-color: #bfd5ed;
  transform: translateY(-1px);
  box-shadow: var(--dd-shadow-sm);
}

.workbench-icon {
  display: grid;
  place-items: center;
  width: 36px;
  height: 36px;
  border-radius: 10px;
  background: var(--dd-primary-soft);
  color: var(--dd-primary);
}

.workbench-copy {
  display: grid;
  gap: 3px;
  min-width: 0;
}

.workbench-copy strong {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
}

.workbench-copy small {
  overflow: hidden;
  color: var(--dd-muted);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.workbench-arrow {
  color: var(--dd-subtle);
}

.nav-admin {
  margin-top: auto;
}

.nav-divider {
  margin: 16px 0 8px;
  border-top: 1px solid rgba(16, 36, 62, 0.1);
}

.nav-item-admin {
  display: flex;
  align-items: center;
  gap: 6px;
}

.content {
  min-width: 0;
}

.topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  min-height: 72px;
  padding: 18px 28px;
  border-bottom: 1px solid var(--dd-line);
  background: rgba(255, 255, 255, 0.78);
  backdrop-filter: blur(12px);
}

.topbar-context {
  min-width: 0;
}

.topbar-breadcrumbs {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 6px;
}

.topbar-crumb {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  color: var(--dd-muted);
  font-size: 12px;
  font-weight: 600;
}

.topbar-crumb:last-child {
  color: var(--dd-ink-2);
}

.topbar-title {
  color: var(--dd-ink);
  font-size: 18px;
  font-weight: 760;
}

.topbar-caption {
  margin-top: 2px;
  color: var(--dd-muted);
  font-size: 12px;
}

.topbar-tools {
  display: flex;
  align-items: center;
  gap: 12px;
}

.search-trigger,
.quick-create {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  min-height: 42px;
  padding: 0 12px;
  border: 1px solid var(--dd-line);
  border-radius: 12px;
  background: #fff;
  color: var(--dd-ink);
  cursor: pointer;
}

.search-trigger {
  min-width: 192px;
  justify-content: space-between;
}

.quick-create {
  background: linear-gradient(180deg, var(--dd-primary), var(--dd-primary-strong));
  border-color: transparent;
  color: #fff;
  box-shadow: 0 10px 22px rgba(18, 75, 135, 0.18);
}

.topbar-signal {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 40px;
  padding: 0 12px;
  border-radius: 999px;
  background: var(--dd-success-soft);
  color: var(--dd-success);
  font-size: 12px;
  font-weight: 700;
}

.user-menu {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  min-height: 44px;
  padding: 6px 10px 6px 6px;
  border: 1px solid var(--dd-line);
  border-radius: 8px;
  background: #fff;
  color: var(--dd-ink);
  cursor: pointer;
}

.user-avatar {
  display: grid;
  place-items: center;
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: var(--dd-primary);
  color: #fff;
  font-weight: 760;
}

.user-meta {
  display: grid;
  min-width: 88px;
  text-align: left;
}

.user-name {
  font-size: 14px;
  font-weight: 750;
}

.user-role {
  color: var(--dd-muted);
  font-size: 12px;
}

@media (max-width: 900px) {
  .layout {
    grid-template-columns: 1fr;
  }

  .sidebar {
    position: static;
    height: auto;
    border-right: 0;
    border-bottom: 1px solid var(--dd-line);
  }

  .nav {
    grid-template-columns: repeat(5, minmax(0, 1fr));
  }

  .topbar {
    padding: 14px 18px;
  }

  .topbar,
  .topbar-tools {
    flex-wrap: wrap;
  }

  .topbar-tools {
    width: 100%;
  }

  .search-trigger {
    min-width: 0;
    flex: 1;
  }

  .sidebar-workbench {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}
</style>
