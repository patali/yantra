<template>
  <v-form>
    <v-select
      v-model="config.provider"
      :items="providerItems"
      label="Mail Server Type"
      placeholder="Use active provider"
      variant="outlined"
      density="compact"
      class="mb-3"
      clearable
      @update:model-value="emitUpdate"
    />
    <v-text-field
      v-model="config.to"
      label="To"
      placeholder="user@example.com"
      variant="outlined"
      density="compact"
      class="mb-3"
      @update:model-value="emitUpdate"
    />

    <v-text-field
      v-model="config.subject"
      label="Subject"
      placeholder="Email subject"
      variant="outlined"
      density="compact"
      class="mb-3"
      @update:model-value="emitUpdate"
    />

    <v-textarea
      v-model="config.body"
      label="Body"
      placeholder="Email body (supports {{input.field}} variables)"
      variant="outlined"
      density="compact"
      rows="6"
      class="mb-3"
      @update:model-value="emitUpdate"
    />

    <v-switch
      v-model="config.isHtml"
      label="HTML Email"
      color="primary"
      density="compact"
      class="mb-3"
      @update:model-value="emitUpdate"
    />

    <v-text-field
      v-model="config.cc"
      label="CC (optional)"
      placeholder="cc@example.com"
      variant="outlined"
      density="compact"
      class="mb-3"
      @update:model-value="emitUpdate"
    />

    <v-text-field
      v-model="config.bcc"
      label="BCC (optional)"
      placeholder="bcc@example.com"
      variant="outlined"
      density="compact"
      class="mb-3"
      @update:model-value="emitUpdate"
    />

    <v-divider class="my-4" />

    <div class="mb-4">
      <div class="d-flex justify-space-between align-center mb-2">
        <span class="text-subtitle-2">Attachments (Optional)</span>
        <v-btn
          size="x-small"
          icon="mdi-plus"
          variant="text"
          @click="addAttachment"
        />
      </div>
      <div class="text-caption text-grey mb-2">
        Attach files from previous node outputs
      </div>

      <div
        v-for="(attachment, index) in config.attachments"
        :key="index"
        class="mb-3 pa-2 attachment-card"
      >
        <div class="d-flex justify-space-between mb-2">
          <span class="text-caption">Attachment {{ index + 1 }}</span>
          <v-btn
            size="x-small"
            icon="mdi-delete"
            variant="text"
            @click="removeAttachment(index)"
          />
        </div>

        <v-text-field
          v-model="attachment.filename"
          label="Filename"
          placeholder="report.csv"
          variant="outlined"
          density="compact"
          class="mb-2"
          @update:model-value="emitUpdate"
        />

        <v-text-field
          v-model="attachment.contentFromPreviousNode"
          label="Content Path"
          placeholder="e.g., csv or input.csv"
          hint="Path to content in previous node output"
          persistent-hint
          variant="outlined"
          density="compact"
          @update:model-value="emitUpdate"
        />
      </div>
    </div>

    <v-divider class="my-4" />

    <v-text-field
      v-model.number="config.maxRetries"
      label="Max Retries"
      type="number"
      min="0"
      max="10"
      hint="Number of retry attempts on failure (0 = no retries, default = 3)"
      persistent-hint
      variant="outlined"
      density="compact"
      class="mb-3"
      @update:model-value="emitUpdate"
    />

    <v-alert
      type="info"
      variant="tonal"
      density="compact"
      class="mt-2 mb-2"
    >
      <div class="text-caption">
        <strong>Tip:</strong> To attach CSV from JSON to CSV node, set Content Path to <code>csv</code>
      </div>
    </v-alert>

    <v-alert
      type="info"
      variant="tonal"
      density="compact"
      class="mt-2"
    >
      <div class="text-caption">
        <strong>Retries:</strong> Set to 0 to disable retries. This allows email failures to be used in conditional logic instead of automatic retries.
      </div>
    </v-alert>
  </v-form>
</template>

<script setup lang="ts">
import { ref, watch } from "vue";

interface Props {
  modelValue: Record<string, any>;
}

const props = defineProps<Props>();
const emit = defineEmits(["update:modelValue"]);

const config = ref(props.modelValue || {
  to: "",
  subject: "",
  body: "",
  isHtml: false,
  cc: "",
  bcc: "",
  attachments: [],
  provider: undefined,
  maxRetries: 3,
});

const providerItems = [
  { title: "Use active (default)", value: undefined },
  { title: "Resend", value: "resend" },
  { title: "Mailgun", value: "mailgun" },
  { title: "AWS SES", value: "ses" },
  { title: "SMTP", value: "smtp" },
];

const addAttachment = () => {
  if (!config.value.attachments) {
    config.value.attachments = [];
  }
  config.value.attachments.push({
    filename: "",
    contentFromPreviousNode: "",
  });
  emitUpdate();
};

const removeAttachment = (index: number) => {
  config.value.attachments.splice(index, 1);
  emitUpdate();
};

const emitUpdate = () => {
  emit("update:modelValue", config.value);
};

watch(() => props.modelValue, (newVal) => {
  config.value = newVal;
}, { deep: true });
</script>

<style scoped>
.attachment-card {
  border: 1px solid rgb(var(--v-theme-surface-variant));
  border-radius: 4px;
  background: rgb(var(--v-theme-surface));
}
</style>

