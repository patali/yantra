<template>
  <v-dialog
    v-model="dialog"
    max-width="600"
  >
    <template #activator="{ props: activatorProps }">
      <v-btn
        v-bind="activatorProps"
        variant="text"
        size="small"
        prepend-icon="mdi-information-outline"
        color="info"
      >
        Workflow Limits
      </v-btn>
    </template>

    <v-card>
      <v-card-title class="d-flex align-center">
        <v-icon
          class="mr-2"
          color="info"
        >
          mdi-shield-check
        </v-icon>
        Workflow Execution Limits
      </v-card-title>

      <v-card-text>
        <v-alert
          type="info"
          variant="tonal"
          density="compact"
          class="mb-4"
        >
          These limits prevent system abuse and ensure fair resource usage for all users.
        </v-alert>

        <v-list density="compact">
          <v-list-subheader>Execution Limits</v-list-subheader>

          <v-list-item>
            <template #prepend>
              <v-icon color="primary">
                mdi-clock-outline
              </v-icon>
            </template>
            <v-list-item-title>Maximum Execution Time</v-list-item-title>
            <v-list-item-subtitle>
              {{ WORKFLOW_LIMITS.MAX_EXECUTION_DURATION_MINUTES }} minutes per workflow
            </v-list-item-subtitle>
          </v-list-item>

          <v-list-item>
            <template #prepend>
              <v-icon color="primary">
                mdi-sitemap
              </v-icon>
            </template>
            <v-list-item-title>Maximum Total Nodes</v-list-item-title>
            <v-list-item-subtitle>
              {{ formattedNumber(WORKFLOW_LIMITS.MAX_TOTAL_NODES) }} nodes executed per workflow
            </v-list-item-subtitle>
          </v-list-item>

          <v-divider class="my-2" />

          <v-list-subheader>Loop Limits</v-list-subheader>

          <v-list-item>
            <template #prepend>
              <v-icon color="success">
                mdi-layers-triple
              </v-icon>
            </template>
            <v-list-item-title>Maximum Nesting Depth</v-list-item-title>
            <v-list-item-subtitle>
              {{ WORKFLOW_LIMITS.MAX_LOOP_DEPTH }} levels of nested loops
            </v-list-item-subtitle>
          </v-list-item>

          <v-list-item>
            <template #prepend>
              <v-icon color="success">
                mdi-repeat
              </v-icon>
            </template>
            <v-list-item-title>Maximum Iterations</v-list-item-title>
            <v-list-item-subtitle>
              Default: {{ formattedNumber(WORKFLOW_LIMITS.DEFAULT_MAX_ITERATIONS) }}
              | Global Max: {{ formattedNumber(WORKFLOW_LIMITS.GLOBAL_MAX_ITERATIONS) }} per loop
            </v-list-item-subtitle>
          </v-list-item>

          <v-list-item>
            <template #prepend>
              <v-icon color="success">
                mdi-timer-sand
              </v-icon>
            </template>
            <v-list-item-title>Maximum Iteration Delay</v-list-item-title>
            <v-list-item-subtitle>
              {{ formatDuration(WORKFLOW_LIMITS.MAX_ITERATION_DELAY) }} between iterations
            </v-list-item-subtitle>
          </v-list-item>

          <v-divider class="my-2" />

          <v-list-subheader>Data Limits</v-list-subheader>

          <v-list-item>
            <template #prepend>
              <v-icon color="warning">
                mdi-database
              </v-icon>
            </template>
            <v-list-item-title>Maximum Data Size</v-list-item-title>
            <v-list-item-subtitle>
              {{ formatBytes(WORKFLOW_LIMITS.MAX_DATA_SIZE) }} per input/output
            </v-list-item-subtitle>
          </v-list-item>

          <v-list-item>
            <template #prepend>
              <v-icon color="warning">
                mdi-database-plus
              </v-icon>
            </template>
            <v-list-item-title>Maximum Accumulator Size</v-list-item-title>
            <v-list-item-subtitle>
              {{ formatBytes(WORKFLOW_LIMITS.MAX_ACCUMULATOR_SIZE) }} accumulated data
            </v-list-item-subtitle>
          </v-list-item>
        </v-list>

        <v-alert
          type="warning"
          variant="tonal"
          density="compact"
          class="mt-4"
        >
          <div class="text-body-2">
            <strong>What happens when limits are exceeded?</strong>
          </div>
          <div class="text-caption mt-2">
            Workflow execution will stop immediately with a clear error message indicating which limit was exceeded.
            You can adjust loop iteration limits in node properties, but global limits cannot be changed.
          </div>
        </v-alert>
      </v-card-text>

      <v-card-actions>
        <v-spacer />
        <v-btn
          color="primary"
          variant="text"
          @click="dialog = false"
        >
          Close
        </v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { WORKFLOW_LIMITS, formatBytes, formatDuration } from "@/constants/workflowLimits";

const dialog = ref(false);

function formattedNumber(num: number): string {
  return num.toLocaleString();
}
</script>

<style scoped>
.v-list-item {
  min-height: 48px;
}

.v-list-subheader {
  font-weight: 600;
  color: rgba(0, 0, 0, 0.87);
}

.v-card-title {
  background-color: rgba(33, 150, 243, 0.05);
  border-bottom: 1px solid rgba(0, 0, 0, 0.12);
}
</style>
