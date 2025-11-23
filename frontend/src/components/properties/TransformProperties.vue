<template>
  <v-form>
    <div class="mb-4">
      <div class="d-flex justify-space-between align-center mb-2">
        <span class="text-subtitle-2">Operations</span>
        <v-btn
          size="x-small"
          icon="mdi-plus"
          variant="text"
          @click="addOperation"
        />
      </div>

      <div
        v-for="(operation, index) in config.operations"
        :key="index"
        class="mb-3 pa-2 operation-card"
      >
        <div class="d-flex justify-space-between mb-2">
          <span class="text-caption">Operation {{ index + 1 }}</span>
          <v-btn
            size="x-small"
            icon="mdi-delete"
            variant="text"
            @click="removeOperation(index)"
          />
        </div>

        <v-select
          v-model="operation.type"
          label="Type"
          :items="operationTypes"
          variant="outlined"
          density="compact"
          class="mb-2"
          @update:model-value="emitUpdate"
        />

        <div v-if="operation.type === 'map'">
          <v-textarea
            v-model="mappingsText[index]"
            label="Mappings (JSON)"
            placeholder="[{&quot;from&quot;: &quot;input.field&quot;, &quot;to&quot;: &quot;output&quot;}]"
            variant="outlined"
            density="compact"
            rows="3"
            @update:model-value="updateMappings(index)"
          />
        </div>

        <div v-else-if="operation.type === 'extract'">
          <v-text-field
            v-model="operation.config.jsonPath"
            label="JSONPath"
            placeholder="$.data.users[*].name"
            variant="outlined"
            density="compact"
            class="mb-2"
            @update:model-value="emitUpdate"
          />
          <v-text-field
            v-model="operation.config.outputKey"
            label="Output Key"
            variant="outlined"
            density="compact"
            @update:model-value="emitUpdate"
          />
        </div>

        <div v-else-if="operation.type === 'concat'">
          <v-text-field
            v-model="operation.config.inputs"
            label="Input Fields (comma-separated)"
            placeholder="firstName,lastName"
            variant="outlined"
            density="compact"
            class="mb-2"
            @update:model-value="emitUpdate"
          />
          <v-text-field
            v-model="operation.config.separator"
            label="Separator"
            variant="outlined"
            density="compact"
            class="mb-2"
            @update:model-value="emitUpdate"
          />
          <v-text-field
            v-model="operation.config.outputKey"
            label="Output Key"
            variant="outlined"
            density="compact"
            @update:model-value="emitUpdate"
          />
        </div>

        <div v-else>
          <v-text-field
            v-model="operation.config.inputKey"
            label="Input Key"
            variant="outlined"
            density="compact"
            class="mb-2"
            @update:model-value="emitUpdate"
          />
          <v-text-field
            v-model="operation.config.outputKey"
            label="Output Key"
            variant="outlined"
            density="compact"
            @update:model-value="emitUpdate"
          />
        </div>
      </div>
    </div>
  </v-form>
</template>

<script setup lang="ts">
import { ref, watch, onMounted } from "vue";

interface Props {
  modelValue: Record<string, any>;
}

const props = defineProps<Props>();
const emit = defineEmits(["update:modelValue"]);

const operationTypes = [
  { title: "Map Fields", value: "map" },
  { title: "Extract (JSONPath)", value: "extract" },
  { title: "Parse JSON", value: "parse" },
  { title: "Stringify", value: "stringify" },
  { title: "Concatenate", value: "concat" },
];

const config = ref(props.modelValue || {
  operations: [],
});

// Store mappings as text for each operation
const mappingsText = ref<Record<number, string>>({});

// Initialize mappingsText from config
onMounted(() => {
  config.value.operations.forEach((operation: any, index: number) => {
    if (operation.type === "map" && operation.config.mappings) {
      // If mappings is already an object/array, stringify it for display
      if (typeof operation.config.mappings === "object") {
        mappingsText.value[index] = JSON.stringify(operation.config.mappings, null, 2);
      } else if (typeof operation.config.mappings === "string") {
        mappingsText.value[index] = operation.config.mappings;
      }
    }
  });
});

const addOperation = () => {
  const newIndex = config.value.operations.length;
  config.value.operations.push({
    type: "map",
    config: {},
  });
  mappingsText.value[newIndex] = "";
  emitUpdate();
};

const removeOperation = (index: number) => {
  config.value.operations.splice(index, 1);
  delete mappingsText.value[index];
  emitUpdate();
};

const updateMappings = (index: number) => {
  const text = mappingsText.value[index];
  if (!text || text.trim() === "") {
    config.value.operations[index].config.mappings = null;
    emitUpdate();
    return;
  }

  try {
    // Parse the JSON string to an actual array/object
    const parsed = JSON.parse(text);
    config.value.operations[index].config.mappings = parsed;
    emitUpdate();
  } catch (e) {
    // Invalid JSON, don't update the config yet
    console.warn("Invalid JSON in mappings:", e);
  }
};

const emitUpdate = () => {
  emit("update:modelValue", config.value);
};

watch(() => props.modelValue, (newVal) => {
  config.value = newVal;
  // Update mappingsText when modelValue changes
  config.value.operations.forEach((operation: any, index: number) => {
    if (operation.type === "map" && operation.config.mappings) {
      if (typeof operation.config.mappings === "object") {
        mappingsText.value[index] = JSON.stringify(operation.config.mappings, null, 2);
      } else if (typeof operation.config.mappings === "string") {
        mappingsText.value[index] = operation.config.mappings;
      }
    }
  });
}, { deep: true });
</script>

<style scoped>
.operation-card {
  border: 1px solid rgb(var(--v-theme-surface-variant));
  border-radius: 4px;
  background: rgb(var(--v-theme-surface));
}
</style>

