<template>
  <v-container
    fluid
    class="pa-6"
    style="height: 100%; overflow-y: auto;"
  >
    <v-row>
      <v-col cols="12">
        <h1 class="text-h4 mb-6">
          App Settings
        </h1>
      </v-col>
    </v-row>

    <!-- Email Provider Settings Card -->
    <v-row>
      <v-col cols="12">
        <v-card>
          <v-card-title class="bg-primary">
            <v-icon
              start
              icon="mdi-email-outline"
            />
            Email Provider Settings
          </v-card-title>

          <v-card-text class="pa-6">
            <v-alert
              v-if="!activeProvider"
              type="warning"
              variant="tonal"
              class="mb-4"
            >
              No email provider configured. Please configure one below to enable email functionality in workflows.
            </v-alert>

            <v-alert
              v-else
              type="success"
              variant="tonal"
              class="mb-4"
            >
              Active provider: <strong>{{ activeProvider.toUpperCase() }}</strong>
            </v-alert>

            <!-- Provider Selector -->
            <v-select
              v-model="selectedProvider"
              label="Email Provider"
              :items="providers"
              item-title="label"
              item-value="value"
              variant="outlined"
              class="mb-4"
              @update:model-value="loadProviderSettings"
            />

            <!-- Resend Configuration -->
            <div v-if="selectedProvider === 'resend'">
              <v-text-field
                v-model="resendConfig.apiKey"
                label="API Key"
                type="password"
                variant="outlined"
                hint="Get your API key from https://resend.com/api-keys"
                persistent-hint
                class="mb-4"
              />
              <v-text-field
                v-model="resendConfig.fromEmail"
                label="From Email"
                type="email"
                variant="outlined"
                hint="e.g., noreply@yourdomain.com"
                persistent-hint
                class="mb-4"
              />
              <v-text-field
                v-model="resendConfig.fromName"
                label="From Name (Optional)"
                variant="outlined"
                hint="e.g., Yantra"
                persistent-hint
                class="mb-4"
              />
            </div>

            <!-- Mailgun Configuration -->
            <div v-if="selectedProvider === 'mailgun'">
              <v-text-field
                v-model="mailgunConfig.apiKey"
                label="API Key"
                type="password"
                variant="outlined"
                hint="Get your API key from Mailgun dashboard"
                persistent-hint
                class="mb-4"
              />
              <v-text-field
                v-model="mailgunConfig.domain"
                label="Domain"
                variant="outlined"
                hint="e.g., mg.yourdomain.com"
                persistent-hint
                class="mb-4"
              />
              <v-text-field
                v-model="mailgunConfig.fromEmail"
                label="From Email"
                type="email"
                variant="outlined"
                hint="e.g., noreply@yourdomain.com"
                persistent-hint
                class="mb-4"
              />
              <v-text-field
                v-model="mailgunConfig.fromName"
                label="From Name (Optional)"
                variant="outlined"
                hint="e.g., Yantra"
                persistent-hint
                class="mb-4"
              />
            </div>

            <!-- AWS SES Configuration -->
            <div v-if="selectedProvider === 'ses'">
              <v-text-field
                v-model="sesConfig.accessKeyId"
                label="AWS Access Key ID"
                type="password"
                variant="outlined"
                hint="IAM access key with SES permissions"
                persistent-hint
                class="mb-4"
              />
              <v-text-field
                v-model="sesConfig.secretAccessKey"
                label="AWS Secret Access Key"
                type="password"
                variant="outlined"
                persistent-hint
                class="mb-4"
              />
              <v-text-field
                v-model="sesConfig.region"
                label="AWS Region"
                variant="outlined"
                hint="e.g., us-east-1, us-west-2"
                persistent-hint
                class="mb-4"
              />
              <v-text-field
                v-model="sesConfig.fromEmail"
                label="From Email"
                type="email"
                variant="outlined"
                hint="Must be verified in SES"
                persistent-hint
                class="mb-4"
              />
              <v-text-field
                v-model="sesConfig.fromName"
                label="From Name (Optional)"
                variant="outlined"
                hint="e.g., Yantra"
                persistent-hint
                class="mb-4"
              />
            </div>

            <!-- SMTP Configuration -->
            <div v-if="selectedProvider === 'smtp'">
              <v-text-field
                v-model="smtpConfig.smtpHost"
                label="SMTP Host"
                variant="outlined"
                hint="e.g., smtp.gmail.com, smtp.office365.com"
                persistent-hint
                class="mb-4"
              />
              <v-text-field
                v-model.number="smtpConfig.smtpPort"
                label="SMTP Port"
                type="number"
                variant="outlined"
                hint="Common ports: 465 (SSL), 587 (TLS), 25"
                persistent-hint
                class="mb-4"
              />
              <v-checkbox
                v-model="smtpConfig.smtpSecure"
                label="Use SSL/TLS (recommended for port 465)"
                hint="Enable for secure connection"
                persistent-hint
                class="mb-4"
              />
              <v-text-field
                v-model="smtpConfig.smtpUser"
                label="SMTP Username"
                variant="outlined"
                hint="Usually your email address"
                persistent-hint
                class="mb-4"
              />
              <v-text-field
                v-model="smtpConfig.smtpPassword"
                label="SMTP Password"
                type="password"
                variant="outlined"
                hint="App password or account password"
                persistent-hint
                class="mb-4"
              />
              <v-text-field
                v-model="smtpConfig.fromEmail"
                label="From Email"
                type="email"
                variant="outlined"
                hint="e.g., noreply@yourdomain.com"
                persistent-hint
                class="mb-4"
              />
              <v-text-field
                v-model="smtpConfig.fromName"
                label="From Name (Optional)"
                variant="outlined"
                hint="e.g., Yantra"
                persistent-hint
                class="mb-4"
              />
            </div>

            <!-- Action Buttons -->
            <div class="d-flex">
              <v-btn
                color="primary"
                class="me-4"
                prepend-icon="mdi-content-save"
                :loading="saving"
                @click="saveProvider"
              >
                Save & Activate
              </v-btn>
              <v-btn
                color="secondary"
                prepend-icon="mdi-email-check"
                :loading="testing"
                :disabled="!canTest"
                @click="testProvider"
              >
                Send Test Email
              </v-btn>
            </div>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <v-snackbar
      v-model="snackbar"
      :color="snackbarColor"
      timeout="3000"
    >
      {{ snackbarText }}
    </v-snackbar>
  </v-container>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from "vue";
