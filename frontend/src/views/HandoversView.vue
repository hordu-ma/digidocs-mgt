<script setup lang="ts">
import { ElMessage } from "element-plus";
import { onMounted, reactive, ref } from "vue";

import AppLayout from "@/components/AppLayout.vue";
import api from "@/api";

const handovers = ref<any[]>([]);
const showDialog = ref(false);
const formLoading = ref(false);
const form = reactive({
  target_user_id: "",
  receiver_user_id: "",
  project_id: "",
  remark: "",
});

const statusLabel: Record<string, string> = {
  generated: "已生成",
  pending_confirm: "待确认",
  completed: "已完成",
  cancelled: "已取消",
};

async function fetchHandovers() {
  const res = await api.get("/handovers");
  handovers.value = res.data?.data ?? [];
}

function openCreate() {
  form.target_user_id = "";
  form.receiver_user_id = "";
  form.project_id = "";
  form.remark = "";
  showDialog.value = true;
}

async function submitCreate() {
  if (!form.target_user_id || !form.receiver_user_id) {
    ElMessage.warning("请填写交接人和接收人 ID");
    return;
  }

  formLoading.value = true;
  try {
    await api.post("/handovers", form);
    ElMessage.success("交接单已创建");
    showDialog.value = false;
    await fetchHandovers();
  } catch (err: any) {
    const msg = err.response?.data?.message ?? "创建失败";
    ElMessage.error(msg);
  } finally {
    formLoading.value = false;
  }
}

onMounted(fetchHandovers);
</script>

<template>
  <AppLayout>
    <div class="page-shell">
      <div class="page-header">
        <div>
          <h1>毕业交接</h1>
          <p>管理课题组人员毕业交接流程，确保文档资产完整移交。</p>
        </div>
        <ElButton type="primary" @click="openCreate">创建交接</ElButton>
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

      <ElDialog v-model="showDialog" title="创建交接单" width="480px">
        <ElForm label-position="top">
          <ElFormItem label="交接人 ID">
            <ElInput
              v-model="form.target_user_id"
              placeholder="即将离开的成员用户 ID"
            />
          </ElFormItem>
          <ElFormItem label="接收人 ID">
            <ElInput
              v-model="form.receiver_user_id"
              placeholder="接手文档的成员用户 ID"
            />
          </ElFormItem>
          <ElFormItem label="课题 ID（可选）">
            <ElInput
              v-model="form.project_id"
              placeholder="限定交接范围的课题 ID"
            />
          </ElFormItem>
          <ElFormItem label="备注">
            <ElInput v-model="form.remark" type="textarea" :rows="2" />
          </ElFormItem>
        </ElForm>
        <template #footer>
          <ElButton @click="showDialog = false">取消</ElButton>
          <ElButton type="primary" :loading="formLoading" @click="submitCreate"
            >确认创建</ElButton
          >
        </template>
      </ElDialog>
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
