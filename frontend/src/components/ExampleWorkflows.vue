<template>
  <v-card
    color="info"
    variant="tonal"
    class="mb-6"
  >
    <v-card-title class="d-flex align-center">
      <v-icon
        start
        size="large"
      >
        mdi-lightbulb-on-outline
      </v-icon>
      <span>Example Workflows</span>
      <v-spacer />
      <v-btn
        icon="mdi-close"
        variant="text"
        size="small"
        @click="$emit('dismiss')"
      />
    </v-card-title>

    <v-card-text>
      <p class="mb-4">
        Get started quickly with these ready-to-use workflow examples. Click "Use Example" to add them to your workspace.
      </p>

      <v-progress-linear
        v-if="loading"
        indeterminate
        color="primary"
      />

      <v-row v-else>
        <v-col
          v-for="example in examples"
          :key="example.id"
          cols="12"
          md="6"
          lg="4"
        >
          <v-card
            elevation="2"
            hover
            class="h-100"
          >
            <v-card-title class="text-subtitle-1">
              {{ example.name }}
            </v-card-title>

            <v-card-subtitle>
              <v-chip
                size="small"
                color="primary"
                variant="flat"
              >
                {{ example.category }}
              </v-chip>
            </v-card-subtitle>

            <v-card-text>
              {{ example.description }}
            </v-card-text>

            <v-card-actions>
              <v-btn
                color="primary"
                variant="text"
                prepend-icon="mdi-content-copy"
                :loading="duplicatingExamples[example.id]"
                @click="duplicateExample(example)"
              >
                Use Example
              </v-btn>
            </v-card-actions>
          </v-card>
        </v-col>
      </v-row>

      <v-alert
        v-if="error"
        type="error"
        class="mt-4"
        closable
        @click:close="error = null"
      >
        {{ error }}
      </v-alert>
    </v-card-text>
  </v-card>
</template>

<script setup lang="ts">
import { ref, onMounted, reactive } from "vue";
import api from "@/services/api";
import { useRouter } from "vue-router";

interface ExampleWorkflow {
  id: string;
  name: string;
  description: string;
  category: string;
  definition: any;
}

defineEmits<{
  dismiss: [];
}>();

const router = useRouter();
const examples = ref<ExampleWorkflow[]>([]);
const loading = ref(false);
const error = ref<string | null>(null);
const duplicatingExamples = reactive<Record<string, boolean>>({});

const fetchExamples = async () => {
  try {
    loading.value = true;
    error.value = null;
    const response = await api.get("/examples/workflows");
    examples.value = response.data;
  } catch (err: any) {
    error.value = err.response?.data?.error || "Failed to load example workflows";
  } finally {
    loading.value = false;
  }
};

const duplicateExample = async (example: ExampleWorkflow) => {
  try {
    duplicatingExamples[example.id] = true;
    const response = await api.post(`/examples/workflows/${example.id}/duplicate`);

    // Navigate to the new workflow
    router.push(`/workflows/${response.data.id}/edit`);
  } catch (err: any) {
    error.value = err.response?.data?.error || "Failed to duplicate example workflow";
  } finally {
    duplicatingExamples[example.id] = false;
  }
};

onMounted(() => {
  fetchExamples();
});
</script>

<style scoped>
.h-100 {
  height: 100%;
}
</style>

