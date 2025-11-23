<template>
  <div class="workflow-editor-container">
    <!-- Node Palette (Left Sidebar) -->
    <div
      v-if="!leftPanelCollapsed"
      class="palette-sidebar"
      :style="{ width: leftPanelWidth + 'px' }"
    >
      <v-toolbar
        color="surface-variant"
        density="compact"
      >
        <v-toolbar-title>Nodes</v-toolbar-title>
        <v-spacer />
        <v-btn
          icon="mdi-chevron-left"
          variant="text"
          size="small"
          @click="leftPanelCollapsed = true"
        />
      </v-toolbar>

      <div class="sidebar-content">
        <v-list>
          <template
            v-for="category in nodeCategories"
            :key="category.name"
          >
            <v-list-subheader class="category-header">
              {{ category.name }}
            </v-list-subheader>
            <v-list-item
              v-for="nodeType in category.nodes"
              :key="nodeType.type"
              class="node-palette-item"
              @click="addNode(nodeType)"
            >
              <template #prepend>
                <v-icon
                  :color="nodeType.color"
                  size="small"
                >
                  {{ nodeType.icon }}
                </v-icon>
              </template>
              <v-list-item-title class="node-item-title">
                {{ nodeType.label }}
              </v-list-item-title>
            </v-list-item>
          </template>
        </v-list>
      </div>
    </div>

    <!-- Left Collapse Button -->
    <div
      v-if="leftPanelCollapsed"
      class="collapse-button collapse-button-left"
      @click="leftPanelCollapsed = false"
    >
      <v-btn
        icon="mdi-chevron-right"
        variant="text"
        size="small"
      />
    </div>

    <!-- Left Resize Handle -->
    <div
      v-if="!leftPanelCollapsed"
      class="resize-handle resize-handle-left"
      @mousedown="startResize('left', $event)"
    />

    <!-- Canvas -->
    <div class="canvas-area">
      <v-toolbar
        color="surface"
      >
        <v-text-field
          v-model="workflowName"
          label="Workflow Name"
          variant="outlined"
          density="compact"
          hide-details
          class="mx-4"
          style="max-width: 300px"
        />

        <v-spacer />

        <v-btn
          icon="mdi-content-save"
          variant="text"
          :loading="saving"
          @click="saveWorkflow"
        />
        <v-btn
          icon="mdi-play"
          variant="text"
          color="success"
          :disabled="!workflowId"
          :loading="executing"
          @click="executeWorkflow"
        />
        <v-btn
          icon="mdi-file-document-outline"
          variant="text"
          :disabled="!workflowId"
          @click="showExecutions"
        />
        <v-btn
          icon="mdi-history"
          variant="text"
          :disabled="!workflowId"
          @click="showVersionHistory"
        />
        <WorkflowLimitsInfo />
        <v-btn
          icon="mdi-delete"
          variant="text"
          color="error"
          :disabled="!selectedNode"
          @click="deleteSelectedNode"
        />
      </v-toolbar>

      <VueFlow
        v-model="elements"
        class="workflow-canvas"
        :default-zoom="1"
        :min-zoom="0.5"
        :max-zoom="2"
        @node-click="onNodeClick"
      >
        <Background />
        <Controls />

        <template #node-http="props">
          <HttpNode v-bind="props" />
        </template>
        <template #node-email="props">
          <EmailNode v-bind="props" />
        </template>
        <template #node-conditional="props">
          <ConditionalNode v-bind="props" />
        </template>
        <template #node-transform="props">
          <TransformNode v-bind="props" />
        </template>
        <template #node-start="props">
          <StartNode v-bind="props" />
        </template>
        <template #node-end="props">
          <EndNode v-bind="props" />
        </template>
        <template #node-loop="props">
          <LoopNode v-bind="props" />
        </template>
        <template #node-loop-accumulator="props">
          <LoopAccumulatorNode v-bind="props" />
        </template>
        <template #node-delay="props">
          <DelayNode v-bind="props" />
        </template>
        <template #node-sleep="props">
          <SleepNode v-bind="props" />
        </template>
        <template #node-slack="props">
          <SlackNode v-bind="props" />
        </template>
        <template #node-json-to-csv="props">
          <JsonToCsvNode v-bind="props" />
        </template>
        <template #node-json-array="props">
          <JsonArrayNode v-bind="props" />
        </template>
        <template #node-json="props">
          <JsonNode v-bind="props" />
        </template>
      </VueFlow>
    </div>

    <!-- Right Resize Handle -->
    <div
      v-if="!rightPanelCollapsed"
      class="resize-handle resize-handle-right"
      @mousedown="startResize('right', $event)"
    />

    <!-- Right Collapse Button -->
    <div
      v-if="rightPanelCollapsed"
      class="collapse-button collapse-button-right"
      @click="rightPanelCollapsed = false"
    >
      <v-btn
        icon="mdi-chevron-left"
        variant="text"
        size="small"
      />
    </div>

    <!-- Property Inspector (Right Sidebar) -->
    <div
      v-if="!rightPanelCollapsed"
      class="properties-sidebar"
      :style="{ width: rightPanelWidth + 'px' }"
    >
      <v-toolbar
        density="compact"
        color="surface-variant"
      >
        <v-btn
          icon="mdi-chevron-right"
          variant="text"
          size="small"
          @click="rightPanelCollapsed = true"
        />
        <v-toolbar-title>Properties</v-toolbar-title>
      </v-toolbar>

      <div class="sidebar-content">
        <div
          v-if="selectedNode"
          class="pa-4"
        >
          <h3 class="mb-4">
            {{ selectedNode.data.label }}
          </h3>

          <component
            :is="getPropertyComponent(selectedNode.type)"
            v-model="selectedNode.data.config"
            :workflow-id="workflowId"
            @update="updateNode"
          />
        </div>
        <div
          v-else
          class="pa-4 text-center text-grey"
        >
          Select a node to view properties
        </div>
      </div>
    </div>

    <!-- Executions Dialog -->
    <v-dialog
      v-model="executionsDialog"
      max-width="800"
    >
      <v-card>
        <v-card-title>Execution History</v-card-title>
        <v-card-text>
          <v-list v-if="executions.length > 0">
            <v-list-item
              v-for="exec in executions"
              :key="exec.id"
              style="cursor: pointer"
              @click="viewExecution(exec.id)"
            >
              <template #prepend>
                <v-icon :color="getExecutionStatusColor(exec.status)">
                  {{
                    exec.status === "success"
                      ? "mdi-check-circle"
                      : exec.status === "error"
                        ? "mdi-alert-circle"
                        : exec.status === "cancelled"
                          ? "mdi-cancel"
                          : exec.status === "running"
                            ? "mdi-loading mdi-spin"
                            : "mdi-help-circle"
                  }}
                </v-icon>
              </template>
              <v-list-item-title>
                Execution {{ exec.id.substring(0, 8) }}
              </v-list-item-title>
              <v-list-item-subtitle>
                {{ exec.triggerType }} - {{ new Date(exec.startedAt).toLocaleString() }}
                {{ exec.error ? ` - Error: ${exec.error.substring(0, 50)}...` : "" }}
              </v-list-item-subtitle>
              <template #append>
                <v-chip
                  size="small"
                  :color="getExecutionStatusColor(exec.status)"
                >
                  {{ exec.status }}
                </v-chip>
              </template>
            </v-list-item>
          </v-list>
          <v-alert
            v-else
            type="info"
            variant="tonal"
          >
            No executions yet. Click the play button to execute this workflow.
          </v-alert>
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn @click="executionsDialog = false">
            Close
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Version History Dialog -->
    <v-dialog
      v-model="versionDialog"
      max-width="600"
    >
      <v-card>
        <v-card-title>Version History</v-card-title>
        <v-card-text>
          <v-list>
            <v-list-item
              v-for="version in versions"
              :key="version.id"
            >
              <v-list-item-title>
                Version {{ version.version }}
              </v-list-item-title>
              <v-list-item-subtitle>
                {{ version.changeLog }} - {{ new Date(version.createdAt).toLocaleString() }}
              </v-list-item-subtitle>
              <template #append>
                <v-btn
                  icon="mdi-restore"
                  variant="text"
                  size="small"
                  @click="restoreVersion(version.version)"
                />
              </template>
            </v-list-item>
          </v-list>
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn @click="versionDialog = false">
            Close
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>


    <v-snackbar
      v-model="snackbar"
      :color="snackbarColor"
      timeout="3000"
    >
      {{ snackbarText }}
    </v-snackbar>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from "vue";
