<template>
  <div class="json-properties">
    <v-textarea
      v-model="localConfig.data"
      label="JSON Data"
      hint="Enter any valid JSON (object, array, string, number, boolean, or null)"
      persistent-hint
      variant="outlined"
      density="compact"
      rows="12"
      class="mb-4"
      :error-messages="validationError"
      @update:model-value="validateAndEmit"
    />

    <v-alert
      v-if="validationSuccess"
      type="success"
      variant="tonal"
      density="compact"
      class="mb-4"
    >
      <div class="text-body-2">
        Valid JSON: {{ dataType }}
      </div>
      <div
        v-if="dataPreview"
        class="text-caption mt-2"
      >
        {{ dataPreview }}
      </div>
    </v-alert>

    <v-alert
      type="info"
      variant="tonal"
      density="compact"
      class="mt-4"
    >
      <div class="text-body-2">
        <strong>JSON Data Node:</strong> Provides static JSON data to the workflow.
      </div>
      <div class="text-caption mt-2">
        This node outputs JSON data that can be used by downstream nodes. Useful for constants, configuration, or test data.
      </div>
      <div class="text-caption mt-2">
        <strong>Examples:</strong>
      </div>
      <pre class="text-caption mt-1">{"key": "value", "number": 123}</pre>
      <pre class="text-caption mt-1">["item1", "item2", "item3"]</pre>
      <pre class="text-caption mt-1">"Just a string"</pre>
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
  data: props.modelValue?.data || "{}",
});

const validationError = ref<string>("");
const validationSuccess = ref(false);
const dataType = ref<string>("");
const dataPreview = ref<string>("");

watch(
  () => props.modelValue,
  (newValue) => {
    if (newValue) {
      localConfig.value = {
        data: newValue.data || "{}",
      };
      validateJSON();
    }
  },
  { deep: true }
);

const validateJSON = () => {
  validationError.value = "";
  validationSuccess.value = false;
  dataType.value = "";
  dataPreview.value = "";

  const jsonStr = localConfig.value.data?.trim();

  if (!jsonStr) {
    validationError.value = "JSON data is required";
    return false;
  }

  try {
    const parsed = JSON.parse(jsonStr);

    if (Array.isArray(parsed)) {
      dataType.value = `Array with ${parsed.length} item(s)`;
      dataPreview.value = `First item: ${JSON.stringify(parsed[0])}`;
    } else if (typeof parsed === "object" && parsed !== null) {
      const keys = Object.keys(parsed);
      dataType.value = `Object with ${keys.length} propert${keys.length === 1 ? "y" : "ies"}`;
      if (keys.length > 0) {
        dataPreview.value = `Keys: ${keys.slice(0, 5).join(", ")}${keys.length > 5 ? "..." : ""}`;
      }
    } else {
      dataType.value = `${typeof parsed}`;
      dataPreview.value = `Value: ${String(parsed)}`;
    }

    validationSuccess.value = true;
    return true;
  } catch (error) {
    validationError.value = `Invalid JSON: ${(error as Error).message}`;
    return false;
  }
};

const validateAndEmit = () => {
  if (validateJSON()) {
    emitUpdate();
  }
};

const emitUpdate = () => {
  emit("update:modelValue", localConfig.value);
  emit("update");
};

// Initial validation
validateJSON();
</script>

<style scoped>
.json-properties {
  padding: 0;
}

pre {
  background: rgba(0, 0, 0, 0.05);
  padding: 4px 8px;
  border-radius: 4px;
  overflow-x: auto;
  margin: 2px 0;
}
</style>
