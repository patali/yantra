<template>
  <div class="loop-properties">
    <v-text-field
      v-model="localConfig.arrayPath"
      label="Array Path"
      hint="Path to array in input (e.g., data.users). Leave empty if input is array."
      persistent-hint
      variant="outlined"
      density="compact"
      class="mb-4"
      @update:model-value="emitUpdate"
    />

    <v-text-field
      v-model="localConfig.itemVariable"
      label="Item Variable Name"
      hint="Variable name for current item (default: 'item')"
      persistent-hint
      variant="outlined"
      density="compact"
      class="mb-4"
      @update:model-value="emitUpdate"
    />

    <v-text-field
      v-model="localConfig.indexVariable"
      label="Index Variable Name"
      hint="Variable name for current index (default: 'index')"
      persistent-hint
      variant="outlined"
      density="compact"
      class="mb-4"
      @update:model-value="emitUpdate"
    />

    <v-text-field
      v-model.number="localConfig.max_iterations"
      label="Maximum Iterations"
      :hint="maxIterationsHint"
      persistent-hint
      variant="outlined"
      density="compact"
      type="number"
      :min="WORKFLOW_LIMITS.MIN_ITERATIONS"
      :max="WORKFLOW_LIMITS.GLOBAL_MAX_ITERATIONS"
      class="mb-4"
      :error="!!iterationValidation.error"
      :error-messages="iterationValidation.error"
      @update:model-value="emitUpdate"
    >
      <template #append-inner>
        <v-tooltip location="top">
          <template #activator="{ props: tooltipProps }">
            <v-icon
              v-bind="tooltipProps"
              size="small"
              color="info"
            >
              mdi-information-outline
            </v-icon>
          </template>
          <div class="text-caption">
            <div><strong>Default:</strong> {{ WORKFLOW_LIMITS.DEFAULT_MAX_ITERATIONS }}</div>
            <div><strong>Maximum:</strong> {{ WORKFLOW_LIMITS.GLOBAL_MAX_ITERATIONS }}</div>
          </div>
        </v-tooltip>
      </template>
    </v-text-field>

    <v-alert
      v-if="iterationValidation.warning"
      type="warning"
      variant="tonal"
      density="compact"
      class="mb-4"
    >
      {{ iterationValidation.warning }}
    </v-alert>

    <v-text-field
      v-model.number="localConfig.iterationDelay"
      label="Iteration Delay (milliseconds)"
      hint="Delay between iterations (default: 0ms = no delay)"
      persistent-hint
      variant="outlined"
      density="compact"
      type="number"
      min="0"
      class="mb-4"
      @update:model-value="emitUpdate"
    />

    <v-alert
      type="info"
      variant="tonal"
      density="compact"
      class="mt-4"
    >
      <div class="text-body-2">
        <strong>Loop Node:</strong> Iterates over an array and executes connected nodes for each item.
      </div>
      <div class="text-caption mt-2">
        Each iteration passes the current item and index to the next node. Use iteration delay to space out executions.
      </div>
    </v-alert>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, computed } from "vue";
import { WORKFLOW_LIMITS, validateIterations } from "@/constants/workflowLimits";

interface Props {
  modelValue: Record<string, any>;
}

const props = defineProps<Props>();
const emit = defineEmits<{
  (e: "update:modelValue", value: Record<string, any>): void;
  (e: "update"): void;
}>();

const localConfig = ref({
  arrayPath: props.modelValue?.arrayPath || "",
  itemVariable: props.modelValue?.itemVariable || "item",
  indexVariable: props.modelValue?.indexVariable || "index",
  iterationDelay: props.modelValue?.iterationDelay || 0,
  max_iterations: props.modelValue?.max_iterations,
});

watch(
  () => props.modelValue,
  (newValue) => {
    if (newValue) {
      localConfig.value = {
        arrayPath: newValue.arrayPath || "",
        itemVariable: newValue.itemVariable || "item",
        indexVariable: newValue.indexVariable || "index",
        iterationDelay: newValue.iterationDelay || 0,
        max_iterations: newValue.max_iterations,
      };
    }
  },
  { deep: true }
);

const iterationValidation = computed(() => {
  return validateIterations(localConfig.value.max_iterations);
});

const maxIterationsHint = computed(() => {
  if (localConfig.value.max_iterations === undefined || localConfig.value.max_iterations === null) {
    return `Leave empty to use default (${WORKFLOW_LIMITS.DEFAULT_MAX_ITERATIONS})`;
  }
  return `Set maximum number of iterations (1 - ${WORKFLOW_LIMITS.GLOBAL_MAX_ITERATIONS})`;
});

const emitUpdate = () => {
  emit("update:modelValue", localConfig.value);
  emit("update");
};
</script>

<style scoped>
.loop-properties {
  padding: 0;
}
</style>

