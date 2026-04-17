<script setup lang="ts">
import { ElMessage } from "element-plus";
import { onMounted, reactive, ref } from "vue";

import api from "@/api";
import AppLayout from "@/components/AppLayout.vue";
import { useAuthStore } from "@/stores/auth";

type UserProfile = {
  id: string;
  username: string;
  display_name: string;
  role: string;
  email?: string;
  phone?: string;
  wechat?: string;
  status: string;
  last_login_at?: string | null;
};

const auth = useAuthStore();
const loading = ref(false);
const saving = ref(false);
const profile = ref<UserProfile | null>(null);
const form = reactive({
  display_name: "",
  email: "",
  phone: "",
  wechat: "",
});

function fillForm(data: UserProfile) {
  profile.value = data;
  form.display_name = data.display_name;
  form.email = data.email ?? "";
  form.phone = data.phone ?? "";
  form.wechat = data.wechat ?? "";
  auth.updateProfile({
    username: data.username,
    displayName: data.display_name,
    role: data.role,
  });
}

async function loadProfile() {
  loading.value = true;
  try {
    const res = await api.get("/auth/me");
    fillForm(res.data?.data);
  } catch (err: any) {
    ElMessage.error(err.response?.data?.message ?? "加载个人信息失败");
  } finally {
    loading.value = false;
  }
}

async function saveProfile() {
  if (!form.display_name.trim()) {
    ElMessage.error("显示姓名不能为空");
    return;
  }

  saving.value = true;
  try {
    const res = await api.patch("/auth/me", {
      display_name: form.display_name,
      email: form.email,
      phone: form.phone,
      wechat: form.wechat,
    });
    fillForm(res.data?.data);
    ElMessage.success("个人信息已更新");
  } catch (err: any) {
    ElMessage.error(err.response?.data?.message ?? "保存个人信息失败");
  } finally {
    saving.value = false;
  }
}

onMounted(loadProfile);
</script>

<template>
  <AppLayout>
    <div class="page-shell">
      <div class="page-header">
        <div>
          <div class="page-eyebrow">账号中心</div>
          <h1>个人信息</h1>
          <p class="page-desc">维护显示姓名和联系方式，账号角色与权限保持只读。</p>
        </div>
      </div>

      <ElSkeleton v-if="loading" :rows="8" animated />
      <div v-else class="profile-grid">
        <section class="page-card profile-panel">
          <div class="section-title">账号信息</div>
          <div class="readonly-list">
            <div class="readonly-item">
              <span>登录账号</span>
              <strong>{{ profile?.username ?? "-" }}</strong>
            </div>
            <div class="readonly-item">
              <span>全局角色</span>
              <strong>{{ profile?.role ?? "-" }}</strong>
            </div>
            <div class="readonly-item">
              <span>账号状态</span>
              <strong>{{ profile?.status ?? "-" }}</strong>
            </div>
            <div class="readonly-item">
              <span>最后登录</span>
              <strong>{{ profile?.last_login_at ?? "-" }}</strong>
            </div>
          </div>
        </section>

        <section class="page-card profile-panel">
          <div class="section-title">联系方式</div>
          <ElForm label-position="top" @submit.prevent="saveProfile">
            <ElFormItem label="显示姓名" required>
              <ElInput v-model="form.display_name" maxlength="64" show-word-limit />
            </ElFormItem>
            <ElFormItem label="邮箱">
              <ElInput v-model="form.email" maxlength="128" placeholder="name@example.com" />
            </ElFormItem>
            <ElFormItem label="手机号">
              <ElInput v-model="form.phone" maxlength="32" />
            </ElFormItem>
            <ElFormItem label="微信号">
              <ElInput v-model="form.wechat" maxlength="64" />
            </ElFormItem>
            <div class="form-actions">
              <ElButton @click="loadProfile">重置</ElButton>
              <ElButton type="primary" :loading="saving" @click="saveProfile">保存</ElButton>
            </div>
          </ElForm>
        </section>
      </div>
    </div>
  </AppLayout>
</template>

<style scoped>
h1 {
  margin: 0;
  font-size: 28px;
}

.page-desc {
  margin: 6px 0 0;
  color: #61748d;
}

.profile-grid {
  display: grid;
  grid-template-columns: minmax(260px, 360px) minmax(0, 1fr);
  gap: 18px;
}

.profile-panel {
  padding: 20px;
}

.section-title {
  margin-bottom: 16px;
  font-size: 16px;
  font-weight: 700;
}

.readonly-list {
  display: grid;
  gap: 12px;
}

.readonly-item {
  display: grid;
  gap: 4px;
  padding-bottom: 12px;
  border-bottom: 1px solid rgba(16, 36, 62, 0.08);
}

.readonly-item span {
  color: #61748d;
  font-size: 13px;
}

.readonly-item strong {
  overflow-wrap: anywhere;
  font-weight: 700;
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}

@media (max-width: 900px) {
  .profile-grid {
    grid-template-columns: 1fr;
  }
}
</style>
