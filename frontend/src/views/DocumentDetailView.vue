<script setup lang="ts">
import { ElCard, ElDescriptions, ElDescriptionsItem, ElTable, ElTableColumn, ElTag } from "element-plus";

import AppLayout from "@/components/AppLayout.vue";

const versions = [
  { version: 3, author: "张三", note: "补充实验数据" },
  { version: 2, author: "李四", note: "修订第三章" },
];

const flows = [
  { from: "张三", to: "李四", status: "待交接", note: "请继续处理第四章" },
  { from: "李四", to: "李四", status: "处理中", note: "已接收" },
];
</script>

<template>
  <AppLayout>
    <div class="page-shell detail-grid">
      <ElCard class="page-card">
        <template #header>文档基本信息</template>
        <ElDescriptions :column="2" border>
          <ElDescriptionsItem label="标题">课题申报书</ElDescriptionsItem>
          <ElDescriptionsItem label="当前责任人">张三</ElDescriptionsItem>
          <ElDescriptionsItem label="状态"><ElTag>处理中</ElTag></ElDescriptionsItem>
          <ElDescriptionsItem label="所属目录">/申报材料</ElDescriptionsItem>
        </ElDescriptions>
      </ElCard>

      <ElCard class="page-card">
        <template #header>AI 摘要与建议</template>
        <div class="summary-text">本版本补充了实验数据、研究目标和阶段成果。</div>
        <div class="tag-row">
          <ElTag>申报材料</ElTag>
          <ElTag type="success">实验数据</ElTag>
          <ElTag type="warning">建议归档</ElTag>
        </div>
      </ElCard>

      <ElCard class="page-card">
        <template #header>版本历史</template>
        <ElTable :data="versions">
          <ElTableColumn prop="version" label="版本号" />
          <ElTableColumn prop="author" label="提交人" />
          <ElTableColumn prop="note" label="说明" />
        </ElTable>
      </ElCard>

      <ElCard class="page-card">
        <template #header>流转历史</template>
        <ElTable :data="flows">
          <ElTableColumn prop="from" label="来源" />
          <ElTableColumn prop="to" label="目标" />
          <ElTableColumn prop="status" label="状态" />
          <ElTableColumn prop="note" label="备注" />
        </ElTable>
      </ElCard>
    </div>
  </AppLayout>
</template>

<style scoped>
.detail-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 18px;
}

.summary-text {
  color: #31465e;
}

.tag-row {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
  margin-top: 14px;
}

@media (max-width: 900px) {
  .detail-grid {
    grid-template-columns: 1fr;
  }
}
</style>
