<template>
  <div class="loop-accumulator-properties">
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
      v-model="localConfig.accumulatorVariable"
      label="Accumulator Variable Name"
      hint="Variable name for accumulated results (default: 'accumulated')"
      persistent-hint
      variant="outlined"
      density="compact"
      class="mb-4"
      @update:model-value="emitUpdate"
    />

    <v-select
      v-model="localConfig.accumulationMode"
      label="Accumulation Mode"
      :items="accumulationModes"
      hint="How to accumulate results from each iteration"
      persistent-hint
      variant="outlined"
      density="compact"
      class="mb-4"
      @update:model-value="emitUpdate"
    />

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

    <v-select
      v-model="localConfig.errorHandling"
      label="Error Handling"
      :items="errorHandlingModes"
      hint="How to handle failed iterations or null/undefined results"
      persistent-hint
      variant="outlined"
      density="compact"
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
        <strong>Loop Accumulator Node:</strong> Iterates over an array, executes loop body nodes, and accumulates results.
      </div>
      <div class="text-caption mt-2">
        <strong>Connections:</strong>
      </div>
      <ul class="text-caption">
        <li><strong>Top (input):</strong> Receives array to iterate</li>
        <li><strong>Left Top (To Loop):</strong> Sends each item to loop body nodes</li>
        <li><strong>Left Bottom (From Loop):</strong> Receives result from loop body nodes</li>
        <li><strong>Right (Final Output):</strong> Outputs accumulated results after all iterations complete</li>
      </ul>
      <div class="text-caption mt-2">
        <strong>Accumulation Modes:</strong>
      </div>
      <ul class="text-caption">
        <li><strong>Array:</strong> Collect all results into an array</li>
        <li><strong>Last:</strong> Keep only the last result</li>
      </ul>
      <div class="text-caption mt-2">
        <strong>Error Handling:</strong>
      </div>
      <ul class="text-caption">
        <li><strong>Skip Failures:</strong> Skip iterations that fail or return null/undefined</li>
        <li><strong>Fail on Error:</strong> Stop and fail the entire loop if any iteration fails</li>
      </ul>
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

const accumulationModes = [
  { title: "Array - Collect all results", value: "array" },
  { title: "Last - Keep only last result", value: "last" },
];

const errorHandlingModes = [
  { title: "Skip Failures - Continue loop on errors", value: "skip" },
  { title: "Fail on Error - Stop loop on first error", value: "fail" },
];

const localConfig = ref({
  arrayPath: props.modelValue?.arrayPath || "",
  itemVariable: props.modelValue?.itemVariable || "item",
  indexVariable: props.modelValue?.indexVariable || "index",
  max_iterations: props.modelValue?.max_iterations,
  accumulatorVariable: props.modelValue?.accumulatorVariable || "accumulated",
  accumulationMode: props.modelValue?.accumulationMode || "array",
  iterationDelay: props.modelValue?.iterationDelay || 0,
  errorHandling: props.modelValue?.errorHandling || "skip",
});

watch(
  () => props.modelValue,
  (newValue) => {
    if (newValue) {
      localConfig.value = {
        arrayPath: newValue.arrayPath || "",
        itemVariable: newValue.itemVariable || "item",
        indexVariable: newValue.indexVariable || "index",
        max_iterations: newValue.max_iterations,
        accumulatorVariable: newValue.accumulatorVariable || "accumulated",
        accumulationMode: newValue.accumulationMode || "array",
        iterationDelay: newValue.iterationDelay || 0,
        errorHandling: newValue.errorHandling || "skip",
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
.loop-accumulator-properties {
  padding: 0;
}

ul {
  margin-left: 16px;
  margin-top: 4px;
}
</style>
