<template>
  <div
    class="custom-node delay-node"
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
        color="green"
      >
        mdi-timer-sand
      </v-icon>
      <span class="node-label">{{ data.label || 'Delay' }}</span>
    </div>

    <div class="node-body">
      <div class="node-duration">
        {{ formatDuration(data.config?.duration || 1000) }}
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

interface Props {
  id: string;
  data: {
    label: string;
    config: Record<string, any>;
  };
  selected?: boolean;
}

defineProps<Props>();

const formatDuration = (ms: number): string => {
  if (ms < 1000) {
    return `${ms}ms`;
  } else if (ms < 60000) {
    return `${ms / 1000}s`;
  } else {
    return `${ms / 60000}m`;
  }
};
</script>

<style scoped>
.custom-node {
  background: rgb(var(--v-theme-surface));
  border: 2px solid #4caf50;
  border-radius: 8px;
  padding: 10px;
  min-width: 180px;
  box-shadow: 0 2px 4px rgba(0,0,0,0.1);
  color: rgb(var(--v-theme-on-surface));
}

.custom-node.selected {
  border-color: #388e3c;
  box-shadow: 0 4px 8px rgba(0,0,0,0.2);
}

.node-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
  font-weight: 600;
  color: #4caf50;
}

.node-label {
  font-size: 14px;
}

.node-body {
  font-size: 12px;
  color: rgb(var(--v-theme-on-surface-variant));
}

.delay-node {
  border-color: #4caf50;
}

.delay-node .node-header {
  color: #4caf50;
}

.node-duration {
  font-weight: bold;
  text-align: center;
  color: rgb(var(--v-theme-on-surface));
}

.handle {
  width: 10px;
  height: 10px;
  background: #4caf50;
  border: 2px solid rgb(var(--v-theme-surface));
}
</style>
