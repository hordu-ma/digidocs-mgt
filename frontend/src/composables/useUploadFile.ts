import { ref } from "vue";

// Reserve 1 MiB for multipart fields beneath the API's 512 MiB request limit.
export const DATA_FILE_MAX_BYTES = 511 * 1024 * 1024;
export const DATA_FILE_LIMIT_LABEL = "511 MiB";

export function useUploadFile(onError: (message: string) => void) {
  const file = ref<File | null>(null);
  function selectFile(candidate: File): boolean {
    if (candidate.size > DATA_FILE_MAX_BYTES) {
      file.value = null;
      onError(`单文件不能超过 ${DATA_FILE_LIMIT_LABEL}`);
      return false;
    }
    file.value = candidate;
    return true;
  }
  return { file, selectFile };
}
