<template>
  <v-container
    fluid
    class="pa-6 execution-page-container"
  >
    <v-row v-if="execution">
      <v-col cols="12">
        <v-btn
          variant="text"
          prepend-icon="mdi-arrow-left"
          class="mb-4"
          @click="$router.back()"
        >
          Back
        </v-btn>

        <v-card
          elevation="2"
          class="mb-6"
        >
          <v-card-title class="d-flex align-center">
            Execution {{ execution.id.substring(0, 8) }}
            <v-spacer />
            <div class="d-flex align-center gap-2">
              <!-- Recovery Actions -->
              <v-btn
                v-if="execution.status === 'error' || execution.status === 'interrupted'"
                color="primary"
                variant="outlined"
                size="small"
                prepend-icon="mdi-play-pause"
                :loading="resumingWorkflow"
                @click="resumeWorkflow"
              >
                Resume
              </v-btn>
              <v-btn
                v-if="(execution.status === 'error' || execution.status === 'partially_failed') && recoveryOptions?.canRestartWorkflow"
                color="warning"
                variant="outlined"
                size="small"
                prepend-icon="mdi-restart"
                :loading="restartingWorkflow"
                @click="restartWorkflow"
              >
                Restart Workflow
              </v-btn>
              <v-chip
                :color="getStatusColor(execution.status)"
                class="ml-2"
              >
                {{ execution.status.toUpperCase() }}
              </v-chip>
            </div>
          </v-card-title>

          <v-card-text>
            <v-row>
              <v-col
                cols="12"
                md="6"
              >
                <div class="mb-2">
                  <strong>Workflow:</strong> {{ execution.workflow?.name }}
                </div>
                <div class="mb-2">
                  <strong>Version:</strong> {{ execution.version }}
                </div>
                <div class="mb-2">
                  <strong>Trigger:</strong> {{ execution.triggerType }}
                </div>
              </v-col>
              <v-col
                cols="12"
                md="6"
              >
                <div class="mb-2">
                  <strong>Started:</strong> {{ new Date(execution.startedAt).toLocaleString() }}
                </div>
                <div
                  v-if="execution.completedAt"
                  class="mb-2"
                >
                  <strong>Completed:</strong> {{ new Date(execution.completedAt).toLocaleString() }}
                </div>
                <div class="mb-2">
                  <strong>Duration:</strong> {{ getDuration(execution) }}
                </div>
              </v-col>
            </v-row>

            <div
              v-if="execution.error"
              class="mt-4"
            >
              <v-alert
                type="error"
                variant="tonal"
                :prominent="isLimitError(execution.error)"
              >
                <div class="d-flex align-center">
                  <v-icon
                    v-if="isLimitError(execution.error)"
                    class="mr-2"
                    size="large"
                  >
                    mdi-shield-alert
                  </v-icon>
                  <div class="flex-grow-1">
                    <div
                      v-if="isLimitError(execution.error)"
                      class="text-h6 mb-2"
                    >
                      Workflow Limit Exceeded
                    </div>
                    <div>{{ execution.error }}</div>
                    <div
                      v-if="isLimitError(execution.error)"
                      class="mt-3"
                    >
                      <WorkflowLimitsInfo />
                    </div>
                  </div>
                </div>
              </v-alert>
            </div>
          </v-card-text>
        </v-card>

        <v-card elevation="2">
          <v-card-title>Node Executions</v-card-title>
          <v-card-text class="node-executions-container">
            <v-timeline
              side="end"
              density="compact"
            >
              <v-timeline-item
                v-for="(nodeGroup, nodeId) in groupedNodeExecutions"
                :key="nodeId"
                :dot-color="getStatusColor(nodeGroup[0].status)"
                size="small"
              >
                <template #opposite>
                  <div class="text-caption">
                    {{ nodeGroup[0].nodeType }}
                  </div>
                </template>

                <!-- Horizontal container for retries -->
                <div class="retry-container">
                  <div
                    v-for="(nodeExec, index) in nodeGroup"
                    :key="nodeExec.id"
                    class="retry-card"
                    :class="{ 'failed-retry': nodeExec.status === 'error' }"
                  >
                    <v-card
                      elevation="2"
                      :color="getStatusColor(nodeExec.status)"
                      variant="tonal"
                    >
                      <v-card-title class="text-subtitle-2 d-flex align-center justify-space-between">
                        <div class="d-flex align-center">
                          <v-icon
                            size="small"
                            class="mr-2"
                          >
                            {{ getNodeIcon(nodeExec.nodeType) }}
                          </v-icon>
                          {{ nodeExec.nodeType }}
                        </div>
                        <v-chip
                          size="x-small"
                          variant="flat"
                        >
                          #{{ nodeGroup.length - index }}
                        </v-chip>
                      </v-card-title>

                      <v-card-subtitle class="text-caption">
                        {{ nodeExec.nodeId.substring(0, 8) }}
                      </v-card-subtitle>

                      <v-card-text>
                        <div class="d-flex align-center justify-space-between mb-2">
                          <v-chip
                            size="x-small"
                            :color="getStatusColor(nodeExec.status)"
                          >
                            {{ nodeExec.status }}
                          </v-chip>
                          <!-- Node Retry Button -->
                          <v-btn
                            v-if="nodeExec.status === 'error' && canRetryNode(nodeExec.nodeId)"
                            color="warning"
                            variant="text"
                            size="x-small"
                            icon="mdi-refresh"
                            :loading="retryingNodes[nodeExec.id]"
                            @click="retryNode(nodeExec)"
                          />
                        </div>

                        <div class="text-caption mb-2">
                          {{ new Date(nodeExec.startedAt).toLocaleString() }}
                        </div>

                        <v-expansion-panels
                          variant="accordion"
                          density="compact"
                        >
                          <v-expansion-panel
                            v-if="nodeExec.input"
                            title="Input"
                          >
                            <v-expansion-panel-text>
                              <pre class="code-output">{{ formatJson(nodeExec.input) }}</pre>
                            </v-expansion-panel-text>
                          </v-expansion-panel>

                          <v-expansion-panel
                            v-if="nodeExec.output"
                            title="Output"
                          >
                            <v-expansion-panel-text>
                              <pre class="code-output">{{ formatJson(nodeExec.output) }}</pre>
                            </v-expansion-panel-text>
                          </v-expansion-panel>

                          <v-expansion-panel
                            v-if="nodeExec.error"
                            bg-color="error"
                          >
                            <template #title>
                              Error
                              <v-chip
                                v-if="isLimitError(nodeExec.error)"
                                size="x-small"
                                color="warning"
                                class="ml-2"
                              >
                                Limit Exceeded
                              </v-chip>
                            </template>
                            <v-expansion-panel-text>
                              <pre class="code-output error-output">{{ nodeExec.error }}</pre>
                            </v-expansion-panel-text>
                          </v-expansion-panel>
                        </v-expansion-panels>
                      </v-card-text>
                    </v-card>
                  </div>
                </div>

                <!-- Loop Body Executions (for loop and loop-accumulator nodes) -->
                <v-expansion-panels
                  v-if="(nodeGroup[0].nodeType === 'loop' || nodeGroup[0].nodeType === 'loop-accumulator') && getLoopBodyExecutions(nodeGroup[0].nodeId).length > 0"
                  variant="accordion"
                  class="mt-3"
                >
                  <v-expansion-panel>
                    <v-expansion-panel-title>
                      <v-icon
                        start
                        size="small"
                        color="primary"
                      >
                        mdi-format-list-bulleted
                      </v-icon>
                      Loop Body Executions ({{ getLoopBodyExecutions(nodeGroup[0].nodeId).length }} iterations)
                    </v-expansion-panel-title>
                    <v-expansion-panel-text>
                      <div
                        v-for="(iterationExec, iterIdx) in getLoopBodyExecutions(nodeGroup[0].nodeId)"
                        :key="iterationExec.id"
                        class="iteration-execution mb-3"
                      >
                        <v-card
                          variant="outlined"
                          :color="getStatusColor(iterationExec.status)"
                        >
                          <v-card-title class="text-subtitle-2 d-flex align-center justify-space-between">
                            <div class="d-flex align-center">
                              <v-icon
                                size="small"
                                class="mr-2"
                              >
                                {{ getNodeIcon(iterationExec.nodeType) }}
                              </v-icon>
                              {{ iterationExec.nodeType }}
                            </div>
                            <v-chip
                              size="x-small"
                              :color="getStatusColor(iterationExec.status)"
                            >
                              Iteration {{ iterIdx + 1 }}
                            </v-chip>
                          </v-card-title>

                          <v-card-subtitle class="text-caption">
                            {{ iterationExec.nodeId.substring(0, 8) }} • {{ new Date(iterationExec.startedAt).toLocaleString() }}
                          </v-card-subtitle>

                          <v-card-text>
                            <v-expansion-panels
                              variant="accordion"
                              density="compact"
                            >
                              <v-expansion-panel
                                v-if="iterationExec.input"
                                title="Input"
                              >
                                <v-expansion-panel-text>
                                  <pre class="code-output">{{ formatJson(iterationExec.input) }}</pre>
                                </v-expansion-panel-text>
                              </v-expansion-panel>

                              <v-expansion-panel
                                v-if="iterationExec.output"
                                title="Output"
                              >
                                <v-expansion-panel-text>
                                  <pre class="code-output">{{ formatJson(iterationExec.output) }}</pre>
                                </v-expansion-panel-text>
                              </v-expansion-panel>

                              <v-expansion-panel
                                v-if="iterationExec.error"
                                title="Error"
                                bg-color="error"
                              >
                                <v-expansion-panel-text>
                                  <pre class="code-output error-output">{{ iterationExec.error }}</pre>
                                </v-expansion-panel-text>
                              </v-expansion-panel>
                            </v-expansion-panels>
                          </v-card-text>
                        </v-card>
                      </div>
                    </v-expansion-panel-text>
                  </v-expansion-panel>
                </v-expansion-panels>
              </v-timeline-item>
            </v-timeline>
          </v-card-text>
        </v-card>

        <!-- Dead Letter Queue Section -->
        <v-card
          v-if="recoveryOptions?.deadLetterMessages && recoveryOptions.deadLetterMessages.length > 0"
          elevation="2"
          class="mt-6"
        >
          <v-card-title class="d-flex align-center">
            <v-icon
              class="mr-2"
              color="error"
            >
              mdi-alert-circle
            </v-icon>
            Dead Letter Queue
            <v-chip
              color="error"
              size="small"
              class="ml-2"
            >
              {{ recoveryOptions?.deadLetterMessages?.length || 0 }}
            </v-chip>
          </v-card-title>
          <v-card-text>
            <v-list density="compact">
              <v-list-item
                v-for="message in recoveryOptions?.deadLetterMessages || []"
                :key="message.id"
                class="mb-2"
              >
                <template #prepend>
                  <v-icon
                    :color="getEventTypeColor(message.eventType)"
                    size="small"
                  >
                    {{ getEventTypeIcon(message.eventType) }}
                  </v-icon>
                </template>
                <v-list-item-title class="text-subtitle-2">
                  {{ message.eventType }}
                </v-list-item-title>
                <v-list-item-subtitle>
                  Attempts: {{ message.attempts }}/{{ message.maxAttempts }}
                  <span v-if="message.lastError">
                    • {{ message.lastError.substring(0, 50) }}...
                  </span>
                </v-list-item-subtitle>
                <template #append>
                  <v-btn
                    color="warning"
                    variant="outlined"
                    size="small"
                    :loading="retryingDeadLetter[message.id]"
                    @click="retryDeadLetterMessage(message)"
                  >
                    Retry
                  </v-btn>
                </template>
              </v-list-item>
            </v-list>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

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
import { ref, onMounted, computed, reactive } from "vue";
import { useRoute } from "vue-router";
import { recoveryApi } from "@/services/api";
import type { WorkflowExecution, RecoveryOptions, DeadLetterMessage } from "@/types";
import WorkflowLimitsInfo from "@/components/WorkflowLimitsInfo.vue";

