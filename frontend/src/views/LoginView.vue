<script setup lang="ts">
import { ElButton, ElCard, ElForm, ElFormItem, ElInput } from "element-plus";
import { reactive } from "vue";
import { useRouter } from "vue-router";

const router = useRouter();
const form = reactive({
  username: "admin",
  password: "",
});

function submit() {
  void router.push("/dashboard");
}
</script>

<template>
  <div class="login-page">
    <ElCard class="login-card">
      <div class="login-title">课题组文档资产平台</div>
      <div class="login-subtitle">成员协作、交接沉淀、负责人总览、OpenClaw 助手</div>
      <ElForm label-position="top" @submit.prevent="submit">
        <ElFormItem label="用户名">
          <ElInput v-model="form.username" />
        </ElFormItem>
        <ElFormItem label="密码">
          <ElInput v-model="form.password" show-password type="password" />
        </ElFormItem>
        <ElButton type="primary" style="width: 100%" @click="submit">登录</ElButton>
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
