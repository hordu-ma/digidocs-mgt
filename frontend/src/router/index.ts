import { createRouter, createWebHistory } from "vue-router";

import AssistantView from "@/views/AssistantView.vue";
import DashboardView from "@/views/DashboardView.vue";
import DocumentDetailView from "@/views/DocumentDetailView.vue";
import DocumentsView from "@/views/DocumentsView.vue";
import HandoversView from "@/views/HandoversView.vue";
import LoginView from "@/views/LoginView.vue";

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: "/login", name: "login", component: LoginView },
    { path: "/", redirect: "/dashboard" },
    { path: "/dashboard", name: "dashboard", component: DashboardView },
    { path: "/documents", name: "documents", component: DocumentsView },
    { path: "/documents/:id", name: "document-detail", component: DocumentDetailView },
    { path: "/handovers", name: "handovers", component: HandoversView },
    { path: "/assistant", name: "assistant", component: AssistantView },
  ],
});

export default router;
