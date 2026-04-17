<script setup lang="ts">
import { ElMessage } from "element-plus";
import { reactive, ref } from "vue";
import { useRouter } from "vue-router";

import axios from "axios";
import { useAuthStore } from "@/stores/auth";

const router = useRouter();
const auth = useAuthStore();
const loading = ref(false);
const form = reactive({
  username: "admin",
  password: "",
});

async function submit() {
  if (!form.username || !form.password) {
    ElMessage.error("请输入用户名和密码");
    return;
  }

  loading.value = true;
  try {
    const res = await axios.post("/api/v1/auth/login", {
      username: form.username,
      password: form.password,
    });
    const data = res.data?.data;
    auth.login({
      token: data.access_token,
      id: data.user.id,
      username: data.user.username,
      displayName: data.user.display_name,
      role: data.user.role,
    });
    void router.push("/dashboard");
  } catch (err: any) {
    const msg = err.response?.data?.message ?? "登录失败，请检查用户名和密码";
    ElMessage.error(msg);
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
        <ElFormItem label="用户名">
          <ElInput v-model="form.username" />
        </ElFormItem>
        <ElFormItem label="密码">
          <ElInput v-model="form.password" show-password type="password" />
        </ElFormItem>
        <ElButton type="primary" style="width: 100%" @click="submit"
          >登录</ElButton
        >
      </ElForm>
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
</style>
