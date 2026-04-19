<script setup lang="ts">
import {
  CircleCheck,
  Coin,
  CopyDocument,
  Plus,
  Refresh,
  Search,
  Timer,
  Warning,
} from "@element-plus/icons-vue";
import { ElMessage } from "element-plus";
import { computed, onMounted, reactive, ref } from "vue";

import api from "@/api";
import AppLayout from "@/components/AppLayout.vue";

type TeamSpaceOption = { id: string; name: string };
type ProjectOption = { id: string; name: string; team_space_id?: string };

type CodeRepository = {
  id: string;
  team_space_id: string;
  project_id: string;
  project_name?: string;
  name: string;
  slug: string;
  description?: string;
  default_branch: string;
  target_folder_path: string;
  repo_storage_path?: string;
  remote_url?: string;
  push_token?: string;
  last_commit_sha?: string;
  last_pushed_at?: string;
  status: string;
  created_by_name?: string;
  created_at: string;
  updated_at: string;
};

type PushEvent = {
  id: string;
  repository_id: string;
  branch: string;
  after_sha?: string;
  commit_message?: string;
  pusher_name?: string;
  sync_status: string;
  error_message?: string;
  created_at: string;
  completed_at?: string;
};

const teamSpaces = ref<TeamSpaceOption[]>([]);
const allProjects = ref<ProjectOption[]>([]);
const repositories = ref<CodeRepository[]>([]);
const total = ref(0);
const keyword = ref("");
const filterProjectID = ref("");
const selectedRepoID = ref("");
const selectedRepo = ref<CodeRepository | null>(null);
const pushEvents = ref<PushEvent[]>([]);
const loading = ref(false);
const detailLoading = ref(false);
const showCreateDialog = ref(false);
const createLoading = ref(false);

const createForm = reactive({
  team_space_id: "",
  project_id: "",
  name: "",
  description: "",
  default_branch: "main",
  target_folder_path: "/code",
});

const filteredProjects = computed(() => {
  if (!createForm.team_space_id) return allProjects.value;
  return allProjects.value.filter((p) => p.team_space_id === createForm.team_space_id);
});

const selectedProjectName = computed(() => {
  const repo = selectedRepo.value;
  if (!repo) return "";
  return repo.project_name || allProjects.value.find((p) => p.id === repo.project_id)?.name || "";
});

function statusLabel(status: string) {
  if (status === "active") return "运行中";
  if (status === "syncing") return "同步中";
  if (status === "failed") return "同步异常";
  return status || "-";
}

function eventStatusLabel(status: string) {
  if (status === "synced") return "已同步";
  if (status === "failed") return "失败";
  if (status === "queued") return "排队中";
  return status || "-";
}

function shortSHA(value?: string) {
  return value ? value.slice(0, 8) : "-";
}

function formatTime(value?: string) {
  if (!value) return "尚未推送";
  return value.replace("T", " ").replace("Z", "").slice(0, 16);
}

function remoteWithToken(repo: CodeRepository | null) {
  if (!repo?.remote_url || !repo.push_token) return repo?.remote_url || "";
  return repo.remote_url.replace("://", `://git:${repo.push_token}@`);
}

async function loadReference() {
  const tsRes = await api.get("/team-spaces");
  teamSpaces.value = tsRes.data?.data ?? [];
  const projects: ProjectOption[] = [];
  for (const ts of teamSpaces.value) {
    const res = await api.get("/projects", { params: { team_space_id: ts.id } });
    projects.push(...(res.data?.data ?? []).map((p: ProjectOption) => ({ ...p, team_space_id: ts.id })));
  }
  allProjects.value = projects;
}

async function fetchRepositories() {
  loading.value = true;
  try {
    const res = await api.get("/code-repositories", {
      params: {
        keyword: keyword.value || undefined,
        project_id: filterProjectID.value || undefined,
        page: 1,
        page_size: 100,
      },
    });
    repositories.value = res.data?.data?.items ?? [];
    total.value = res.data?.data?.total ?? 0;
    if (!selectedRepoID.value && repositories.value.length) {
      await selectRepository(repositories.value[0].id);
    } else if (selectedRepoID.value && !repositories.value.some((item) => item.id === selectedRepoID.value)) {
      selectedRepoID.value = "";
      selectedRepo.value = null;
      pushEvents.value = [];
    }
  } finally {
    loading.value = false;
  }
}

