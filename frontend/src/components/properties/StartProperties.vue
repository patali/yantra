<template>
  <div class="start-properties">
    <!-- Active Configuration Summary -->
    <v-card
      variant="outlined"
      class="mb-4"
      color="primary"
    >
      <v-card-title class="text-subtitle-1 d-flex align-center">
        <v-icon
          size="small"
          class="mr-2"
        >
          mdi-play-circle
        </v-icon>
        Active Trigger Configuration
      </v-card-title>
      <v-card-text>
        <!-- Webhook Active Status -->
        <div
          v-if="hasActiveWebhook"
          class="mb-3 pa-3 rounded"
          style="background-color: rgba(var(--v-theme-success), 0.1)"
        >
          <div class="d-flex align-center mb-2">
            <v-icon
              color="success"
              size="small"
              class="mr-2"
            >
              mdi-webhook
            </v-icon>
            <span class="text-subtitle-2 font-weight-bold">Webhook Active</span>
            <v-spacer />
            <v-btn
              size="x-small"
              variant="text"
              color="error"
              @click="clearWebhook"
            >
              <v-icon size="small">
                mdi-close
              </v-icon>
              Remove
            </v-btn>
          </div>
          <div class="ml-7">
            <div class="text-caption mb-1">
              <strong>URL:</strong>
            </div>
            <div class="font-monospace text-caption mb-2 d-flex align-center">
              <span class="text-truncate">{{ webhookUrl }}</span>
              <v-btn
                size="x-small"
                variant="text"
                icon="mdi-content-copy"
                @click="copyWebhookUrl"
              />
            </div>
            <div
              v-if="localConfig.webhookPath"
              class="text-caption mb-1"
            >
              <strong>Custom Path:</strong> {{ localConfig.webhookPath }}
            </div>
            <div class="text-caption">
              <v-icon
                :color="hasSecretConfigured ? 'success' : 'warning'"
                size="x-small"
                class="mr-1"
              >
                {{ hasSecretConfigured ? 'mdi-check-circle' : 'mdi-alert-circle' }}
              </v-icon>
              <strong>Secret:</strong>
              <span v-if="hasSecretConfigured">Configured</span>
              <span
                v-else
                class="text-warning"
              >Not configured - Generate below</span>
            </div>
          </div>
        </div>

        <!-- Cron Active Status -->
        <div
          v-if="hasActiveCron"
          class="mb-3 pa-3 rounded"
          style="background-color: rgba(var(--v-theme-success), 0.1)"
        >
          <div class="d-flex align-center mb-2">
            <v-icon
              color="success"
              size="small"
              class="mr-2"
            >
              mdi-calendar-clock
            </v-icon>
            <span class="text-subtitle-2 font-weight-bold">Cron Schedule Active</span>
            <v-spacer />
            <v-btn
              size="x-small"
              variant="text"
              color="error"
              @click="clearCron"
            >
              <v-icon size="small">
                mdi-close
              </v-icon>
              Remove
            </v-btn>
          </div>
          <div class="ml-7">
            <div class="text-caption mb-1">
              <strong>Schedule:</strong> <code>{{ localConfig.cronSchedule }}</code>
            </div>
            <div class="text-caption mb-1">
              <strong>Timezone:</strong> {{ localConfig.timezone }}
            </div>
            <div class="text-caption text-medium-emphasis">
              {{ cronDescription }}
            </div>
          </div>
        </div>

        <!-- No Active Triggers -->
        <v-alert
          v-if="!hasActiveWebhook && !hasActiveCron"
          type="info"
          variant="tonal"
          density="compact"
        >
          <div class="text-caption">
            <strong>Manual trigger only:</strong> This workflow can only be started manually using the Run button.
          </div>
        </v-alert>
      </v-card-text>
    </v-card>

    <!-- Trigger Type Selection -->
    <v-card
      variant="outlined"
      class="mb-4"
    >
      <v-card-title class="text-subtitle-1">
        Configure Triggers
      </v-card-title>
      <v-card-text>
        <v-radio-group
          v-model="triggerType"
          @update:model-value="updateTriggerType"
        >
          <v-radio
            label="Manual (Run button only)"
            value="manual"
          />
          <v-radio
            label="Webhook (External HTTP requests)"
            value="webhook"
          />
          <v-radio
            label="Cron Schedule (Automatic execution)"
            value="cron"
          />
          <v-radio
            label="Webhook + Cron (Both enabled)"
            value="both"
          />
        </v-radio-group>
      </v-card-text>
    </v-card>

    <!-- Webhook Configuration -->
    <v-card
      v-if="triggerType === 'webhook' || triggerType === 'both'"
      variant="outlined"
      class="mb-4"
    >
      <v-card-title class="text-subtitle-1">
        <v-icon
          size="small"
          class="mr-2"
        >
          mdi-webhook
        </v-icon>
        Webhook Configuration
      </v-card-title>
      <v-card-text>
        <v-alert
          v-if="webhookUrl"
          type="success"
          variant="tonal"
          density="compact"
          class="mb-3"
        >
          <div class="text-caption">
            <strong>Webhook URL:</strong>
          </div>
          <div class="text-body-2 font-monospace mt-1">
            {{ webhookUrl }}
          </div>
          <v-btn
            size="small"
            variant="text"
            class="mt-2"
            @click="copyWebhookUrl"
          >
            <v-icon size="small">
              mdi-content-copy
            </v-icon>
            Copy URL
          </v-btn>
        </v-alert>

        <v-text-field
          v-model="localConfig.webhookPath"
          label="Webhook Path (optional)"
          hint="Custom path segment (e.g., 'my-webhook' creates /api/webhooks/{workflowId}/my-webhook). Leave empty for default."
          persistent-hint
          variant="outlined"
          density="compact"
          placeholder="Leave empty for default"
          class="mb-3"
          @update:model-value="emitUpdate"
        />

        <v-alert
          type="info"
          variant="tonal"
          density="compact"
          class="mb-3"
        >
          <div class="text-caption">
            <strong>Security:</strong> All webhooks require authentication. Generate a secret below to secure your webhook endpoint.
          </div>
        </v-alert>

        <!-- Webhook Secret Section -->
        <div class="mb-3">
          <div class="d-flex align-center mb-2">
            <v-label class="text-caption font-weight-medium">
              Webhook Secret
            </v-label>
            <v-spacer />
            <v-btn
              size="small"
              variant="outlined"
              :color="hasSecretConfigured ? 'warning' : 'primary'"
              :loading="generatingSecret"
              @click="generateSecret"
            >
              <v-icon
                size="small"
                class="mr-1"
              >
                {{ hasSecretConfigured ? 'mdi-refresh' : 'mdi-key-plus' }}
              </v-icon>
              {{ hasSecretConfigured ? 'Regenerate Secret' : 'Generate Secret' }}
            </v-btn>
          </div>

          <v-alert
            v-if="webhookSecret"
            type="warning"
            variant="tonal"
            density="compact"
            class="mb-2"
          >
            <div class="text-caption font-weight-bold mb-1">
              ⚠️ Save this secret now! It will not be shown again.
            </div>
            <div class="d-flex align-center flex-wrap">
              <code class="font-monospace text-body-2 mr-2 flex-grow-1">{{ webhookSecret }}</code>
              <div>
                <v-btn
                  size="x-small"
                  variant="text"
                  @click="copySecret"
                >
                  <v-icon size="small">
                    mdi-content-copy
                  </v-icon>
                  Copy
                </v-btn>
                <v-btn
                  size="x-small"
                  variant="text"
                  @click="hideSecret"
                >
                  <v-icon size="small">
                    mdi-eye-off
                  </v-icon>
                  Hide
                </v-btn>
              </div>
            </div>
          </v-alert>

          <v-alert
            v-else-if="hasSecretConfigured"
            type="success"
            variant="tonal"
            density="compact"
            class="mb-2"
          >
            <div class="d-flex align-center">
              <v-icon
                size="small"
                class="mr-2"
              >
                mdi-check-circle
              </v-icon>
              <div class="text-caption">
                <strong>Webhook secret is configured and active.</strong>
              </div>
            </div>
            <div class="text-caption mt-1">
              The secret cannot be retrieved. Click "Regenerate Secret" to create a new one (this will invalidate the old secret).
            </div>
          </v-alert>

          <v-alert
            v-else
            type="warning"
            variant="tonal"
            density="compact"
            class="mb-2"
          >
            <div class="d-flex align-center">
              <v-icon
                size="small"
                class="mr-2"
              >
                mdi-alert
              </v-icon>
              <div class="text-caption">
                <strong>No webhook secret configured - webhook will not work!</strong>
              </div>
            </div>
            <div class="text-caption mt-1">
              Generate a secret to secure your webhook. Include it in the Authorization header as a Bearer token when calling the webhook.
            </div>
          </v-alert>
        </div>

        <v-alert
          type="info"
          variant="tonal"
          density="compact"
        >
          <div class="text-caption">
            <strong>Usage:</strong> Send HTTP POST requests to the webhook URL with JSON body.
          </div>
          <div
            v-if="webhookSecret"
            class="text-caption mt-1"
          >
            <strong>Usage with Secret:</strong>
            <code class="d-block mt-1">
              curl -X POST {{ webhookUrl }} \<br>
              &nbsp;&nbsp;-H "Content-Type: application/json" \<br>
              &nbsp;&nbsp;-H "Authorization: Bearer {{ webhookSecret }}" \<br>
              &nbsp;&nbsp;-d '{"key":"value"}'
            </code>
          </div>
          <div
            v-else
            class="text-caption mt-1"
          >
            <strong>Usage:</strong> Generate a secret below, then include it in the Authorization header:
            <code class="d-block mt-1">
              curl -X POST {{ webhookUrl || 'https://api.example.com/api/webhooks/{id}' }} \<br>
              &nbsp;&nbsp;-H "Content-Type: application/json" \<br>
              &nbsp;&nbsp;-H "Authorization: Bearer YOUR_SECRET" \<br>
              &nbsp;&nbsp;-d '{"key":"value"}'
            </code>
          </div>
        </v-alert>
      </v-card-text>
    </v-card>

    <!-- Cron Schedule Configuration -->
    <v-card
      v-if="triggerType === 'cron' || triggerType === 'both'"
      variant="outlined"
      class="mb-4"
    >
      <v-card-title class="text-subtitle-1">
        <v-icon
          size="small"
          class="mr-2"
        >
          mdi-calendar-clock
        </v-icon>
        Cron Schedule
      </v-card-title>
      <v-card-text>
        <v-text-field
          v-model="localConfig.cronSchedule"
          label="Cron Expression"
          hint="Standard cron format (5 or 6 fields). Example: '0 */5 * * * *' (every 5 minutes)"
          persistent-hint
          variant="outlined"
          density="compact"
          placeholder="0 */5 * * * *"
          class="mb-3"
          @update:model-value="emitUpdate"
        />

        <v-select
          v-model="localConfig.timezone"
          label="Timezone"
          :items="timezones"
          variant="outlined"
          density="compact"
          class="mb-3"
          @update:model-value="emitUpdate"
        />

        <v-row class="mb-2">
          <v-col cols="12">
            <div class="text-caption mb-2">
              Quick Schedule Templates:
            </div>
            <v-chip-group>
              <v-chip
                variant="outlined"
                size="small"
                @click="setCron('0 */5 * * * *')"
              >
                Every 5 min
              </v-chip>
              <v-chip
                variant="outlined"
                size="small"
                @click="setCron('0 */15 * * * *')"
              >
                Every 15 min
              </v-chip>
              <v-chip
                variant="outlined"
                size="small"
                @click="setCron('0 */30 * * * *')"
              >
                Every 30 min
              </v-chip>
              <v-chip
                variant="outlined"
                size="small"
                @click="setCron('0 0 * * * *')"
              >
                Every hour
              </v-chip>
              <v-chip
                variant="outlined"
                size="small"
                @click="setCron('0 0 */6 * * *')"
              >
                Every 6 hours
              </v-chip>
              <v-chip
                variant="outlined"
                size="small"
                @click="setCron('0 0 0 * * *')"
              >
                Daily
              </v-chip>
            </v-chip-group>
          </v-col>
        </v-row>

        <v-alert
          type="info"
          variant="tonal"
          density="compact"
        >
          <div class="text-caption">
            <strong>Cron Format:</strong> 6 fields (seconds minutes hours day month weekday)
          </div>
          <div class="text-caption mt-1">
            <strong>Example:</strong> <code>0 0 9 * * 1-5</code> = Weekdays at 9:00 AM
          </div>
        </v-alert>
      </v-card-text>
    </v-card>

    <v-alert
      type="warning"
      variant="tonal"
      density="compact"
      class="mt-4"
    >
      <div class="text-body-2">
        <strong>Note:</strong> Changes to trigger configuration require saving the workflow to take effect.
      </div>
    </v-alert>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, computed } from "vue";
