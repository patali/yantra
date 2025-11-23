import { defineStore } from "pinia";
import { ref, computed } from "vue";
import api from "@/services/api";
import type { Admin } from "@/types";

export const useAuthStore = defineStore("auth", () => {
  const token = ref<string | null>(localStorage.getItem("token"));
  const admin = ref<Admin | null>(null);
  const loading = ref(false);

  const isAuthenticated = computed(() => !!token.value);

  const setToken = (newToken: string) => {
    token.value = newToken;
    localStorage.setItem("token", newToken);
    api.defaults.headers.common["Authorization"] = `Bearer ${newToken}`;
  };

  const clearToken = () => {
    token.value = null;
    admin.value = null;
    localStorage.removeItem("token");
    delete api.defaults.headers.common["Authorization"];
  };

  const login = async (username: string, password: string) => {
    try {
      loading.value = true;
      const response = await api.post("/auth/login", { username, password });
      setToken(response.data.token);
      admin.value = response.data.user;
      return true;
    } catch (error: any) {
      throw new Error(error.response?.data?.error || "Login failed");
    } finally {
      loading.value = false;
    }
  };

  const logout = () => {
    clearToken();
  };

  const checkAuth = async () => {
    if (!token.value) return;

    try {
      api.defaults.headers.common["Authorization"] = `Bearer ${token.value}`;
      const response = await api.get("/users/me");
      admin.value = response.data;
    } catch (_error) {
      clearToken();
    }
  };

  const updateTheme = async (theme: string) => {
    try {
      const response = await api.post("/users/theme", { theme });
      if (admin.value) {
        admin.value.theme = response.data.theme;
      }
      return response.data;
    } catch (error: any) {
      throw new Error(error.response?.data?.error || "Failed to update theme");
    }
  };

  const requestPasswordReset = async (email: string) => {
    try {
      loading.value = true;
      const response = await api.post("/auth/request-password-reset", { email });
      return response.data;
    } catch (error: any) {
      throw new Error(error.response?.data?.error || "Failed to request password reset");
    } finally {
      loading.value = false;
    }
  };

  const resetPassword = async (token: string, newPassword: string) => {
    try {
      loading.value = true;
      const response = await api.post("/auth/reset-password", {
        token,
        newPassword
      });
      return response.data;
    } catch (error: any) {
      throw new Error(error.response?.data?.error || "Failed to reset password");
    } finally {
      loading.value = false;
    }
  };

  const changePassword = async (currentPassword: string, newPassword: string) => {
    try {
      loading.value = true;
      const response = await api.post("/auth/change-password", {
        currentPassword,
        newPassword
      });
      return response.data;
    } catch (error: any) {
      throw new Error(error.response?.data?.error || "Failed to change password");
    } finally {
      loading.value = false;
    }
  };

  return {
    token,
    admin,
    loading,
    isAuthenticated,
    setToken,
    login,
    logout,
    checkAuth,
    updateTheme,
    requestPasswordReset,
    resetPassword,
    changePassword,
  };
});

