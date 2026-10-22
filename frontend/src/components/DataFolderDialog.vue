<script setup lang="ts">
import type { ProjectOption } from "@/types";
const showFolderDialog = defineModel<boolean>({ required: true });
const newFolderForm = defineModel<{ project_id: string; parent_id: string; name: string }>("form", { required: true });
defineProps<{ folderProjects: ProjectOption[]; folderParentOptions: { id: string; name: string }[] }>();
const emit = defineEmits<{ submit: []; projectChange: [id: string] }>();
</script>

<template>
<ElDialog v-model="showFolderDialog" title="新建文件夹" width="420px">
        <ElForm label-position="top">
          <ElFormItem label="课题" required>
            <ElSelect v-model="newFolderForm.project_id" @change="emit('projectChange', $event)" style="width:100%">
              <ElOption v-for="p in folderProjects" :key="p.id" :label="p.name" :value="p.id" />
            </ElSelect>
          </ElFormItem>
          <ElFormItem label="父文件夹">
            <ElSelect v-model="newFolderForm.parent_id" clearable style="width:100%" placeholder="顶层文件夹">
              <ElOption v-for="f in folderParentOptions" :key="f.id" :label="f.name" :value="f.id" />
            </ElSelect>
            <p class="field-hint">最多 2 层，选择父文件夹后可创建子文件夹</p>
          </ElFormItem>
          <ElFormItem label="名称" required>
            <ElInput v-model="newFolderForm.name" placeholder="文件夹名称" maxlength="128" />
          </ElFormItem>
        </ElForm>
        <template #footer>
          <ElButton @click="showFolderDialog = false">取消</ElButton>
          <ElButton type="primary" @click="emit('submit')">创建</ElButton>
        </template>
      </ElDialog>
</template>

<style scoped>
.field-hint { margin: 6px 0 0; color: var(--dd-muted); font-size: 12px; }
</style>
