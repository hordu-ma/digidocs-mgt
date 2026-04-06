<script setup lang="ts">
import { ElMessage } from "element-plus";
import { onBeforeUnmount, ref } from "vue";

import AppLayout from "@/components/AppLayout.vue";
import api from "@/api";

const question = ref("课题A 最近一个月有哪些文档在流转？");
const loading = ref(false);
const timeline = ref<{ title: string; content: string }[]>([]);
const activeRequestID = ref("");
const submittedQuestion = ref("");
let pollTimer: number | null = null;

function stopPolling() {
  if (pollTimer !== null) {
    window.clearTimeout(pollTimer);
    pollTimer = null;
  }
}

async function pollRequest(requestID: string) {
  try {
    const res = await api.get(`/assistant/requests/${requestID}`);
    const data = res.data?.data;
    const answer = data?.output?.answer;
    timeline.value = [
      {
        title: "已提交",
        content: `问题「${submittedQuestion.value}」已提交至 AI 助手（request_id: ${requestID}）`,
      },
      {
        title: "状态",
        content:
          data?.status === "completed"
            ? answer || "任务已完成，但未返回回答内容。"
            : data?.status === "failed"
              ? data?.error_message || "任务执行失败。"
              : "任务仍在处理中，正在轮询最新状态。",
      },
    ];

    if (data?.status === "completed" || data?.status === "failed") {
      stopPolling();
      return;
    }

    pollTimer = window.setTimeout(() => {
      void pollRequest(requestID);
    }, 2000);
  } catch (err: any) {
    stopPolling();
    ElMessage.error(err.response?.data?.message ?? "查询 AI 任务状态失败");
  }
}

async function submitQuestion() {
  if (!question.value.trim()) {
    ElMessage.warning("请输入问题");
    return;
  }

  loading.value = true;
  try {
    stopPolling();
    const res = await api.post("/assistant/ask", {
      question: question.value,
      scope: {
        project_id: null,
        document_id: null,
      },
    });
    const data = res.data?.data;
    activeRequestID.value = data?.request_id ?? "";
    submittedQuestion.value = data?.question ?? question.value;
    timeline.value = [
      {
        title: "已提交",
        content: `问题「${submittedQuestion.value}」已提交至 AI 助手（request_id: ${data?.request_id ?? "-"}）`,
      },
      {
        title: "状态",
        content:
          data?.status === "queued"
            ? "任务已排队，结果将在后台处理完成后更新。"
            : data?.answer
              ? data.answer
              : "任务状态未知，请稍后刷新。",
      },
    ];
    if (activeRequestID.value) {
      await pollRequest(activeRequestID.value);
    }
    ElMessage.success("问题已提交");
  } catch (err: any) {
    const msg = err.response?.data?.message ?? "提交失败";
    ElMessage.error(msg);
  } finally {
    loading.value = false;
  }
}

onBeforeUnmount(() => {
  stopPolling();
});
</script>

<template>
  <AppLayout>
    <div class="page-shell assistant-grid">
      <ElCard class="page-card">
        <template #header>OpenClaw 助手</template>
        <ElInput v-model="question" :rows="4" type="textarea" />
        <ElButton
          type="primary"
          :loading="loading"
          style="margin-top: 16px"
          @click="submitQuestion"
          >发起问答</ElButton
        >
      </ElCard>

      <ElCard class="page-card">
        <template #header>结果与建议</template>
        <ElTimeline>
          <ElTimelineItem
            v-for="item in timeline"
            :key="item.title"
            :timestamp="item.title"
          >
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
