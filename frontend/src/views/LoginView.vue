<script setup lang="ts">
import { ElMessage } from "element-plus";
import { reactive, ref } from "vue";
import { useRoute, useRouter } from "vue-router";

import axios from "axios";
import { useAuthStore } from "@/stores/auth";
import { extractError } from "@/utils/error";

const router = useRouter();
const route = useRoute();
const auth = useAuthStore();
const loading = ref(false);
const form = reactive({
  identifier: "",
  password: "",
});

async function submit() {
  if (!form.identifier || !form.password) {
    ElMessage.error("请输入账号和密码");
    return;
  }

  loading.value = true;
  try {
    const res = await axios.post("/api/v1/auth/login", {
      identifier: form.identifier,
      password: form.password,
    });
    const data = res.data?.data;
    auth.login({
      token: data.access_token,
      id: data.user.id,
      username: data.user.username ?? "",
      displayName: data.user.display_name,
      role: data.user.role,
    });
    const redirect = typeof route.query.redirect === "string" ? route.query.redirect : "";
    void router.push(redirect.startsWith("/") ? redirect : "/dashboard");
  } catch (err: unknown) {
    ElMessage.error(extractError(err, "登录失败，请检查账号和密码"));
  } finally {
    loading.value = false;
  }
}
</script>

<template>
  <div class="login-page">
    <ElCard class="login-card">
      <div class="login-title">课题组文档资产平台</div>
      <div class="login-subtitle">
        成员协作、交接沉淀、负责人总览、OpenClaw 助手
      </div>
      <ElForm label-position="top" @submit.prevent="submit">
        <ElFormItem label="账号（用户名 / 手机号 / 邮箱）">
          <ElInput v-model="form.identifier" placeholder="用户名、手机号或邮箱" />
        </ElFormItem>
        <ElFormItem label="密码">
          <ElInput v-model="form.password" show-password type="password" />
        </ElFormItem>
        <ElButton type="primary" style="width: 100%" :loading="loading" @click="submit">
          登录
        </ElButton>
      </ElForm>
      <div class="auth-switch">
        没有账号？
        <RouterLink to="/register">去注册</RouterLink>
      </div>
    </ElCard>
    <footer class="auth-footer">
      系统管理员：<a href="mailto:maliguo@outlook.com">maliguo@outlook.com</a>
    </footer>
  </div>
</template>

<style scoped>
.login-page {
  display: grid;
  place-items: center;
  min-height: 100vh;
  padding: 24px;
}

.login-card {
  width: min(420px, 100%);
  border-radius: 24px;
}

.login-title {
  font-size: 28px;
  font-weight: 700;
  margin-bottom: 10px;
}

.login-subtitle {
  margin-bottom: 24px;
  color: #61748d;
}

.auth-switch {
  margin-top: 16px;
  text-align: center;
  color: #61748d;
}

.auth-switch a {
  color: var(--dd-primary, #124b87);
  font-weight: 600;
}

.auth-footer {
  margin-top: 20px;
  color: #94a3b8;
  font-size: 12px;
  text-align: center;
}

.auth-footer a {
  color: #61748d;
  font-weight: 600;
  text-decoration: underline;
  text-underline-offset: 2px;
}

.auth-footer a:hover {
  color: var(--dd-primary, #124b87);
}
</style>
