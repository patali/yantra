<template>
  <v-container
    fluid
    class="fill-height"
  >
    <v-row
      align="center"
      justify="center"
    >
      <v-col
        cols="12"
        sm="8"
        md="4"
      >
        <v-card elevation="8">
          <v-card-title class="text-h4 text-center pa-6 bg-primary">
            <div class="d-flex align-center justify-center text-white">
              <v-icon
                icon="mdi-lock-reset"
                size="36"
                class="mr-2"
              />
              <span>Reset Password</span>
            </div>
          </v-card-title>

          <v-card-text class="pa-8">
            <div v-if="!resetSuccess">
              <p class="text-body-1 mb-6">
                Enter your new password below.
              </p>

              <v-form @submit.prevent="handleSubmit">
                <v-text-field
                  v-model="newPassword"
                  label="New Password"
                  type="password"
                  prepend-icon="mdi-lock"
                  variant="outlined"
                  :rules="[rules.required, rules.minLength]"
                  class="mb-4"
                />

                <v-text-field
                  v-model="confirmPassword"
                  label="Confirm Password"
                  type="password"
                  prepend-icon="mdi-lock-check"
                  variant="outlined"
                  :rules="[rules.required, rules.match]"
                  class="mb-4"
                />

                <v-alert
                  v-if="error"
                  type="error"
                  class="mb-4"
                >
                  {{ error }}
                </v-alert>

                <v-btn
                  type="submit"
                  color="primary"
                  size="large"
                  block
                  :loading="loading"
                  :disabled="!token"
                >
                  Reset Password
                </v-btn>

                <v-divider class="my-4" />

                <div class="text-center">
                  <v-btn
                    variant="text"
                    color="primary"
                    size="small"
                    @click="goToLogin"
                  >
                    Back to Login
                  </v-btn>
                </div>
              </v-form>
            </div>

            <div v-else>
              <v-alert
                type="success"
                class="mb-4"
              >
                <p class="mb-0">
                  Your password has been reset successfully! You can now log in with your new password.
                </p>
              </v-alert>

              <v-btn
                color="primary"
                size="large"
                block
                @click="goToLogin"
              >
                Go to Login
              </v-btn>
            </div>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>
  </v-container>
</template>

<script setup lang="ts">
import { ref, onMounted } from "vue";
import { useRouter, useRoute } from "vue-router";
import { useAuthStore } from "@/stores/auth";

const router = useRouter();
const route = useRoute();
const authStore = useAuthStore();

const token = ref("");
const newPassword = ref("");
const confirmPassword = ref("");
const error = ref("");
const loading = ref(false);
const resetSuccess = ref(false);

const rules = {
  required: (value: string) => !!value || "This field is required",
  minLength: (value: string) => value.length >= 6 || "Password must be at least 6 characters",
  match: (value: string) => value === newPassword.value || "Passwords do not match",
};

onMounted(() => {
  // Get token from URL query parameter
  token.value = route.query.token as string || "";

  if (!token.value) {
    error.value = "Invalid or missing reset token. Please request a new password reset.";
  }
});

const handleSubmit = async () => {
  if (!newPassword.value || !confirmPassword.value) {
    error.value = "Please fill in all fields";
    return;
  }

  if (newPassword.value.length < 6) {
    error.value = "Password must be at least 6 characters";
    return;
  }

  if (newPassword.value !== confirmPassword.value) {
    error.value = "Passwords do not match";
    return;
  }

  if (!token.value) {
    error.value = "Invalid reset token";
    return;
  }

  try {
    loading.value = true;
    error.value = "";
    await authStore.resetPassword(token.value, newPassword.value);
    resetSuccess.value = true;
  } catch (err: any) {
    error.value = err.message || "Failed to reset password";
  } finally {
    loading.value = false;
  }
};

const goToLogin = () => {
  router.push("/login");
};
</script>
