<script setup lang="ts">
import { ElMessage, ElMessageBox } from "element-plus";
import { computed, onMounted, ref, watch } from "vue";

import AppLayout from "@/components/AppLayout.vue";
import api from "@/api";
import { formatDateTime } from "@/utils/format";
import { extractError } from "@/utils/error";
import { useAuthStore } from "@/stores/auth";

type ProjectItem = {
  id: string;
  name: string;
  my_role?: string;
  owner?: { display_name?: string };
};

type MemberItem = {
  id: string;
  user_id: string;
  display_name: string;
  username: string;
  project_role: string;
};

type InviteItem = {
  id: string;
  token: string;
  project_role: string;
  status: string;
  expires_at: string;
  created_at: string;
};

const auth = useAuthStore();
const projects = ref<ProjectItem[]>([]);
const projectId = ref("");
const members = ref<MemberItem[]>([]);
const invites = ref<InviteItem[]>([]);
const users = ref<{ id: string; display_name: string; username?: string }[]>([]);
const loading = ref(false);
const addVisible = ref(false);
const addUserId = ref("");
const addRole = ref("contributor");
const submitting = ref(false);

const roleLabel: Record<string, string> = {
  owner: "负责人",
  manager: "协管人",
  contributor: "参与人",
  viewer: "只读",
};

const selectedProject = computed(() => projects.value.find((p) => p.id === projectId.value));
const canManage = computed(() => {
  if (auth.role === "admin") return true;
  const role = selectedProject.value?.my_role;
  return role === "owner" || role === "manager";
});

function inviteLink(token: string) {
  return `${window.location.origin}/join/${token}`;
}

function inviteStatus(item: InviteItem) {
  if (item.status === "revoked") return "已撤销";
  if (new Date(item.expires_at).getTime() < Date.now()) return "已过期";
  return "有效";
}

async function loadProjects() {
  const res = await api.get("/projects");
  projects.value = res.data?.data ?? [];
  if (!projectId.value && projects.value[0]) {
    projectId.value = projects.value[0].id;
  }
}

async function loadMembers() {
  if (!projectId.value) return;
  loading.value = true;
  try {
    const [memberRes, userRes] = await Promise.all([
      api.get(`/projects/${projectId.value}/members`),
      api.get("/users"),
    ]);
    members.value = memberRes.data?.data ?? [];
    users.value = userRes.data?.data ?? [];
    if (canManage.value) {
      const inviteRes = await api.get(`/projects/${projectId.value}/invites`);
      invites.value = inviteRes.data?.data ?? [];
    } else {
      invites.value = [];
    }
  } catch (err: unknown) {
    ElMessage.error(extractError(err, "加载成员失败"));
  } finally {
    loading.value = false;
  }
}

async function addMember() {
  if (!addUserId.value) {
    ElMessage.error("请选择要加入的用户");
    return;
  }
  submitting.value = true;
  try {
    await api.post(`/projects/${projectId.value}/members`, {
      user_id: addUserId.value,
      project_role: addRole.value,
    });
    ElMessage.success("已加入课题");
    addVisible.value = false;
    addUserId.value = "";
    addRole.value = "contributor";
    await loadMembers();
  } catch (err: unknown) {
    ElMessage.error(extractError(err, "添加成员失败"));
  } finally {
    submitting.value = false;
  }
}

async function removeMember(member: MemberItem) {
  try {
    await ElMessageBox.confirm(`确定将 ${member.display_name} 移出本课题？`, "移出成员");
  } catch {
    return;
  }
  try {
    await api.delete(`/projects/${projectId.value}/members/${member.id}`);
    ElMessage.success("已移出");
    await loadMembers();
  } catch (err: unknown) {
    ElMessage.error(extractError(err, "移出失败"));
  }
}

async function createInvite() {
  try {
    const res = await api.post(`/projects/${projectId.value}/invites`);
    const token = res.data?.data?.token;
    if (token) {
      const link = inviteLink(token);
      try {
        await navigator.clipboard.writeText(link);
        ElMessage.success("邀请链接已复制");
      } catch {
        await ElMessageBox.alert(link, "请复制邀请链接");
      }
    }
    await loadMembers();
  } catch (err: unknown) {
    ElMessage.error(extractError(err, "生成邀请失败"));
  }
}

