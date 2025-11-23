<template>
  <div class="slack-properties">
    <v-text-field
      v-model="localConfig.apiKey"
      label="Slack API Key"
      hint="Bot User OAuth Token (starts with xoxb-)"
      persistent-hint
      variant="outlined"
      density="compact"
      type="password"
      class="mb-4"
      @update:model-value="emitUpdate"
    />

    <v-text-field
      v-model="localConfig.channel"
      label="Channel"
      hint="Channel name (e.g., #general) or ID"
      persistent-hint
      variant="outlined"
      density="compact"
      class="mb-4"
      @update:model-value="emitUpdate"
    />

    <v-textarea
      v-model="localConfig.message"
      label="Message"
      hint="Use {{variable.path}} to insert data from previous nodes"
      persistent-hint
      variant="outlined"
      density="compact"
      rows="4"
      class="mb-4"
      @update:model-value="emitUpdate"
    />

    <v-expansion-panels
      variant="accordion"
      class="mb-4"
    >
      <v-expansion-panel>
        <v-expansion-panel-title>
          <v-icon
            start
            size="small"
          >
            mdi-cog
          </v-icon>
          Advanced Options
        </v-expansion-panel-title>
        <v-expansion-panel-text>
          <v-text-field
            v-model="localConfig.username"
            label="Bot Username (Optional)"
            variant="outlined"
            density="compact"
            class="mb-3"
            @update:model-value="emitUpdate"
          />

          <v-text-field
            v-model="localConfig.iconEmoji"
            label="Icon Emoji (Optional)"
            hint="e.g., :robot_face:"
            persistent-hint
            variant="outlined"
            density="compact"
            class="mb-3"
            @update:model-value="emitUpdate"
          />

          <v-text-field
            v-model="localConfig.iconUrl"
            label="Icon URL (Optional)"
            variant="outlined"
            density="compact"
            class="mb-3"
            @update:model-value="emitUpdate"
          />

          <v-text-field
            v-model="localConfig.threadTs"
            label="Thread Timestamp (Optional)"
            hint="Reply to a thread"
            persistent-hint
            variant="outlined"
            density="compact"
            @update:model-value="emitUpdate"
          />

          <v-text-field
            v-model.number="localConfig.maxRetries"
            label="Max Retries"
            type="number"
            min="0"
            max="10"
            hint="Number of retry attempts on failure (0 = no retries, default = 3)"
            persistent-hint
            variant="outlined"
            density="compact"
            class="mt-3"
            @update:model-value="emitUpdate"
          />
        </v-expansion-panel-text>
      </v-expansion-panel>
    </v-expansion-panels>

    <v-alert
      type="info"
      variant="tonal"
      density="compact"
      class="mt-4"
    >
      <div class="text-body-2">
        <strong>Slack Message:</strong> Send messages to Slack channels using a Bot Token.
      </div>
      <div class="text-caption mt-2">
        Use <code>{{ '{input.field}' }}</code> syntax to insert dynamic data.
      </div>
    </v-alert>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from "vue";

interface Props {
  modelValue: Record<string, any>;
}

const props = defineProps<Props>();
const emit = defineEmits<{
  (e: "update:modelValue", value: Record<string, any>): void;
  (e: "update"): void;
}>();

const localConfig = ref({
  apiKey: props.modelValue?.apiKey || "",
  channel: props.modelValue?.channel || "",
  message: props.modelValue?.message || "",
  username: props.modelValue?.username || "",
  iconEmoji: props.modelValue?.iconEmoji || "",
  iconUrl: props.modelValue?.iconUrl || "",
  threadTs: props.modelValue?.threadTs || "",
  maxRetries: props.modelValue?.maxRetries ?? 3,
});

watch(
  () => props.modelValue,
  (newValue) => {
    if (newValue) {
      localConfig.value = {
        apiKey: newValue.apiKey || "",
        channel: newValue.channel || "",
        message: newValue.message || "",
        username: newValue.username || "",
        iconEmoji: newValue.iconEmoji || "",
        iconUrl: newValue.iconUrl || "",
        threadTs: newValue.threadTs || "",
        maxRetries: newValue.maxRetries ?? 3,
      };
    }
  },
  { deep: true }
);

const emitUpdate = () => {
  emit("update:modelValue", localConfig.value);
  emit("update");
};
</script>

<style scoped>
.slack-properties {
  padding: 0;
}
</style>