import { useRoute, useRouter } from "vue-router";
import { VueFlow, useVueFlow } from "@vue-flow/core";
import { Background } from "@vue-flow/background";
import { Controls } from "@vue-flow/controls";
import api from "@/services/api";
import type { WorkflowVersion } from "@/types";

// Import node components
import HttpNode from "@/components/nodes/HttpNode.vue";
import EmailNode from "@/components/nodes/EmailNode.vue";
import ConditionalNode from "@/components/nodes/ConditionalNode.vue";
import TransformNode from "@/components/nodes/TransformNode.vue";
import StartNode from "@/components/nodes/StartNode.vue";
import EndNode from "@/components/nodes/EndNode.vue";
import LoopNode from "@/components/nodes/LoopNode.vue";
import LoopAccumulatorNode from "@/components/nodes/LoopAccumulatorNode.vue";
import DelayNode from "@/components/nodes/DelayNode.vue";
import SleepNode from "@/components/nodes/SleepNode.vue";
import SlackNode from "@/components/nodes/SlackNode.vue";
import JsonToCsvNode from "@/components/nodes/JsonToCsvNode.vue";
import JsonArrayNode from "@/components/nodes/JsonArrayNode.vue";
import JsonNode from "@/components/nodes/JsonNode.vue";

// Import property components
import HttpProperties from "@/components/properties/HttpProperties.vue";
import EmailProperties from "@/components/properties/EmailProperties.vue";
import ConditionalProperties from "@/components/properties/ConditionalProperties.vue";
import TransformProperties from "@/components/properties/TransformProperties.vue";
import LoopProperties from "@/components/properties/LoopProperties.vue";
import LoopAccumulatorProperties from "@/components/properties/LoopAccumulatorProperties.vue";
import DelayProperties from "@/components/properties/DelayProperties.vue";
import SleepProperties from "@/components/properties/SleepProperties.vue";
import SlackProperties from "@/components/properties/SlackProperties.vue";
import JsonToCsvProperties from "@/components/properties/JsonToCsvProperties.vue";
import WorkflowLimitsInfo from "@/components/WorkflowLimitsInfo.vue";
import JsonArrayTriggerProperties from "@/components/properties/JsonArrayTriggerProperties.vue";
import JsonProperties from "@/components/properties/JsonProperties.vue";
import StartProperties from "@/components/properties/StartProperties.vue";

