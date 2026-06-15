import axios from "axios";
import { useAuthStore } from "@/stores/auth";
import router from "@/router";

const api = axios.create({
    baseURL: "/api/v1",
    headers: { "Content-Type": "application/json" },
});

api.interceptors.request.use((config) => {
    const auth = useAuthStore();
    if (auth.token) {
        config.headers.Authorization = `Bearer ${auth.token}`;
    }
    return config;
});

// Centralized handling for expired/invalid sessions: a 401 on any authenticated
// request clears the session and routes back to login. The login request itself
// is exempt so wrong-credential errors stay on the login screen.
api.interceptors.response.use(
    (response) => response,
    (error) => {
        const status = error?.response?.status;
        const url = String(error?.config?.url ?? "");
        if (status === 401 && !url.includes("/auth/login")) {
            const auth = useAuthStore();
            if (auth.token) {
                auth.logout();
                if (router.currentRoute.value.name !== "login") {
                    void router.push({ name: "login" });
                }
            }
        }
        return Promise.reject(error);
    },
);

export default api;
