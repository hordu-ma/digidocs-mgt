import { createRouter, createWebHistory } from "vue-router";

const AssistantView = () => import("@/views/AssistantView.vue");
const AdminView = () => import("@/views/AdminView.vue");
const DashboardView = () => import("@/views/DashboardView.vue");
const DataView = () => import("@/views/DataView.vue");
const CodeView = () => import("@/views/CodeView.vue");
const DocumentDetailView = () => import("@/views/DocumentDetailView.vue");
const DocumentsView = () => import("@/views/DocumentsView.vue");
const HandoversView = () => import("@/views/HandoversView.vue");
const LoginView = () => import("@/views/LoginView.vue");
const ProfileView = () => import("@/views/ProfileView.vue");

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: "/login", name: "login", component: LoginView },
    { path: "/", redirect: "/dashboard" },
    { path: "/dashboard", name: "dashboard", component: DashboardView, meta: { requiresAuth: true } },
    { path: "/documents", name: "documents", component: DocumentsView, meta: { requiresAuth: true } },
    { path: "/documents/:id", name: "document-detail", component: DocumentDetailView, meta: { requiresAuth: true } },
    { path: "/data", name: "data", component: DataView, meta: { requiresAuth: true } },
    { path: "/code", name: "code", component: CodeView, meta: { requiresAuth: true } },
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