const route = useRoute();
const execution = ref<WorkflowExecution | null>(null);
const recoveryOptions = ref<RecoveryOptions | null>(null);

const snackbar = ref(false);
const snackbarText = ref("");
const snackbarColor = ref("success");

// Import SSE composable
import { useSSE } from "@/composables/useSSE";

// Loading states for recovery operations
const restartingWorkflow = ref(false);
const resumingWorkflow = ref(false);
const retryingNodes = reactive<Record<string, boolean>>({});
const retryingDeadLetter = reactive<Record<string, boolean>>({});

// Group node executions by nodeId (for showing retries horizontally)
// Filter out loop body executions (they have parentLoopNodeId set)
const groupedNodeExecutions = computed(() => {
  if (!execution.value?.nodeExecutions) return {};

  const grouped: Record<string, any[]> = {};

  // Group by nodeId, excluding loop body executions
  execution.value.nodeExecutions.forEach((nodeExec) => {
    // Skip loop body executions - they will be shown separately
    if (nodeExec.parentLoopNodeId) {
      return;
    }

    if (!grouped[nodeExec.nodeId]) {
      grouped[nodeExec.nodeId] = [];
    }
    grouped[nodeExec.nodeId].push(nodeExec);
  });

  // Sort each group by startedAt DESC (most recent first/leftmost)
  Object.keys(grouped).forEach((nodeId) => {
    grouped[nodeId].sort((a, b) =>
      new Date(b.startedAt).getTime() - new Date(a.startedAt).getTime()
    );
  });

  // Convert to array and sort by earliest execution time (chronological order)
  const sortedEntries = Object.entries(grouped).sort((a, b) => {
    const aEarliest = Math.min(...a[1].map(node => new Date(node.startedAt).getTime()));
    const bEarliest = Math.min(...b[1].map(node => new Date(node.startedAt).getTime()));
    return aEarliest - bEarliest;
  });

  // Convert back to object
  return Object.fromEntries(sortedEntries);
});