async function selectRepository(id: string) {
  selectedRepoID.value = id;
  detailLoading.value = true;
  try {
    const [detailRes, eventRes] = await Promise.all([
      api.get(`/code-repositories/${id}`),
      api.get(`/code-repositories/${id}/push-events`),
    ]);
    selectedRepo.value = detailRes.data?.data ?? null;
    pushEvents.value = eventRes.data?.data ?? [];
  } catch (err: any) {
    ElMessage.error(err.response?.data?.message ?? "加载代码仓库失败");
  } finally {
    detailLoading.value = false;
  }
}

function openCreateDialog() {
  const firstTS = teamSpaces.value[0];
  createForm.team_space_id = firstTS?.id ?? "";
  const firstProject = allProjects.value.find((p) => p.team_space_id === createForm.team_space_id) || allProjects.value[0];
  createForm.project_id = firstProject?.id ?? "";
  createForm.name = "";
  createForm.description = "";
  createForm.default_branch = "main";
  createForm.target_folder_path = firstProject ? `/projects/${firstProject.name}/code` : "/code";
  showCreateDialog.value = true;
}

function onTeamSpaceChange() {
  createForm.project_id = filteredProjects.value[0]?.id ?? "";
}

async function submitCreate() {
  createLoading.value = true;
  try {
    const res = await api.post("/code-repositories", { ...createForm });
    const repo = res.data?.data as CodeRepository;
    ElMessage.success("代码仓库已创建");
    showCreateDialog.value = false;
    await fetchRepositories();
    selectedRepoID.value = repo.id;
    selectedRepo.value = repo;
    pushEvents.value = [];
  } catch (err: any) {
    ElMessage.error(err.response?.data?.message ?? "创建失败");
  } finally {
    createLoading.value = false;
  }
}

async function copyText(value: string, label: string) {
  if (!value) return;
  await navigator.clipboard.writeText(value);
  ElMessage.success(`${label}已复制`);
}

onMounted(async () => {
  await loadReference();
  await fetchRepositories();
});
</script>

