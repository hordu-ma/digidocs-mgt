import { afterEach, describe, expect, it, vi } from "vitest";
import { createApp, defineComponent, h, nextTick, type Component } from "vue";
import AssistantComposer from "./AssistantComposer.vue";
import DocumentTransferDialog from "./DocumentTransferDialog.vue";
import HandoverCreateDialog from "./HandoverCreateDialog.vue";
import DocumentCreateDialog from "./DocumentCreateDialog.vue";
import { DATA_FILE_MAX_BYTES, useUploadFile } from "@/composables/useUploadFile";

const cleanups: (() => void)[] = [];
afterEach(() => cleanups.splice(0).forEach((cleanup) => cleanup()));

function mount(component: Component, props: Record<string, unknown>) {
  const root = document.createElement("div");
  document.body.append(root);
  const app = createApp(component, props);
  const wrapper = defineComponent({ setup(_, { slots }) { return () => h("div", [slots.default?.(), slots.footer?.()]); } });
  for (const name of ["ElDialog", "ElForm", "ElFormItem", "ElIcon", "ElOption", "ElUpload"]) app.component(name, wrapper);
  app.component("ElButton", defineComponent({ props: ["disabled", "loading"], setup(p, { slots }) { return () => h("button", { disabled: p.disabled || p.loading }, slots.default?.()); } }));
  app.component("ElInput", defineComponent({ props: ["modelValue", "disabled"], emits: ["update:modelValue"], setup(p, { emit }) { return () => h("input", { value: p.modelValue, disabled: p.disabled, onInput: (e: Event) => emit("update:modelValue", (e.target as HTMLInputElement).value) }); } }));
  app.component("ElSelect", wrapper);
  app.directive("loading", {});
  app.mount(root);
  cleanups.push(() => { app.unmount(); root.remove(); });
  return root;
}

function button(root: HTMLElement, label: string) {
  const item = [...root.querySelectorAll("button")].find((b) => b.textContent?.includes(label));
  if (!item) throw new Error(`Missing button ${label}`);
  return item;
}

describe("review interaction regressions", () => {
  it("rejects oversized selected/dropped files before any upload", () => {
    const error = vi.fn();
    const { file, selectFile } = useUploadFile(error);
    const candidate = new File(["ok"], "small.txt");
    expect(selectFile(candidate)).toBe(true);
    const oversized = new File([], "large.bin");
    Object.defineProperty(oversized, "size", { value: DATA_FILE_MAX_BYTES + 1 });
    expect(selectFile(oversized)).toBe(false);
    expect(file.value).toBeNull();
    expect(error).toHaveBeenCalledWith("单文件不能超过 511 MiB");
  });

  it("accepts the documented file boundary", () => {
    const { selectFile } = useUploadFile(vi.fn());
    const candidate = new File([], "boundary.bin");
    Object.defineProperty(candidate, "size", { value: DATA_FILE_MAX_BYTES });
    expect(selectFile(candidate)).toBe(true);
  });

  it("preserves transfer submit and cancel events", () => {
    const submit = vi.fn(), close = vi.fn();
    const root = mount(DocumentTransferDialog, { modelValue: true, form: { to_user_id: "receiver" }, selectableUsers: [], transferLoading: false, onSubmit: submit, "onUpdate:modelValue": close });
    button(root, "确认转交").click();
    expect(submit).toHaveBeenCalledOnce();
    button(root, "取消").click();
    expect(close).toHaveBeenCalledWith(false);
  });

  it("does not submit a handover while saving", () => {
    const submit = vi.fn();
    const root = mount(HandoverCreateDialog, { modelValue: true, form: { target_user_id: "a", receiver_user_id: "b", project_id: "p", remark: "" }, users: [], projects: [], referenceLoading: false, formLoading: true, onSubmit: submit });
    button(root, "确认创建").click();
    expect(submit).not.toHaveBeenCalled();
  });

  it("submits the document form through its parent action", () => {
    const submit = vi.fn();
    const root = mount(DocumentCreateDialog, { modelValue: true, form: {}, users: [], projects: [], teamSpaces: [], folderOptions: [], referenceLoading: false, createLoading: false, onSubmit: submit });
    button(root, "创建").click();
    expect(submit).toHaveBeenCalledOnce();
  });

  it("does not send a second question while thinking, including Ctrl Enter", async () => {
    const submit = vi.fn();
    const root = mount(AssistantComposer, { modelValue: "question", recommendedPrompts: [], composerScopeType: "project", composerScopeLabel: "project", thinking: true, loading: false, onSubmit: submit });
    button(root, "处理中").click();
    root.querySelector("input")!.dispatchEvent(new KeyboardEvent("keyup", { key: "Enter", ctrlKey: true, bubbles: true }));
    await nextTick();
    expect(submit).not.toHaveBeenCalled();
  });

  it("sends a question and forwards a suggested prompt", () => {
    const submit = vi.fn(), prompt = vi.fn();
    const root = mount(AssistantComposer, { modelValue: "question", recommendedPrompts: ["summary"], composerScopeType: "project", composerScopeLabel: "project", thinking: false, loading: false, onSubmit: submit, onPrompt: prompt });
    button(root, "发送").click();
    button(root, "summary").click();
    expect(submit).toHaveBeenCalledOnce();
    expect(prompt).toHaveBeenCalledWith("summary");
  });
});