// Get loop body executions for a loop node using the parentLoopNodeId field
const getLoopBodyExecutions = (loopNodeId: string) => {
  if (!execution.value?.nodeExecutions) return [];

  // Filter executions that have this loop as their parent
  const loopBodyExecs = execution.value.nodeExecutions.filter((nodeExec) => {
    return nodeExec.parentLoopNodeId === loopNodeId;
  });

  // Sort by startedAt to show iterations in chronological order
  return loopBodyExecs.sort((a, b) =>
    new Date(a.startedAt).getTime() - new Date(b.startedAt).getTime()
  );
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
    case "cancelled":
      return "grey";
    case "interrupted":
      return "warning";
    default:
      return "grey";
  }
};

// Check if error message is related to workflow limits
const isLimitError = (errorMessage?: string): boolean => {
  if (!errorMessage) return false;

  const limitKeywords = [
    "limit",
    "exceeded",
    "maximum",
    "timeout",
    "depth",
    "iterations",
    "nodes executed",
    "nesting depth",
    "accumulator size",
    "data size",
  ];

  const lowerError = errorMessage.toLowerCase();
  return limitKeywords.some((keyword) => lowerError.includes(keyword));
};

const getDuration = (exec: WorkflowExecution): string => {
  if (!exec.completedAt) return "In progress";

  const start = new Date(exec.startedAt).getTime();
  const end = new Date(exec.completedAt).getTime();
  const durationMs = end - start;

  if (durationMs < 1000) return `${durationMs}ms`;
  if (durationMs < 60000) return `${(durationMs / 1000).toFixed(2)}s`;
  return `${(durationMs / 60000).toFixed(2)}m`;
};

