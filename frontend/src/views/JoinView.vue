<script setup lang="ts">
import { loadRegistrationMode, registrationMode } from "@/utils/registration";
import { ElMessage } from "element-plus";
import { computed, onMounted, ref } from "vue";
import { useRoute, useRouter } from "vue-router";

import axios from "axios";
import api from "@/api";
import { useAuthStore } from "@/stores/auth";
import { extractError } from "@/utils/error";

const route = useRoute();
const router = useRouter();
const auth = useAuthStore();

const token = computed(() => String(route.params.token || ""));
const loading = ref(true);
const joining = ref(false);
const preview = ref<{
  project_name?: string;
  team_space_name?: string;
  project_role?: string;
  status?: string;
  expires_at?: string;
  expired?: boolean;
} | null>(null);
const errorText = ref("");

const inviteInvalid = computed(() => {
  if (!preview.value) return Boolean(errorText.value);
  return preview.value.status !== "active" || preview.value.expired;
});

const loginPath = computed(() => ({ path: "/login", query: { redirect: `/join/${token.value}` } }));
const registerPath = computed(() => ({ path: "/register", query: { redirect: `/join/${token.value}` } }));

const roleLabel: Record<string, string> = {
  owner: "负责人",
  manager: "协管人",
  contributor: "参与人",
  viewer: "只读",
};

onMounted(async () => {
  if (!token.value) {
    errorText.value = "邀请链接无效";
    loading.value = false;
    return;
  }
  try {
    const res = await axios.get(`/api/v1/invites/${token.value}`);
    preview.value = res.data?.data ?? null;
    if (auth.token && preview.value && !inviteInvalid.value) {
      await accept();
    }
  } catch (err: unknown) {
    errorText.value = extractError(err, "邀请不存在或已失效");
  } finally {
    loading.value = false;
  }
});

async function accept() {
  joining.value = true;
  try {
    await api.post(`/invites/${token.value}/accept`);
    ElMessage.success("已加入课题");
    void router.replace("/dashboard");
  } catch (err: unknown) {
    ElMessage.error(extractError(err, "加入课题失败"));
  } finally {
    joining.value = false;
  }
}

void loadRegistrationMode();
</script>

<template>
  <div class="login-page">
    <ElCard class="login-card">
      <div class="login-title">加入课题</div>
      <ElSkeleton v-if="loading" :rows="4" animated />
      <template v-else-if="preview && !inviteInvalid">
        <div class="login-subtitle">
          {{ preview.team_space_name }} · {{ preview.project_name }}
        </div>
        <p class="join-note">
          加入后你的课题角色为{{ roleLabel[preview.project_role || ""] || "参与人" }}。
        </p>
        <template v-if="auth.token">
          <ElButton type="primary" style="width: 100%" :loading="joining" @click="accept">
            确认加入
          </ElButton>
        </template>
        <template v-else>
          <p class="join-note">请先登录或注册，然后再加入该课题。</p>
          <ElButton type="primary" style="width: 100%; margin-bottom: 10px" @click="router.push(loginPath)">
            去登录
          </ElButton>
          <ElButton v-if="registrationMode !== 'closed'" style="width: 100%" @click="router.push(registerPath)">去注册</ElButton>
        </template>
      </template>
      <template v-else>
        <div class="login-subtitle">{{ errorText || "邀请已过期或已撤销" }}</div>
        <ElButton style="width: 100%" @click="router.push('/login')">返回登录</ElButton>
      </template>
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
  margin-bottom: 16px;
  color: #61748d;
}
.join-note {
  margin: 0 0 16px;
  color: #61748d;
  font-size: 14px;
}
</style>
