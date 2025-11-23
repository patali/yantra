<template>
  <v-form>
    <v-text-field
      v-model="config.url"
      label="URL"
      placeholder="https://api.example.com/{{input.id}}"
      hint="Use {{variable.path}} to insert data from previous nodes"
      persistent-hint
      variant="outlined"
      density="compact"
      class="mb-3"
      @update:model-value="emitUpdate"
    />

    <v-select
      v-model="config.method"
      label="Method"
      :items="['GET', 'POST', 'PUT', 'PATCH', 'DELETE']"
      variant="outlined"
      density="compact"
      class="mb-3"
      @update:model-value="emitUpdate"
    />

    <v-textarea
      v-model="headersText"
      label="Headers (JSON)"
      placeholder="{&quot;Authorization&quot;: &quot;Bearer {{input.token}}&quot;}"
      hint="Use {{variable.path}} in header values to insert data from previous nodes"
      persistent-hint
      variant="outlined"
      density="compact"
      rows="3"
      class="mb-3"
      @update:model-value="updateHeaders"
    />

    <v-textarea
      v-model="bodyText"
      label="Body (JSON or Text)"
      placeholder="{&quot;name&quot;: &quot;{{input.name}}&quot;, &quot;email&quot;: &quot;{{input.email}}&quot;}"
      hint="Use {{variable.path}} to insert data from previous nodes. Can be JSON or plain text."
      persistent-hint
      variant="outlined"
      density="compact"
      rows="4"
      class="mb-3"
      @update:model-value="updateBody"
    />

    <v-text-field
      v-model="config.timeout"
      label="Timeout (ms)"
      type="number"
      variant="outlined"
      density="compact"
      class="mb-3"
      @update:model-value="emitUpdate"
    />

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
      class="mt-2 mb-3"
    >
      <div class="text-caption">
        <strong>Template Variables:</strong> Use <code>{{ '{input.field}' }}</code> syntax in URL, headers, and body to insert dynamic data from previous nodes.
      </div>
      <div class="text-caption mt-1">
        Example: URL = <code>https://api.example.com/users/{{ '{input.userId}' }}</code>
      </div>
    </v-alert>

    <v-alert
      type="info"
      variant="tonal"
      density="compact"
      class="mt-2"
    >
      <div class="text-caption">
        <strong>Retries:</strong> Set to 0 to disable retries. This allows HTTP failures to be used in conditional logic instead of automatic retries.
      </div>
    </v-alert>
  </v-form>
</template>

<script setup lang="ts">
import { ref, watch, onMounted } from "vue";

interface Props {
  modelValue: Record<string, any>;
}

const props = defineProps<Props>();
const emit = defineEmits(["update:modelValue"]);

const config = ref(props.modelValue || {
  url: "",
  method: "GET",
  headers: {},
  body: null,
  timeout: 30000,
  maxRetries: 3,
});

const headersText = ref("");
const bodyText = ref("");

onMounted(() => {
  if (config.value.headers && typeof config.value.headers === "object") {
    headersText.value = JSON.stringify(config.value.headers, null, 2);
  }
  if (config.value.body) {
    bodyText.value = typeof config.value.body === "string"
      ? config.value.body
      : JSON.stringify(config.value.body, null, 2);
  }
});

const updateHeaders = () => {
  try {
    if (headersText.value.trim()) {
      config.value.headers = JSON.parse(headersText.value);
    } else {
      config.value.headers = {};
    }
    emitUpdate();
  } catch (_e) {
    // Invalid JSON, don't update
  }
};

const updateBody = () => {
  if (!bodyText.value.trim()) {
    config.value.body = null;
    emitUpdate();
    return;
  }

  try {
    // Try to parse as JSON
    config.value.body = JSON.parse(bodyText.value);
    emitUpdate();
  } catch (_e) {
    // Not valid JSON, treat as plain text string
    config.value.body = bodyText.value;
    emitUpdate();
  }
};

const emitUpdate = () => {
  emit("update:modelValue", config.value);
};

watch(() => props.modelValue, (newVal) => {
  config.value = newVal;
}, { deep: true });
</script>