// Import Vue Flow styles
import "@vue-flow/core/dist/style.css";
import "@vue-flow/core/dist/theme-default.css";
import "@vue-flow/controls/dist/style.css";
import "@vue-flow/minimap/dist/style.css";

const route = useRoute();
const router = useRouter();
const { addNodes, addEdges, onConnect, removeNodes } = useVueFlow();

const workflowId = ref<string | null>(null);
const workflowName = ref("New Workflow");
const elements = ref<any[]>([]);
const selectedNode = ref<any | null>(null);
const saving = ref(false);
const executing = ref(false);
const versionDialog = ref(false);
const versions = ref<WorkflowVersion[]>([]);
const executionsDialog = ref(false);
const executions = ref<any[]>([]);

const snackbar = ref(false);
const snackbarText = ref("");
const snackbarColor = ref("success");

// Schedule state (removed - now handled in Start node properties)
const workflowTimezone = ref("UTC");

const nodeCategories = [
  {
    name: "Data Blocks",
    nodes: [
      { type: "json", label: "JSON Data", icon: "mdi-code-braces", color: "orange" },
      { type: "json-array", label: "JSON Array", icon: "mdi-code-json", color: "orange" },
    ],
  },
  {
    name: "Flow Control",
    nodes: [
      { type: "start", label: "Start", icon: "mdi-play-circle", color: "green" },
      { type: "end", label: "End", icon: "mdi-stop-circle", color: "green" },
      { type: "conditional", label: "Conditional", icon: "mdi-call-split", color: "green" },
      { type: "loop", label: "Loop", icon: "mdi-repeat", color: "green" },
      { type: "loop-accumulator", label: "Loop Accumulator", icon: "mdi-format-list-bulleted-square", color: "green" },
      { type: "delay", label: "Delay", icon: "mdi-timer-sand", color: "green" },
      { type: "sleep", label: "Sleep", icon: "mdi-sleep", color: "indigo" },
    ],
  },
  {
    name: "Actions",
    nodes: [
      { type: "http", label: "HTTP Request", icon: "mdi-web", color: "blue" },
      { type: "email", label: "Send Email", icon: "mdi-email", color: "blue" },
      { type: "slack", label: "Slack Message", icon: "mdi-slack", color: "blue" },
    ],
  },
  {
    name: "Data Processing",
    nodes: [
      { type: "transform", label: "Transform", icon: "mdi-shuffle-variant", color: "purple" },
      { type: "json-to-csv", label: "JSON to CSV", icon: "mdi-table-arrow-left", color: "purple" },
    ],
  },
];