const getNodeIcon = (nodeType: string): string => {
  const iconMap: Record<string, string> = {
    start: "mdi-play-circle",
    end: "mdi-stop-circle",
    http: "mdi-web",
    email: "mdi-email",
    slack: "mdi-slack",
    transform: "mdi-shuffle-variant",
    conditional: "mdi-call-split",
    loop: "mdi-repeat",
    delay: "mdi-timer-sand",
    "json-to-csv": "mdi-table-arrow-left",
  };
  return iconMap[nodeType] || "mdi-cog";
};

const formatJson = (jsonString?: string): string => {
  if (!jsonString) return "No data";
  try {
    const parsed = JSON.parse(jsonString);
    return JSON.stringify(parsed, null, 2);
  } catch {
    return jsonString;
  }
};

const fetchExecution = async () => {
  try {
    const executionId = route.params.executionId as string;
    const workflowId = route.params.id as string;
    const response = await recoveryApi.getExecutionWithRecovery(workflowId, executionId);
    execution.value = response.data.execution;
    recoveryOptions.value = response.data.recoveryOptions;
  } catch (_error) {
    showSnackbar("Failed to fetch execution", "error");
  }
};

// Setup SSE connection for real-time updates using composable
const setupSSE = () => {
  const executionId = route.params.executionId as string;
  const workflowId = route.params.id as string;

  const { connect } = useSSE({
    url: `/api/workflows/${workflowId}/executions/${executionId}/stream`,
    maxReconnectAttempts: 5,
    reconnectDelay: 5000,
    onMessage: {
      update: (data: WorkflowExecution) => {
        // Update execution data
        execution.value = data;

        // Update recovery options if needed
        if (execution.value) {
          recoveryOptions.value = {
            canRestartWorkflow: execution.value.status === "error" || execution.value.status === "partially_failed",
            canRetryNodes: getRetryableNodesFromExecution(execution.value),
            deadLetterMessages: recoveryOptions.value?.deadLetterMessages || [],
          };
        }
      },
      complete: (_data: { status: string }) => {
        // Connection will auto-close after server sends complete event
      },
      error: (data: { error: string }) => {
        console.error("SSE error event:", data);
        showSnackbar(data.error || "Stream connection error", "error");
      },
    },
    onConnected: () => {
      // Connection established
    },
    onClosed: () => {
      // Connection closed
    },
    onError: (error: string) => {
      showSnackbar(error, "error");
    },
    onMaxReconnectAttemptsReached: () => {
      showSnackbar("Connection lost. Please refresh the page.", "error");
    },
  });

  // Start the connection
  connect();
};

