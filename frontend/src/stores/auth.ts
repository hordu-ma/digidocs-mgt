import { defineStore } from "pinia";
import { ref } from "vue";

export const useAuthStore = defineStore("auth", () => {
  const token = ref<string | null>(localStorage.getItem("access_token"));
  const displayName = ref(localStorage.getItem("display_name") ?? "");
  const role = ref(localStorage.getItem("role") ?? "");
  const userId = ref(localStorage.getItem("user_id") ?? "");

  function login(payload: { token: string; id: string; displayName: string; role: string }) {
    token.value = payload.token;
    userId.value = payload.id;
    displayName.value = payload.displayName;
    role.value = payload.role;
    localStorage.setItem("access_token", payload.token);
    localStorage.setItem("user_id", payload.id);
    localStorage.setItem("display_name", payload.displayName);
    localStorage.setItem("role", payload.role);
  }

  function logout() {
    token.value = null;
    userId.value = "";
    displayName.value = "";
    role.value = "";
    localStorage.removeItem("access_token");
    localStorage.removeItem("user_id");
    localStorage.removeItem("display_name");
    localStorage.removeItem("role");
  }

  return {
    token,
    userId,
    displayName,
    role,
    login,
    logout,
  };
});