// Panel resize and collapse functionality
const leftPanelWidth = ref(250);
const rightPanelWidth = ref(320);
const leftPanelCollapsed = ref(false);
const rightPanelCollapsed = ref(false);
const isResizing = ref(false);
const resizingPanel = ref<"left" | "right" | null>(null);
const startX = ref(0);
const startWidth = ref(0);

const startResize = (panel: "left" | "right", event: MouseEvent) => {
  isResizing.value = true;
  resizingPanel.value = panel;
  startX.value = event.clientX;
  startWidth.value = panel === "left" ? leftPanelWidth.value : rightPanelWidth.value;

  document.addEventListener("mousemove", handleResize);
  document.addEventListener("mouseup", stopResize);
  event.preventDefault();
};

const handleResize = (event: MouseEvent) => {
  if (!isResizing.value || !resizingPanel.value) return;

  const delta = event.clientX - startX.value;

  if (resizingPanel.value === "left") {
    const newWidth = startWidth.value + delta;
    leftPanelWidth.value = Math.max(200, Math.min(500, newWidth));
  } else {
    const newWidth = startWidth.value - delta;
    rightPanelWidth.value = Math.max(250, Math.min(600, newWidth));
  }
};

const stopResize = () => {
  isResizing.value = false;
  resizingPanel.value = null;
  document.removeEventListener("mousemove", handleResize);
  document.removeEventListener("mouseup", stopResize);
};

let nodeIdCounter = 1;

const addNode = (nodeType: any) => {
  const id = `node-${nodeIdCounter++}`;
  const newNode = {
    id,
    type: nodeType.type,
    position: { x: 250, y: 100 + (nodeIdCounter * 50) },
    data: {
      label: nodeType.label,
      config: getDefaultConfig(nodeType.type),
    },
  };

  addNodes([newNode]);
  showSnackbar(`Added ${nodeType.label}`, "success");
};

const getDefaultConfig = (type: string): Record<string, any> => {
  switch (type) {
    case "json":
      return { data: "{}" };
    case "json-array":
      return { jsonArray: "[]", validateSchema: true };
    case "http":
      return { url: "", method: "GET", headers: {}, body: null };
    case "email":
      return { to: "", subject: "", body: "", isHtml: false, cc: "", bcc: "", attachments: [], provider: undefined };
    case "conditional":
      return { conditions: [], logicalOperator: "AND" };
    case "transform":
      return { operations: [] };
    case "loop":
      return { arrayPath: "", itemVariable: "item", indexVariable: "index" };
    case "loop-accumulator":
      return { arrayPath: "", itemVariable: "item", indexVariable: "index", accumulatorVariable: "accumulated", accumulationMode: "array", errorHandling: "skip" };
    case "sleep":
      return { mode: "relative", duration_value: 1, duration_unit: "hours", target_date: "", timezone: "UTC" };
    case "slack":
      return { apiKey: "", channel: "", message: "" };
    case "json-to-csv":
      return { arrayPath: "", delimiter: ",", includeHeaders: true, outputFormat: "string", columns: [] };
    case "start":
      return { triggerType: "manual", cronSchedule: "", timezone: "UTC", webhookPath: "", webhookRequireAuth: false, webhookSecretConfigured: false };
    default:
      return {};
  }
};

const onNodeClick = (event: any) => {
  selectedNode.value = event.node;
};

const updateNode = () => {
  // Node is updated via v-model binding
  showSnackbar("Node updated", "success");
};

