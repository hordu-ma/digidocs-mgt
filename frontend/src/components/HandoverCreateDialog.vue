<script setup lang="ts">
import type { UserOption, ProjectOption } from "@/types";
const showDialog = defineModel<boolean>({ required: true });
const form = defineModel<{ target_user_id: string; receiver_user_id: string; project_id: string; remark: string }>("form", { required: true });
defineProps<{ users: UserOption[]; projects: ProjectOption[]; referenceLoading: boolean; formLoading: boolean }>();
const emit = defineEmits<{ submit: [] }>();
</script>

<template>
<ElDialog v-model="showDialog" title="创建交接单" width="520px">
        <ElForm label-position="top" v-loading="referenceLoading">
          <ElFormItem label="交接人" required>
            <ElSelect
              v-model="form.target_user_id"
              filterable
              placeholder="选择即将离开的成员"
            >
              <ElOption
                v-for="item in users"
                :key="item.id"
                :label="`${item.display_name} (${item.role})`"
                :value="item.id"
              />
            </ElSelect>
          </ElFormItem>
          <ElFormItem label="接收人" required>
            <ElSelect
              v-model="form.receiver_user_id"
              filterable
              placeholder="选择接手成员"
            >
              <ElOption
                v-for="item in users"
                :key="item.id"
                :label="`${item.display_name} (${item.role})`"
                :value="item.id"
              />
            </ElSelect>
          </ElFormItem>
          <ElFormItem label="课题">
            <ElSelect
              v-model="form.project_id"
              clearable
              filterable
              placeholder="限定交接范围（可选）"
            >
              <ElOption
                v-for="item in projects"
                :key="item.id"
                :label="item.name"
                :value="item.id"
              />
            </ElSelect>
          </ElFormItem>
          <ElFormItem label="备注">
            <ElInput v-model="form.remark" type="textarea" :rows="2" />
          </ElFormItem>
        </ElForm>
        <template #footer>
          <ElButton @click="showDialog = false">取消</ElButton>
          <ElButton type="primary" :loading="formLoading" @click="emit('submit')">
            确认创建
          </ElButton>
        </template>
      </ElDialog>
</template>
