<script setup lang="ts">
import { ref, computed, onMounted, watch } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { Setting, User, OfficeBuilding, FolderOpened } from "@element-plus/icons-vue";
import { useAuthStore } from "@/stores/auth";
import AppLayout from "@/components/AppLayout.vue";

const auth = useAuthStore();
const API = "/api/v1";
const headers = computed(() => ({ Authorization: `Bearer ${auth.token}` }));
const activeTab = ref("team-spaces");

// ───── Team Spaces ─────
const teamSpaces = ref<any[]>([]);
const tsLoading = ref(false);
const tsDialogVisible = ref(false);
const tsForm = ref({ name: "", code: "", description: "" });
const tsSubmitting = ref(false);

async function loadTeamSpaces() {
  tsLoading.value = true;
  try {
    const res = await fetch(`${API}/team-spaces`, { headers: headers.value });
    const json = await res.json();
    teamSpaces.value = json.data ?? [];
  } finally {
    tsLoading.value = false;
  }
}

async function createTeamSpace() {
  tsSubmitting.value = true;
  try {
    const res = await fetch(`${API}/admin/team-spaces`, {
      method: "POST",
      headers: { ...headers.value, "Content-Type": "application/json" },
      body: JSON.stringify(tsForm.value),
    });
    if (!res.ok) {
      const json = await res.json();
      ElMessage.error(json.message || "创建失败");
      return;
    }
    ElMessage.success("团队空间创建成功");
    tsDialogVisible.value = false;
    tsForm.value = { name: "", code: "", description: "" };
    await loadTeamSpaces();
  } finally {
    tsSubmitting.value = false;
  }
}

// ───── Projects ─────
const projects = ref<any[]>([]);
const projLoading = ref(false);
const projDialogVisible = ref(false);
const projForm = ref({ team_space_id: "", name: "", code: "", owner_id: "", description: "" });
const projSubmitting = ref(false);

async function loadProjects() {
  projLoading.value = true;
  try {
    const res = await fetch(`${API}/projects`, { headers: headers.value });
    const json = await res.json();
    projects.value = json.data ?? [];
  } finally {
    projLoading.value = false;
  }
}

async function createProject() {
  projSubmitting.value = true;
  try {
    const res = await fetch(`${API}/admin/projects`, {
      method: "POST",
      headers: { ...headers.value, "Content-Type": "application/json" },
      body: JSON.stringify(projForm.value),
    });
    if (!res.ok) {
      const json = await res.json();
      ElMessage.error(json.message || "创建失败");
      return;
    }
    ElMessage.success("项目创建成功");
    projDialogVisible.value = false;
    projForm.value = { team_space_id: "", name: "", code: "", owner_id: "", description: "" };
    await loadProjects();
  } finally {
    projSubmitting.value = false;
  }
}

// ───── Users ─────
const users = ref<any[]>([]);
const userLoading = ref(false);
const userDialogVisible = ref(false);
const userEditMode = ref(false);
const userForm = ref({ id: "", username: "", password: "", confirm_password: "", display_name: "", role: "member", email: "", phone: "" });
const userSubmitting = ref(false);

async function loadUsers() {
  userLoading.value = true;
  try {
    const res = await fetch(`${API}/admin/users`, { headers: headers.value });
    const json = await res.json();
    users.value = json.data ?? [];
  } finally {
    userLoading.value = false;
  }
}

function openCreateUser() {
  userEditMode.value = false;
  userForm.value = { id: "", username: "", password: "", confirm_password: "", display_name: "", role: "member", email: "", phone: "" };
  userDialogVisible.value = true;
}

function openEditUser(u: any) {
  userEditMode.value = true;
  userForm.value = { id: u.id, username: u.username, password: "", confirm_password: "", display_name: u.display_name, role: u.role, email: u.email || "", phone: u.phone || "" };
  userDialogVisible.value = true;
}

