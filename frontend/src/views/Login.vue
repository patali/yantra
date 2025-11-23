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
                icon="mdi-graph-outline"
                size="36"
                class="mr-2"
              />
              <span>Yantra</span>
            </div>
          </v-card-title>

          <v-card-text class="pa-8">
            <v-form @submit.prevent="handleLogin">
              <v-text-field
                v-model="username"
                label="Username"
                prepend-icon="mdi-account"
                variant="outlined"
                :rules="[rules.required]"
                class="mb-4"
              />

              <v-text-field
                v-model="password"
                label="Password"
                type="password"
                prepend-icon="mdi-lock"
                variant="outlined"
                :rules="[rules.required]"
                class="mb-2"
              />

              <div class="text-right mb-4">
                <v-btn
                  variant="text"
                  color="primary"
                  size="small"
                  @click="goToForgotPassword"
                >
                  Forgot Password?
                </v-btn>
              </div>

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
                Login
              </v-btn>

              <v-divider class="my-4" />

              <div class="text-center">
                <span class="text-body-2">Don't have an account?</span>
                <v-btn
                  variant="text"
                  color="primary"
                  size="small"
                  @click="goToSignup"
                >
                  Sign Up
                </v-btn>
              </div>
            </v-form>
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

const username = ref("");
const password = ref("");
const error = ref("");
const loading = ref(false);

const rules = {
  required: (value: string) => !!value || "This field is required",
};

const handleLogin = async () => {
  if (!username.value || !password.value) {
    error.value = "Please fill in all fields";
    return;
  }

  try {
    loading.value = true;
    error.value = "";
    await authStore.login(username.value, password.value);
    router.push("/");
  } catch (err: any) {
    error.value = err.message || "Login failed";
  } finally {
    loading.value = false;
  }
};

const goToSignup = () => {
  router.push("/signup");
};

const goToForgotPassword = () => {
  router.push("/forgot-password");
};
</script>