const deleteSelectedNode = () => {
  if (!selectedNode.value) return;

  const nodeId = selectedNode.value.id;

  // Remove the node
  removeNodes([nodeId]);

  // Clear selection
  selectedNode.value = null;

  showSnackbar("Node deleted", "info");
};

const getPropertyComponent = (nodeType: string) => {
  switch (nodeType) {
    case "json":
      return JsonProperties;
    case "json-array":
      return JsonArrayTriggerProperties;
    case "http":
      return HttpProperties;
    case "email":
      return EmailProperties;
    case "conditional":
      return ConditionalProperties;
    case "transform":
      return TransformProperties;
    case "loop":
      return LoopProperties;
    case "loop-accumulator":
      return LoopAccumulatorProperties;
    case "delay":
      return DelayProperties;
    case "sleep":
      return SleepProperties;
    case "slack":
      return SlackProperties;
    case "json-to-csv":
      return JsonToCsvProperties;
    case "start":
      return StartProperties;
    default:
      return null;
  }
};

const saveWorkflow = async () => {
  try {
    saving.value = true;

    const definition = {
      nodes: elements.value.filter(el => !el.source), // Nodes don't have source
      edges: elements.value.filter(el => el.source), // Edges have source
    };

    // Extract trigger configuration from start node
    const startNode = elements.value.find((el: any) => el.type === "start" && !el.source);
    let schedule: string | undefined = undefined;
    let timezone = "UTC";
    let webhookPath: string | null | undefined = undefined;
    let webhookRequireAuth = false;

    if (startNode && startNode.data?.config) {
      const config = startNode.data.config;
      const triggerType = config.triggerType || "manual";

      if (triggerType === "cron" || triggerType === "both") {
        schedule = config.cronSchedule || undefined;
        timezone = config.timezone || "UTC";
      } else {
        // Clear schedule if not cron
        schedule = undefined;
      }

      if (triggerType === "webhook" || triggerType === "both") {
        // Only send webhookPath if user has set a custom path, otherwise send null to use default
        webhookPath = (config.webhookPath && config.webhookPath.trim() !== "") ? config.webhookPath : null;
        // Always require auth for webhooks
        webhookRequireAuth = true;
      } else {
        // Clear webhook config if not webhook - send null to clear
        webhookPath = null;
        webhookRequireAuth = false;
      }
    } else {
      // No start node config, clear all trigger settings
      schedule = undefined;
      webhookPath = undefined;
      webhookRequireAuth = false;
    }

    if (workflowId.value) {
      // Update existing workflow
      const updatePayload: any = {
        name: workflowName.value,
        definition,
        change_log: "Updated via editor",
        // Send empty string to clear schedule, or the actual schedule value
        schedule: schedule || "",
        timezone,
        // Send null for webhookPath to use default, or the custom path, or empty string to clear
        webhookPath: webhookPath !== null ? webhookPath : "",
        webhookRequireAuth,
      };
      await api.put(`/workflows/${workflowId.value}`, updatePayload);
      showSnackbar("Workflow saved", "success");
    } else {
      // Create new workflow
      const createPayload: any = {
        name: workflowName.value,
        definition,
        // Send empty string to clear schedule, or the actual schedule value
        schedule: schedule || "",
        timezone,
        // Send null for webhookPath to use default, or the custom path
        webhookPath: webhookPath,
        webhookRequireAuth,
      };
      const response = await api.post("/workflows", createPayload);
      workflowId.value = response.data.id;
      router.replace(`/workflows/${workflowId.value}/edit`);
      showSnackbar("Workflow created", "success");
    }
  } catch (error: any) {
    console.error("Failed to save workflow:", error);
    console.error("Error details:", error.response?.data);
    showSnackbar(
      error.response?.data?.error || "Failed to save workflow",
      "error"
    );
  } finally {
    saving.value = false;
  }
};

const executeWorkflow = async () => {
  if (!workflowId.value) return;

  try {
    executing.value = true;
    const response = await api.post(`/workflows/${workflowId.value}/execute`, {});
    const executionId = response.data.execution_id;

    showSnackbar("Workflow execution started! Click to view details", "success");

    // Navigate to execution details after a short delay
    setTimeout(() => {
      router.push(`/workflows/${workflowId.value}/executions/${executionId}`);
    }, 1500);
  } catch (error: any) {
    showSnackbar(
      error.response?.data?.error || "Failed to execute workflow",
      "error"
    );
  } finally {
    executing.value = false;
  }
};

