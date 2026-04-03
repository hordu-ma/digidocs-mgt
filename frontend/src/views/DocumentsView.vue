<script setup lang="ts">
import { ElButton, ElCard, ElInput, ElTable, ElTableColumn, ElTag } from "element-plus";

import AppLayout from "@/components/AppLayout.vue";

const rows = [
  { id: "1", title: "课题申报书", owner: "张三", version: 3, status: "处理中" },
  { id: "2", title: "中期汇报PPT", owner: "李四", version: 5, status: "定稿" },
];
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
          <ElInput placeholder="搜索文档标题 / 版本说明" />
        </div>
        <ElTable :data="rows" style="width: 100%">
          <ElTableColumn prop="title" label="文档标题" />
          <ElTableColumn prop="owner" label="当前责任人" />
          <ElTableColumn prop="version" label="当前版本" />
          <ElTableColumn label="状态">
            <template #default="{ row }">
              <ElTag>{{ row.status }}</ElTag>
            </template>
          </ElTableColumn>
        </ElTable>
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
