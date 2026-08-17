<script setup lang="ts">
import { useRouter } from "vue-router";

const router = useRouter();

function goHome() {
  void router.push("/dashboard");
}

function goBack() {
  if (window.history.length > 1) {
    router.back();
    return;
  }
  goHome();
}
</script>

<template>
  <div class="notfound-page">
    <div class="notfound-card">
      <div class="notfound-code">404</div>
      <h1 class="notfound-title">页面不存在</h1>
      <p class="notfound-hint">
        链接可能已经失效，或者你没有访问该页面的权限。
      </p>
      <div class="notfound-actions">
        <ElButton type="primary" @click="goHome">返回工作台</ElButton>
        <ElButton @click="goBack">返回上一页</ElButton>
      </div>
    </div>
  </div>
</template>

<style scoped>
.notfound-page {
  display: grid;
  place-items: center;
  min-height: 100vh;
  padding: 24px;
}

.notfound-card {
  width: min(440px, 100%);
  padding: 40px 32px;
  border: 1px solid var(--dd-line);
  border-radius: 24px;
  background: #fff;
  box-shadow: var(--dd-shadow-sm);
  text-align: center;
}

.notfound-code {
  color: var(--dd-primary, #124b87);
  font-size: 56px;
  font-weight: 760;
  line-height: 1;
}

.notfound-title {
  margin: 16px 0 8px;
  font-size: 20px;
  font-weight: 700;
}

.notfound-hint {
  margin: 0 0 24px;
  color: var(--dd-muted, #61748d);
  font-size: 14px;
}

.notfound-actions {
  display: flex;
  justify-content: center;
  gap: 12px;
}
</style>