async function copyInvite(item: InviteItem) {
  await navigator.clipboard.writeText(inviteLink(item.token));
  ElMessage.success("已复制邀请链接");
}

async function revokeInvite(item: InviteItem) {
  try {
    await api.post(`/projects/${projectId.value}/invites/${item.id}/revoke`);
    ElMessage.success("已撤销");
    await loadMembers();
  } catch (err: unknown) {
    ElMessage.error(extractError(err, "撤销失败"));
  }
}

watch(projectId, () => {
  void loadMembers();
});

onMounted(async () => {
  try {
    await loadProjects();
    await loadMembers();
  } catch (err: unknown) {
    ElMessage.error(extractError(err, "加载课题失败"));
  }
});
</script>

<template>
  <AppLayout>
    <div class="page-shell">
      <div class="page-header">
        <div>
          <div class="page-eyebrow">课题授权</div>
          <h1>课题成员</h1>
          <p class="page-desc">查看名单；负责人可添加已有账号，或生成邀请链接给新同学。</p>
        </div>
        <div class="header-actions">
          <ElSelect v-model="projectId" placeholder="选择课题" style="min-width: 220px">
            <ElOption v-for="p in projects" :key="p.id" :label="p.name" :value="p.id" />
          </ElSelect>
          <ElButton v-if="canManage" type="primary" @click="addVisible = true">添加成员</ElButton>
          <ElButton v-if="canManage" @click="createInvite">生成邀请链接</ElButton>
        </div>
      </div>

      <section class="page-card">
        <div class="section-title">成员名单</div>
        <ElTable :data="members" v-loading="loading" empty-text="暂无成员">
          <ElTableColumn prop="display_name" label="姓名" />
          <ElTableColumn label="账号">
            <template #default="{ row }">{{ row.account || row.username || "暂无账号" }}</template>
          </ElTableColumn>
          <ElTableColumn label="课题角色">
            <template #default="{ row }">{{ roleLabel[row.project_role] || row.project_role }}</template>
          </ElTableColumn>
          <ElTableColumn v-if="canManage" label="操作" width="100">
            <template #default="{ row }">
              <ElButton
                v-if="row.project_role !== 'owner'"
                link
                type="danger"
                @click="removeMember(row)"
              >
                移出
              </ElButton>
            </template>
          </ElTableColumn>
        </ElTable>
      </section>

      <section v-if="canManage" class="page-card" style="margin-top: 16px">
        <div class="section-title">邀请链接</div>
        <p class="page-desc">链接 7 天内有效，可多人使用。发给尚未加入本课题的同学。</p>
        <ElTable :data="invites" empty-text="尚未生成邀请">
          <ElTableColumn label="状态" width="100">
            <template #default="{ row }">{{ inviteStatus(row) }}</template>
          </ElTableColumn>
          <ElTableColumn label="过期时间" width="200">
            <template #default="{ row }">{{ formatDateTime(row.expires_at) }}</template>
          </ElTableColumn>
          <ElTableColumn label="操作" width="180">
            <template #default="{ row }">
              <ElButton v-if="inviteStatus(row) === '有效'" link type="primary" @click="copyInvite(row)">
                复制
              </ElButton>
              <ElButton v-if="row.status === 'active'" link type="danger" @click="revokeInvite(row)">
                撤销
              </ElButton>
            </template>
          </ElTableColumn>
        </ElTable>
      </section>
    </div>

    <ElDialog v-model="addVisible" title="添加已有账号" width="420px">
      <ElForm label-position="top">
        <ElFormItem label="用户">
          <ElSelect v-model="addUserId" filterable placeholder="选择用户" style="width: 100%">
            <ElOption
              v-for="u in users"
              :key="u.id"
              :label="`${u.display_name}${u.username ? ' (' + u.username + ')' : ''}`"
              :value="u.id"
            />
          </ElSelect>
        </ElFormItem>
        <ElFormItem label="课题角色">
          <ElSelect v-model="addRole" style="width: 100%">
            <ElOption label="参与人" value="contributor" />
            <ElOption label="协管人" value="manager" />
            <ElOption label="只读" value="viewer" />
          </ElSelect>
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton @click="addVisible = false">取消</ElButton>
        <ElButton type="primary" :loading="submitting" @click="addMember">添加</ElButton>
      </template>
    </ElDialog>
  </AppLayout>
</template>

<style scoped>
.header-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}
</style>
