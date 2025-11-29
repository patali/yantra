<template>
  <v-container
    fluid
    class="pa-6"
  >
    <v-row>
      <v-col cols="12">
        <div class="d-flex justify-space-between align-center mb-4">
          <h1 class="text-h4">
            Workflows
          </h1>
          <v-btn
            color="primary"
            prepend-icon="mdi-plus"
            @click="createNewWorkflow"
          >
            Create Workflow
          </v-btn>
        </div>
      </v-col>
    </v-row>

    <v-row v-if="!examplesDismissed">
      <v-col cols="12">
        <ExampleWorkflows @dismiss="dismissExamples" />
      </v-col>
    </v-row>

    <v-row>
      <v-col
        v-for="workflow in workflows"
        :key="workflow.id"
        cols="12"
        md="6"
        lg="4"
      >
        <v-card
          elevation="2"
          hover
        >
          <v-card-title>
            {{ workflow.name }}
            <v-chip
              size="small"
              class="ml-2"
              :color="workflow.isActive ? 'success' : 'grey'"
            >
              {{ workflow.isActive ? 'Active' : 'Inactive' }}
            </v-chip>
            <v-chip
              v-if="workflow.schedule"
              size="small"
              color="primary"
              class="ml-2"
            >
              <v-icon
                start
                size="small"
              >
                mdi-clock-outline
              </v-icon>
              Scheduled
            </v-chip>
          </v-card-title>

          <v-card-subtitle class="mt-2">
            {{ workflow.description || 'No description' }}
          </v-card-subtitle>

          <v-card-text>
            <div class="text-caption mb-2">
              <strong>Created by:</strong> {{ workflow.creator?.username }}
            </div>
            <div class="text-caption mb-2">
              <strong>Version:</strong> {{ workflow.currentVersion }}
            </div>
            <div
              v-if="workflow.schedule"
              class="text-caption mb-2"
            >
              <v-icon
                size="small"
                class="mr-1"
              >
                mdi-clock-outline
              </v-icon>
              <strong>Schedule:</strong> <code>{{ workflow.schedule }}</code> ({{ workflow.timezone || 'UTC' }})
            </div>
            <div class="text-caption">
              <strong>Executions:</strong> {{ workflow._count?.executions || 0 }}
            </div>
          </v-card-text>

          <v-card-actions>
            <v-btn
              color="success"
              variant="text"
              prepend-icon="mdi-play"
              :loading="executingWorkflows[workflow.id]"
              @click="executeWorkflow(workflow)"
            >
              Execute
            </v-btn>
            <v-btn
              color="primary"
              variant="text"
              prepend-icon="mdi-pencil"
              :to="`/workflows/${workflow.id}/edit`"
            >
              Edit
            </v-btn>
            <v-btn
              color="info"
              variant="text"
              prepend-icon="mdi-content-copy"
              :loading="duplicatingWorkflows[workflow.id]"
              @click="duplicateWorkflow(workflow)"
            >
              Duplicate
            </v-btn>
            <v-spacer />
            <v-btn
              color="error"
              variant="text"
              icon="mdi-delete"
              @click="confirmDelete(workflow)"
            />
          </v-card-actions>
        </v-card>
      </v-col>
    </v-row>

    <!-- Delete Confirmation Dialog -->
    <v-dialog
      v-model="deleteDialog"
      max-width="400"
    >
      <v-card>
        <v-card-title>Delete Workflow</v-card-title>
        <v-card-text>
          Are you sure you want to delete "{{ workflowToDelete?.name }}"?
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn @click="deleteDialog = false">
            Cancel
          </v-btn>
          <v-btn
            color="error"
            :loading="deleting"
            @click="deleteWorkflow"
          >
            Delete
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
  </v-container>
</template>

<script setup lang="ts">
import { ref, onMounted, reactive } from "vue";
import ExampleWorkflows from "@/components/ExampleWorkflows.vue";
import { useRouter } from "vue-router";
import api from "@/services/api";
import type { Workflow } from "@/types";

const router = useRouter();
const workflows = ref<Workflow[]>([]);
const deleteDialog = ref(false);
const deleting = ref(false);
const executingWorkflows = reactive<Record<string, boolean>>({});
const duplicatingWorkflows = reactive<Record<string, boolean>>({});
const workflowToDelete = ref<Workflow | null>(null);

const snackbar = ref(false);
const snackbarText = ref("");
const snackbarColor = ref("success");

const EXAMPLES_DISMISSED_KEY = "yantra_examples_dismissed";
const examplesDismissed = ref(localStorage.getItem(EXAMPLES_DISMISSED_KEY) === "true");

const fetchWorkflows = async () => {
  try {
    const response = await api.get("/workflows");
    workflows.value = response.data;
  } catch (_error) {
    showSnackbar("Failed to fetch workflows", "error");
  }
};

const createNewWorkflow = () => {
  router.push("/workflows/new");
};

const executeWorkflow = async (workflow: Workflow) => {
  try {
    executingWorkflows[workflow.id] = true;
    await api.post(`/workflows/${workflow.id}/execute`, {});
    showSnackbar("Workflow execution started", "success");
    fetchWorkflows();
  } catch (error: any) {
    showSnackbar(
      error.response?.data?.error || "Failed to execute workflow",
      "error"
    );
  } finally {
    executingWorkflows[workflow.id] = false;
  }
};

const duplicateWorkflow = async (workflow: Workflow) => {
  try {
    duplicatingWorkflows[workflow.id] = true;
    const response = await api.post(`/workflows/${workflow.id}/duplicate`, {});
    showSnackbar(`Workflow duplicated: ${response.data.name}`, "success");
    fetchWorkflows();
  } catch (error: any) {
    showSnackbar(
      error.response?.data?.error || "Failed to duplicate workflow",
      "error"
    );
  } finally {
    duplicatingWorkflows[workflow.id] = false;
  }
};

const confirmDelete = (workflow: Workflow) => {
  workflowToDelete.value = workflow;
  deleteDialog.value = true;
};

const deleteWorkflow = async () => {
  if (!workflowToDelete.value) return;

  try {
    deleting.value = true;
    await api.delete(`/workflows/${workflowToDelete.value.id}`);
    showSnackbar("Workflow deleted successfully", "success");
    deleteDialog.value = false;
    workflowToDelete.value = null;
    fetchWorkflows();
  } catch (_error) {
    showSnackbar("Failed to delete workflow", "error");
  } finally {
    deleting.value = false;
  }
};

const dismissExamples = () => {
  examplesDismissed.value = true;
  localStorage.setItem(EXAMPLES_DISMISSED_KEY, "true");
};

const showSnackbar = (text: string, color: string) => {
  snackbarText.value = text;
  snackbarColor.value = color;
  snackbar.value = true;
};

onMounted(() => {
  fetchWorkflows();
});
</script>

