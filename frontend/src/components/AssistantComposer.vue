<script setup lang="ts">
import { Document, FolderOpened, Position } from "@element-plus/icons-vue";
const question = defineModel<string>({ required: true });
const props = defineProps<{ recommendedPrompts: string[]; composerScopeType: string; composerScopeLabel: string; thinking: boolean; loading: boolean }>();
const emit = defineEmits<{ submit: []; prompt: [text: string] }>();
function submit() { if (!props.thinking && !props.loading) emit("submit"); }
</script>
<template>
<footer class="composer">
          <div class="prompt-suggestions">
            <button
              v-for="item in recommendedPrompts"
              :key="item"
              class="prompt-chip"
              type="button"
              @click="emit('prompt', item)"
            >
              {{ item }}
            </button>
          </div>
          <div class="composer-scope">
            <span v-if="composerScopeType" class="scope-chip active">
              <ElIcon>
                <component :is="composerScopeType === 'document' ? Document : FolderOpened" />
              </ElIcon>
              {{ composerScopeLabel }}
            </span>
            <span v-else class="assistant-hint">请先选择项目或文档范围</span>
          </div>
          <div class="composer-input">
            <ElInput
              v-model="question"
              :rows="3"
              type="textarea"
              :disabled="thinking"
              resize="none"
              :placeholder="thinking ? 'DocsAgent 正在处理上一条问题…' : '输入需要整理、核对或追问的问题'"
              @keyup.ctrl.enter="submit"
            />
            <ElButton
              type="primary"
              :loading="loading"
              :disabled="thinking"
              @click="submit"
            >
              <ElIcon><Position /></ElIcon>
              {{ thinking ? "处理中" : "发送" }}
            </ElButton>
          </div>
        </footer>
</template>
<style scoped>
.scope-chip {
  display: inline-flex;
  align-items: center;
  max-width: 100%;
  gap: 6px;
  min-height: 26px;
  padding: 0 9px;
  border: 1px solid #dbe3ed;
  border-radius: 999px;
  background: #f8fafc;
  color: var(--dd-ink-2);
  font-size: 12px;
  font-weight: 700;
}
.scope-chip.document {
  border-color: #f1d18b;
  background: var(--dd-warning-soft);
  color: var(--dd-warning);
}
.scope-chip.active,
.scope-chip.project {
  border-color: #c8ddf2;
  background: var(--dd-primary-soft);
  color: var(--dd-primary-strong);
}
.assistant-hint,
.message-meta,
.message-extra {
  color: var(--dd-muted);
  font-size: 13px;
}
.composer {
  display: grid;
  gap: 10px;
  padding: 16px 20px 20px;
  border-top: 1px solid var(--dd-line-soft);
  background: rgba(255, 255, 255, 0.94);
}
.prompt-suggestions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.prompt-chip {
  display: inline-flex;
  align-items: center;
  min-height: 34px;
  padding: 0 12px;
  border: 1px solid var(--dd-line);
  border-radius: 999px;
  background: #fff;
  color: var(--dd-ink-2);
  font-size: 12px;
  font-weight: 700;
  cursor: pointer;
}
.prompt-chip:hover {
  border-color: #bfd5ed;
  background: var(--dd-primary-soft);
  color: var(--dd-primary-strong);
}
.composer-scope {
  display: flex;
  align-items: center;
  min-height: 28px;
}
.composer-input {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 104px;
  gap: 12px;
  align-items: end;
}
@media (max-width: 720px) {
  .composer-input { grid-template-columns: 1fr; }
}
</style>