import api from "@/services/api";

interface Props {
  modelValue: Record<string, any>;
  workflowId?: string;
}

const props = defineProps<Props>();
const emit = defineEmits<{
  (e: "update:modelValue", value: Record<string, any>): void;
  (e: "update"): void;
}>();

const timezones = [
  // UTC/GMT
  "UTC",
  "GMT",

  // Americas - North America
  "America/New_York",        // Eastern Time (US & Canada)
  "America/Chicago",         // Central Time (US & Canada)
  "America/Denver",          // Mountain Time (US & Canada)
  "America/Phoenix",         // Arizona (no DST)
  "America/Los_Angeles",     // Pacific Time (US & Canada)
  "America/Anchorage",       // Alaska
  "Pacific/Honolulu",        // Hawaii
  "America/Toronto",         // Toronto
  "America/Vancouver",       // Vancouver
  "America/Halifax",         // Atlantic Time (Canada)
  "America/St_Johns",        // Newfoundland

  // Americas - Latin America
  "America/Mexico_City",     // Mexico City
  "America/Bogota",          // Bogota, Lima, Quito
  "America/Lima",
  "America/Santiago",        // Santiago
  "America/Caracas",         // Caracas
  "America/Sao_Paulo",       // Brasilia, São Paulo
  "America/Buenos_Aires",    // Buenos Aires
  "America/Montevideo",      // Montevideo

  // Europe - Western
  "Europe/London",           // London, Dublin, Lisbon
  "Europe/Dublin",
  "Europe/Lisbon",
  "Atlantic/Reykjavik",      // Reykjavik

  // Europe - Central
  "Europe/Paris",            // Paris, Brussels, Madrid
  "Europe/Brussels",
  "Europe/Madrid",
  "Europe/Berlin",           // Berlin, Rome, Vienna
  "Europe/Rome",
  "Europe/Vienna",
  "Europe/Amsterdam",        // Amsterdam
  "Europe/Stockholm",        // Stockholm
  "Europe/Oslo",             // Oslo
  "Europe/Copenhagen",       // Copenhagen
  "Europe/Zurich",           // Zurich
  "Europe/Prague",           // Prague
  "Europe/Warsaw",           // Warsaw

  // Europe - Eastern
  "Europe/Athens",           // Athens, Bucharest
  "Europe/Bucharest",
  "Europe/Helsinki",         // Helsinki, Kiev, Sofia
  "Europe/Kiev",
  "Europe/Sofia",
  "Europe/Istanbul",         // Istanbul
  "Europe/Moscow",           // Moscow, St. Petersburg

  // Middle East
  "Asia/Dubai",              // Dubai, Abu Dhabi
  "Asia/Jerusalem",          // Jerusalem
  "Asia/Riyadh",             // Riyadh, Kuwait
  "Asia/Tehran",             // Tehran
  "Asia/Baghdad",            // Baghdad

  // Africa
  "Africa/Cairo",            // Cairo
  "Africa/Johannesburg",     // Johannesburg, Pretoria
  "Africa/Nairobi",          // Nairobi
  "Africa/Lagos",            // Lagos
  "Africa/Casablanca",       // Casablanca

  // Asia - South
  "Asia/Kolkata",            // India Standard Time
  "Asia/Karachi",            // Karachi
  "Asia/Dhaka",              // Dhaka
  "Asia/Colombo",            // Colombo

  // Asia - Southeast
  "Asia/Bangkok",            // Bangkok, Hanoi, Jakarta
  "Asia/Jakarta",
  "Asia/Singapore",          // Singapore
  "Asia/Kuala_Lumpur",       // Kuala Lumpur
  "Asia/Manila",             // Manila
  "Asia/Ho_Chi_Minh",        // Ho Chi Minh

  // Asia - East
  "Asia/Hong_Kong",          // Hong Kong
  "Asia/Shanghai",           // Beijing, Shanghai
  "Asia/Taipei",             // Taipei
  "Asia/Tokyo",              // Tokyo, Osaka
  "Asia/Seoul",              // Seoul

  // Australia & Pacific
  "Australia/Perth",         // Perth
  "Australia/Adelaide",      // Adelaide
  "Australia/Darwin",        // Darwin
  "Australia/Brisbane",      // Brisbane
  "Australia/Sydney",        // Sydney, Melbourne
  "Australia/Melbourne",
  "Australia/Hobart",        // Hobart
  "Pacific/Auckland",        // Auckland, Wellington
  "Pacific/Fiji",            // Fiji
  "Pacific/Guam",            // Guam
];

