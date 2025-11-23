<template>
  <div class="delay-properties">
    <v-text-field
      v-model.number="localConfig.duration"
      label="Delay Duration (milliseconds)"
      hint="How long to delay execution (e.g., 1000 = 1 second)"
      persistent-hint
      variant="outlined"
      density="compact"
      type="number"
      min="0"
      class="mb-4"
      @update:model-value="emitUpdate"
    />

    <v-row class="mb-4">
      <v-col cols="12">
        <v-chip-group>
          <v-chip
            variant="outlined"
            size="small"
            @click="setDuration(1000)"
          >
            1 second
          </v-chip>
          <v-chip
            variant="outlined"
            size="small"
            @click="setDuration(5000)"
          >
            5 seconds
          </v-chip>
          <v-chip
            variant="outlined"
            size="small"
            @click="setDuration(10000)"
          >
            10 seconds
          </v-chip>
          <v-chip
            variant="outlined"
            size="small"
            @click="setDuration(30000)"
          >
            30 seconds
          </v-chip>
          <v-chip
            variant="outlined"
            size="small"
            @click="setDuration(60000)"
          >
            1 minute
          </v-chip>
        </v-chip-group>
      </v-col>
    </v-row>

    <v-alert
      type="info"
      variant="tonal"
      density="compact"
      class="mt-4"
    >
      <div class="text-body-2">
        <strong>Delay Node:</strong> Pauses workflow execution for a specified duration.
      </div>
      <div class="text-caption mt-2">
        Use this to introduce delays between actions or wait for external processes.
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

const localConfig = ref({
  duration: props.modelValue?.duration || 1000,
});

watch(
  () => props.modelValue,
  (newValue) => {
    if (newValue) {
      localConfig.value = {
        duration: newValue.duration || 1000,
      };
    }
  },
  { deep: true }
);

const emitUpdate = () => {
  emit("update:modelValue", localConfig.value);
  emit("update");
};

const setDuration = (ms: number) => {
  localConfig.value.duration = ms;
  emitUpdate();
};
</script>

<style scoped>
.delay-properties {
  padding: 0;
}
</style>
