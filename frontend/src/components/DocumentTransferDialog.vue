<script setup lang="ts">
import type { UserOption } from "@/types";
const showTransferDialog = defineModel<boolean>({ required: true });
const transferForm = defineModel<{ to_user_id: string }>("form", { required: true });
defineProps<{ selectableUsers: UserOption[]; transferLoading: boolean }>();
const emit = defineEmits<{ submit: [] }>();
</script>

<template>
<ElDialog v-model="showTransferDialog" title="转交文档" width="420px">
      <ElForm label-position="top">
        <ElFormItem label="接收人" required>
          <ElSelect
            v-model="transferForm.to_user_id"
            filterable
            placeholder="选择接收人"
          >
            <ElOption
              v-for="item in selectableUsers"
              :key="item.id"
              :label="`${item.display_name} (${item.role})`"
              :value="item.id"
            />
          </ElSelect>
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton @click="showTransferDialog = false">取消</ElButton>
        <ElButton
          type="primary"
          :loading="transferLoading"
          @click="emit('submit')"
        >
          确认转交
        </ElButton>
      </template>
    </ElDialog>
</template>
