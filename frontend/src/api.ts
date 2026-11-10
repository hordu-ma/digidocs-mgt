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
        // change-password answers 401 for a wrong *current* password; that is
        // a form error, not an expired session, and must not log the user out.
        if (
            status === 401 &&
            !url.includes("/auth/login") &&
            !url.includes("/auth/register") &&
            !url.includes("/auth/change-password")
        ) {
            const auth = useAuthStore();
            if (auth.token) {
                const current = router.currentRoute.value;
                auth.logout();
                if (current.name !== "login") {
                    // Carry the page the session expired on so the user lands
                    // back where they were instead of on the dashboard.
                    void router.push({
                        name: "login",
                        query: current.fullPath && current.fullPath !== "/"
                            ? { redirect: current.fullPath }
                            : undefined,
                    });
                }
            }
        }
        return Promise.reject(error);
    },
);

export default api;
