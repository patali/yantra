import { createRouter, createWebHistory, RouteRecordRaw } from "vue-router";
import { useAuthStore } from "@/stores/auth";

const routes: RouteRecordRaw[] = [
  {
    path: "/login",
    name: "Login",
    component: () => import("@/views/Login.vue"),
    meta: { requiresAuth: false },
  },
  {
    path: "/signup",
    name: "Signup",
    component: () => import("@/views/Signup.vue"),
    meta: { requiresAuth: false },
  },
  {
    path: "/forgot-password",
    name: "ForgotPassword",
    component: () => import("@/views/ForgotPassword.vue"),
    meta: { requiresAuth: false },
  },
  {
    path: "/reset-password",
    name: "ResetPassword",
    component: () => import("@/views/ResetPassword.vue"),
    meta: { requiresAuth: false },
  },
  {
    path: "/",
    component: () => import("@/layouts/Dashboard.vue"),
    meta: { requiresAuth: true },
    children: [
      {
        path: "",
        redirect: "/workflows",
      },
      {
        path: "/users",
        name: "Users",
        component: () => import("@/views/Users.vue"),
      },
      {
        path: "/workflows",
        name: "Workflows",
        component: () => import("@/views/Workflows.vue"),
      },
      {
        path: "/workflows/new",
        name: "WorkflowNew",
        component: () => import("@/views/WorkflowEditor.vue"),
      },
      {
        path: "/workflows/:id/edit",
        name: "WorkflowEditor",
        component: () => import("@/views/WorkflowEditor.vue"),
      },
      {
        path: "/workflows/:id/executions/:executionId",
        name: "WorkflowExecution",
        component: () => import("@/views/WorkflowExecution.vue"),
      },
      {
        path: "/runs",
        name: "Runs",
        component: () => import("@/views/Runs.vue"),
      },
      {
        path: "/settings/account",
        name: "AccountSettings",
        component: () => import("@/views/AccountSettings.vue"),
      },
      {
        path: "/settings",
        name: "Settings",
        component: () => import("@/views/AppSettings.vue"),
      },
    ],
  },
];

const router = createRouter({
  history: createWebHistory(),
  routes,
});

router.beforeEach((to, _from, next) => {
  const authStore = useAuthStore();
  const requiresAuth = to.matched.some((record) => record.meta.requiresAuth);

  if (requiresAuth && !authStore.isAuthenticated) {
    next("/login");
  } else if (to.path === "/login" && authStore.isAuthenticated) {
    next("/");
  } else {
    next();
  }
});

export default router;
