<template>
  <v-container
    fluid
    class="pa-6"
    style="height: 100%; overflow-y: auto;"
  >
    <v-row>
      <v-col cols="12">
        <h1 class="text-h4 mb-6">
          Account Settings
        </h1>
      </v-col>
    </v-row>

    <!-- Change Password Card -->
    <v-row>
      <v-col cols="12">
        <v-card>
          <v-card-title class="bg-primary">
            <v-icon
              start
              icon="mdi-lock-outline"
            />
            Change Password
          </v-card-title>

          <v-card-text class="pa-6">
            <v-form @submit.prevent="handleChangePassword">
              <v-text-field
                v-model="currentPassword"
                label="Current Password"
                type="password"
                prepend-icon="mdi-lock"
                variant="outlined"
                class="mb-4"
              />

              <v-text-field
                v-model="newPassword"
                label="New Password"
                type="password"
                prepend-icon="mdi-lock-plus"
                variant="outlined"
                hint="Password must be at least 6 characters"
                persistent-hint
                class="mb-4"
              />

              <v-text-field
                v-model="confirmNewPassword"
                label="Confirm New Password"
                type="password"
                prepend-icon="mdi-lock-check"
                variant="outlined"
                class="mb-4"
              />

              <v-alert
                v-if="passwordError"
                type="error"
                class="mb-4"
              >
                {{ passwordError }}
              </v-alert>

              <v-alert
                v-if="passwordSuccess"
                type="success"
                class="mb-4"
              >
                {{ passwordSuccess }}
              </v-alert>

              <v-btn
                type="submit"
                color="primary"
                prepend-icon="mdi-content-save"
                :loading="changingPassword"
              >
                Change Password
              </v-btn>
            </v-form>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>
  </v-container>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { useAuthStore } from "@/stores/auth";

const authStore = useAuthStore();

// Password change state
const currentPassword = ref("");
const newPassword = ref("");
const confirmNewPassword = ref("");
const passwordError = ref("");
const passwordSuccess = ref("");
const changingPassword = ref(false);

const handleChangePassword = async () => {
  // Reset messages
  passwordError.value = "";
  passwordSuccess.value = "";

  // Validation
  if (!currentPassword.value || !newPassword.value || !confirmNewPassword.value) {
    passwordError.value = "Please fill in all fields";
    return;
  }

  if (newPassword.value.length < 6) {
    passwordError.value = "New password must be at least 6 characters";
    return;
  }

  if (newPassword.value !== confirmNewPassword.value) {
    passwordError.value = "New passwords do not match";
    return;
  }

  try {
    changingPassword.value = true;
    await authStore.changePassword(currentPassword.value, newPassword.value);

    passwordSuccess.value = "Password changed successfully!";

    // Clear form
    currentPassword.value = "";
    newPassword.value = "";
    confirmNewPassword.value = "";
  } catch (error: any) {
    passwordError.value = error.message || "Failed to change password";
  } finally {
    changingPassword.value = false;
  }
};
</script>
