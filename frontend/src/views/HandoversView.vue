<script setup lang="ts">
import { ElButton, ElCard, ElSelect, ElOption, ElSteps, ElStep, ElTable, ElTableColumn } from "element-plus";
import { ref } from "vue";

import AppLayout from "@/components/AppLayout.vue";

const member = ref("");
const rows = [
  { title: "实验记录", selected: "是", status: "处理中" },
  { title: "汇报PPT", selected: "是", status: "定稿" },
];
</script>

<template>
  <AppLayout>
    <div class="page-shell handover-grid">
      <ElCard class="page-card">
        <template #header>发起毕业交接</template>
        <ElSelect v-model="member" placeholder="选择成员" style="width: 100%">
          <ElOption label="张三" value="zhangsan" />
          <ElOption label="李四" value="lisi" />
        </ElSelect>
        <ElButton type="primary" style="margin-top: 16px">生成候选清单</ElButton>
      </ElCard>

      <ElCard class="page-card">
        <template #header>交接状态</template>
        <ElSteps :active="2" finish-status="success">
          <ElStep title="生成交接单" />
          <ElStep title="接收确认" />
          <ElStep title="完成更新" />
        </ElSteps>
      </ElCard>

      <ElCard class="page-card handover-table">
        <template #header>候选文档</template>
        <ElTable :data="rows">
          <ElTableColumn prop="title" label="文档" />
          <ElTableColumn prop="selected" label="是否纳入" />
          <ElTableColumn prop="status" label="当前状态" />
        </ElTable>
      </ElCard>
    </div>
  </AppLayout>
</template>

<style scoped>
.handover-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 18px;
}

.handover-table {
  grid-column: 1 / -1;
}

@media (max-width: 900px) {
  .handover-grid {
    grid-template-columns: 1fr;
  }
}
</style>