<template>
  <AppLayout>
    <div class="page-shell">
      <div class="page-header">
        <div>
          <div class="page-eyebrow">代码资产库</div>
          <h1>代码管理</h1>
          <p>为课题配置受控 Git 远程仓库，成员提交后可推送到平台维护的代码目录，形成可追踪的代码资产记录。</p>
        </div>
        <div class="header-actions">
          <ElButton @click="fetchRepositories">
            <ElIcon><Refresh /></ElIcon>
            刷新
          </ElButton>
          <ElButton type="primary" @click="openCreateDialog">
            <ElIcon><Plus /></ElIcon>
            新建代码仓库
          </ElButton>
        </div>
      </div>

      <div class="code-workspace">
        <aside class="repo-rail page-card">
          <div class="rail-title">
            <ElIcon><Coin /></ElIcon>
            代码仓库
          </div>
          <div class="rail-summary">
            <div>
              <span>当前查看</span>
              <strong>{{ selectedRepo?.name || "未选择仓库" }}</strong>
            </div>
            <div>
              <span>课题数</span>
              <strong>{{ allProjects.length }}</strong>
            </div>
          </div>
          <ElInput
            v-model="keyword"
            placeholder="搜索仓库"
            clearable
            @keyup.enter="fetchRepositories"
            @clear="fetchRepositories"
          >
            <template #prefix><ElIcon><Search /></ElIcon></template>
          </ElInput>
          <ElSelect v-model="filterProjectID" clearable placeholder="全部课题" @change="fetchRepositories">
            <ElOption v-for="p in allProjects" :key="p.id" :label="p.name" :value="p.id" />
          </ElSelect>

          <div class="repo-count">共 {{ total }} 个仓库</div>
          <div v-loading="loading" class="repo-list">
            <button
              v-for="repo in repositories"
              :key="repo.id"
              class="repo-tab"
              :class="{ active: selectedRepoID === repo.id }"
              type="button"
              @click="selectRepository(repo.id)"
            >
              <strong>{{ repo.name }}</strong>
              <div class="repo-tab-head">
                <span>{{ repo.project_name || repo.slug }}</span>
                <span class="repo-state-pill" :class="repo.status">
                  {{ statusLabel(repo.status) }}
                </span>
              </div>
              <small>{{ formatTime(repo.last_pushed_at) }}</small>
            </button>
          </div>
        </aside>

        <main v-loading="detailLoading" class="code-main">
          <section v-if="!selectedRepo" class="page-card empty-code">
            <ElIcon><Coin /></ElIcon>
            <strong>还没有代码仓库</strong>
            <span>创建仓库后，页面会生成 remote 地址和推送命令。</span>
          </section>

          <template v-else>
            <section class="repo-hero page-card">
              <div class="repo-hero-main">
                <div class="repo-title-block">
                  <div class="repo-avatar"><ElIcon><Coin /></ElIcon></div>
                  <div>
                    <div class="repo-kicker">{{ selectedProjectName }}</div>
                    <h2>{{ selectedRepo.name }}</h2>
                    <p>{{ selectedRepo.description || "该仓库用于沉淀课题代码并记录 push 同步历史。" }}</p>
                  </div>
                </div>
                <div class="repo-hero-actions">
                  <ElButton @click="selectRepository(selectedRepo.id)">
                    <ElIcon><Refresh /></ElIcon>
                    刷新仓库
                  </ElButton>
                  <ElButton type="primary" plain @click="copyText(remoteWithToken(selectedRepo), 'Remote URL')">
                    <ElIcon><CopyDocument /></ElIcon>
                    复制 remote
                  </ElButton>
                </div>
              </div>
              <div class="repo-metrics">
                <div>
                  <span>默认分支</span>
                  <strong>{{ selectedRepo.default_branch }}</strong>
                </div>
                <div>
                  <span>最近提交</span>
                  <strong>{{ shortSHA(selectedRepo.last_commit_sha) }}</strong>
                </div>
                <div>
                  <span>仓库状态</span>
                  <strong>{{ statusLabel(selectedRepo.status) }}</strong>
                </div>
                <div>
                  <span>推送记录</span>
                  <strong>{{ pushEvents.length }}</strong>
                </div>
              </div>
            </section>

            <section class="setup-grid">
              <div class="page-card command-card">
                <div class="section-head">
                  <div>
                    <h3>推送命令</h3>
                    <p>在本地项目完成 git add 和 git commit 后，添加 remote 并推送默认分支。</p>
                  </div>
                  <ElButton text @click="copyText(remoteWithToken(selectedRepo), 'Remote URL')">
                    <ElIcon><CopyDocument /></ElIcon>
                  </ElButton>
                </div>
                <pre><code>git remote add digidocs {{ remoteWithToken(selectedRepo) }}
