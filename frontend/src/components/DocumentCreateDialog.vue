<script setup lang="ts">
import type { UploadFile } from "element-plus";
import type { UserOption, ProjectOption } from "@/types";
const showCreateDialog = defineModel<boolean>({ required: true });
const createForm = defineModel<{ team_space_id: string; project_id: string; folder_id: string; current_owner_id: string; title: string; description: string; commit_message: string }>("form", { required: true });
defineProps<{ users: UserOption[]; projects: ProjectOption[]; teamSpaces: { id: string; name: string; code: string }[]; folderOptions: {id: string; path: string}[]; referenceLoading: boolean; createLoading: boolean }>();
const emit = defineEmits<{ submit: []; teamSpaceChange: [id: string]; projectChange: [id: string]; fileChange: [file: UploadFile] }>();
function handleFileChange(file: UploadFile) { emit("fileChange", file); }
</script>

<template>
<ElDialog v-model="showCreateDialog" title="新建文档" width="520px">
        <ElForm label-position="top" v-loading="referenceLoading">
          <ElFormItem label="标题" required>
            <ElInput v-model="createForm.title" placeholder="文档标题" />
          </ElFormItem>
          <ElFormItem label="团队空间" required>
            <ElSelect
              v-model="createForm.team_space_id"
              filterable
              placeholder="选择团队空间"
              @change="emit('teamSpaceChange', $event)"
            >
              <ElOption
                v-for="item in teamSpaces"
                :key="item.id"
                :label="`${item.name} (${item.code})`"
                :value="item.id"
              />
            </ElSelect>
          </ElFormItem>
          <ElFormItem label="课题" required>
            <ElSelect
              v-model="createForm.project_id"
              filterable
              placeholder="选择课题"
              :disabled="projects.length === 0"
              @change="emit('projectChange', $event)"
            >
              <ElOption
                v-for="item in projects"
                :key="item.id"
                :label="item.name"
                :value="item.id"
              />
            </ElSelect>
          </ElFormItem>
          <ElFormItem label="目录">
            <ElSelect
              v-model="createForm.folder_id"
              clearable
              filterable
              placeholder="选择目录（可选）"
              :disabled="folderOptions.length === 0"
            >
              <ElOption
                v-for="item in folderOptions"
                :key="item.id"
                :label="item.path"
                :value="item.id"
              />
            </ElSelect>
          </ElFormItem>
          <ElFormItem label="当前责任人" required>
            <ElSelect
              v-model="createForm.current_owner_id"
              filterable
              placeholder="选择责任人"
            >
              <ElOption
                v-for="item in users"
                :key="item.id"
                :label="`${item.display_name} (${item.role})`"
                :value="item.id"
              />
            </ElSelect>
          </ElFormItem>
          <ElFormItem label="描述">
            <ElInput
              v-model="createForm.description"
              type="textarea"
              :rows="2"
            />
          </ElFormItem>
          <ElFormItem label="提交说明">
            <ElInput
              v-model="createForm.commit_message"
              placeholder="首版本提交说明"
            />
          </ElFormItem>
          <ElFormItem label="文件" required>
            <ElUpload
              :auto-upload="false"
              :limit="1"
              :on-change="handleFileChange"
            >
              <ElButton>选择文件</ElButton>
            </ElUpload>
          </ElFormItem>
        </ElForm>
        <template #footer>
          <ElButton @click="showCreateDialog = false">取消</ElButton>
          <ElButton
            type="primary"
            :loading="createLoading"
            @click="emit('submit')"
            >创建</ElButton
          >
        </template>
      </ElDialog>
</template>
