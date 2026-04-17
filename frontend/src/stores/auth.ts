import { defineStore } from "pinia";
import { ref } from "vue";

export const useAuthStore = defineStore("auth", () => {
  const token = ref<string | null>(localStorage.getItem("access_token"));
  const username = ref(localStorage.getItem("username") ?? "");
  const displayName = ref(localStorage.getItem("display_name") ?? "");
  const role = ref(localStorage.getItem("role") ?? "");
  const userId = ref(localStorage.getItem("user_id") ?? "");

  function login(payload: { token: string; id: string; username: string; displayName: string; role: string }) {
    token.value = payload.token;
    userId.value = payload.id;
    username.value = payload.username;
    displayName.value = payload.displayName;
    role.value = payload.role;
    localStorage.setItem("access_token", payload.token);
    localStorage.setItem("user_id", payload.id);
    localStorage.setItem("username", payload.username);
    localStorage.setItem("display_name", payload.displayName);
    localStorage.setItem("role", payload.role);
  }

  function updateProfile(payload: { username: string; displayName: string; role: string }) {
    username.value = payload.username;
    displayName.value = payload.displayName;
    role.value = payload.role;
    localStorage.setItem("username", payload.username);
    localStorage.setItem("display_name", payload.displayName);
    localStorage.setItem("role", payload.role);
  }

  function logout() {
    token.value = null;
    userId.value = "";
    username.value = "";
    displayName.value = "";
    role.value = "";
    localStorage.removeItem("access_token");
    localStorage.removeItem("user_id");
    localStorage.removeItem("username");
    localStorage.removeItem("display_name");
    localStorage.removeItem("role");
  }

  return {
    token,
    userId,
    username,
    displayName,
    role,
    login,
    updateProfile,
    logout,
  };
});
