<template>
  <div class="json-to-csv-properties">
    <v-text-field
      v-model="localConfig.arrayPath"
      label="Array Path (Optional)"
      hint="JSONPath to array (e.g., data.users). Leave empty if input is already an array"
      persistent-hint
      variant="outlined"
      density="compact"
      class="mb-4"
      @update:model-value="emitUpdate"
    />

    <v-text-field
      v-model="localConfig.delimiter"
      label="Delimiter"
      hint="CSV delimiter character"
      persistent-hint
      variant="outlined"
      density="compact"
      class="mb-4"
      placeholder=","
      @update:model-value="emitUpdate"
    />

    <v-switch
      v-model="localConfig.includeHeaders"
      label="Include Header Row"
      color="primary"
      density="compact"
      class="mb-4"
      hide-details
      @update:model-value="emitUpdate"
    />

    <v-select
      v-model="localConfig.outputFormat"
      label="Output Format"
      :items="outputFormats"
      variant="outlined"
      density="compact"
      class="mb-4"
      @update:model-value="emitUpdate"
    />

    <v-divider class="my-4" />

    <div class="mb-4">
      <div class="d-flex justify-space-between align-center mb-2">
        <span class="text-subtitle-2">Column Mapping (Optional)</span>
        <v-btn
          size="x-small"
          icon="mdi-plus"
          variant="text"
          @click="addColumn"
        />
      </div>
      <div class="text-caption text-grey mb-2">
        Leave empty to include all fields from JSON objects
      </div>

      <div
        v-for="(column, index) in localConfig.columns"
        :key="index"
        class="mb-3 pa-2 column-card"
      >
        <div class="d-flex justify-space-between mb-2">
          <span class="text-caption">Column {{ index + 1 }}</span>
          <v-btn
            size="x-small"
            icon="mdi-delete"
            variant="text"
            @click="removeColumn(index)"
          />
        </div>

        <v-text-field
          v-model="column.key"
          label="JSON Field"
          placeholder="e.g., user.name"
          variant="outlined"
          density="compact"
          class="mb-2"
          @update:model-value="emitUpdate"
        />

        <v-text-field
          v-model="column.header"
          label="CSV Header"
          placeholder="e.g., User Name"
          variant="outlined"
          density="compact"
          @update:model-value="emitUpdate"
        />
      </div>
    </div>

    <v-alert
      type="info"
      variant="tonal"
      density="compact"
      class="mt-4"
    >
      <div class="text-body-2">
        <strong>JSON to CSV:</strong> Convert an array of JSON objects to CSV format.
      </div>
      <div class="text-caption mt-2">
        Use <code>{{ '{input.field}' }}</code> syntax to reference previous node data.
      </div>
      <div class="text-caption mt-1">
        Output includes <code>csv</code> (string or array) and <code>rowCount</code> fields.
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

const outputFormats = [
  { title: "CSV String", value: "string" },
  { title: "Array of Arrays", value: "array" },
];

const localConfig = ref({
  arrayPath: props.modelValue?.arrayPath || "",
  delimiter: props.modelValue?.delimiter || ",",
  includeHeaders: props.modelValue?.includeHeaders !== false,
  outputFormat: props.modelValue?.outputFormat || "string",
  columns: props.modelValue?.columns || [],
});

watch(
  () => props.modelValue,
  (newValue) => {
    if (newValue) {
      localConfig.value = {
        arrayPath: newValue.arrayPath || "",
        delimiter: newValue.delimiter || ",",
        includeHeaders: newValue.includeHeaders !== false,
        outputFormat: newValue.outputFormat || "string",
        columns: newValue.columns || [],
      };
    }
  },
  { deep: true }
);

const addColumn = () => {
  localConfig.value.columns.push({
    key: "",
    header: "",
  });
  emitUpdate();
};

const removeColumn = (index: number) => {
  localConfig.value.columns.splice(index, 1);
  emitUpdate();
};

const emitUpdate = () => {
  emit("update:modelValue", localConfig.value);
  emit("update");
};
</script>

<style scoped>
.json-to-csv-properties {
  padding: 0;
}

.column-card {
  border: 1px solid rgb(var(--v-theme-surface-variant));
  border-radius: 4px;
  background: rgb(var(--v-theme-surface));
}
</style>

