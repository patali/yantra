<template>
  <div class="json-array-trigger-properties">
    <v-textarea
      v-model="localConfig.jsonArray"
      label="JSON Array"
      hint="Enter a JSON array of objects. All objects must have the same structure."
      persistent-hint
      variant="outlined"
      density="compact"
      rows="10"
      class="mb-4"
      :error-messages="validationError"
      @update:model-value="validateAndEmit"
    />

    <v-checkbox
      v-model="localConfig.validateSchema"
      label="Validate uniform object types"
      hint="Ensure all objects in the array have the same properties"
      persistent-hint
      density="compact"
      class="mb-4"
      @update:model-value="emitUpdate"
    />

    <v-alert
      v-if="validationSuccess"
      type="success"
      variant="tonal"
      density="compact"
      class="mb-4"
    >
      <div class="text-body-2">
        Valid JSON array with {{ arrayLength }} object(s)
      </div>
      <div
        v-if="detectedSchema"
        class="text-caption mt-2"
      >
        <strong>Detected properties:</strong> {{ detectedSchema }}
      </div>
    </v-alert>

    <v-alert
      type="info"
      variant="tonal"
      density="compact"
      class="mt-4"
    >
      <div class="text-body-2">
        <strong>JSON Array Trigger:</strong> Starts a workflow with an array of uniform objects.
      </div>
      <div class="text-caption mt-2">
        This trigger validates that all objects in the array have the same structure before passing them to the workflow.
      </div>
      <div class="text-caption mt-2">
        <strong>Example:</strong>
      </div>
      <pre class="text-caption mt-1">[
  {"id": 1, "name": "Item 1"},
  {"id": 2, "name": "Item 2"}
]</pre>
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
  jsonArray: props.modelValue?.jsonArray || "[]",
  validateSchema: props.modelValue?.validateSchema !== false,
});

const validationError = ref<string>("");
const validationSuccess = ref(false);
const arrayLength = ref(0);
const detectedSchema = ref<string>("");

watch(
  () => props.modelValue,
  (newValue) => {
    if (newValue) {
      localConfig.value = {
        jsonArray: newValue.jsonArray || "[]",
        validateSchema: newValue.validateSchema !== false,
      };
      validateJSON();
    }
  },
  { deep: true }
);

const validateJSON = () => {
  validationError.value = "";
  validationSuccess.value = false;
  arrayLength.value = 0;
  detectedSchema.value = "";

  const jsonStr = localConfig.value.jsonArray.trim();

  if (!jsonStr) {
    validationError.value = "JSON array is required";
    return false;
  }

  try {
    const parsed = JSON.parse(jsonStr);

    if (!Array.isArray(parsed)) {
      validationError.value = "Input must be a JSON array";
      return false;
    }

    if (parsed.length === 0) {
      validationError.value = "Array cannot be empty";
      return false;
    }

    // Check if all elements are objects
    const allObjects = parsed.every(item => typeof item === "object" && item !== null && !Array.isArray(item));
    if (!allObjects) {
      validationError.value = "All array elements must be objects";
      return false;
    }

    // Validate uniform schema if enabled
    if (localConfig.value.validateSchema) {
      const firstKeys = Object.keys(parsed[0]).sort();
      const allSameSchema = parsed.every(item => {
        const keys = Object.keys(item).sort();
        return keys.length === firstKeys.length &&
               keys.every((key, idx) => key === firstKeys[idx]);
      });

      if (!allSameSchema) {
        validationError.value = "All objects must have the same properties";
        return false;
      }

      detectedSchema.value = firstKeys.join(", ");
    }

    arrayLength.value = parsed.length;
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
.json-array-trigger-properties {
  padding: 0;
}

pre {
  background: rgba(0, 0, 0, 0.05);
  padding: 8px;
  border-radius: 4px;
  overflow-x: auto;
}
</style>
