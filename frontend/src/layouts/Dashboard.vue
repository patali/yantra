<template>
  <v-layout>
    <v-navigation-drawer
      v-model="drawer"
      app
      :permanent="$vuetify.display.mdAndUp"
      :rail="rail && $vuetify.display.mdAndUp"
      :width="rail ? 72 : 220"
    >
      <!-- Header with logo and collapse button -->
      <v-list-item
        v-if="!rail"
        class="pa-4"
        title="Yantra"
        :subtitle="authStore.admin?.username"
      >
        <template #prepend>
          <v-icon
            icon="mdi-graph-outline"
            size="40"
            color="primary"
          />
        </template>
        <template #append>
          <v-btn
            icon="mdi-chevron-left"
            variant="text"
            size="small"
            @click="rail = true"
          />
        </template>
      </v-list-item>

      <!-- Collapsed header -->
      <v-list-item
        v-else
        class="pa-2"
      >
        <template #prepend>
          <v-btn
            icon="mdi-menu"
            variant="text"
            @click="rail = false"
          />
        </template>
      </v-list-item>

      <v-divider />

      <v-list
        density="compact"
        nav
      >
        <v-list-item
          prepend-icon="mdi-sitemap"
          :title="rail ? undefined : 'Workflows'"
          value="workflows"
          to="/workflows"
        >
          <v-tooltip
            v-if="rail"
            activator="parent"
            location="end"
          >
            Workflows
          </v-tooltip>
        </v-list-item>
        <v-list-item
          prepend-icon="mdi-account-multiple"
          :title="rail ? undefined : 'Users'"
          value="users"
          to="/users"
        >
          <v-tooltip
            v-if="rail"
            activator="parent"
            location="end"
          >
            Users
          </v-tooltip>
        </v-list-item>
        <v-list-item
          prepend-icon="mdi-play-circle-outline"
          :title="rail ? undefined : 'Runs'"
          value="runs"
          to="/runs"
        >
          <v-tooltip
            v-if="rail"
            activator="parent"
            location="end"
          >
            Runs
          </v-tooltip>
        </v-list-item>
      </v-list>

      <template #append>
        <div>
          <v-list
            density="compact"
            nav
          >
            <v-divider class="my-2" />
            <v-list-item
              prepend-icon="mdi-cog"
              :title="rail ? undefined : 'Settings'"
              value="settings"
              to="/settings"
            >
              <v-tooltip
                v-if="rail"
                activator="parent"
                location="end"
              >
                Settings
              </v-tooltip>
            </v-list-item>
            <v-list-item
              prepend-icon="mdi-logout"
              :title="rail ? undefined : 'Logout'"
              value="logout"
              class="logout-button"
              @click="handleLogout"
            >
              <v-tooltip
                v-if="rail"
                activator="parent"
                location="end"
              >
                Logout
              </v-tooltip>
            </v-list-item>
          </v-list>
        </div>
      </template>
    </v-navigation-drawer>

    <v-app-bar
      color="surface"
      density="compact"
    >
      <v-app-bar-nav-icon
        v-if="$vuetify.display.smAndDown"
        @click="drawer = !drawer"
      />
      <v-toolbar-title>{{ currentPageTitle }}</v-toolbar-title>
      <v-spacer />
      <v-btn
        icon
        :loading="themeSwitching"
        class="mr-2"
        @click="toggleTheme"
      >
        <v-icon>{{ isDark ? 'mdi-white-balance-sunny' : 'mdi-weather-night' }}</v-icon>
      </v-btn>
      <v-menu>
        <template #activator="{ props }">
          <v-chip
            v-bind="props"
            class="mr-4"
            variant="elevated"
          >
            <v-icon
              start
              icon="mdi-account-circle"
            />
            {{ authStore.admin?.username }}
            <v-icon
              end
              icon="mdi-chevron-down"
              size="small"
            />
          </v-chip>
        </template>
        <v-list>
          <v-list-item
            prepend-icon="mdi-account-cog"
            title="Account Settings"
            @click="router.push('/settings/account')"
          />
          <v-divider />
          <v-list-item
            prepend-icon="mdi-logout"
            title="Logout"
            @click="handleLogout"
          />
        </v-list>
      </v-menu>
    </v-app-bar>

    <v-main>
      <router-view />
    </v-main>
  </v-layout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from "vue";
import { useRouter, useRoute } from "vue-router";
import { useTheme } from "vuetify";
import { useAuthStore } from "@/stores/auth";

const router = useRouter();
const route = useRoute();
const theme = useTheme();
const authStore = useAuthStore();

const drawer = ref(true);
const rail = ref(false);
const themeSwitching = ref(false);

const currentPageTitle = computed(() => {
  const name = route.name as string;
  return name || "Dashboard";
});

const isDark = computed(() => theme.global.name.value === "dark");

const toggleTheme = async () => {
  try {
    themeSwitching.value = true;
    const newTheme = isDark.value ? "light" : "dark";

    // Update Vuetify theme using the new API
    theme.global.name.value = newTheme;

    // Save to backend
    await authStore.updateTheme(newTheme);
  } catch (_error) {
    // Revert on error
    const revertTheme = isDark.value ? "dark" : "light";
    theme.global.name.value = revertTheme;
  } finally {
    themeSwitching.value = false;
  }
};

const handleLogout = () => {
  authStore.logout();
  router.push("/login");
};

// Apply saved theme on mount
onMounted(() => {
  if (authStore.admin?.theme) {
    // Use the Vuetify 3 API
    theme.global.name.value = authStore.admin.theme;
  }
});
</script>

<style scoped>
.logout-button {
  color: white;
  font-weight: 800;
  background-color: rgb(221, 54, 54);
  &:hover {
    background-color: rgb(221, 54, 54);
  }
}
</style>