// Helper to get retryable nodes from execution
const getRetryableNodesFromExecution = (exec: WorkflowExecution): string[] => {
  if (!exec.nodeExecutions) return [];

  const retryableNodes: string[] = [];
  for (const nodeExec of exec.nodeExecutions) {
    if (nodeExec.status === "error") {
      // Only asynchronous nodes can be retried individually
      if (["email", "http", "slack"].includes(nodeExec.nodeType)) {
        retryableNodes.push(nodeExec.nodeId);
      }
    }
  }
  return retryableNodes;
};

// Cleanup SSE connection
// SSE cleanup is now handled automatically by the useSSE composable

// Recovery functions
const restartWorkflow = async () => {
  if (!execution.value) return;

  try {
    restartingWorkflow.value = true;
    await recoveryApi.restartWorkflow(execution.value.id);
    showSnackbar("Workflow restart initiated", "success");
    // Refresh execution data
    await fetchExecution();
  } catch (error: any) {
    showSnackbar(
      error.response?.data?.error || "Failed to restart workflow",
      "error"
    );
  } finally {
    restartingWorkflow.value = false;
  }
};

const resumeWorkflow = async () => {
  if (!execution.value) return;

  try {
    resumingWorkflow.value = true;
    await recoveryApi.resumeWorkflow(execution.value.workflowId, execution.value.id);
    showSnackbar("Workflow resumption initiated - continuing from checkpoint", "success");
    // Refresh execution data
    await fetchExecution();
    // Restart SSE connection since workflow is now running again
    setupSSE();
  } catch (error: any) {
    showSnackbar(
      error.response?.data?.error || "Failed to resume workflow",
      "error"
    );
  } finally {
    resumingWorkflow.value = false;
  }
};