async function submitUser() {
  userSubmitting.value = true;
  try {
    if (userForm.value.password !== userForm.value.confirm_password) {
      ElMessage.error("两次输入的密码不一致");
      return;
    }

    if (userEditMode.value) {
      const body: any = {};
      if (userForm.value.display_name) body.display_name = userForm.value.display_name;
      if (userForm.value.role) body.role = userForm.value.role;
      if (userForm.value.email !== undefined) body.email = userForm.value.email;
      if (userForm.value.phone !== undefined) body.phone = userForm.value.phone;
      if (userForm.value.password) body.password = userForm.value.password;
      const res = await fetch(`${API}/admin/users/${userForm.value.id}`, {
        method: "PATCH",
        headers: { ...headers.value, "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      if (!res.ok) {
        const json = await res.json();
        ElMessage.error(json.message || "更新失败");
        return;
      }
      ElMessage.success(userForm.value.password ? "用户更新成功，密码已重置" : "用户更新成功");
    } else {
      const res = await fetch(`${API}/admin/users`, {
        method: "POST",
        headers: { ...headers.value, "Content-Type": "application/json" },
        body: JSON.stringify({
          username: userForm.value.username,
          password: userForm.value.password,
          display_name: userForm.value.display_name,
          role: userForm.value.role,
          email: userForm.value.email,
          phone: userForm.value.phone,
        }),
      });
      if (!res.ok) {
        const json = await res.json();
        ElMessage.error(json.message || "创建失败");
        return;
      }
      ElMessage.success("用户创建成功");
    }
    userDialogVisible.value = false;
    await loadUsers();
  } finally {
    userSubmitting.value = false;
  }
}

const roleLabel: Record<string, string> = { admin: "管理员", project_lead: "课题负责人", member: "成员" };
const statusLabel: Record<string, string> = { active: "活跃", inactive: "已停用" };

// ───── Members ─────
const memberProjectId = ref("");
const members = ref<any[]>([]);
const memberLoading = ref(false);
const memberDialogVisible = ref(false);
const memberForm = ref({ user_id: "", project_role: "contributor" });
const memberSubmitting = ref(false);

async function loadMembers() {
  if (!memberProjectId.value) return;
  memberLoading.value = true;
  try {
    const res = await fetch(`${API}/admin/projects/${memberProjectId.value}/members`, { headers: headers.value });
    const json = await res.json();
    members.value = json.data ?? [];
  } finally {
    memberLoading.value = false;
  }
}

watch(memberProjectId, () => loadMembers());

async function addMember() {
  memberSubmitting.value = true;
  try {
    const res = await fetch(`${API}/admin/projects/${memberProjectId.value}/members`, {
      method: "POST",
      headers: { ...headers.value, "Content-Type": "application/json" },
      body: JSON.stringify(memberForm.value),
    });
    if (!res.ok) {
      const json = await res.json();
      ElMessage.error(json.message || "添加失败");
      return;
    }
    ElMessage.success("成员添加成功");
    memberDialogVisible.value = false;
    memberForm.value = { user_id: "", project_role: "contributor" };
    await loadMembers();
  } finally {
    memberSubmitting.value = false;
  }
}

async function updateMemberRole(m: any, newRole: string) {
  await fetch(`${API}/admin/projects/${memberProjectId.value}/members/${m.id}`, {
    method: "PATCH",
    headers: { ...headers.value, "Content-Type": "application/json" },
    body: JSON.stringify({ project_role: newRole }),
  });
  await loadMembers();
}

async function removeMember(m: any) {
  await ElMessageBox.confirm(`确定移除成员 ${m.display_name}？`, "确认", { type: "warning" });
  await fetch(`${API}/admin/projects/${memberProjectId.value}/members/${m.id}`, {
    method: "DELETE",
    headers: headers.value,
  });
  ElMessage.success("已移除");
  await loadMembers();
}

const projRoleLabel: Record<string, string> = { owner: "负责人", manager: "管理者", contributor: "贡献者", viewer: "查看者" };

// ───── Init ─────
onMounted(() => {
  loadTeamSpaces();
  loadProjects();
  loadUsers();
});
</script>

<template>
  <AppLayout>
  <div class="page-shell">
    <div class="page-header">
      <div>
        <div class="page-eyebrow">平台治理</div>
        <h1>系统管理</h1>
        <p>统一维护团队空间、项目、用户和项目成员关系。</p>
      </div>
    </div>

    <div class="admin-view page-card">
    <ElTabs v-model="activeTab" class="admin-tabs">
      <!-- ===== 团队空间 ===== -->
      <ElTabPane label="团队空间" name="team-spaces">
        <div class="tab-toolbar">
          <span class="tab-desc">管理团队空间，每个空间可包含多个项目</span>
          <ElButton type="primary" @click="tsDialogVisible = true">
            <ElIcon><OfficeBuilding /></ElIcon> 新建空间
          </ElButton>
        </div>
        <ElTable :data="teamSpaces" v-loading="tsLoading" stripe style="width:100%" empty-text="暂无团队空间">
          <ElTableColumn prop="name" label="名称" />
          <ElTableColumn prop="code" label="代码" width="180" />
        </ElTable>

        <ElDialog v-model="tsDialogVisible" title="新建团队空间" width="480">
          <ElForm label-width="80px">
            <ElFormItem label="名称"><ElInput v-model="tsForm.name" placeholder="如：智能系统实验室" /></ElFormItem>
            <ElFormItem label="代码"><ElInput v-model="tsForm.code" placeholder="如：smart-lab" /></ElFormItem>
            <ElFormItem label="描述"><ElInput v-model="tsForm.description" type="textarea" :rows="2" /></ElFormItem>
          </ElForm>
          <template #footer>
            <ElButton @click="tsDialogVisible = false">取消</ElButton>
            <ElButton type="primary" :loading="tsSubmitting" @click="createTeamSpace">创建</ElButton>
          </template>
        </ElDialog>
      </ElTabPane>

      <!-- ===== 项目管理 ===== -->
      <ElTabPane label="项目管理" name="projects">
        <div class="tab-toolbar">
          <span class="tab-desc">管理项目，每个项目归属于一个团队空间</span>
          <ElButton type="primary" @click="projDialogVisible = true">
            <ElIcon><FolderOpened /></ElIcon> 新建项目
          </ElButton>
        </div>
        <ElTable :data="projects" v-loading="projLoading" stripe style="width:100%" empty-text="暂无项目">
          <ElTableColumn prop="name" label="项目名称" />
          <ElTableColumn prop="code" label="代码" width="160" />
          <ElTableColumn label="负责人" width="140">
            <template #default="{ row }">{{ row.owner?.display_name || "-" }}</template>
          </ElTableColumn>
        </ElTable>

        <ElDialog v-model="projDialogVisible" title="新建项目" width="520">
          <ElForm label-width="100px">
            <ElFormItem label="团队空间">
              <ElSelect v-model="projForm.team_space_id" placeholder="选择空间" style="width:100%">
                <ElOption v-for="ts in teamSpaces" :key="ts.id" :label="ts.name" :value="ts.id" />
              </ElSelect>
            </ElFormItem>
            <ElFormItem label="项目名称"><ElInput v-model="projForm.name" placeholder="如：wuhao-ai" /></ElFormItem>
            <ElFormItem label="项目代码"><ElInput v-model="projForm.code" placeholder="如：wuhao-ai" /></ElFormItem>
            <ElFormItem label="负责人">
              <ElSelect v-model="projForm.owner_id" placeholder="选择负责人" filterable style="width:100%">
                <ElOption v-for="u in users" :key="u.id" :label="`${u.display_name} (${u.username})`" :value="u.id" />
              </ElSelect>
            </ElFormItem>
            <ElFormItem label="描述"><ElInput v-model="projForm.description" type="textarea" :rows="2" /></ElFormItem>
          </ElForm>
          <template #footer>
            <ElButton @click="projDialogVisible = false">取消</ElButton>
            <ElButton type="primary" :loading="projSubmitting" @click="createProject">创建</ElButton>
          </template>
        </ElDialog>
      </ElTabPane>

      <!-- ===== 用户管理 ===== -->
      <ElTabPane label="用户管理" name="users">
        <div class="tab-toolbar">
          <span class="tab-desc">创建和编辑系统用户，管理角色与状态</span>
          <ElButton type="primary" @click="openCreateUser">
            <ElIcon><User /></ElIcon> 新建用户
          </ElButton>
        </div>
        <ElTable :data="users" v-loading="userLoading" stripe style="width:100%" empty-text="暂无用户">
          <ElTableColumn prop="display_name" label="姓名" width="120" />
          <ElTableColumn prop="username" label="账号" width="120" />
          <ElTableColumn label="角色" width="120">
            <template #default="{ row }">
              <ElTag :type="row.role === 'admin' ? 'danger' : row.role === 'project_lead' ? 'warning' : 'info'" size="small">
                {{ roleLabel[row.role] || row.role }}
              </ElTag>
            </template>
          </ElTableColumn>
          <ElTableColumn prop="email" label="邮箱" />
          <ElTableColumn prop="phone" label="电话" width="140" />
          <ElTableColumn label="状态" width="80">
            <template #default="{ row }">
              <ElTag :type="row.status === 'active' ? 'success' : 'info'" size="small">
                {{ statusLabel[row.status] || row.status }}
              </ElTag>
            </template>
          </ElTableColumn>
          <ElTableColumn label="操作" width="80" fixed="right">
            <template #default="{ row }">
              <ElButton link type="primary" size="small" @click="openEditUser(row)">编辑</ElButton>
            </template>
          </ElTableColumn>
        </ElTable>

        <ElDialog v-model="userDialogVisible" :title="userEditMode ? '编辑用户' : '新建用户'" width="500">
          <ElForm label-width="80px">
            <ElFormItem label="账号">
              <ElInput v-model="userForm.username" :disabled="userEditMode" placeholder="登录用的用户名" />
            </ElFormItem>
            <ElFormItem :label="userEditMode ? '新密码' : '密码'">
              <ElInput v-model="userForm.password" type="password" show-password :placeholder="userEditMode ? '留空表示不修改' : '初始密码'" />
            </ElFormItem>
            <ElFormItem :label="userEditMode ? '确认新密码' : '确认密码'">
              <ElInput v-model="userForm.confirm_password" type="password" show-password :placeholder="userEditMode ? '再次输入新密码' : '再次输入初始密码'" />
            </ElFormItem>
            <ElFormItem label="姓名"><ElInput v-model="userForm.display_name" /></ElFormItem>
            <ElFormItem label="角色">
              <ElSelect v-model="userForm.role" style="width:100%">
                <ElOption label="管理员" value="admin" />
                <ElOption label="课题负责人" value="project_lead" />
                <ElOption label="成员" value="member" />
              </ElSelect>
            </ElFormItem>
            <ElFormItem label="邮箱"><ElInput v-model="userForm.email" /></ElFormItem>
            <ElFormItem label="电话"><ElInput v-model="userForm.phone" /></ElFormItem>
          </ElForm>
          <template #footer>
            <ElButton @click="userDialogVisible = false">取消</ElButton>
            <ElButton type="primary" :loading="userSubmitting" @click="submitUser">
              {{ userEditMode ? "保存" : "创建" }}
            </ElButton>
          </template>
        </ElDialog>
      </ElTabPane>

      <!-- ===== 成员分配 ===== -->
      <ElTabPane label="成员分配" name="members">
        <div class="tab-toolbar">
          <ElSelect v-model="memberProjectId" placeholder="选择项目" filterable style="width:300px">
            <ElOption v-for="p in projects" :key="p.id" :label="p.name" :value="p.id" />
          </ElSelect>
          <ElButton type="primary" :disabled="!memberProjectId" @click="memberDialogVisible = true">
            添加成员
          </ElButton>
        </div>

        <ElTable v-if="memberProjectId" :data="members" v-loading="memberLoading" stripe style="width:100%" empty-text="暂无成员">
          <ElTableColumn prop="display_name" label="姓名" width="120" />
          <ElTableColumn prop="username" label="账号" width="120" />
          <ElTableColumn label="项目角色" width="180">
            <template #default="{ row }">
              <ElSelect :model-value="row.project_role" size="small" @change="(v: string) => updateMemberRole(row, v)">
                <ElOption label="负责人" value="owner" />
                <ElOption label="管理者" value="manager" />
                <ElOption label="贡献者" value="contributor" />
                <ElOption label="查看者" value="viewer" />
              </ElSelect>
            </template>
          </ElTableColumn>
          <ElTableColumn label="操作" width="80" fixed="right">
            <template #default="{ row }">
              <ElButton link type="danger" size="small" @click="removeMember(row)">移除</ElButton>
            </template>
          </ElTableColumn>
        </ElTable>
        <div v-else class="empty-hint">
          <ElIcon :size="40" color="#c0c4cc"><FolderOpened /></ElIcon>
          <p>请先选择一个项目以管理成员</p>
        </div>

        <ElDialog v-model="memberDialogVisible" title="添加项目成员" width="440">
          <ElForm label-width="80px">
            <ElFormItem label="用户">
              <ElSelect v-model="memberForm.user_id" placeholder="选择用户" filterable style="width:100%">
                <ElOption v-for="u in users" :key="u.id" :label="`${u.display_name} (${u.username})`" :value="u.id" />
              </ElSelect>
            </ElFormItem>
            <ElFormItem label="角色">
              <ElSelect v-model="memberForm.project_role" style="width:100%">
                <ElOption label="负责人" value="owner" />
                <ElOption label="管理者" value="manager" />
                <ElOption label="贡献者" value="contributor" />
                <ElOption label="查看者" value="viewer" />
              </ElSelect>
            </ElFormItem>
          </ElForm>
          <template #footer>
            <ElButton @click="memberDialogVisible = false">取消</ElButton>
            <ElButton type="primary" :loading="memberSubmitting" @click="addMember">添加</ElButton>
          </template>
        </ElDialog>
      </ElTabPane>
    </ElTabs>
    </div>
  </div>
  </AppLayout>
</template>

<style scoped>
.admin-view {
  padding: 20px;
}

.page-title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0 0 20px;
  font-size: 20px;
  font-weight: 700;
}

.admin-tabs :deep(.el-tabs__header) {
  margin-bottom: 16px;
}

.tab-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
  gap: 12px;
}

.tab-desc {
  color: #61748d;
  font-size: 13px;
}

.empty-hint {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 48px 0;
  color: #909399;
}

@media (max-width: 720px) {
  .tab-toolbar {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
