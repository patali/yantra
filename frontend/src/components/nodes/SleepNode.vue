<template>
  <div
    class="custom-node sleep-node"
    :class="{ selected: selected }"
  >
    <Handle
      type="target"
      :position="Position.Top"
      class="handle"
    />

    <div class="node-header">
      <v-icon
        size="small"
        color="indigo"
      >
        mdi-sleep
      </v-icon>
      <span class="node-label">{{ data.label || 'Sleep' }}</span>
    </div>

    <div class="node-body">
      <div class="node-sleep-info">
        {{ sleepInfoText }}
      </div>
    </div>

    <Handle
      type="source"
      :position="Position.Bottom"
      class="handle"
    />
  </div>
</template>

<script setup lang="ts">
import { Handle, Position } from "@vue-flow/core";
import { computed } from "vue";

interface Props {
  id: string;
  data: {
    label: string;
    config: Record<string, any>;
  };
  selected?: boolean;
}

const props = defineProps<Props>();

const sleepInfoText = computed(() => {
  const config = props.data.config;
  const mode = config?.mode;

  if (!mode) {
    return "Not configured";
  }

  if (mode === "relative") {
    const duration = config?.duration_value || 0;
    const unit = config?.duration_unit || "seconds";
    return `Sleep for ${duration} ${unit}`;
  } else if (mode === "absolute") {
    const targetDate = config?.target_date;
    if (!targetDate) {
      return "No target date set";
    }
    // Format the date for display
    try {
      const date = new Date(targetDate);
      return `Until ${date.toLocaleDateString()}`;
    } catch {
      return "Invalid date";
    }
  }

  return "Not configured";
});
</script>

<style scoped>
.custom-node {
  background: rgb(var(--v-theme-surface));
  border: 2px solid #5c6bc0;
  border-radius: 8px;
  padding: 10px;
  min-width: 180px;
  box-shadow: 0 2px 4px rgba(0,0,0,0.1);
  color: rgb(var(--v-theme-on-surface));
}

.custom-node.selected {
  border-color: #3949ab;
  box-shadow: 0 4px 8px rgba(0,0,0,0.2);
}

.node-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
  font-weight: 600;
  color: #5c6bc0;
}

.node-label {
  font-size: 14px;
}

.node-body {
  font-size: 12px;
  color: rgb(var(--v-theme-on-surface-variant));
}

.sleep-node {
  border-color: #5c6bc0;
}

.sleep-node .node-header {
  color: #5c6bc0;
}

.node-sleep-info {
  text-align: center;
  font-size: 11px;
  color: rgb(var(--v-theme-on-surface));
}

.handle {
  width: 10px;
  height: 10px;
  background: #5c6bc0;
  border: 2px solid rgb(var(--v-theme-surface));
}
</style>