const localConfig = ref({
  triggerType: props.modelValue?.triggerType || "manual",
  cronSchedule: props.modelValue?.cronSchedule || "",
  timezone: props.modelValue?.timezone || "UTC",
  webhookPath: props.modelValue?.webhookPath || "",
  webhookRequireAuth: props.modelValue?.webhookRequireAuth !== undefined
    ? props.modelValue.webhookRequireAuth
    : false,
  webhookSecretConfigured: props.modelValue?.webhookSecretConfigured || false,
});

const triggerType = ref(localConfig.value.triggerType);
const webhookSecret = ref<string | null>(null);
const hasSecretConfigured = ref(props.modelValue?.webhookSecretConfigured || false);
const generatingSecret = ref(false);

const webhookUrl = computed(() => {
  if (!props.workflowId) return "";

  const baseUrl = window.location.origin;
  const path = localConfig.value.webhookPath
    ? `/api/webhooks/${props.workflowId}/${localConfig.value.webhookPath}`
    : `/api/webhooks/${props.workflowId}`;

  return `${baseUrl}${path}`;
});

const hasActiveWebhook = computed(() => {
  return triggerType.value === "webhook" || triggerType.value === "both";
});

const hasActiveCron = computed(() => {
  return (triggerType.value === "cron" || triggerType.value === "both") &&
    localConfig.value.cronSchedule !== "";
});

