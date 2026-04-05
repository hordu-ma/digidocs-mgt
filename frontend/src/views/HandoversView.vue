<script setup lang="ts">
import {
  ElButton,
  ElCard,
  ElEmpty,
  ElTable,
  ElTableColumn,
  ElTag,
} from "element-plus";
import { onMounted, ref } from "vue";

import AppLayout from "@/components/AppLayout.vue";
import api from "@/api";

const handovers = ref<any[]>([]);

const statusLabel: Record<string, string> = {
  generated: "已生成",
  pending_confirm: "待确认",
  completed: "已完成",
  cancelled: "已取消",
};

onMounted(async () => {
  const res = await api.get("/handovers");
  handovers.value = res.data?.data ?? [];
});
</script>

<template>
  <AppLayout>
    <div class="page-shell">
      <div class="page-header">
        <div>
          <h1>毕业交接</h1>
          <p>管理课题组人员毕业交接流程，确保文档资产完整移交。</p>
        </div>
      </div>

      <ElCard class="page-card">
        <template #header>交接记录</template>
        <ElTable
          v-if="handovers.length > 0"
          :data="handovers"
          style="width: 100%"
        >
          <ElTableColumn prop="id" label="交接单 ID" width="320" />
          <ElTableColumn prop="target_user_id" label="交接人" />
          <ElTableColumn prop="receiver_user_id" label="接收人" />
          <ElTableColumn label="状态">
            <template #default="{ row }">
              <ElTag>{{ statusLabel[row.status] ?? row.status }}</ElTag>
            </template>
          </ElTableColumn>
          <ElTableColumn prop="remark" label="备注" />
        </ElTable>
        <ElEmpty v-else description="暂无交接记录" />
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
</style>