git push digidocs {{ selectedRepo.default_branch }}</code></pre>
              </div>

              <div class="page-card target-card">
                <div class="section-head">
                  <div>
                    <h3>同步目标</h3>
                    <p>推送事件会记录到平台，目标目录用于后续受控同步到群晖代码文件夹。</p>
                  </div>
                </div>
                <div class="target-line">
                  <span>目标文件夹</span>
                  <strong>{{ selectedRepo.target_folder_path }}</strong>
                </div>
                <div class="target-line">
                  <span>服务端仓库</span>
                  <strong>{{ selectedRepo.slug }}</strong>
                </div>
                <div class="target-line">
                  <span>最近推送</span>
                  <strong>{{ formatTime(selectedRepo.last_pushed_at) }}</strong>
                </div>
              </div>
            </section>

            <section class="page-card event-panel">
              <div class="section-head">
                <div>
                  <h3>推送记录</h3>
                  <p>每次 Git push 后会写入事件，便于负责人查看代码资产变化。</p>
                </div>
                <ElButton @click="selectRepository(selectedRepo.id)">
                  <ElIcon><Refresh /></ElIcon>
                  更新记录
                </ElButton>
              </div>
              <div v-if="pushEvents.length === 0" class="event-empty">
                <ElIcon><Timer /></ElIcon>
                <span>暂无 push 记录</span>
              </div>
              <div v-else class="event-list">
                <div v-for="event in pushEvents" :key="event.id" class="event-row">
                  <div class="event-status" :class="event.sync_status">
                    <ElIcon><CircleCheck v-if="event.sync_status === 'synced'" /><Warning v-else /></ElIcon>
                  </div>
                  <div class="event-main">
                    <div class="event-main-head">
                      <strong>{{ event.commit_message || "Git push" }}</strong>
                      <ElTag :type="event.sync_status === 'synced' ? 'success' : 'danger'" effect="light">
                        {{ eventStatusLabel(event.sync_status) }}
                      </ElTag>
                    </div>
                    <span class="event-meta-line">
                      <span>{{ event.branch }}</span>
                      <span>{{ shortSHA(event.after_sha) }}</span>
                      <span>{{ formatTime(event.created_at) }}</span>
                    </span>
                    <small v-if="event.pusher_name">推送人：{{ event.pusher_name }}</small>
                    <small v-if="event.error_message">{{ event.error_message }}</small>
                  </div>
                </div>
              </div>
            </section>
          </template>
        </main>
      </div>

      <ElDialog v-model="showCreateDialog" title="新建代码仓库" width="560px" :close-on-click-modal="false">
        <ElForm label-position="top">
          <ElFormItem label="团队空间" required>
            <ElSelect v-model="createForm.team_space_id" style="width:100%" @change="onTeamSpaceChange">
              <ElOption v-for="ts in teamSpaces" :key="ts.id" :label="ts.name" :value="ts.id" />
            </ElSelect>
          </ElFormItem>
          <ElFormItem label="关联课题" required>
            <ElSelect v-model="createForm.project_id" style="width:100%">
              <ElOption v-for="p in filteredProjects" :key="p.id" :label="p.name" :value="p.id" />
            </ElSelect>
          </ElFormItem>
          <ElFormItem label="仓库名称" required>
            <ElInput v-model="createForm.name" placeholder="例如：五好爱学前端代码" maxlength="128" />
          </ElFormItem>
          <ElFormItem label="默认分支" required>
            <ElInput v-model="createForm.default_branch" placeholder="main" />
          </ElFormItem>
          <ElFormItem label="目标代码文件夹" required>
            <ElInput v-model="createForm.target_folder_path" placeholder="/projects/课题/code" />
          </ElFormItem>
          <ElFormItem label="说明">
            <ElInput v-model="createForm.description" type="textarea" :rows="3" placeholder="可选" />
          </ElFormItem>
        </ElForm>
        <template #footer>
          <ElButton @click="showCreateDialog = false">取消</ElButton>
          <ElButton type="primary" :loading="createLoading" @click="submitCreate">创建仓库</ElButton>
        </template>
      </ElDialog>
    </div>
  </AppLayout>
</template>

<style scoped>
.code-workspace {
  display: grid;
  grid-template-columns: 280px minmax(0, 1fr);
  gap: 18px;
  align-items: start;
}

.repo-rail {
  position: sticky;
  top: 92px;
  display: grid;
  gap: 12px;
  padding: 16px;
  min-width: 0;
}

.rail-title,
.section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.rail-title {
  justify-content: flex-start;
  font-weight: 800;
  color: var(--dd-ink);
}

.rail-summary {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

.rail-summary div {
  min-width: 0;
  padding: 12px;
  border: 1px solid var(--dd-line);
  border-radius: 10px;
  background: var(--dd-surface-soft);
  display: grid;
  gap: 6px;
}

.rail-summary span,
.repo-tab small,
.repo-kicker,
.section-head p,
.target-line span,
.event-main span,
.event-main small {
  color: var(--dd-muted);
  font-size: 13px;
}

.rail-summary strong {
  color: var(--dd-ink);
  font-size: 14px;
  line-height: 1.4;
}

.repo-count {
  color: var(--dd-muted);
  font-size: 13px;
}

.repo-list {
  display: grid;
  gap: 8px;
  min-height: 120px;
  min-width: 0;
}

.repo-tab {
  width: 100%;
  min-width: 0;
  border: 1px solid var(--dd-line);
  border-radius: 8px;
  background: #fff;
  padding: 12px;
  display: grid;
  gap: 4px;
  text-align: left;
  cursor: pointer;
}

.repo-tab strong {
  color: var(--dd-ink);
  font-size: 14px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.repo-tab-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.repo-tab.active {
  border-color: var(--dd-primary);
  background: #eef6ff;
}

.repo-state-pill {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 24px;
  padding: 0 8px;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 700;
  white-space: nowrap;
}

.repo-state-pill.active {
  background: #ecfdf5;
  color: #047857;
}

.repo-state-pill.syncing {
  background: #eff6ff;
  color: #1d4ed8;
}

.repo-state-pill.failed {
  background: #fef2f2;
  color: #dc2626;
}

.code-main {
  display: grid;
  gap: 18px;
}

.empty-code {
  min-height: 360px;
  display: grid;
  place-items: center;
  align-content: center;
  gap: 10px;
  color: var(--dd-muted);
}

.empty-code .el-icon {
  font-size: 44px;
  color: var(--dd-primary);
}

.repo-hero {
  display: grid;
  gap: 24px;
  padding: 22px;
}

.repo-hero-main {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 18px;
}

.repo-title-block {
  display: flex;
  gap: 14px;
  align-items: flex-start;
}

.repo-avatar {
  width: 48px;
  height: 48px;
  border-radius: 8px;
  display: grid;
  place-items: center;
  background: #e8f4ff;
  color: var(--dd-primary);
  font-size: 22px;
}

.repo-title-block h2,
.section-head h3 {
  margin: 0;
}

.repo-title-block p {
  margin: 6px 0 0;
  color: var(--dd-ink-2);
}

.repo-hero-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  justify-content: flex-end;
}

.repo-metrics {
  display: grid;
  grid-template-columns: repeat(4, minmax(120px, 1fr));
  gap: 10px;
}

.repo-metrics div,
.target-line {
  border: 1px solid var(--dd-line);
  border-radius: 8px;
  padding: 12px;
  display: grid;
  gap: 4px;
}

.repo-metrics span {
  color: var(--dd-muted);
  font-size: 12px;
}

.setup-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.4fr) minmax(280px, 0.8fr);
  gap: 18px;
}

