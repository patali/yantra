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
            <div v-if="!submitted">
              <p class="text-body-1 mb-6">
                Enter your email address and we'll send you a link to reset your password.
              </p>

              <v-form @submit.prevent="handleSubmit">
                <v-text-field
                  v-model="email"
                  label="Email"
                  type="email"
                  prepend-icon="mdi-email"
                  variant="outlined"
                  :rules="[rules.required, rules.email]"
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
                >
                  Send Reset Link
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
                  If an account with that email exists, a password reset link has been sent.
                  Please check your email inbox.
                </p>
              </v-alert>

              <v-btn
                color="primary"
                size="large"
                block
                @click="goToLogin"
              >
                Return to Login
              </v-btn>
            </div>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>
  </v-container>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { useRouter } from "vue-router";
import { useAuthStore } from "@/stores/auth";

const router = useRouter();
const authStore = useAuthStore();

const email = ref("");
const error = ref("");
const loading = ref(false);
const submitted = ref(false);

const rules = {
  required: (value: string) => !!value || "This field is required",
  email: (value: string) => {
    const pattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    return pattern.test(value) || "Invalid email address";
  },
};

const handleSubmit = async () => {
  if (!email.value) {
    error.value = "Please enter your email address";
    return;
  }

  try {
    loading.value = true;
    error.value = "";
    await authStore.requestPasswordReset(email.value);
    submitted.value = true;
  } catch (err: any) {
    error.value = err.message || "Failed to send reset link";
  } finally {
    loading.value = false;
  }
};

const goToLogin = () => {
  router.push("/login");
};
</script>