const showVersionHistory = async () => {
  if (!workflowId.value) return;

  try {
    const response = await api.get(`/workflows/${workflowId.value}/versions`);
    versions.value = response.data;
    versionDialog.value = true;
  } catch (_error) {
    showSnackbar("Failed to load version history", "error");
  }
};

const showExecutions = async () => {
  if (!workflowId.value) return;

  try {
    const response = await api.get(`/workflows/${workflowId.value}/executions`);
    executions.value = response.data;
    executionsDialog.value = true;
  } catch (_error) {
    showSnackbar("Failed to load executions", "error");
  }
};

const viewExecution = (executionId: string) => {
  executionsDialog.value = false;
  router.push(`/workflows/${workflowId.value}/executions/${executionId}`);
};

const getExecutionStatusColor = (status: string): string => {
  switch (status) {
    case "success":
      return "success";
    case "error":
      return "error";
    case "running":
      return "primary";
    default:
      return "warning";
  }
};

const restoreVersion = async (version: number) => {
  if (!workflowId.value) return;

  try {
    await api.post(`/workflows/${workflowId.value}/versions/restore`, { version });
    showSnackbar(`Restored to version ${version}`, "success");
    versionDialog.value = false;
    loadWorkflow();
  } catch (_error) {
    showSnackbar("Failed to restore version", "error");
  }
};


const loadWorkflow = async () => {
  const id = route.params.id as string;
  if (!id || id === "new") return;

  try {
    const response = await api.get(`/workflows/${id}`);
    const workflow = response.data;

    workflowId.value = workflow.id;
    workflowName.value = workflow.name;
    workflowTimezone.value = workflow.timezone || "UTC";

    // Load latest version
    if (workflow.versions && workflow.versions.length > 0) {
      const definition = JSON.parse(workflow.versions[0].definition);
      elements.value = [...definition.nodes, ...definition.edges];

      // Populate start node config from workflow trigger settings
      const startNodeIndex = elements.value.findIndex((el: any) => el.type === "start" && !el.source);
      if (startNodeIndex !== -1) {
        const startNode = elements.value[startNodeIndex];

        // Determine trigger type based on workflow settings
        const hasSchedule = workflow.schedule && workflow.schedule !== "";
        // Webhook is enabled ONLY if webhookRequireAuth is explicitly set to true
        // This ensures that removing webhook clears it properly even if secret still exists
        const hasWebhook = workflow.webhookRequireAuth === true;

        let triggerType = "manual";
        if (hasSchedule && hasWebhook) {
          triggerType = "both";
        } else if (hasSchedule) {
          triggerType = "cron";
        } else if (hasWebhook) {
          triggerType = "webhook";
        }

        // Create new config object to trigger reactivity
        const newConfig = {
          triggerType,
          cronSchedule: workflow.schedule || "",
          timezone: workflow.timezone || "UTC",
          webhookPath: workflow.webhookPath || "",
          webhookRequireAuth: workflow.webhookRequireAuth || false,
          webhookSecretConfigured: workflow.hasWebhookSecret || false,
        };

        // Update the start node with new config (create new object for reactivity)
        elements.value[startNodeIndex] = {
          ...startNode,
          data: {
            ...startNode.data,
            config: newConfig,
          },
        };
      }

      // Update node counter
      const maxId = Math.max(...definition.nodes.map((n: any) => {
        const match = n.id.match(/node-(\d+)/);
        return match ? parseInt(match[1]) : 0;
      }));
      nodeIdCounter = maxId + 1;
    }
  } catch (_error) {
    showSnackbar("Failed to load workflow", "error");
  }
};

const showSnackbar = (text: string, color: string) => {
  snackbarText.value = text;
  snackbarColor.value = color;
  snackbar.value = true;
};

// Handle keyboard shortcuts
const handleKeyDown = (event: KeyboardEvent) => {
  if ((event.key === "Delete" || event.key === "Backspace") && selectedNode.value) {
    // Don't delete if user is typing in an input
    const target = event.target as HTMLElement;
    if (target.tagName === "INPUT" || target.tagName === "TEXTAREA") return;

    event.preventDefault();
    deleteSelectedNode();
  }
};

