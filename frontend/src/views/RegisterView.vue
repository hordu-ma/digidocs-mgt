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
  email: "",
  phone: "",
  displayName: "",
  password: "",
  confirmPassword: "",
});

function validate(): string | null {
  if (!form.email.trim() && !form.phone.trim()) {
    return "请填写手机号或邮箱，至少一项";
  }
  if (form.email.trim() && !form.email.includes("@")) {
    return "邮箱格式不正确";
  }
  if (form.phone.trim() && !/^1[3-9]\d{9}$/.test(form.phone.trim())) {
    return "请输入有效的中国大陆手机号";
  }
  if (!form.displayName.trim()) {
    return "请填写显示姓名";
  }
  if (form.password.length < 8) {
    return "密码至少 8 位";
  }
  if (form.password !== form.confirmPassword) {
    return "两次输入的密码不一致";
  }
  return null;
}

async function submit() {
  const message = validate();
  if (message) {
    ElMessage.error(message);
    return;
  }

  loading.value = true;
  try {
    const res = await axios.post("/api/v1/auth/register", {
      email: form.email.trim(),
      phone: form.phone.trim(),
      password: form.password,
      display_name: form.displayName.trim(),
    });
    const data = res.data?.data;
    auth.login({
      token: data.access_token,
      id: data.user.id,
      username: data.user.username ?? "",
      displayName: data.user.display_name,
      role: data.user.role,
    });
    ElMessage.success("注册成功");
    const redirect = typeof route.query.redirect === "string" ? route.query.redirect : "";
    void router.push(redirect.startsWith("/") ? redirect : "/dashboard");
  } catch (err: unknown) {
    ElMessage.error(extractError(err, "注册失败，请检查填写信息"));
  } finally {
    loading.value = false;
  }
}
</script>

<template>
  <div class="login-page">
    <ElCard class="login-card">
      <div class="login-title">创建账号</div>
      <div class="login-subtitle">使用手机号或邮箱自助注册，注册成功后将自动登录</div>
      <ElForm label-position="top" @submit.prevent="submit">
        <ElFormItem label="显示姓名">
          <ElInput v-model="form.displayName" maxlength="64" placeholder="将展示在平台中的名称" />
        </ElFormItem>
        <ElFormItem label="手机号">
          <ElInput v-model="form.phone" maxlength="11" placeholder="中国大陆手机号，与邮箱至少填一项" />
        </ElFormItem>
        <ElFormItem label="邮箱">
          <ElInput v-model="form.email" placeholder="与手机号至少填一项" />
        </ElFormItem>
        <ElFormItem label="密码">
          <ElInput v-model="form.password" show-password type="password" placeholder="至少 8 位" />
        </ElFormItem>
        <ElFormItem label="确认密码">
          <ElInput v-model="form.confirmPassword" show-password type="password" placeholder="请再次输入密码" />
        </ElFormItem>
        <ElButton type="primary" style="width: 100%" :loading="loading" @click="submit">
          注册并登录
        </ElButton>
      </ElForm>
      <div class="auth-switch">
        已有账号？
        <RouterLink to="/login">去登录</RouterLink>
      </div>
    </ElCard>
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
</style>
