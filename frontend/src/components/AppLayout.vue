<script setup lang="ts">
import { ArrowDown } from "@element-plus/icons-vue";
import { useRoute, useRouter } from "vue-router";

import { useAuthStore } from "@/stores/auth";

const route = useRoute();
const router = useRouter();
const auth = useAuthStore();

const menus = [
  { label: "总览", path: "/dashboard" },
  { label: "文档", path: "/documents" },
  { label: "交接", path: "/handovers" },
  { label: "助手", path: "/assistant" },
];

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
</script>

<template>
  <div class="layout">
    <aside class="sidebar">
      <div class="brand">
        <div class="brand-mark">DG</div>
        <div>
          <div class="brand-title">DigiDocs</div>
          <div class="brand-subtitle">课题组文档平台</div>
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
          {{ menu.label }}
        </RouterLink>
      </nav>
    </aside>
    <main class="content">
      <header class="topbar">
        <div></div>
        <ElDropdown trigger="click" @command="handleUserCommand">
          <button class="user-menu" type="button">
            <span class="user-avatar">{{ auth.displayName.slice(0, 1) || "用" }}</span>
            <span class="user-meta">
              <span class="user-name">{{ auth.displayName || auth.username || "当前用户" }}</span>
              <span class="user-role">{{ auth.role || "-" }}</span>
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
      </header>
      <slot />
    </main>
  </div>
</template>

<style scoped>
.layout {
  display: grid;
  grid-template-columns: 240px 1fr;
  min-height: 100vh;
}

.sidebar {
  padding: 28px 18px;
  border-right: 1px solid rgba(16, 36, 62, 0.08);
  background:
    radial-gradient(circle at top, rgba(52, 120, 246, 0.16), transparent 45%),
    #f7fafc;
}

.brand {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 28px;
}

.brand-mark {
  display: grid;
  place-items: center;
  width: 44px;
  height: 44px;
  border-radius: 14px;
  background: #123e73;
  color: #fff;
  font-weight: 700;
}

.brand-title {
  font-size: 18px;
  font-weight: 700;
}

.brand-subtitle {
  color: #61748d;
  font-size: 12px;
}

.nav {
  display: grid;
  gap: 8px;
}

.nav-item {
  padding: 12px 14px;
  border-radius: 12px;
  color: #48607e;
}

.nav-item.active {
  background: #123e73;
  color: #fff;
}

.content {
  padding: 24px;
}

.topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}

.user-menu {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  min-height: 44px;
  padding: 6px 10px 6px 6px;
  border: 1px solid rgba(16, 36, 62, 0.1);
  border-radius: 12px;
  background: #fff;
  color: #10243e;
  cursor: pointer;
}

.user-avatar {
  display: grid;
  place-items: center;
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: #123e73;
  color: #fff;
  font-weight: 700;
}

.user-meta {
  display: grid;
  min-width: 88px;
  text-align: left;
}

.user-name {
  font-size: 14px;
  font-weight: 700;
}

.user-role {
  color: #61748d;
  font-size: 12px;
}

@media (max-width: 900px) {
  .layout {
    grid-template-columns: 1fr;
  }

  .sidebar {
    border-right: 0;
    border-bottom: 1px solid rgba(16, 36, 62, 0.08);
  }
}
</style>
