<template>
  <v-app>
    <router-view />
  </v-app>
</template>

<script setup lang="ts">
import { onMounted, watch } from "vue";
import { useTheme } from "vuetify";
import { useAuthStore } from "./stores/auth";

const theme = useTheme();
const authStore = useAuthStore();

onMounted(() => {
  authStore.checkAuth();
});

// Watch for admin data and apply theme
watch(
  () => authStore.admin?.theme,
  (newTheme) => {
    if (newTheme) {
      theme.global.name.value = newTheme;
    }
  },
  { immediate: true }
);
</script>