.command-card,
.target-card,
.event-panel {
  padding: 18px;
}

pre {
  margin: 16px 0 0;
  padding: 16px;
  border-radius: 8px;
  background: #111827;
  color: #e5e7eb;
  overflow-x: hidden;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  word-break: break-word;
  line-height: 1.7;
}

.target-card {
  display: grid;
  gap: 12px;
}

.event-list {
  display: grid;
  gap: 10px;
  margin-top: 14px;
}

.event-row {
  display: grid;
  grid-template-columns: 36px minmax(0, 1fr);
  gap: 12px;
  align-items: center;
  border: 1px solid var(--dd-line);
  border-radius: 8px;
  padding: 12px;
}

.event-status {
  width: 32px;
  height: 32px;
  border-radius: 999px;
  display: grid;
  place-items: center;
  background: #fef2f2;
  color: #dc2626;
}

.event-status.synced {
  background: #ecfdf5;
  color: #059669;
}

.event-main {
  display: grid;
  gap: 3px;
}

.event-main-head,
.event-meta-line {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  flex-wrap: wrap;
}

.event-main-head strong {
  color: var(--dd-ink);
}

.event-meta-line span {
  position: relative;
}

.event-meta-line span:not(:last-child)::after {
  content: "·";
  margin-left: 10px;
  color: var(--dd-subtle);
}

.event-empty {
  min-height: 140px;
  display: grid;
  place-items: center;
  align-content: center;
  gap: 8px;
  color: var(--dd-muted);
}

@media (max-width: 960px) {
  .code-workspace,
  .setup-grid {
    grid-template-columns: 1fr;
  }

  .repo-rail {
    position: static;
  }

  .repo-hero-main {
    flex-direction: column;
  }

  .repo-hero-actions {
    width: 100%;
    justify-content: flex-start;
  }

  .repo-metrics {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 767px) {
  .repo-rail {
    padding: 14px;
  }

  .rail-summary {
    grid-template-columns: 1fr;
  }

  .repo-list {
    grid-template-columns: 1fr;
  }

  .repo-tab {
    min-height: 0;
  }

  .repo-tab-head {
    align-items: flex-start;
    flex-direction: column;
  }

  .repo-hero,
  .command-card,
  .target-card,
  .event-panel {
    padding: 16px;
  }

  .repo-title-block {
    gap: 12px;
  }

  .repo-avatar {
    width: 42px;
    height: 42px;
    font-size: 20px;
  }

  .repo-metrics {
    grid-template-columns: 1fr;
  }

  .section-head {
    align-items: flex-start;
    flex-direction: column;
  }

  pre {
    margin-top: 12px;
    padding: 14px;
    font-size: 12px;
  }

  .event-row {
    grid-template-columns: 1fr;
    align-items: flex-start;
  }

  .event-status {
    width: 28px;
    height: 28px;
  }
}
</style>
