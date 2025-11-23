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
        md="5"
      >
        <v-card elevation="8">
          <v-card-title class="text-h4 text-center pa-6 bg-primary">
            <div class="d-flex align-center justify-center text-white">
              <v-icon
                icon="mdi-graph-outline"
                size="36"
                class="mr-2"
              />
              <span>Sign Up for Yantra</span>
            </div>
          </v-card-title>

          <v-card-text class="pa-8">
            <v-form @submit.prevent="handleSignup">
              <v-text-field
                v-model="accountName"
                label="Account Name"
                prepend-icon="mdi-domain"
                variant="outlined"
                :rules="[rules.required]"
                hint="Your organization or team name"
                persistent-hint
                class="mb-4"
              />

              <v-text-field
                v-model="username"
                label="Username"
                prepend-icon="mdi-account"
                variant="outlined"
                :rules="[rules.required, rules.minLength(3)]"
                class="mb-4"
              />

              <v-text-field
                v-model="email"
                label="Email"
                type="email"
                prepend-icon="mdi-email"
                variant="outlined"
                :rules="[rules.required, rules.email]"
                class="mb-4"
              />

              <v-text-field
                v-model="password"
                label="Password"
                type="password"
                prepend-icon="mdi-lock"
                variant="outlined"
                :rules="[rules.required, rules.minLength(8)]"
                class="mb-4"
              />

              <v-alert
                v-if="error"
                type="error"
                class="mb-4"
              >
                {{ error }}
              </v-alert>

              <v-alert
                v-if="success"
                type="success"
                class="mb-4"
              >
                {{ success }}
              </v-alert>

              <v-btn
                type="submit"
                color="primary"
                size="large"
                block
                :loading="loading"
                class="mb-4"
              >
                Create Account
              </v-btn>

              <v-divider class="my-4" />

              <div class="text-center">
                <span class="text-body-2">Already have an account?</span>
                <v-btn
                  variant="text"
                  color="primary"
                  size="small"
                  @click="goToLogin"
                >
                  Sign In
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
import api from "@/services/api";

const router = useRouter();
const authStore = useAuthStore();

const accountName = ref("");
const username = ref("");
const email = ref("");
const password = ref("");
const error = ref("");
const success = ref("");
const loading = ref(false);

const rules = {
  required: (value: string) => !!value || "This field is required",
  minLength: (min: number) => (value: string) =>
    value.length >= min || `Minimum ${min} characters`,
  email: (value: string) =>
    /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value) || "Invalid email",
};

const handleSignup = async () => {
  if (!accountName.value || !username.value || !email.value || !password.value) {
    error.value = "Please fill in all fields";
    return;
  }

  try {
    loading.value = true;
    error.value = "";
    success.value = "";

    const response = await api.post("/auth/signup", {
      name: accountName.value,
      username: username.value,
      email: email.value,
      password: password.value,
    });

    // Store token and user data
    authStore.setToken(response.data.token);
    authStore.admin = response.data.user;

    success.value = "Account created successfully! Redirecting...";

    // Redirect to dashboard after a short delay
    setTimeout(() => {
      router.push("/");
    }, 1500);
  } catch (err: any) {
    error.value = err.response?.data?.error || "Signup failed. Please try again.";
  } finally {
    loading.value = false;
  }
};

const goToLogin = () => {
  router.push("/login");
};
</script>

