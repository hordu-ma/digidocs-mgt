import { createRouter, createWebHistory } from "vue-router";

import AssistantView from "@/views/AssistantView.vue";
import AdminView from "@/views/AdminView.vue";
import DashboardView from "@/views/DashboardView.vue";
import DocumentDetailView from "@/views/DocumentDetailView.vue";
import DocumentsView from "@/views/DocumentsView.vue";
import HandoversView from "@/views/HandoversView.vue";
import LoginView from "@/views/LoginView.vue";
import ProfileView from "@/views/ProfileView.vue";

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: "/login", name: "login", component: LoginView },
    { path: "/", redirect: "/dashboard" },
    { path: "/dashboard", name: "dashboard", component: DashboardView, meta: { requiresAuth: true } },
    { path: "/documents", name: "documents", component: DocumentsView, meta: { requiresAuth: true } },
    { path: "/documents/:id", name: "document-detail", component: DocumentDetailView, meta: { requiresAuth: true } },
    { path: "/handovers", name: "handovers", component: HandoversView, meta: { requiresAuth: true } },
    { path: "/assistant", name: "assistant", component: AssistantView, meta: { requiresAuth: true } },
    { path: "/admin", name: "admin", component: AdminView, meta: { requiresAuth: true, requiresAdmin: true } },
    { path: "/profile", name: "profile", component: ProfileView, meta: { requiresAuth: true } },
  ],
});

router.beforeEach((to) => {
  const token = localStorage.getItem("access_token");
  if (to.meta.requiresAuth && !token) {
    return { name: "login" };
  }
  if (to.meta.requiresAdmin && localStorage.getItem("role") !== "admin") {
    return { name: "dashboard" };
  }
});

export default router;