import api from "@/services/api";

const selectedProvider = ref<"resend" | "mailgun" | "ses" | "smtp">("resend");
const activeProvider = ref<string | null>(null);

const resendConfig = ref({
  apiKey: "",
  fromEmail: "",
  fromName: "",
});

const mailgunConfig = ref({
  apiKey: "",
  domain: "",
  fromEmail: "",
  fromName: "",
});

const sesConfig = ref({
  accessKeyId: "",
  secretAccessKey: "",
  region: "us-east-1",
  fromEmail: "",
  fromName: "",
});

const smtpConfig = ref({
  smtpHost: "",
  smtpPort: 587,
  smtpSecure: false,
  smtpUser: "",
  smtpPassword: "",
  fromEmail: "",
  fromName: "",
});

const saving = ref(false);
const testing = ref(false);
const snackbar = ref(false);
const snackbarText = ref("");
const snackbarColor = ref("success");

const providers = [
  { label: "Resend", value: "resend" },
  { label: "Mailgun", value: "mailgun" },
  { label: "AWS SES", value: "ses" },
  { label: "SMTP (Universal)", value: "smtp" },
];

const canTest = computed(() => {
  if (selectedProvider.value === "resend") {
    return !!resendConfig.value.apiKey && !!resendConfig.value.fromEmail;
  } else if (selectedProvider.value === "mailgun") {
    return !!mailgunConfig.value.apiKey && !!mailgunConfig.value.domain;
  } else if (selectedProvider.value === "ses") {
    return (
      !!sesConfig.value.accessKeyId &&
      !!sesConfig.value.secretAccessKey &&
      !!sesConfig.value.region &&
      !!sesConfig.value.fromEmail
    );
  } else if (selectedProvider.value === "smtp") {
    return (
      !!smtpConfig.value.smtpHost &&
      !!smtpConfig.value.smtpPort &&
      !!smtpConfig.value.smtpUser &&
      !!smtpConfig.value.smtpPassword &&
      !!smtpConfig.value.fromEmail
    );
  }
  return false;
});