const cronDescription = computed(() => {
  const schedule = localConfig.value.cronSchedule;
  if (!schedule) return "";

  // Simple descriptions for common patterns
  const descriptions: Record<string, string> = {
    "0 */5 * * * *": "Runs every 5 minutes",
    "0 */15 * * * *": "Runs every 15 minutes",
    "0 */30 * * * *": "Runs every 30 minutes",
    "0 0 * * * *": "Runs every hour",
    "0 0 */6 * * *": "Runs every 6 hours",
    "0 0 0 * * *": "Runs daily at midnight",
    "0 0 9 * * 1-5": "Runs weekdays at 9:00 AM",
  };

  return descriptions[schedule] || "Custom schedule";
});

watch(
  () => props.modelValue,
  (newValue) => {
    if (newValue) {
      localConfig.value = {
        triggerType: newValue.triggerType || "manual",
        cronSchedule: newValue.cronSchedule || "",
        timezone: newValue.timezone || "UTC",
        webhookPath: newValue.webhookPath || "",
        webhookRequireAuth: newValue.webhookRequireAuth !== undefined
          ? newValue.webhookRequireAuth
          : false,
        webhookSecretConfigured: newValue.webhookSecretConfigured || false,
      };
      triggerType.value = localConfig.value.triggerType;
      // Check if secret is configured (indicated by a flag in config or loaded from workflow)
      hasSecretConfigured.value = newValue.webhookSecretConfigured === true;
    }
  },
  { immediate: true, deep: true }
);