const canRetryNode = (nodeId: string): boolean => {
  return recoveryOptions.value?.canRetryNodes?.includes(nodeId) || false;
};

const retryNode = async (nodeExec: any) => {
  if (!execution.value) return;

  try {
    retryingNodes[nodeExec.id] = true;
    await recoveryApi.reExecuteNode(execution.value.id, nodeExec.nodeId);
    showSnackbar("Node retry initiated", "success");
    // Refresh execution data
    await fetchExecution();
  } catch (error: any) {
    showSnackbar(
      error.response?.data?.error || "Failed to retry node",
      "error"
    );
  } finally {
    retryingNodes[nodeExec.id] = false;
  }
};

const retryDeadLetterMessage = async (message: DeadLetterMessage) => {
  try {
    retryingDeadLetter[message.id] = true;
    await recoveryApi.retryDeadLetterMessage(message.id);
    showSnackbar("Dead letter message retry initiated", "success");
    // Refresh execution data
    await fetchExecution();
  } catch (error: any) {
    showSnackbar(
      error.response?.data?.error || "Failed to retry dead letter message",
      "error"
    );
  } finally {
    retryingDeadLetter[message.id] = false;
  }
};

const getEventTypeIcon = (eventType: string): string => {
  const iconMap: Record<string, string> = {
    "email.send": "mdi-email",
    "http.request": "mdi-web",
    "slack.send": "mdi-slack",
  };
  return iconMap[eventType] || "mdi-cog";
};

const getEventTypeColor = (eventType: string): string => {
  const colorMap: Record<string, string> = {
    "email.send": "blue",
    "http.request": "green",
    "slack.send": "purple",
  };
  return colorMap[eventType] || "grey";
};

const showSnackbar = (text: string, color: string) => {
  snackbarText.value = text;
  snackbarColor.value = color;
  snackbar.value = true;
};

onMounted(() => {
  fetchExecution().then(() => {
    // Setup SSE after initial fetch
    // Connect if execution is not in a final state (might still change)
    // Note: "interrupted" is resumable, so we keep connection open for it too
    if (execution.value) {
      const finalStatuses = ["success", "error", "partially_failed", "cancelled"];
      if (!finalStatuses.includes(execution.value.status)) {
        setupSSE();
      }
    }
  });
});

// SSE cleanup is now handled automatically by the useSSE composable (onUnmounted hook)
</script>

<style scoped>
.execution-page-container {
  height: calc(100vh - 64px);
  overflow-y: auto;
  overflow-x: auto;
}

.node-executions-container {
  overflow-x: auto;
  overflow-y: visible;
}

.retry-container {
  display: flex;
  gap: 16px;
  overflow-x: auto;
  padding: 8px 0;
  align-items: flex-start;
  scroll-behavior: smooth;
}

.retry-card {
  flex: 0 0 300px;
  max-width: 300px;
  min-width: 300px;
  transition: opacity 0.3s ease;
}

.retry-card.failed-retry {
  opacity: 0.6;
}

.retry-card:hover {
  opacity: 1;
}

.error-text {
  color: rgb(var(--v-theme-error));
  word-break: break-word;
}

.code-output {
  background: rgb(var(--v-theme-surface-variant));
  color: rgb(var(--v-theme-on-surface-variant));
  padding: 12px;
  border-radius: 4px;
  overflow-x: auto;
  overflow-y: auto;
  max-height: 300px;
  font-size: 12px;
  font-family: 'Courier New', monospace;
  border: 1px solid rgb(var(--v-theme-outline-variant));
}

.error-output {
  background: rgba(255, 255, 255, 0.1);
  color: white;
  border-color: rgba(255, 255, 255, 0.2);
}

.gap-2 {
  gap: 8px;
}

.iteration-execution {
  padding-left: 16px;
  border-left: 2px solid rgb(var(--v-theme-primary));
}

.iteration-execution:last-child {
  margin-bottom: 0 !important;
}
</style>

