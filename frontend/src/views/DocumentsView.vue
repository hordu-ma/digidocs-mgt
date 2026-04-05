<script setup lang="ts">
import { onMounted, ref } from "vue";
import { useRouter } from "vue-router";

import AppLayout from "@/components/AppLayout.vue";
import api from "@/api";

const router = useRouter();
const rows = ref<any[]>([]);
const total = ref(0);
const page = ref(1);
const pageSize = ref(20);
const keyword = ref("");

const statusLabel: Record<string, string> = {
  draft: "草稿",
  in_progress: "处理中",
  pending_handover: "待交接",
  handed_over: "已交接",
  finalized: "定稿",
  archived: "已归档",
};

async function fetchDocuments() {
  const res = await api.get("/documents", {
    params: {
      page: page.value,
      page_size: pageSize.value,
      keyword: keyword.value,
    },
  });
  rows.value = res.data?.data ?? [];
  total.value = res.data?.meta?.total ?? 0;
}

function handleSearch() {
  page.value = 1;
  fetchDocuments();
}

function handlePageChange(p: number) {
  page.value = p;
  fetchDocuments();
}

function goDetail(row: any) {
  router.push(`/documents/${row.id}`);
}

onMounted(fetchDocuments);
</script>

<template>
  <AppLayout>
    <div class="page-shell">
      <div class="page-header">
        <div>
          <h1>文档管理</h1>
          <p>按团队空间、课题和目录组织文档，并管理责任人和版本。</p>
        </div>
        <ElButton type="primary">新建文档</ElButton>
      </div>
      <ElCard class="page-card">
        <div class="toolbar">
          <ElInput
            v-model="keyword"
            placeholder="搜索文档标题"
            @keyup.enter="handleSearch"
          />
        </div>
        <ElTable :data="rows" style="width: 100%" @row-click="goDetail">
          <ElTableColumn prop="title" label="文档标题" />
          <ElTableColumn label="当前责任人">
            <template #default="{ row }">{{
              row.current_owner?.display_name ?? "-"
            }}</template>
          </ElTableColumn>
          <ElTableColumn label="当前版本">
            <template #default="{ row }">{{
              row.current_version_no ?? "-"
            }}</template>
          </ElTableColumn>
          <ElTableColumn label="状态">
            <template #default="{ row }">
              <ElTag>{{
                statusLabel[row.current_status] ?? row.current_status
              }}</ElTag>
            </template>
          </ElTableColumn>
          <ElTableColumn prop="updated_at" label="更新时间" />
        </ElTable>
        <ElPagination
          v-if="total > pageSize"
          :current-page="page"
          :page-size="pageSize"
          :total="total"
          layout="prev, pager, next"
          style="margin-top: 16px; justify-content: flex-end"
          @current-change="handlePageChange"
        />
      </ElCard>
    </div>
  </AppLayout>
</template>

<style scoped>
h1 {
  margin: 0;
  font-size: 32px;
}

p {
  color: #61748d;
}

.toolbar {
  margin-bottom: 16px;
}
</style>
