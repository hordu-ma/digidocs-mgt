<script setup lang="ts">
// Shared frame for the login and register pages: a brand panel next to the
// form on wide screens, stacked above it on phones. These pages were a lone
// card on a blank background — the first screen external users see.
const year = new Date().getFullYear();
const highlights = [
  { title: "文档不随人走", text: "版本、责任人和流转全程留痕，毕业交接自动归集名下资料" },
  { title: "负责人一屏掌握", text: "待办、风险提示和下载活跃度集中呈现" },
  { title: "AI 助手可溯源", text: "问答先检索正文再回答，每句结论都标注出处" },
];
</script>

<template>
  <div class="auth-shell">
    <section class="auth-brand">
      <div class="auth-logo">
        <span class="auth-mark">DG</span>
        <span>
          <strong>DigiDocs</strong>
          <small>课题组文档资产平台</small>
        </span>
      </div>
      <h1>让课题资料成为团队资产</h1>
      <ul class="auth-points">
        <li v-for="item in highlights" :key="item.title">
          <strong>{{ item.title }}</strong>
          <span>{{ item.text }}</span>
        </li>
      </ul>
    </section>
    <section class="auth-form">
      <slot />
      <footer class="auth-footer">
        <div>系统管理员：<a href="mailto:maliguo@outlook.com">maliguo@outlook.com</a></div>
        <div>© {{ year }} DigiDocs · 私有化部署，数据留在本单位</div>
      </footer>
    </section>
  </div>
</template>

<style scoped>
.auth-shell {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(360px, 480px);
  min-height: 100vh;
  background: #f4f7fb;
}

.auth-brand {
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 28px;
  padding: 56px clamp(24px, 6vw, 96px);
  color: #fff;
  background: linear-gradient(150deg, #0f3d6e 0%, #124b87 55%, #1d6aa8 100%);
}

.auth-logo {
  display: flex;
  align-items: center;
  gap: 12px;
}

.auth-logo strong {
  display: block;
  font-size: 22px;
}

.auth-logo small {
  color: rgba(255, 255, 255, 0.75);
}

.auth-mark {
  display: inline-grid;
  place-items: center;
  width: 44px;
  height: 44px;
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.16);
  font-weight: 800;
}

.auth-brand h1 {
  margin: 0;
  font-size: clamp(26px, 3vw, 38px);
  line-height: 1.25;
}

.auth-points {
  display: grid;
  gap: 18px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.auth-points li {
  display: grid;
  gap: 4px;
  padding-left: 14px;
  border-left: 3px solid rgba(255, 255, 255, 0.4);
}

.auth-points span {
  color: rgba(255, 255, 255, 0.8);
  font-size: 14px;
}

.auth-form {
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 20px;
  padding: 32px 24px;
}

.auth-footer {
  display: grid;
  gap: 4px;
  color: #94a3b8;
  font-size: 12px;
  text-align: center;
}

.auth-footer a {
  color: #61748d;
  font-weight: 600;
}

@media (max-width: 820px) {
  .auth-shell {
    grid-template-columns: 1fr;
  }

  .auth-brand {
    gap: 16px;
    padding: 28px 16px;
  }

  .auth-points {
    display: none;
  }

  .auth-form {
    padding: 20px 16px 32px;
  }
}
</style>
