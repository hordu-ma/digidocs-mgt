import { defineStore } from "pinia";
import { ref } from "vue";

export const useAuthStore = defineStore("auth", () => {
  const token = ref<string | null>(null);
  const displayName = ref("系统管理员");
  const role = ref("admin");

  function login(nextToken: string) {
    token.value = nextToken;
  }

  function logout() {
    token.value = null;
  }

  return {
    token,
    displayName,
    role,
    login,
    logout,
  };
});
