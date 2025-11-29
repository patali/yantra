<template>
  <v-container
    fluid
    class="pa-6 runs-container"
  >
    <v-row>
      <v-col cols="12">
        <div class="d-flex justify-space-between align-center mb-4">
          <h1 class="text-h4">
            Workflow Runs
          </h1>
          <v-btn
            color="primary"
            variant="outlined"
            prepend-icon="mdi-refresh"
            @click="fetchRuns"
          >
            Refresh
          </v-btn>
        </div>
      </v-col>
    </v-row>

    <v-row>
      <v-col cols="12">
        <v-card elevation="2">
          <v-card-title class="d-flex align-center">
            <v-icon class="mr-2">
              mdi-play-circle-outline
            </v-icon>
            All Executions
            <v-spacer />

            <!-- Status Filter -->
            <v-chip-group
              v-model="selectedStatus"
              selected-class="text-primary"
              mandatory
              @update:model-value="onStatusFilterChange"
            >
              <v-chip value="all">
                All
              </v-chip>
              <v-chip value="queued">
                Queued
              </v-chip>
              <v-chip value="running">
                Running
              </v-chip>
              <v-chip value="interrupted">
                Interrupted
              </v-chip>
              <v-chip value="success">
                Success
              </v-chip>
              <v-chip value="error">
                Failed
              </v-chip>
              <v-chip value="partially_failed">
                Partial
              </v-chip>
            </v-chip-group>
          </v-card-title>

          <v-data-table
            :headers="headers"
            :items="runs"
            :loading="loading"
            :items-per-page="25"
            class="elevation-0"
            hover
          >
            <!-- Workflow Name Column -->
            <template #[`item.workflow`]="{ item }">
              <div class="d-flex align-center">
                <v-icon
                  size="small"
                  class="mr-2"
                >
                  mdi-file-tree
                </v-icon>
                {{ item.workflow?.name || 'Unknown' }}
              </div>
            </template>

            <!-- Status Column -->
            <template #[`item.status`]="{ item }">
              <v-chip
                :color="getStatusColor(item.status)"
                size="small"
                class="text-uppercase"
              >
                <v-icon
                  v-if="item.status === 'running'"
                  start
                  size="small"
                  class="rotating"
                >
                  mdi-loading
                </v-icon>
                {{ getStatusLabel(item.status) }}
              </v-chip>
            </template>

            <!-- Trigger Type Column -->
            <template #[`item.triggerType`]="{ item }">
              <v-chip
                size="small"
                variant="outlined"
              >
                <v-icon
                  start
                  size="small"
                >
                  {{ getTriggerIcon(item.triggerType) }}
                </v-icon>
                {{ item.triggerType }}
              </v-chip>
            </template>

            <!-- Started At Column -->
            <template #[`item.startedAt`]="{ item }">
              <div class="text-caption">
                {{ formatDateTime(item.startedAt) }}
              </div>
            </template>

            <!-- Duration Column -->
            <template #[`item.duration`]="{ item }">
              <div class="text-caption">
                {{ getDuration(item) }}
              </div>
            </template>

            <!-- Node Stats Column -->
            <template #[`item.nodeStats`]="{ item }">
              <div class="text-caption">
                <v-icon
                  v-if="getSuccessNodeCount(item) > 0"
                  size="small"
                  color="success"
                >
                  mdi-check-circle
                </v-icon>
                {{ getSuccessNodeCount(item) }}
                <v-icon
                  v-if="getFailedNodeCount(item) > 0"
                  size="small"
                  color="error"
                  class="ml-2"
                >
                  mdi-alert-circle
                </v-icon>
                <span v-if="getFailedNodeCount(item) > 0">
                  {{ getFailedNodeCount(item) }}
                </span>
              </div>
            </template>

            <!-- Actions Column -->
            <template #[`item.actions`]="{ item }">
              <div class="d-flex gap-1">
                <v-tooltip location="top">
                  <template #activator="{ props }">
                    <v-btn
                      v-bind="props"
                      icon="mdi-eye"
                      size="small"
                      variant="text"
                      :to="`/workflows/${item.workflowId}/executions/${item.id}`"
                    />
                  </template>
                  <span>View Details</span>
                </v-tooltip>

                <v-tooltip
                  v-if="item.status === 'running' || item.status === 'queued'"
                  location="top"
                >
                  <template #activator="{ props }">
                    <v-btn
                      v-bind="props"
                      icon="mdi-stop-circle"
                      size="small"
                      variant="text"
                      color="error"
                      :loading="cancellingRuns[item.id]"
                      @click="cancelRun(item)"
                    />
                  </template>
                  <span>Cancel Execution</span>
                </v-tooltip>

                <v-tooltip
                  v-if="item.status === 'error' || item.status === 'partially_failed'"
                  location="top"
                >
                  <template #activator="{ props }">
                    <v-btn
                      v-bind="props"
                      icon="mdi-restart"
                      size="small"
                      variant="text"
                      color="warning"
                      :loading="retryingRuns[item.id]"
                      @click="restartRun(item)"
                    />
                  </template>
                  <span>Restart Workflow</span>
                </v-tooltip>

                <v-tooltip
                  v-if="item.status === 'interrupted'"
                  location="top"
                >
                  <template #activator="{ props }">
                    <v-btn
                      v-bind="props"
                      icon="mdi-play-pause"
                      size="small"
                      variant="text"
                      color="primary"
                      :loading="resumingRuns[item.id]"
                      @click="resumeRun(item)"
                    />
                  </template>
                  <span>Resume from Checkpoint</span>
                </v-tooltip>
              </div>
            </template>
          </v-data-table>
        </v-card>
      </v-col>
    </v-row>

    <!-- Snackbar for notifications -->
    <v-snackbar
      v-model="snackbar"
      :color="snackbarColor"
      timeout="3000"
    >
      {{ snackbarText }}
    </v-snackbar>
  </v-container>
