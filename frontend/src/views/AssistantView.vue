<script setup lang="ts">
import { ElButton, ElCard, ElInput, ElTimeline, ElTimelineItem } from "element-plus";
import { ref } from "vue";

import AppLayout from "@/components/AppLayout.vue";

const question = ref("课题A 最近一个月有哪些文档在流转？");
const timeline = [
  {
    title: "问答结果",
    content: "最近一个月共有 6 份文档发生流转，主要集中在申报材料目录。",
  },
  {
    title: "建议",
    content: "建议优先处理超过 30 天未更新的阶段报告。",
  },
];
</script>

<template>
  <AppLayout>
    <div class="page-shell assistant-grid">
      <ElCard class="page-card">
        <template #header>OpenClaw 助手</template>
        <ElInput v-model="question" :rows="4" type="textarea" />
        <ElButton type="primary" style="margin-top: 16px">发起问答</ElButton>
      </ElCard>

      <ElCard class="page-card">
        <template #header>结果与建议</template>
        <ElTimeline>
          <ElTimelineItem v-for="item in timeline" :key="item.title" :timestamp="item.title">
            {{ item.content }}
          </ElTimelineItem>
        </ElTimeline>
      </ElCard>
    </div>
  </AppLayout>
</template>

<style scoped>
.assistant-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 18px;
}

@media (max-width: 900px) {
  .assistant-grid {
    grid-template-columns: 1fr;
  }
}
</style>