onMounted(() => {
  loadWorkflow();

  // Handle connections
  onConnect((connection) => {
    addEdges([connection]);
  });

  // Add keyboard listener
  window.addEventListener("keydown", handleKeyDown);
});

onUnmounted(() => {
  window.removeEventListener("keydown", handleKeyDown);
});
</script>

<style scoped>
.workflow-editor-container {
  display: flex;
  height: calc(100vh - 64px);
  width: 100%;
  overflow: hidden;
}

.palette-sidebar {
  display: flex;
  flex-direction: column;
  border-right: 1px solid rgb(var(--v-theme-surface-variant));
  background: rgb(var(--v-theme-surface));
  flex-shrink: 0;
  height: 100%;
}

.properties-sidebar {
  display: flex;
  flex-direction: column;
  border-left: 1px solid rgb(var(--v-theme-surface-variant));
  background: rgb(var(--v-theme-surface));
  flex-shrink: 0;
  height: 100%;
}

.sidebar-content {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
  min-height: 0; /* Important for flex scrolling */
}

/* Custom scrollbar styling */
.sidebar-content::-webkit-scrollbar {
  width: 8px;
}

.sidebar-content::-webkit-scrollbar-track {
  background: rgb(var(--v-theme-surface));
}

.sidebar-content::-webkit-scrollbar-thumb {
  background: rgb(var(--v-theme-surface-variant));
  border-radius: 4px;
}

.sidebar-content::-webkit-scrollbar-thumb:hover {
  background: rgb(var(--v-theme-primary));
}

.collapse-button {
  width: 32px;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgb(var(--v-theme-surface-variant));
  cursor: pointer;
  transition: background-color 0.2s;
  flex-shrink: 0;
}

.collapse-button:hover {
  background: rgb(var(--v-theme-primary));
}

.collapse-button-left {
  border-right: 1px solid rgb(var(--v-theme-surface-variant));
}

.collapse-button-right {
  border-left: 1px solid rgb(var(--v-theme-surface-variant));
}

.resize-handle {
  width: 4px;
  background: rgb(var(--v-theme-surface-variant));
  cursor: col-resize;
  flex-shrink: 0;
  transition: background-color 0.2s;
  position: relative;
}

.resize-handle:hover {
  background: rgb(var(--v-theme-primary));
}

.resize-handle:active {
  background: rgb(var(--v-theme-primary));
}

.canvas-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  position: relative;
  min-width: 400px;
}

.workflow-canvas {
  flex: 1;
  background: rgb(var(--v-theme-background));
}

.category-header {
  font-weight: 600;
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: rgb(var(--v-theme-primary));
  padding-top: 12px;
  padding-bottom: 4px;
}

.node-palette-item {
  cursor: pointer;
  transition: background-color 0.2s;
  padding: 8px 16px;
  min-height: 40px;
}

.node-palette-item:hover {
  background: rgb(var(--v-theme-surface-variant));
}

.node-item-title {
  font-size: 13px;
}

/* Dark mode specific styles for Vue Flow */
:deep(.vue-flow__node) {
  color: rgb(var(--v-theme-on-surface));
}

:deep(.vue-flow__edge-path) {
  stroke: rgb(var(--v-theme-primary));
}

:deep(.vue-flow__edge.selected .vue-flow__edge-path) {
  stroke: rgb(var(--v-theme-secondary));
}

:deep(.vue-flow__controls) {
  background: rgb(var(--v-theme-surface)) !important;
  border: 1px solid rgb(var(--v-theme-surface-variant)) !important;
  box-shadow: 0 2px 8px rgba(0,0,0,0.2);
}

:deep(.vue-flow__controls button) {
  background: rgb(var(--v-theme-surface)) !important;
  border-bottom: 1px solid rgb(var(--v-theme-surface-variant)) !important;
  color: rgb(var(--v-theme-on-surface)) !important;
  fill: rgb(var(--v-theme-on-surface)) !important;
}

:deep(.vue-flow__controls button:hover) {
  background: rgb(var(--v-theme-surface-variant)) !important;
}

:deep(.vue-flow__controls button svg) {
  fill: rgb(var(--v-theme-on-surface)) !important;
}
</style>