const updateTriggerType = () => {
  localConfig.value.triggerType = triggerType.value;

  // Always require auth for webhooks
  if (triggerType.value === "webhook" || triggerType.value === "both") {
    localConfig.value.webhookRequireAuth = true;
  }

  // Clear unrelated fields when switching trigger types
  if (triggerType.value === "manual") {
    localConfig.value.cronSchedule = "";
    localConfig.value.webhookPath = "";
    localConfig.value.webhookRequireAuth = false;
    // Note: Don't clear webhookSecretConfigured - secret remains in backend
    // User needs to regenerate if they re-enable webhook
  } else if (triggerType.value === "webhook") {
    localConfig.value.cronSchedule = "";
    localConfig.value.webhookRequireAuth = true;
  } else if (triggerType.value === "cron") {
    localConfig.value.webhookPath = "";
    localConfig.value.webhookRequireAuth = false;
    // Note: Don't clear webhookSecretConfigured - secret remains in backend
  } else if (triggerType.value === "both") {
    localConfig.value.webhookRequireAuth = true;
  }

  emitUpdate();
};

const emitUpdate = () => {
  emit("update:modelValue", localConfig.value);
  emit("update");
};

const setCron = (cron: string) => {
  localConfig.value.cronSchedule = cron;
  emitUpdate();
};