const showSnackbar = (message: string, color: string = "success") => {
  snackbarText.value = message;
  snackbarColor.value = color;
  snackbar.value = true;
};

const loadProviderSettings = async () => {
  if (!selectedProvider.value) {
    return; // Don't load if no provider selected
  }

  try {
    const response = await api.get(
      `/settings/email-providers/${selectedProvider.value}`
    );
    if (response.data) {
      // Populate form but don't show sensitive data
      if (selectedProvider.value === "resend") {
        resendConfig.value.fromEmail = response.data.fromEmail || "";
        resendConfig.value.fromName = response.data.fromName || "";
      } else if (selectedProvider.value === "mailgun") {
        mailgunConfig.value.domain = response.data.domain || "";
        mailgunConfig.value.fromEmail = response.data.fromEmail || "";
        mailgunConfig.value.fromName = response.data.fromName || "";
      } else if (selectedProvider.value === "ses") {
        sesConfig.value.region = response.data.region || "us-east-1";
        sesConfig.value.fromEmail = response.data.fromEmail || "";
        sesConfig.value.fromName = response.data.fromName || "";
      } else if (selectedProvider.value === "smtp") {
        smtpConfig.value.smtpHost = response.data.smtpHost || "";
        smtpConfig.value.smtpPort = response.data.smtpPort || 587;
        smtpConfig.value.smtpSecure = response.data.smtpSecure ?? false;
        smtpConfig.value.fromEmail = response.data.fromEmail || "";
        smtpConfig.value.fromName = response.data.fromName || "";
      }
    }
  } catch (_error) {
    // Provider not configured yet, that's ok
    // Provider not configured yet
  }
};

const saveProvider = async () => {
  saving.value = true;
  try {
    let config: any = {
      provider: selectedProvider.value,
      isActive: true // Activate the provider when saving
    };

    if (selectedProvider.value === "resend") {
      config = { ...config, ...resendConfig.value };
    } else if (selectedProvider.value === "mailgun") {
      config = { ...config, ...mailgunConfig.value };
    } else if (selectedProvider.value === "ses") {
      config = { ...config, ...sesConfig.value };
    } else if (selectedProvider.value === "smtp") {
      config = { ...config, ...smtpConfig.value };
    }

    await api.post("/settings/email-providers", config);
    activeProvider.value = selectedProvider.value;
    showSnackbar("Email provider saved and activated successfully");
  } catch (error: any) {
    showSnackbar(
      error.response?.data?.error || "Failed to save email provider",
      "error"
    );
  } finally {
    saving.value = false;
  }
};

const testProvider = async () => {
  testing.value = true;
  try {
    let config: any = { provider: selectedProvider.value };

    if (selectedProvider.value === "resend") {
      config = { ...config, ...resendConfig.value };
    } else if (selectedProvider.value === "mailgun") {
      config = { ...config, ...mailgunConfig.value };
    } else if (selectedProvider.value === "ses") {
      config = { ...config, ...sesConfig.value };
    } else if (selectedProvider.value === "smtp") {
      config = { ...config, ...smtpConfig.value };
    }

    const response = await api.post("/settings/email-providers/test", config);

    if (response.data.success) {
      showSnackbar("Test email sent successfully! Check your inbox.");
    } else {
      showSnackbar(`Test failed: ${response.data.error}`, "error");
    }
  } catch (error: any) {
    showSnackbar(
      error.response?.data?.error || "Failed to send test email",
      "error"
    );
  } finally {
    testing.value = false;
  }
};

const loadActiveProvider = async () => {
  try {
    const response = await api.get("/settings/email-providers");
    const active = response.data.find((p: any) => p.isActive);
    if (active) {
      activeProvider.value = active.provider;
      selectedProvider.value = active.provider;
      await loadProviderSettings();
    } else {
      // No active provider, set defaults
      activeProvider.value = "";
      selectedProvider.value = "resend"; // Default to resend
    }
  } catch (error: any) {
    console.error("Failed to load active provider:", error);
    console.error("Error details:", error.response?.data);
    // Set defaults on error
    activeProvider.value = "";
    selectedProvider.value = "resend";
  }
};

onMounted(() => {
  loadActiveProvider();
});
</script>