</template>

<script setup lang="ts">
import { ref, onMounted, reactive } from "vue";
import { recoveryApi } from "@/services/api";
import type { WorkflowExecution } from "@/types";

const runs = ref<WorkflowExecution[]>([]);
const loading = ref(false);
const selectedStatus = ref("all");
const retryingRuns = reactive<Record<string, boolean>>({});
const resumingRuns = reactive<Record<string, boolean>>({});
const cancellingRuns = reactive<Record<string, boolean>>({});

const snackbar = ref(false);
const snackbarText = ref("");
const snackbarColor = ref("success");

const headers = [
  { title: "Workflow", key: "workflow", sortable: true },
  { title: "Status", key: "status", sortable: true },
  { title: "Trigger", key: "triggerType", sortable: true },
  { title: "Started", key: "startedAt", sortable: true },
  { title: "Duration", key: "duration", sortable: false },
  { title: "Nodes", key: "nodeStats", sortable: false },
  { title: "Actions", key: "actions", sortable: false, align: "end" as const },
];

const fetchRuns = async () => {
  try {
    loading.value = true;
    const response = await recoveryApi.getAllRuns(selectedStatus.value);
    runs.value = response.data;
  } catch (_error) {
    showSnackbar("Failed to fetch workflow runs", "error");
  } finally {
    loading.value = false;
  }
};

const onStatusFilterChange = () => {
  fetchRuns();
};

const cancelRun = async (run: WorkflowExecution) => {
  try {
    cancellingRuns[run.id] = true;
    await recoveryApi.cancelWorkflow(run.id);
    showSnackbar("Workflow execution cancelled", "success");
    await fetchRuns();
  } catch (error: any) {
    showSnackbar(
      error.response?.data?.error || "Failed to cancel workflow",
      "error"
    );
  } finally {
    cancellingRuns[run.id] = false;
  }
};

const restartRun = async (run: WorkflowExecution) => {
  try {
    retryingRuns[run.id] = true;
    await recoveryApi.restartWorkflow(run.id);
    showSnackbar("Workflow restart initiated", "success");
    await fetchRuns();
  } catch (error: any) {
    showSnackbar(
      error.response?.data?.error || "Failed to restart workflow",
      "error"
    );
  } finally {
    retryingRuns[run.id] = false;
  }
};

const resumeRun = async (run: WorkflowExecution) => {
  try {
    resumingRuns[run.id] = true;
    await recoveryApi.resumeWorkflow(run.workflowId, run.id);
    showSnackbar("Workflow resumption initiated - continuing from checkpoint", "success");
    await fetchRuns();
  } catch (error: any) {
    showSnackbar(
      error.response?.data?.error || "Failed to resume workflow",
      "error"
    );
  } finally {
    resumingRuns[run.id] = false;
  }
};

const getStatusColor = (status: string): string => {
  switch (status) {
    case "success":
      return "success";
    case "error":
      return "error";
    case "partially_failed":
      return "warning";
    case "running":
      return "primary";
    case "queued":
      return "info";
    case "interrupted":
      return "warning";
    case "cancelled":
      return "grey";
    default:
      return "grey";
  }
};

const getStatusLabel = (status: string): string => {
  if (status === "partially_failed") return "partial";
  return status;
};

const getTriggerIcon = (triggerType: string): string => {
  const iconMap: Record<string, string> = {
    manual: "mdi-hand-pointing-up",
    scheduled: "mdi-clock-outline",
    webhook: "mdi-webhook",
  };
  return iconMap[triggerType] || "mdi-help-circle";
};

const formatDateTime = (dateStr: string): string => {
  const date = new Date(dateStr);
  return date.toLocaleString();
};

const getDuration = (run: WorkflowExecution): string => {
  if (!run.completedAt) {
    if (run.status === "running") {
      const start = new Date(run.startedAt).getTime();
      const now = Date.now();
      const duration = Math.floor((now - start) / 1000);
      return `${duration}s (running)`;
    }
    return "—";
  }

  const start = new Date(run.startedAt).getTime();
  const end = new Date(run.completedAt).getTime();
  const durationMs = end - start;

  if (durationMs < 1000) {
    return `${durationMs}ms`;
  } else if (durationMs < 60000) {
    return `${Math.floor(durationMs / 1000)}s`;
  } else {
    const minutes = Math.floor(durationMs / 60000);
    const seconds = Math.floor((durationMs % 60000) / 1000);
    return `${minutes}m ${seconds}s`;
  }
};

const getSuccessNodeCount = (run: WorkflowExecution): number => {
  if (!run.nodeExecutions) return 0;
  return run.nodeExecutions.filter((node) => node.status === "success").length;
};

const getFailedNodeCount = (run: WorkflowExecution): number => {
  if (!run.nodeExecutions) return 0;
  return run.nodeExecutions.filter((node) => node.status === "error").length;
};

const showSnackbar = (text: string, color: string) => {
  snackbarText.value = text;
  snackbarColor.value = color;
  snackbar.value = true;
};

onMounted(() => {
  fetchRuns();
});
</script>

<style scoped>
.runs-container {
  height: 100%;
  overflow-y: auto;
}

.gap-1 {
  gap: 4px;
}

@keyframes rotate {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

.rotating {
  animation: rotate 1s linear infinite;
}
</style>