const copyWebhookUrl = async () => {
  try {
    await navigator.clipboard.writeText(webhookUrl.value);
    // You might want to show a toast notification here
  } catch (err) {
    console.error("Failed to copy webhook URL:", err);
  }
};

const generateSecret = async () => {
  if (!props.workflowId) {
    alert("Please save the workflow first before generating a secret");
    return;
  }

  try {
    generatingSecret.value = true;
    const response = await api.post(`/workflows/${props.workflowId}/webhook-secret/regenerate`);
    webhookSecret.value = response.data.secret;
    hasSecretConfigured.value = true;
    // Update config to indicate secret is configured
    localConfig.value.webhookSecretConfigured = true;
    emitUpdate();
  } catch (error: any) {
    console.error("Failed to generate secret:", error);
    alert(error.response?.data?.error || "Failed to generate webhook secret");
  } finally {
    generatingSecret.value = false;
  }
};

const copySecret = async () => {
  if (!webhookSecret.value) return;

  try {
    await navigator.clipboard.writeText(webhookSecret.value);
    // Could show a toast here
  } catch (err) {
    console.error("Failed to copy secret:", err);
  }
};

const hideSecret = () => {
  webhookSecret.value = null;
  // Secret is still configured, just hidden
};

const clearWebhook = () => {
  if (triggerType.value === "both") {
    // If both are enabled, switch to cron only
    triggerType.value = "cron";
  } else {
    // If only webhook is enabled, switch to manual
    triggerType.value = "manual";
  }
  updateTriggerType();
};

const clearCron = () => {
  if (triggerType.value === "both") {
    // If both are enabled, switch to webhook only
    triggerType.value = "webhook";
  } else {
    // If only cron is enabled, switch to manual
    triggerType.value = "manual";
  }
  updateTriggerType();
};

</script>

<style scoped>
.start-properties {
  padding: 0;
}

.font-monospace {
  font-family: monospace;
  word-break: break-all;
}
</style>
